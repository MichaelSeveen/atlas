package persistence

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var errRetryMemberRevocation = errors.New("retry member revocation transaction")

// RevokeMember removes only direct viewer/operator authority. Administrator
// targets remain fail-closed until their step-up and last-administrator policy
// has an executable owner.
func (store *OrganizationStore) RevokeMember(
	ctx context.Context,
	command identity.RevokeOrganizationMemberCommand,
) (identity.RevokeOrganizationMemberResult, error) {
	var lastRetry error
	for range 3 {
		result, err := store.revokeMemberOnce(ctx, command)
		if !errors.Is(err, errRetryMemberRevocation) {
			return result, err
		}
		lastRetry = err
	}
	return identity.RevokeOrganizationMemberResult{}, errors.Join(identity.ErrIdentityUnavailable, lastRetry)
}

func (store *OrganizationStore) revokeMemberOnce(
	ctx context.Context,
	command identity.RevokeOrganizationMemberCommand,
) (identity.RevokeOrganizationMemberResult, error) {
	if command.Actor.SessionID.IsZero() || command.Actor.PrincipalID.IsZero() ||
		command.Actor.Population != identity.PopulationMerchant ||
		command.RevocationID.IsZero() || command.RevocationID.Prefix() != "mrv" ||
		command.OrganizationID.IsZero() || command.OrganizationID.Prefix() != "ten" ||
		command.MembershipID.IsZero() || command.MembershipID.Prefix() != "mem" ||
		command.ExpectedVersion < 1 || command.Now.IsZero() || command.Now.Location() != time.UTC {
		return identity.RevokeOrganizationMemberResult{}, identity.ErrInputInvalid
	}

	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, identity.ErrIdentityUnavailable
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
FOR SHARE`, command.Actor.SessionID.String(), command.Actor.PrincipalID.String()).Scan(
		&status, &population, &tenantIDText, &authorizationVersion, &rotationVersion,
		&idleExpiresAt, &absoluteExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.RevokeOrganizationMemberResult{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, identity.ErrIdentityUnavailable
	}
	if status != "active" || population != string(identity.PopulationMerchant) ||
		tenantID != command.Actor.TenantID ||
		authorizationVersion != command.Actor.AuthorizationVersion ||
		rotationVersion != command.Actor.RotationVersion ||
		!command.Now.Before(idleExpiresAt) || !command.Now.Before(absoluteExpiresAt) {
		return identity.RevokeOrganizationMemberResult{}, identity.ErrAuthenticationRequired
	}

	_, actorRole, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx, transaction, command.Actor.PrincipalID, identity.PopulationMerchant, tenantID,
	)
	if err != nil || currentAuthorizationVersion != authorizationVersion {
		if errors.Is(err, identity.ErrIdentityUnavailable) {
			return identity.RevokeOrganizationMemberResult{}, err
		}
		return identity.RevokeOrganizationMemberResult{}, identity.ErrAuthenticationRequired
	}
	if command.OrganizationID != tenantID {
		return store.commitMemberRevocationDenial(
			ctx, transaction, command, "tenant_concealed", identity.ErrMembershipNotFound,
		)
	}
	permissions, err := permissionsForRole(ctx, transaction, actorRole)
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, err
	}
	if !containsPermission(permissions, "organization.members.remove") {
		return store.commitMemberRevocationDenial(
			ctx, transaction, command, "permission_denied", identity.ErrActionNotAuthorized,
		)
	}

	if replayed, storedRequest, replayErr := loadMemberRevocationReplay(ctx, transaction, command); replayErr == nil {
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return store.commitMemberRevocationDenial(
				ctx, transaction, command, "idempotency_conflict", identity.ErrIdempotencyConflict,
			)
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
		}
		replayed.Replay = true
		return replayed, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(replayErr)
	}

	member, err := scanOrganizationMember(transaction.QueryRow(ctx, `
SELECT membership_id, tenant_id, principal_id, role_id, status, version, created_at, revoked_at
FROM atlas_identity.memberships
WHERE tenant_id = $1 AND membership_id = $2 AND population = 'merchant'
FOR UPDATE`, command.OrganizationID.String(), command.MembershipID.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return store.commitMemberRevocationDenial(
			ctx, transaction, command, "membership_concealed", identity.ErrMembershipNotFound,
		)
	}
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
	}

	// Recheck replay after acquiring the target lock so concurrent response-loss
	// retries observe the first committed revocation.
	if replayed, storedRequest, replayErr := loadMemberRevocationReplay(ctx, transaction, command); replayErr == nil {
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return store.commitMemberRevocationDenial(
				ctx, transaction, command, "idempotency_conflict", identity.ErrIdempotencyConflict,
			)
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
		}
		replayed.Replay = true
		return replayed, nil
	} else if !errors.Is(replayErr, pgx.ErrNoRows) {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(replayErr)
	}

	if member.Status != "active" {
		return store.commitMemberRevocationDenial(
			ctx, transaction, command, "membership_concealed", identity.ErrMembershipNotFound,
		)
	}
	if member.Version != command.ExpectedVersion {
		return store.commitMemberRevocationDenial(
			ctx, transaction, command, "membership_version_mismatch",
			identity.ErrMembershipPreconditionFailed,
		)
	}
	if administratorMerchantRole(member.Role) {
		return store.commitMemberRevocationDenial(
			ctx, transaction, command, "administrator_removal_unavailable",
			identity.ErrMembershipAdministratorRemovalUnavailable,
		)
	}
	if member.Role != "merchant_viewer" && member.Role != "merchant_operator" {
		return identity.RevokeOrganizationMemberResult{}, identity.ErrIdentityUnavailable
	}

	member, err = scanOrganizationMember(transaction.QueryRow(ctx, `
UPDATE atlas_identity.memberships
SET status = 'revoked',
    authorization_version = authorization_version + 1,
    version = version + 1,
    revoked_at = $4,
    updated_at = $4
WHERE tenant_id = $1
  AND membership_id = $2
  AND version = $3
  AND status = 'active'
RETURNING membership_id, tenant_id, principal_id, role_id, status, version, created_at, revoked_at`,
		command.OrganizationID.String(), command.MembershipID.String(),
		command.ExpectedVersion, command.Now,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.RevokeOrganizationMemberResult{}, errRetryMemberRevocation
	}
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
	}

	if _, err := transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $3, version = version + 1
WHERE principal_id = $1
  AND tenant_id = $2
  AND status = 'active'`, member.PrincipalID.String(), command.OrganizationID.String(), command.Now); err != nil {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
	}

	event := command.AuditEvent
	event.SafeBeforeReference = "membership-status:active;role=" + member.Role
	event.SafeAfterReference = "membership-status:revoked"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.RevokeOrganizationMemberResult{}, fmt.Errorf(
			"record membership revocation audit: %w", identity.ErrIdentityUnavailable,
		)
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.membership_revocations (
    membership_revocation_id, tenant_id, membership_id, target_principal_id, population,
    actor_principal_id, actor_session_id, idempotency_key_sha256, request_sha256,
    expected_membership_version, target_role_id, result_membership_version,
    membership_created_at, revoked_at, decision_id, audit_event_id, created_at
) VALUES (
    $1, $2, $3, $4, 'merchant',
    $5, $6, $7, $8,
    $9, $10, $11,
    $12, $13, $14, $15, $16
)
ON CONFLICT (tenant_id, actor_principal_id, idempotency_key_sha256) DO NOTHING`,
		command.RevocationID.String(), command.OrganizationID.String(), command.MembershipID.String(),
		member.PrincipalID.String(), command.Actor.PrincipalID.String(), command.Actor.SessionID.String(),
		command.IdempotencyDigest[:], command.RequestDigest[:], command.ExpectedVersion,
		member.Role, member.Version, member.CreatedAt, command.Now,
		event.DecisionID.String(), event.AuditEventID.String(), command.Now,
	)
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
	}
	if inserted.RowsAffected() != 1 {
		return identity.RevokeOrganizationMemberResult{}, errRetryMemberRevocation
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
	}
	return identity.RevokeOrganizationMemberResult{Member: member, DecisionID: event.DecisionID}, nil
}

func loadMemberRevocationReplay(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.RevokeOrganizationMemberCommand,
) (identity.RevokeOrganizationMemberResult, []byte, error) {
	var (
		membershipIDText, tenantIDText, principalIDText, role, decisionIDText string
		version                                                               int64
		createdAt, revokedAt                                                  time.Time
		storedRequest                                                         []byte
	)
	err := transaction.QueryRow(ctx, `
