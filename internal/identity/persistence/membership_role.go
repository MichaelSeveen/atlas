package persistence

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var errRetryMemberRoleChange = errors.New("retry member role change transaction")

// UpdateMemberRole executes only a direct viewer/operator transition. Any
// administrator transition remains denied until the Operations approval owner
// can create and execute the typed ADR 0014 action.
func (store *OrganizationStore) UpdateMemberRole(
	ctx context.Context,
	command identity.UpdateOrganizationMemberRoleCommand,
) (identity.UpdateOrganizationMemberRoleResult, error) {
	for range 3 {
		result, err := store.updateMemberRoleOnce(ctx, command)
		if !errors.Is(err, errRetryMemberRoleChange) {
			return result, err
		}
	}
	return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrIdentityUnavailable
}

func (store *OrganizationStore) updateMemberRoleOnce(
	ctx context.Context,
	command identity.UpdateOrganizationMemberRoleCommand,
) (identity.UpdateOrganizationMemberRoleResult, error) {
	if command.Actor.SessionID.IsZero() || command.Actor.PrincipalID.IsZero() ||
		command.Actor.Population != identity.PopulationMerchant ||
		command.RoleChangeID.IsZero() || command.RoleChangeID.Prefix() != "mrc" ||
		command.OrganizationID.IsZero() || command.OrganizationID.Prefix() != "ten" ||
		command.MembershipID.IsZero() || command.MembershipID.Prefix() != "mem" ||
		command.ExpectedVersion < 1 || !persistedMerchantRole(command.Role) ||
		command.Now.IsZero() || command.Now.Location() != time.UTC {
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrInputInvalid
	}

	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var (
		status, population, tenantIDText      string
		authorizationVersion, rotationVersion int64
		idleExpiresAt, absoluteExpiresAt      time.Time
	)
	err = transaction.QueryRow(ctx, `
SELECT status, population, tenant_id, authorization_version, rotation_version,
       idle_expires_at, absolute_expires_at
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR SHARE`,
		command.Actor.SessionID.String(), command.Actor.PrincipalID.String(),
	).Scan(
		&status, &population, &tenantIDText, &authorizationVersion, &rotationVersion,
		&idleExpiresAt, &absoluteExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrIdentityUnavailable
	}
	if status != "active" || population != string(identity.PopulationMerchant) ||
		tenantID != command.Actor.TenantID ||
		authorizationVersion != command.Actor.AuthorizationVersion ||
		rotationVersion != command.Actor.RotationVersion ||
		!command.Now.Before(idleExpiresAt) || !command.Now.Before(absoluteExpiresAt) {
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrAuthenticationRequired
	}

	_, actorRole, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx, transaction, command.Actor.PrincipalID, identity.PopulationMerchant, tenantID,
	)
	if err != nil || currentAuthorizationVersion != authorizationVersion {
		if errors.Is(err, identity.ErrIdentityUnavailable) {
			return identity.UpdateOrganizationMemberRoleResult{}, err
		}
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrAuthenticationRequired
	}
	if command.OrganizationID != tenantID {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "tenant_concealed", identity.ErrMembershipNotFound,
		)
	}
	permissions, err := permissionsForRole(ctx, transaction, actorRole)
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, err
	}
	if !containsPermission(permissions, "organization.members.roles.update") {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "permission_denied", identity.ErrActionNotAuthorized,
		)
	}

	if replayed, storedRequest, replayErr := loadMemberRoleChangeReplay(ctx, transaction, command); replayErr == nil {
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return store.commitMemberRoleChangeDenial(
				ctx, transaction, command, "idempotency_conflict", identity.ErrIdempotencyConflict,
			)
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
		}
		replayed.Replay = true
		return replayed, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(replayErr)
	}

	member, err := scanOrganizationMember(transaction.QueryRow(ctx, `
SELECT membership_id, tenant_id, principal_id, role_id, status, version, created_at, revoked_at
FROM atlas_identity.memberships
WHERE tenant_id = $1 AND membership_id = $2 AND population = 'merchant'
FOR UPDATE`, command.OrganizationID.String(), command.MembershipID.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "membership_concealed", identity.ErrMembershipNotFound,
		)
	}
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}

	// The target lock serializes same-membership requests. Recheck replay after
	// acquiring it so a concurrent response-loss retry observes the first result.
	if replayed, storedRequest, replayErr := loadMemberRoleChangeReplay(ctx, transaction, command); replayErr == nil {
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return store.commitMemberRoleChangeDenial(
				ctx, transaction, command, "idempotency_conflict", identity.ErrIdempotencyConflict,
			)
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
		}
		replayed.Replay = true
		return replayed, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(replayErr)
	}

	if member.Status != "active" {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "membership_concealed", identity.ErrMembershipNotFound,
		)
	}
	if member.Version != command.ExpectedVersion {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "membership_version_mismatch",
			identity.ErrMembershipPreconditionFailed,
		)
	}
	if member.Role == command.Role {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "role_unchanged", identity.ErrValidationFailed,
		)
	}

	var delegable bool
	if err := transaction.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM atlas_identity.role_delegations
    WHERE role_id = $1 AND delegable_role_id = $2
)`, actorRole, command.Role).Scan(&delegable); err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}
	if !delegable {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "delegation_denied", identity.ErrActionNotAuthorized,
		)
	}
	if administratorMerchantRole(member.Role) || administratorMerchantRole(command.Role) {
		return store.commitMemberRoleChangeDenial(
			ctx, transaction, command, "approval_required", identity.ErrMembershipApprovalRequired,
		)
	}

	beforeRole := member.Role
	member, err = scanOrganizationMember(transaction.QueryRow(ctx, `
