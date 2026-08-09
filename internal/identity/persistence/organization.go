package persistence

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const defaultOrganizationMemberPageSize = 25

var errRetryInvitationAcceptance = errors.New("retry invitation acceptance transaction")

const listOrganizationsSQL = `
SELECT organization.tenant_id, organization.display_name, membership.role_id, membership.version
FROM atlas_identity.memberships AS membership
JOIN atlas_identity.organizations AS organization
  ON organization.tenant_id = membership.tenant_id
WHERE membership.principal_id = $1
  AND membership.population = 'merchant'
  AND membership.status = 'active'
  AND organization.organization_type = 'merchant'
  AND organization.status = 'active'
ORDER BY organization.tenant_id
LIMIT 100`

// OrganizationStore persists merchant organization context within Identity's PostgreSQL boundary.
type OrganizationStore struct {
	pool          *pgxpool.Pool
	recorder      audit.Recorder
	authorization *identity.AuthorizationPolicy
}

// NewOrganizationStore constructs the tenant-context store.
func NewOrganizationStore(pool *pgxpool.Pool, recorder audit.Recorder) (*OrganizationStore, error) {
	if pool == nil || recorder == nil {
		return nil, errors.New("organization store dependencies are incomplete")
	}
	return &OrganizationStore{
		pool: pool, recorder: recorder, authorization: identity.DefaultAuthorizationPolicy(),
	}, nil
}

// ListOrganizations returns only active merchant memberships owned by the principal.
func (store *OrganizationStore) ListOrganizations(
	ctx context.Context,
	actor identity.Session,
) ([]identity.Organization, error) {
	if actor.PrincipalID.IsZero() || actor.Population != identity.PopulationMerchant {
		return nil, identity.ErrActionNotAuthorized
	}
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var status, population, tenantIDText string
	var authorizationVersion, rotationVersion int64
	err = transaction.QueryRow(ctx, `
SELECT status, population, tenant_id, authorization_version, rotation_version
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR SHARE`,
		actor.SessionID.String(),
		actor.PrincipalID.String(),
	).Scan(&status, &population, &tenantIDText, &authorizationVersion, &rotationVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	if status != "active" ||
		population != string(identity.PopulationMerchant) ||
		tenantID != actor.TenantID ||
		authorizationVersion != actor.AuthorizationVersion ||
		rotationVersion != actor.RotationVersion {
		return nil, identity.ErrAuthenticationRequired
	}
	_, currentRole, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx,
		transaction,
		actor.PrincipalID,
		identity.PopulationMerchant,
		tenantID,
	)
	if err != nil || currentAuthorizationVersion != authorizationVersion {
		if errors.Is(err, identity.ErrIdentityUnavailable) {
			return nil, err
		}
		return nil, identity.ErrAuthenticationRequired
	}
	permissions, err := permissionsForRole(ctx, transaction, currentRole)
	if err != nil {
		return nil, err
	}
	if !containsPermission(permissions, "organization.list") {
		return nil, identity.ErrActionNotAuthorized
	}

	rows, err := transaction.Query(ctx, listOrganizationsSQL, actor.PrincipalID.String())
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	defer rows.Close()
	organizations := make([]identity.Organization, 0)
	for rows.Next() {
		var organizationIDText string
		var organization identity.Organization
		if err := rows.Scan(
			&organizationIDText,
			&organization.DisplayName,
			&organization.PrincipalRole,
			&organization.MembershipVersion,
		); err != nil {
			return nil, identity.ErrIdentityUnavailable
		}
		organization.OrganizationID, err = identifier.Parse(organizationIDText)
		if err != nil || organization.OrganizationID.Prefix() != "ten" {
			return nil, identity.ErrIdentityUnavailable
		}
		organizations = append(organizations, organization)
	}
	if rows.Err() != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	return organizations, nil
}