SELECT membership_id, tenant_id, target_principal_id, target_role_id,
       result_membership_version, membership_created_at, revoked_at,
       decision_id, request_sha256
FROM atlas_identity.membership_revocations
WHERE tenant_id = $1
  AND actor_principal_id = $2
  AND idempotency_key_sha256 = $3`, command.OrganizationID.String(), command.Actor.PrincipalID.String(), command.IdempotencyDigest[:]).Scan(
		&membershipIDText, &tenantIDText, &principalIDText, &role,
		&version, &createdAt, &revokedAt, &decisionIDText, &storedRequest,
	)
	if err != nil {
		return identity.RevokeOrganizationMemberResult{}, nil, err
	}
	membershipID, err := identifier.Parse(membershipIDText)
	if err != nil || membershipID.Prefix() != "mem" {
		return identity.RevokeOrganizationMemberResult{}, nil, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil || tenantID.Prefix() != "ten" {
		return identity.RevokeOrganizationMemberResult{}, nil, identity.ErrIdentityUnavailable
	}
	principalID, err := identifier.Parse(principalIDText)
	if err != nil || principalID.Prefix() != "usr" {
		return identity.RevokeOrganizationMemberResult{}, nil, identity.ErrIdentityUnavailable
	}
	decisionID, err := identifier.Parse(decisionIDText)
	if err != nil || decisionID.Prefix() != "dec" {
		return identity.RevokeOrganizationMemberResult{}, nil, identity.ErrIdentityUnavailable
	}
	return identity.RevokeOrganizationMemberResult{
		Member: identity.OrganizationMember{
			MembershipID: membershipID, OrganizationID: tenantID, PrincipalID: principalID,
			Role: role, Status: "revoked", Version: version, CreatedAt: createdAt, RevokedAt: &revokedAt,
		},
		DecisionID: decisionID,
	}, storedRequest, nil
}

func (store *OrganizationStore) commitMemberRevocationDenial(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.RevokeOrganizationMemberCommand,
	reason string,
	denial error,
) (identity.RevokeOrganizationMemberResult, error) {
	event := command.AuditEvent
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeBeforeReference = "membership-status:concealed"
	event.SafeAfterReference = "membership-status:unchanged"
	if reason == "tenant_concealed" || reason == "membership_concealed" {
		event.TargetID = "membership:concealed"
	}
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.RevokeOrganizationMemberResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.RevokeOrganizationMemberResult{}, memberRevocationDatabaseError(err)
	}
	return identity.RevokeOrganizationMemberResult{DecisionID: event.DecisionID}, denial
}

func memberRevocationDatabaseError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) {
		if databaseError.Code == "40001" || databaseError.Code == "40P01" {
			return fmt.Errorf("%w: sqlstate %s", errRetryMemberRevocation, databaseError.Code)
		}
		return fmt.Errorf(
			"membership revocation database sqlstate %s: %w",
			databaseError.Code, identity.ErrIdentityUnavailable,
		)
	}
	return fmt.Errorf(
		"membership revocation database error type %T: %w",
		err, identity.ErrIdentityUnavailable,
	)
}