UPDATE atlas_identity.memberships
SET role_id = $4,
    authorization_version = authorization_version + 1,
    version = version + 1,
    updated_at = $5
WHERE tenant_id = $1
  AND membership_id = $2
  AND version = $3
  AND status = 'active'
RETURNING membership_id, tenant_id, principal_id, role_id, status, version, created_at, revoked_at`,
		command.OrganizationID.String(), command.MembershipID.String(),
		command.ExpectedVersion, command.Role, command.Now,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.UpdateOrganizationMemberRoleResult{}, errRetryMemberRoleChange
	}
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}

	if _, err := transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $3, version = version + 1
WHERE principal_id = $1
  AND tenant_id = $2
  AND status = 'active'`,
		member.PrincipalID.String(), command.OrganizationID.String(), command.Now,
	); err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}

	event := command.AuditEvent
	event.SafeBeforeReference = "membership-role:" + beforeRole
	event.SafeAfterReference = "membership-role:" + command.Role
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrIdentityUnavailable
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.membership_role_changes (
    role_change_id, tenant_id, membership_id, target_principal_id, population,
    actor_principal_id, actor_session_id, idempotency_key_sha256, request_sha256,
    expected_membership_version, before_role_id, after_role_id,
    result_membership_version, membership_created_at,
    decision_id, audit_event_id, created_at
) VALUES (
    $1, $2, $3, $4, 'merchant',
    $5, $6, $7, $8,
    $9, $10, $11,
    $12, $13,
    $14, $15, $16
)
ON CONFLICT (tenant_id, actor_principal_id, idempotency_key_sha256) DO NOTHING`,
		command.RoleChangeID.String(), command.OrganizationID.String(), command.MembershipID.String(),
		member.PrincipalID.String(), command.Actor.PrincipalID.String(), command.Actor.SessionID.String(),
		command.IdempotencyDigest[:], command.RequestDigest[:], command.ExpectedVersion,
		beforeRole, command.Role, member.Version, member.CreatedAt,
		event.DecisionID.String(), event.AuditEventID.String(), command.Now,
	)
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}
	if inserted.RowsAffected() != 1 {
		return identity.UpdateOrganizationMemberRoleResult{}, errRetryMemberRoleChange
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}
	return identity.UpdateOrganizationMemberRoleResult{
		Member: member, DecisionID: event.DecisionID,
	}, nil
}

func loadMemberRoleChangeReplay(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.UpdateOrganizationMemberRoleCommand,
) (identity.UpdateOrganizationMemberRoleResult, []byte, error) {
	var (
		membershipIDText, tenantIDText, principalIDText, role, decisionIDText string
		version                                                               int64
		createdAt                                                             time.Time
		storedRequest                                                         []byte
	)
	err := transaction.QueryRow(ctx, `
SELECT membership_id, tenant_id, target_principal_id, after_role_id,
       result_membership_version, membership_created_at, decision_id, request_sha256
FROM atlas_identity.membership_role_changes
WHERE tenant_id = $1
  AND actor_principal_id = $2
  AND idempotency_key_sha256 = $3`,
		command.OrganizationID.String(), command.Actor.PrincipalID.String(),
		command.IdempotencyDigest[:],
	).Scan(
		&membershipIDText, &tenantIDText, &principalIDText, &role,
		&version, &createdAt, &decisionIDText, &storedRequest,
	)
	if err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, nil, err
	}
	membershipID, err := identifier.Parse(membershipIDText)
	if err != nil || membershipID.Prefix() != "mem" {
		return identity.UpdateOrganizationMemberRoleResult{}, nil, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil || tenantID.Prefix() != "ten" {
		return identity.UpdateOrganizationMemberRoleResult{}, nil, identity.ErrIdentityUnavailable
	}
	principalID, err := identifier.Parse(principalIDText)
	if err != nil || principalID.Prefix() != "usr" {
		return identity.UpdateOrganizationMemberRoleResult{}, nil, identity.ErrIdentityUnavailable
	}
	decisionID, err := identifier.Parse(decisionIDText)
	if err != nil || decisionID.Prefix() != "dec" {
		return identity.UpdateOrganizationMemberRoleResult{}, nil, identity.ErrIdentityUnavailable
	}
	return identity.UpdateOrganizationMemberRoleResult{
		Member: identity.OrganizationMember{
			MembershipID: membershipID, OrganizationID: tenantID, PrincipalID: principalID,
			Role: role, Status: "active", Version: version, CreatedAt: createdAt,
		},
		DecisionID: decisionID,
	}, storedRequest, nil
}

func (store *OrganizationStore) commitMemberRoleChangeDenial(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.UpdateOrganizationMemberRoleCommand,
	reason string,
	denial error,
) (identity.UpdateOrganizationMemberRoleResult, error) {
	event := command.AuditEvent
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeBeforeReference = "membership-role:concealed"
	event.SafeAfterReference = "membership-role:unchanged"
	if reason == "tenant_concealed" || reason == "membership_concealed" {
		event.TargetID = "membership:concealed"
	}
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.UpdateOrganizationMemberRoleResult{}, memberRoleChangeDatabaseError(err)
	}
	return identity.UpdateOrganizationMemberRoleResult{DecisionID: event.DecisionID}, denial
}

func persistedMerchantRole(role string) bool {
	switch role {
	case "merchant_viewer", "merchant_operator", "merchant_admin", "merchant_security_admin":
		return true
	default:
		return false
	}
}

func administratorMerchantRole(role string) bool {
	return role == "merchant_admin" || role == "merchant_security_admin"
}

func memberRoleChangeDatabaseError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) &&
		(databaseError.Code == "40001" || databaseError.Code == "40P01" ||
			databaseError.Code == "55P03") {
		return errRetryMemberRoleChange
	}
	return identity.ErrIdentityUnavailable
}