// ListMembers authorizes before cursor evaluation and returns no count metadata.
func (store *OrganizationStore) ListMembers(
	ctx context.Context,
	command identity.ListOrganizationMembersCommand,
) (identity.ListOrganizationMembersResult, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var (
		status, population, tenantIDText      string
		sessionAssurance                      string
		authorizationVersion, rotationVersion int64
		idleExpiresAt, absoluteExpiresAt      time.Time
	)
	err = transaction.QueryRow(ctx, `
SELECT status, population, tenant_id, assurance, authorization_version, rotation_version,
       idle_expires_at, absolute_expires_at
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR SHARE`,
		command.Actor.SessionID.String(),
		command.Actor.PrincipalID.String(),
	).Scan(
		&status, &population, &tenantIDText, &sessionAssurance, &authorizationVersion, &rotationVersion,
		&idleExpiresAt, &absoluteExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.ListOrganizationMembersResult{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}
	if status != "active" || population != string(identity.PopulationMerchant) ||
		tenantID != command.Actor.TenantID || sessionAssurance != string(command.Actor.Assurance) ||
		authorizationVersion != command.Actor.AuthorizationVersion ||
		rotationVersion != command.Actor.RotationVersion || !command.Now.Before(idleExpiresAt) ||
		!command.Now.Before(absoluteExpiresAt) {
		return identity.ListOrganizationMembersResult{}, identity.ErrAuthenticationRequired
	}
	_, actorRole, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx, transaction, command.Actor.PrincipalID, identity.PopulationMerchant, tenantID,
	)
	if err != nil || currentAuthorizationVersion != authorizationVersion {
		if errors.Is(err, identity.ErrIdentityUnavailable) {
			return identity.ListOrganizationMembersResult{}, err
		}
		return identity.ListOrganizationMembersResult{}, identity.ErrAuthenticationRequired
	}
	permissions, err := permissionsForRole(ctx, transaction, actorRole)
	if err != nil {
		return identity.ListOrganizationMembersResult{}, err
	}
	resourceStatus := "active"
	resourceVersion := int64(1)
	if command.OrganizationID == tenantID {
		err = transaction.QueryRow(ctx, `
SELECT status, version
FROM atlas_identity.organizations
WHERE tenant_id = $1
FOR SHARE`, command.OrganizationID.String()).Scan(&resourceStatus, &resourceVersion)
		if err != nil {
			return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
		}
	}
	decision := store.authorization.Evaluate(identity.AuthorizationFacts{
		PrincipalID: command.Actor.PrincipalID, PrincipalType: command.Actor.PrincipalType,
		Population: command.Actor.Population, TenantID: tenantID,
		ResourceTenantID: command.OrganizationID, Role: actorRole,
		CurrentPermissions: permissions, Action: "organization.members.list",
		Resource: "organization_member", Field: "email_hint", Purpose: command.Purpose,
		Assurance:                   command.Actor.Assurance,
		SessionAuthorizationVersion: command.Actor.AuthorizationVersion,
		CurrentAuthorizationVersion: currentAuthorizationVersion,
		ResourceVersion:             resourceVersion, ResourceStatus: resourceStatus,
	})
	if decision.Effect == identity.AuthorizationConceal {
		return store.commitMemberListDenial(
			ctx, transaction, command, decision.Reason, identity.ErrOrganizationNotFound,
		)
	}
	if decision.Effect != identity.AuthorizationAllow {
		return store.commitMemberListDenial(
			ctx, transaction, command, decision.Reason, identity.ErrActionNotAuthorized,
		)
	}
	revealEmailHint := decision.FieldAccess == identity.FieldAccessReveal

	pageSize, err := organizationMemberPageSize(command.PageSize, command.PageSizeProvided)
	if err != nil {
		return identity.ListOrganizationMembersResult{DecisionID: command.AuditEvent.DecisionID}, err
	}
	afterPrincipal, err := decodeOrganizationMemberCursor(
		command.Cursor, command.CursorProvided, command.OrganizationID,
	)
	if err != nil {
		return identity.ListOrganizationMembersResult{DecisionID: command.AuditEvent.DecisionID}, err
	}
	query := `
SELECT membership_id, tenant_id, principal_id, NULL::text AS email_hint,
       role_id, status, version, created_at, revoked_at
FROM atlas_identity.memberships
WHERE tenant_id = $1
  AND population = 'merchant'
ORDER BY principal_id
LIMIT $2`
	arguments := []any{command.OrganizationID.String(), pageSize + 1}
	if revealEmailHint {
		query = `
SELECT membership_id, tenant_id, principal_id, email_hint,
       role_id, status, version, created_at, revoked_at
FROM atlas_identity.memberships
WHERE tenant_id = $1
  AND population = 'merchant'
ORDER BY principal_id
LIMIT $2`
	}
	if afterPrincipal != "" && !revealEmailHint {
		query = `
SELECT membership_id, tenant_id, principal_id, NULL::text AS email_hint,
       role_id, status, version, created_at, revoked_at
FROM atlas_identity.memberships
WHERE tenant_id = $1
  AND population = 'merchant'
  AND principal_id > $2
ORDER BY principal_id
LIMIT $3`
		arguments = []any{command.OrganizationID.String(), afterPrincipal, pageSize + 1}
	}
	if afterPrincipal != "" && revealEmailHint {
		query = `
SELECT membership_id, tenant_id, principal_id, email_hint,
       role_id, status, version, created_at, revoked_at
FROM atlas_identity.memberships
WHERE tenant_id = $1
  AND population = 'merchant'
  AND principal_id > $2
ORDER BY principal_id
LIMIT $3`
		arguments = []any{command.OrganizationID.String(), afterPrincipal, pageSize + 1}
	}
	rows, err := transaction.Query(ctx, query, arguments...)
	if err != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}
	defer rows.Close()
	members := make([]identity.OrganizationMember, 0, pageSize+1)
	for rows.Next() {
		var member identity.OrganizationMember
		var membershipIDText, organizationIDText, principalIDText string
		if err := rows.Scan(
			&membershipIDText, &organizationIDText, &principalIDText, &member.EmailHint, &member.Role,
			&member.Status, &member.Version, &member.CreatedAt, &member.RevokedAt,
		); err != nil {
			return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
		}
		member.MembershipID, err = identifier.Parse(membershipIDText)
		if err != nil || member.MembershipID.Prefix() != "mem" {
			return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
		}
		member.OrganizationID, err = identifier.Parse(organizationIDText)
		if err != nil || member.OrganizationID != command.OrganizationID {
			return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
		}
		member.PrincipalID, err = identifier.Parse(principalIDText)
		if err != nil || member.PrincipalID.Prefix() != "usr" {
			return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
		}
		members = append(members, member)
	}
	if rows.Err() != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}

	page := identity.OrganizationMemberPage{Members: members}
	if len(page.Members) > pageSize {
		page.HasMore = true
		page.Members = page.Members[:pageSize]
		page.NextCursor = encodeOrganizationMemberCursor(
			command.OrganizationID,
			page.Members[len(page.Members)-1].PrincipalID,
		)
	}
	if revealEmailHint {
		event := command.AuditEvent
		event.Action = "identity.organization.members.sensitive.list"
		event.ReasonCode = "organization_members_sensitive_listed"
		event.SafeAfterReference = "organization-members:masked-email-hint"
		if err := store.recorder.Record(ctx, transaction, event); err != nil {
			return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}
	return identity.ListOrganizationMembersResult{
		Page: page, DecisionID: command.AuditEvent.DecisionID,
	}, nil
}

func (store *OrganizationStore) commitMemberListDenial(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.ListOrganizationMembersCommand,
	reason string,
	denial error,
) (identity.ListOrganizationMembersResult, error) {
	event := command.AuditEvent
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeBeforeReference = "organization-members:concealed"
	event.SafeAfterReference = "organization-members:concealed"
	if reason == "tenant_concealed" {
		event.TargetID = "organization:concealed"
	}
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.ListOrganizationMembersResult{}, identity.ErrIdentityUnavailable
	}
	return identity.ListOrganizationMembersResult{DecisionID: event.DecisionID}, denial
}

func organizationMemberPageSize(value string, provided bool) (int, error) {
	if !provided {
		return defaultOrganizationMemberPageSize, nil
	}
	if value == "" || value != strings.TrimSpace(value) {
		return 0, identity.ErrInputInvalid
	}
	parsed, err := strconv.ParseUint(value, 10, 7)
	if err != nil || parsed < 1 || parsed > 100 {
		return 0, identity.ErrInputInvalid
	}
	return int(parsed), nil
}

func encodeOrganizationMemberCursor(organizationID, principalID identifier.ID) string {
	payload := "v1\n" + organizationID.String() + "\n" + principalID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeOrganizationMemberCursor(
	value string,
	provided bool,
	organizationID identifier.ID,
) (string, error) {
	if !provided {
		return "", nil
	}
	if len(value) < 16 || len(value) > 2048 {
		return "", identity.ErrInputInvalid
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return "", identity.ErrInputInvalid
	}
	parts := strings.Split(string(decoded), "\n")
	if len(parts) != 3 || parts[0] != "v1" {
		return "", identity.ErrInputInvalid
	}
	cursorOrganizationID, err := identifier.Parse(parts[1])
	if err != nil || cursorOrganizationID != organizationID {
		return "", identity.ErrInputInvalid
	}
	principalID, err := identifier.Parse(parts[2])
	if err != nil || principalID.Prefix() != "usr" {
		return "", identity.ErrInputInvalid
	}
	return principalID.String(), nil
}

// SwitchOrganization rotates the active tenant with zero grace for the old session.
func (store *OrganizationStore) SwitchOrganization(
	ctx context.Context,
	command identity.SwitchOrganizationCommand,
) (identity.Session, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var (
		status, population, currentTenantText, assurance, principalType, displayName string
		sessionAuthorizationVersion, rotationVersion                                 int64
		absoluteExpiresAt                                                            time.Time
		clientLabel                                                                  *string
		verifiedEmailDigest                                                          []byte
	)
	err = transaction.QueryRow(ctx, `
SELECT session.status, session.population, session.tenant_id, session.assurance,
       session.authorization_version, session.rotation_version,
       session.absolute_expires_at, session.client_label, session.verified_email_sha256,
       principal.principal_type, principal.display_name
FROM atlas_identity.sessions AS session
JOIN atlas_identity.principals AS principal
  ON principal.principal_id = session.principal_id
WHERE session.session_id = $1
  AND session.principal_id = $2
FOR UPDATE OF session`,
		command.Actor.SessionID.String(),
		command.Actor.PrincipalID.String(),
	).Scan(
		&status,
		&population,
		&currentTenantText,
		&assurance,
		&sessionAuthorizationVersion,
		&rotationVersion,
		&absoluteExpiresAt,
		&clientLabel,
		&verifiedEmailDigest,
		&principalType,
		&displayName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.Session{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	currentTenant, err := identifier.Parse(currentTenantText)
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if status != "active" ||
		population != string(identity.PopulationMerchant) ||
		currentTenant != command.Actor.TenantID ||
		sessionAuthorizationVersion != command.Actor.AuthorizationVersion ||
		rotationVersion != command.Actor.RotationVersion ||
		!command.Now.Before(absoluteExpiresAt) {
		return identity.Session{}, identity.ErrAuthenticationRequired
	}

	_, _, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx,
		transaction,
		command.Actor.PrincipalID,
		identity.PopulationMerchant,
		currentTenant,
	)
	if err != nil || currentAuthorizationVersion != sessionAuthorizationVersion {
		if errors.Is(err, identity.ErrIdentityUnavailable) {
			return identity.Session{}, err
		}
		return identity.Session{}, identity.ErrAuthenticationRequired
	}
	targetTenant, targetRole, targetAuthorizationVersion, err := authorityForPrincipal(
		ctx,
		transaction,
		command.Actor.PrincipalID,
		identity.PopulationMerchant,
		command.OrganizationID,
	)
	if errors.Is(err, identity.ErrAuthenticationRequired) {
		return identity.Session{}, identity.ErrOrganizationNotFound
	}
	if err != nil {
		return identity.Session{}, err
	}
	permissions, err := permissionsForRole(ctx, transaction, targetRole)
	if err != nil {
		return identity.Session{}, err
	}
	if !containsPermission(permissions, "organization.active.switch") {
		return identity.Session{}, identity.ErrActionNotAuthorized
	}

	_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $2, version = version + 1
WHERE session_id = $1 AND status = 'active'`,
		command.Actor.SessionID.String(),
		command.Now,
	)
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	var clientLabelValue any
	if clientLabel != nil {
		clientLabelValue = *clientLabel
	}
	_, err = transaction.Exec(
		ctx,
		insertSessionSQL,
		command.NewSessionID.String(),
		command.Actor.PrincipalID.String(),
		string(identity.PopulationMerchant),
		targetTenant.String(),
		nil,
		command.VerifierDigest[:],
		assurance,
		targetAuthorizationVersion,
		rotationVersion+1,
		command.Now,
		command.IdleExpiresAt,
		absoluteExpiresAt,
		clientLabelValue,
		nil,
		nil,
		nil,
		verifiedEmailDigest,
	)
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if err := store.recorder.Record(ctx, transaction, command.AuditEvent); err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}

	return identity.Session{
		SessionID: command.NewSessionID, PrincipalID: command.Actor.PrincipalID,
		PrincipalType: principalType, DisplayName: displayName,
		Population: identity.PopulationMerchant, TenantID: targetTenant,
		Assurance: identity.Assurance(assurance), AuthorizationVersion: targetAuthorizationVersion,
		RotationVersion: rotationVersion + 1, CreatedAt: command.Now, LastSeenAt: command.Now,
		IdleExpiresAt: command.IdleExpiresAt, AbsoluteExpiresAt: absoluteExpiresAt,
		ClientLabel: valueOrEmpty(clientLabel), Permissions: permissions,
		VerifiedEmailDigest: command.Actor.VerifiedEmailDigest,
	}, nil
}

// CreateInvitation authorizes and records invitation issuance in the same transaction as Audit.
func (store *OrganizationStore) CreateInvitation(
	ctx context.Context,
	command identity.CreateInvitationCommand,
) (identity.CreateInvitationStoreResult, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var (
		status, population, tenantIDText, assurance string
		authorizationVersion, rotationVersion       int64
		idleExpiresAt, absoluteExpiresAt            time.Time
		stepUpAction                                *string
		stepUpVerifiedAt                            *time.Time
	)
	err = transaction.QueryRow(ctx, `
SELECT status, population, tenant_id, assurance, authorization_version, rotation_version,
       idle_expires_at, absolute_expires_at, step_up_action, step_up_verified_at
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR SHARE`,
		command.Actor.SessionID.String(),
		command.Actor.PrincipalID.String(),
	).Scan(
		&status, &population, &tenantIDText, &assurance,
		&authorizationVersion, &rotationVersion, &idleExpiresAt, &absoluteExpiresAt,
		&stepUpAction, &stepUpVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.CreateInvitationStoreResult{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	if status != "active" || population != string(identity.PopulationMerchant) ||
		tenantID != command.Actor.TenantID || authorizationVersion != command.Actor.AuthorizationVersion ||
		rotationVersion != command.Actor.RotationVersion || !command.Now.Before(idleExpiresAt) ||
		!command.Now.Before(absoluteExpiresAt) {
		return identity.CreateInvitationStoreResult{}, identity.ErrAuthenticationRequired
	}
	_, actorRole, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx, transaction, command.Actor.PrincipalID, identity.PopulationMerchant, tenantID,
	)
	if err != nil || currentAuthorizationVersion != authorizationVersion {
		if errors.Is(err, identity.ErrIdentityUnavailable) {
			return identity.CreateInvitationStoreResult{}, err
		}
		return identity.CreateInvitationStoreResult{}, identity.ErrAuthenticationRequired
	}
	if command.OrganizationID != tenantID {
		return store.commitInvitationDenial(
			ctx, transaction, command, "tenant_concealed", identity.ErrOrganizationNotFound,
		)
	}
	permissions, err := permissionsForRole(ctx, transaction, actorRole)
	if err != nil {
		return identity.CreateInvitationStoreResult{}, err
	}
	if !containsPermission(permissions, "organization.invitations.create") {
		return store.commitInvitationDenial(
			ctx, transaction, command, "permission_denied", identity.ErrActionNotAuthorized,
		)
	}
	var delegable bool
	if err := transaction.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM atlas_identity.role_delegations
    WHERE role_id = $1 AND delegable_role_id = $2
)`, actorRole, command.Role).Scan(&delegable); err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	if !delegable {
		return store.commitInvitationDenial(
			ctx, transaction, command, "delegation_denied", identity.ErrActionNotAuthorized,
		)
	}
	replayed, storedRequest, err := loadInvitationReplay(ctx, transaction, command)
	if err == nil {
		result := identity.CreateInvitationStoreResult{
			Invitation: replayed.Invitation, DecisionID: replayed.DecisionID, Replay: true,
		}
		if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
			return result, identity.ErrIdempotencyConflict
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
		}
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	if command.Role == "merchant_admin" {
		stepUpSatisfied := assurance == string(identity.AssurancePhishingResistant) &&
			stepUpAction != nil && *stepUpAction == "identity.organization.invitation.create_admin" &&
			stepUpVerifiedAt != nil &&
			!stepUpVerifiedAt.Before(command.Now.Add(-5*time.Minute)) &&
			!stepUpVerifiedAt.After(command.Now.Add(time.Minute))
		if !stepUpSatisfied {
			return store.commitInvitationDenial(
				ctx, transaction, command, "step_up_required", identity.ErrStepUpRequired,
			)
		}
	}

	inserted, err := scanOrganizationInvitation(transaction.QueryRow(ctx, `
INSERT INTO atlas_identity.organization_invitations (
    invitation_id, tenant_id, invited_by_principal_id, invited_by_session_id,
    email_sha256, email_hint, role_id, population, token_sha256,
    idempotency_key_sha256, request_sha256, creation_decision_id,
    status, expires_at, created_at, version
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, 'merchant', $8,
    $9, $10, $11,
    'pending', $12, $13, 1
)
ON CONFLICT (tenant_id, invited_by_principal_id, idempotency_key_sha256) DO NOTHING
RETURNING invitation_id, tenant_id, email_hint, role_id, status, expires_at, created_at,
          creation_decision_id`,
		command.InvitationID.String(), command.OrganizationID.String(),
		command.Actor.PrincipalID.String(), command.Actor.SessionID.String(),
		command.EmailDigest[:], command.EmailHint, command.Role, command.TokenDigest[:],
		command.IdempotencyDigest[:], command.RequestDigest[:],
		command.AuditEvent.DecisionID.String(), command.ExpiresAt, command.Now,
	))
	if err == nil {
		if err := store.recorder.Record(ctx, transaction, command.AuditEvent); err != nil {
			return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
		}
		return identity.CreateInvitationStoreResult{
			Invitation: inserted.Invitation, DecisionID: inserted.DecisionID,
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}

	replayed, storedRequest, err = loadInvitationReplay(ctx, transaction, command)
	if err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	result := identity.CreateInvitationStoreResult{
		Invitation: replayed.Invitation, DecisionID: replayed.DecisionID, Replay: true,
	}
	if !bytes.Equal(storedRequest, command.RequestDigest[:]) {
		return result, identity.ErrIdempotencyConflict
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	return result, nil
}

func loadInvitationReplay(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.CreateInvitationCommand,
) (scannedInvitation, []byte, error) {
	var storedRequest []byte
	replayed, err := scanOrganizationInvitation(transaction.QueryRow(ctx, `
SELECT invitation_id, tenant_id, email_hint, role_id, status, expires_at, created_at,
       creation_decision_id, request_sha256
FROM atlas_identity.organization_invitations
WHERE tenant_id = $1
  AND invited_by_principal_id = $2
  AND idempotency_key_sha256 = $3
FOR SHARE`,
		command.OrganizationID.String(), command.Actor.PrincipalID.String(),
		command.IdempotencyDigest[:],
	), &storedRequest)
	return replayed, storedRequest, err
}

// AcceptInvitation consumes a recipient-bound invitation, creates the
// membership, rotates the bootstrap session, and records Audit atomically.
func (store *OrganizationStore) AcceptInvitation(
	ctx context.Context,
	command identity.AcceptInvitationCommand,
) (identity.AcceptInvitationStoreResult, error) {
	for range 3 {
		result, err := store.acceptInvitationOnce(ctx, command)
		if !errors.Is(err, errRetryInvitationAcceptance) {
			return result, err
		}
	}
	return identity.AcceptInvitationStoreResult{}, identity.ErrIdentityUnavailable
}

func (store *OrganizationStore) acceptInvitationOnce(
	ctx context.Context,
	command identity.AcceptInvitationCommand,
) (identity.AcceptInvitationStoreResult, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var (
		sessionStatus, population, assurance, principalType, displayName            string
		globalScope, sessionInvitationText, clientLabel, tenantText                 *string
		sessionAuthorizationVersion, rotationVersion, principalAuthorizationVersion int64
		idleExpiresAt, absoluteExpiresAt                                            time.Time
		verifiedEmail                                                               []byte
	)
	err = transaction.QueryRow(ctx, `
SELECT session.status, session.population, session.tenant_id, session.global_scope,
       session.invitation_id, session.assurance, session.authorization_version,
       session.rotation_version, session.idle_expires_at, session.absolute_expires_at,
       session.client_label, session.verified_email_sha256,
       principal.principal_type, principal.display_name, principal.authorization_version
FROM atlas_identity.sessions AS session
JOIN atlas_identity.principals AS principal
  ON principal.principal_id = session.principal_id
WHERE session.session_id = $1
  AND session.principal_id = $2
  AND principal.status = 'active'
FOR UPDATE OF session, principal`,
		command.Actor.SessionID.String(), command.Actor.PrincipalID.String(),
	).Scan(
		&sessionStatus, &population, &tenantText, &globalScope,
		&sessionInvitationText, &assurance, &sessionAuthorizationVersion,
		&rotationVersion, &idleExpiresAt, &absoluteExpiresAt,
		&clientLabel, &verifiedEmail, &principalType, &displayName,
		&principalAuthorizationVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.AcceptInvitationStoreResult{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	if (sessionStatus != "active" && sessionStatus != "revoked") ||
		population != string(identity.PopulationMerchant) ||
		sessionAuthorizationVersion != command.Actor.AuthorizationVersion ||
		rotationVersion != command.Actor.RotationVersion ||
		principalAuthorizationVersion > sessionAuthorizationVersion ||
		!command.Now.Before(idleExpiresAt) || !command.Now.Before(absoluteExpiresAt) ||
		!equalDigest(verifiedEmail, command.Actor.VerifiedEmailDigest) {
		return identity.AcceptInvitationStoreResult{}, identity.ErrAuthenticationRequired
	}

	var (
		invitationTenantText, role, invitationStatus  string
		invitationEmailHint                           string
		storedToken, storedEmail                      []byte
		invitationExpiresAt                           time.Time
		acceptedPrincipalText, acceptedMembershipText *string
		acceptedIdempotency, acceptedRequest          []byte
		acceptanceDecisionText                        *string
	)
	err = transaction.QueryRow(ctx, `
SELECT invitation.tenant_id, invitation.role_id, invitation.status,
       invitation.token_sha256, invitation.email_sha256, invitation.email_hint,
       invitation.expires_at,
       invitation.accepted_by_principal_id, invitation.accepted_membership_id,
       invitation.acceptance_idempotency_key_sha256, invitation.acceptance_request_sha256,
       invitation.acceptance_decision_id
FROM atlas_identity.organization_invitations AS invitation
WHERE invitation.invitation_id = $1
FOR UPDATE`, command.InvitationID.String()).Scan(
		&invitationTenantText, &role, &invitationStatus,
		&storedToken, &storedEmail, &invitationEmailHint, &invitationExpiresAt,
		&acceptedPrincipalText, &acceptedMembershipText,
		&acceptedIdempotency, &acceptedRequest, &acceptanceDecisionText,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.AcceptInvitationStoreResult{}, identity.ErrInvitationNotFound
	}
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	invitationTenant, err := identifier.Parse(invitationTenantText)
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}

	if invitationStatus == "accepted" {
		if acceptedPrincipalText == nil || *acceptedPrincipalText != command.Actor.PrincipalID.String() ||
			!equalDigest(storedToken, command.TokenDigest) ||
			!equalDigest(storedEmail, command.Actor.VerifiedEmailDigest) {
			return identity.AcceptInvitationStoreResult{}, identity.ErrInvitationNotFound
		}
		if acceptedMembershipText == nil || acceptanceDecisionText == nil {
			return identity.AcceptInvitationStoreResult{}, identity.ErrIdentityUnavailable
		}
		if !equalDigest(acceptedIdempotency, command.IdempotencyDigest) ||
			!equalDigest(acceptedRequest, command.RequestDigest) {
			return identity.AcceptInvitationStoreResult{}, identity.ErrIdempotencyConflict
		}
		member, err := loadAcceptedMembership(
			ctx, transaction, invitationTenant, *acceptedMembershipText,
		)
		if err != nil {
			return identity.AcceptInvitationStoreResult{}, err
		}
		decisionID, err := identifier.Parse(*acceptanceDecisionText)
		if err != nil {
			return identity.AcceptInvitationStoreResult{}, identity.ErrIdentityUnavailable
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
		}
		return identity.AcceptInvitationStoreResult{
			Member: member, Session: command.Actor, DecisionID: decisionID, Replay: true,
		}, nil
	}

	bootstrapBound := sessionStatus == "active" &&
		globalScope != nil && *globalScope == "invitation-acceptance" &&
		sessionInvitationText != nil && *sessionInvitationText == command.InvitationID.String() &&
		tenantText == nil && command.Actor.InvitationAcceptanceOnly
	if !bootstrapBound || invitationStatus != "pending" ||
		!equalDigest(storedToken, command.TokenDigest) ||
		!equalDigest(storedEmail, command.Actor.VerifiedEmailDigest) {
		return store.commitInvitationAcceptanceDenial(
			ctx, transaction, command, invitationTenant, "invitation_concealed",
		)
	}
	if !command.Now.Before(invitationExpiresAt) {
		_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.organization_invitations
SET status = 'expired', terminal_at = $2, version = version + 1
WHERE invitation_id = $1 AND status = 'pending'`,
			command.InvitationID.String(), command.Now,
		)
		if err != nil {
			return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
		}
		return store.commitInvitationAcceptanceDenial(
			ctx, transaction, command, invitationTenant, "invitation_expired",
		)
	}

	var organizationStatus string
	var organizationAuthorizationVersion int64
	err = transaction.QueryRow(ctx, `
SELECT status, authorization_version
FROM atlas_identity.organizations
WHERE tenant_id = $1
FOR SHARE`, invitationTenant.String()).Scan(
		&organizationStatus, &organizationAuthorizationVersion,
	)
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	if organizationStatus != "active" {
		return store.commitInvitationAcceptanceDenial(
			ctx, transaction, command, invitationTenant, "organization_inactive",
		)
	}
	var membershipExists bool
	if err := transaction.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM atlas_identity.memberships
    WHERE tenant_id = $1 AND principal_id = $2
)`, invitationTenant.String(), command.Actor.PrincipalID.String()).Scan(&membershipExists); err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	if membershipExists {
		return identity.AcceptInvitationStoreResult{}, identity.ErrSessionConflict
	}

	authorizationVersion := principalAuthorizationVersion
	if organizationAuthorizationVersion > authorizationVersion {
		authorizationVersion = organizationAuthorizationVersion
	}
	member, err := scanOrganizationMember(transaction.QueryRow(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at, email_hint
) VALUES ($1, $2, $3, $4, 'merchant', 'active', 1, 1, $5, $5, $6)
RETURNING membership_id, tenant_id, principal_id, role_id, status, version, created_at, revoked_at`,
		command.MembershipID.String(), invitationTenant.String(),
		command.Actor.PrincipalID.String(), role, command.Now, invitationEmailHint,
	))
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.organization_invitations
SET status = 'accepted', terminal_at = $2,
    accepted_by_principal_id = $3, accepted_membership_id = $4,
    acceptance_idempotency_key_sha256 = $5, acceptance_request_sha256 = $6,
    acceptance_decision_id = $7, version = version + 1
WHERE invitation_id = $1 AND status = 'pending'`,
		command.InvitationID.String(), command.Now, command.Actor.PrincipalID.String(),
		command.MembershipID.String(), command.IdempotencyDigest[:], command.RequestDigest[:],
		command.AuditEvent.DecisionID.String(),
	)
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $2, version = version + 1
WHERE session_id = $1 AND status = 'active'`, command.Actor.SessionID.String(), command.Now)
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	permissions, err := permissionsForRole(ctx, transaction, role)
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, err
	}
	var clientLabelValue any
	if clientLabel != nil {
		clientLabelValue = *clientLabel
	}
	_, err = transaction.Exec(ctx, insertSessionSQL,
		command.NewSessionID.String(), command.Actor.PrincipalID.String(),
		string(identity.PopulationMerchant), invitationTenant.String(), nil,
		command.NewSessionVerifierDigest[:], assurance, authorizationVersion,
		rotationVersion+1, command.Now, command.IdleExpiresAt, command.AbsoluteExpiresAt,
		clientLabelValue, nil, nil, nil, verifiedEmail,
	)
	if err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	event := command.AuditEvent
	event.TenantID = invitationTenant
	event.GlobalScope = ""
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.AcceptInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	return identity.AcceptInvitationStoreResult{
		Member: member,
		Session: identity.Session{
			SessionID: command.NewSessionID, PrincipalID: command.Actor.PrincipalID,
			PrincipalType: principalType, DisplayName: displayName,
			Population: identity.PopulationMerchant, TenantID: invitationTenant,
			Assurance: identity.Assurance(assurance), AuthorizationVersion: authorizationVersion,
			RotationVersion: rotationVersion + 1, CreatedAt: command.Now, LastSeenAt: command.Now,
			IdleExpiresAt: command.IdleExpiresAt, AbsoluteExpiresAt: command.AbsoluteExpiresAt,
			ClientLabel: valueOrEmpty(clientLabel), Permissions: permissions,
			VerifiedEmailDigest: command.Actor.VerifiedEmailDigest,
		},
		DecisionID: command.AuditEvent.DecisionID,
	}, nil
}

func (store *OrganizationStore) commitInvitationAcceptanceDenial(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.AcceptInvitationCommand,
	tenantID identifier.ID,
	reason string,
) (identity.AcceptInvitationStoreResult, error) {
	event := command.AuditEvent
	event.TenantID = tenantID
	event.GlobalScope = ""
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeBeforeReference = "invitation:concealed"
	event.SafeAfterReference = "invitation:concealed"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.AcceptInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.AcceptInvitationStoreResult{}, invitationAcceptanceDatabaseError(err)
	}
	return identity.AcceptInvitationStoreResult{DecisionID: event.DecisionID}, identity.ErrInvitationNotFound
}

func invitationAcceptanceDatabaseError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) &&
		(databaseError.Code == "40001" || databaseError.Code == "40P01") {
		return errRetryInvitationAcceptance
	}
	return identity.ErrIdentityUnavailable
}

func loadAcceptedMembership(
	ctx context.Context,
	transaction pgx.Tx,
	tenantID identifier.ID,
	membershipIDText string,
) (identity.OrganizationMember, error) {
	member, err := scanOrganizationMember(transaction.QueryRow(ctx, `
SELECT membership_id, tenant_id, principal_id, role_id, status, version, created_at, revoked_at
FROM atlas_identity.memberships
WHERE tenant_id = $1 AND membership_id = $2`, tenantID.String(), membershipIDText))
	if err != nil {
		return identity.OrganizationMember{}, identity.ErrIdentityUnavailable
	}
	return member, nil
}

func scanOrganizationMember(row pgx.Row) (identity.OrganizationMember, error) {
	var member identity.OrganizationMember
	var membershipIDText, tenantIDText, principalIDText string
	if err := row.Scan(
		&membershipIDText, &tenantIDText, &principalIDText, &member.Role,
		&member.Status, &member.Version, &member.CreatedAt, &member.RevokedAt,
	); err != nil {
		return identity.OrganizationMember{}, err
	}
	var err error
	member.MembershipID, err = identifier.Parse(membershipIDText)
	if err != nil {
		return identity.OrganizationMember{}, err
	}
	member.OrganizationID, err = identifier.Parse(tenantIDText)
	if err != nil {
		return identity.OrganizationMember{}, err
	}
	member.PrincipalID, err = identifier.Parse(principalIDText)
	if err != nil {
		return identity.OrganizationMember{}, err
	}
	return member, nil
}

func (store *OrganizationStore) commitInvitationDenial(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.CreateInvitationCommand,
	reason string,
	denial error,
) (identity.CreateInvitationStoreResult, error) {
	event := command.AuditEvent
	event.TenantID = command.Actor.TenantID
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeAfterReference = "invitation:not_created"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CreateInvitationStoreResult{}, identity.ErrIdentityUnavailable
	}
	return identity.CreateInvitationStoreResult{DecisionID: event.DecisionID}, denial
}

type scannedInvitation struct {
	Invitation identity.OrganizationInvitation
	DecisionID identifier.ID
}

func scanOrganizationInvitation(row pgx.Row, trailing ...any) (scannedInvitation, error) {
	var result scannedInvitation
	var invitationIDText, tenantIDText, decisionIDText string
	destinations := []any{
		&invitationIDText, &tenantIDText, &result.Invitation.EmailHint,
		&result.Invitation.Role, &result.Invitation.Status,
		&result.Invitation.ExpiresAt, &result.Invitation.CreatedAt, &decisionIDText,
	}
	destinations = append(destinations, trailing...)
	if err := row.Scan(destinations...); err != nil {
		return scannedInvitation{}, err
	}
	var err error
	result.Invitation.InvitationID, err = identifier.Parse(invitationIDText)
	if err != nil || result.Invitation.InvitationID.Prefix() != "inv" {
		return scannedInvitation{}, identity.ErrIdentityUnavailable
	}
	result.Invitation.OrganizationID, err = identifier.Parse(tenantIDText)
	if err != nil || result.Invitation.OrganizationID.Prefix() != "ten" {
		return scannedInvitation{}, identity.ErrIdentityUnavailable
	}
	result.DecisionID, err = identifier.Parse(decisionIDText)
	if err != nil || result.DecisionID.Prefix() != "dec" {
		return scannedInvitation{}, identity.ErrIdentityUnavailable
	}
	return result, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
