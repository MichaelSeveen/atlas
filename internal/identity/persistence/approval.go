package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

// ApprovalBoundary implements the Identity-owned half of the Operations
// approval protocol. Every method uses the caller-owned transaction and never
// commits, rolls back, or begins target work independently.
type ApprovalBoundary struct {
	recorder      audit.Recorder
	authorization *identity.AuthorizationPolicy
}

func NewApprovalBoundary(recorder audit.Recorder) (*ApprovalBoundary, error) {
	if recorder == nil {
		return nil, errors.New("approval identity boundary requires Audit")
	}
	return &ApprovalBoundary{
		recorder: recorder, authorization: identity.DefaultAuthorizationPolicy(),
	}, nil
}

func (boundary *ApprovalBoundary) AuthorizeApprovalSession(
	ctx context.Context,
	transaction identity.ApprovalTransaction,
	request identity.ApprovalSessionAuthorizationRequest,
) (identity.ApprovalAuthorizationResult, error) {
	denied := func(reason string) identity.ApprovalAuthorizationResult {
		return identity.ApprovalAuthorizationResult{Effect: identity.AuthorizationDeny, Reason: reason}
	}
	if boundary == nil || boundary.authorization == nil || request.Actor.SessionID.IsZero() ||
		request.Actor.PrincipalID.IsZero() || request.Actor.Population != identity.PopulationMerchant ||
		request.OrganizationID.IsZero() || request.OrganizationID.Prefix() != "ten" ||
		request.Action == "" || request.Resource == "" || request.ResourceVersion < 1 ||
		request.ResourceStatus == "" || request.Now.IsZero() || request.Now.Location() != time.UTC {
		return denied("invalid_authorization_facts"), nil
	}

	var (
		status, population, tenantIDText, assurance string
		authorizationVersion, rotationVersion       int64
		idleExpiresAt, absoluteExpiresAt            time.Time
		stepUpAction                                *string
		stepUpVerifiedAt                            *time.Time
	)
	err := transaction.QueryRow(ctx, `
SELECT status, population, tenant_id, assurance, authorization_version, rotation_version,
       idle_expires_at, absolute_expires_at, step_up_action, step_up_verified_at
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR SHARE`, request.Actor.SessionID.String(), request.Actor.PrincipalID.String()).Scan(
		&status, &population, &tenantIDText, &assurance, &authorizationVersion,
		&rotationVersion, &idleExpiresAt, &absoluteExpiresAt, &stepUpAction, &stepUpVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return denied("authority_stale_or_revoked"), nil
	}
	if err != nil {
		return identity.ApprovalAuthorizationResult{}, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.ApprovalAuthorizationResult{}, identity.ErrIdentityUnavailable
	}
	if status != "active" || population != string(identity.PopulationMerchant) ||
		tenantID != request.Actor.TenantID || assurance != string(request.Actor.Assurance) ||
		authorizationVersion != request.Actor.AuthorizationVersion ||
		rotationVersion != request.Actor.RotationVersion ||
		!request.Now.Before(idleExpiresAt) || !request.Now.Before(absoluteExpiresAt) {
		return denied("authority_stale_or_revoked"), nil
	}
	if request.OrganizationID != tenantID {
		return identity.ApprovalAuthorizationResult{
			Effect: identity.AuthorizationConceal, Reason: "tenant_concealed",
		}, nil
	}

	_, role, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx, transaction, request.Actor.PrincipalID, identity.PopulationMerchant, tenantID,
	)
	if errors.Is(err, identity.ErrAuthenticationRequired) {
		return denied("authority_stale_or_revoked"), nil
	}
	if err != nil {
		return identity.ApprovalAuthorizationResult{}, err
	}
	if currentAuthorizationVersion != authorizationVersion {
		return denied("stale_authority"), nil
	}
	permissions, err := permissionsForRole(ctx, transaction, role)
	if err != nil {
		return identity.ApprovalAuthorizationResult{}, err
	}
	decision := boundary.authorization.Evaluate(identity.AuthorizationFacts{
		PrincipalID: request.Actor.PrincipalID, PrincipalType: request.Actor.PrincipalType,
		Population: identity.PopulationMerchant, TenantID: tenantID,
		ResourceTenantID: request.OrganizationID, Role: role,
		CurrentPermissions: permissions, Action: request.Action, Resource: request.Resource,
		Purpose: request.Purpose, Assurance: request.Actor.Assurance,
		SessionAuthorizationVersion: authorizationVersion,
		CurrentAuthorizationVersion: currentAuthorizationVersion,
		ResourceVersion:             request.ResourceVersion, ResourceStatus: request.ResourceStatus,
	})
	if decision.Effect != identity.AuthorizationAllow {
		return identity.ApprovalAuthorizationResult{
			Effect: decision.Effect, Reason: decision.Reason, Role: role,
		}, nil
	}
	if request.AdditionalAction != "" {
		additional := boundary.authorization.Evaluate(identity.AuthorizationFacts{
			PrincipalID: request.Actor.PrincipalID, PrincipalType: request.Actor.PrincipalType,
			Population: identity.PopulationMerchant, TenantID: tenantID,
			ResourceTenantID: request.OrganizationID, Role: role,
			CurrentPermissions: permissions, Action: request.AdditionalAction, Resource: request.Resource,
			Purpose: request.Purpose, Assurance: request.Actor.Assurance,
			SessionAuthorizationVersion: authorizationVersion,
			CurrentAuthorizationVersion: currentAuthorizationVersion,
			ResourceVersion:             request.ResourceVersion, ResourceStatus: request.ResourceStatus,
		})
		if additional.Effect != identity.AuthorizationAllow {
			return identity.ApprovalAuthorizationResult{
				Effect: additional.Effect, Reason: additional.Reason, Role: role,
			}, nil
		}
	}
	if request.RequiredStepUpAction != "" {
		stepUpSatisfied := assurance == string(identity.AssurancePhishingResistant) &&
			stepUpAction != nil && *stepUpAction == request.RequiredStepUpAction &&
			stepUpVerifiedAt != nil &&
			!stepUpVerifiedAt.Before(request.Now.Add(-5*time.Minute)) &&
			!stepUpVerifiedAt.After(request.Now.Add(time.Minute))
		if !stepUpSatisfied {
			return identity.ApprovalAuthorizationResult{
				Effect: identity.AuthorizationDeny, Reason: "step_up_required", Role: role,
			}, nil
		}
	}
	return identity.ApprovalAuthorizationResult{
		Effect: identity.AuthorizationAllow, Reason: "allowed", Role: role,
	}, nil
}

func (boundary *ApprovalBoundary) AuthorizeApprovalPrincipal(
	ctx context.Context,
	transaction identity.ApprovalTransaction,
	request identity.ApprovalPrincipalAuthorizationRequest,
) (identity.ApprovalAuthorizationResult, error) {
	denied := func(reason string) identity.ApprovalAuthorizationResult {
		return identity.ApprovalAuthorizationResult{Effect: identity.AuthorizationDeny, Reason: reason}
	}
	if boundary == nil || boundary.authorization == nil || request.PrincipalID.IsZero() ||
		request.OrganizationID.IsZero() || request.OrganizationID.Prefix() != "ten" ||
		request.Action == "" || request.Resource == "" || request.ResourceVersion < 1 ||
		request.ResourceStatus == "" {
		return denied("invalid_authorization_facts"), nil
	}
	tenantID, role, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx, transaction, request.PrincipalID, identity.PopulationMerchant, request.OrganizationID,
	)
	if errors.Is(err, identity.ErrAuthenticationRequired) {
		return denied("authority_stale_or_revoked"), nil
	}
	if err != nil {
		return identity.ApprovalAuthorizationResult{}, err
	}
	permissions, err := permissionsForRole(ctx, transaction, role)
	if err != nil {
		return identity.ApprovalAuthorizationResult{}, err
	}
	decision := boundary.authorization.Evaluate(identity.AuthorizationFacts{
		PrincipalID: request.PrincipalID, PrincipalType: "merchant",
		Population: identity.PopulationMerchant, TenantID: tenantID,
		ResourceTenantID: request.OrganizationID, Role: role,
		CurrentPermissions: permissions, Action: request.Action, Resource: request.Resource,
		Purpose: request.Purpose, Assurance: identity.AssuranceBaseline,
		SessionAuthorizationVersion: currentAuthorizationVersion,
		CurrentAuthorizationVersion: currentAuthorizationVersion,
		ResourceVersion:             request.ResourceVersion, ResourceStatus: request.ResourceStatus,
	})
	return identity.ApprovalAuthorizationResult{
		Effect: decision.Effect, Reason: decision.Reason, Role: role,
	}, nil
}

func (boundary *ApprovalBoundary) ValidateMembershipRoleChangeTarget(
	ctx context.Context,
	transaction identity.ApprovalTransaction,
	request identity.MembershipRoleChangeTargetRequest,
) (identity.MembershipRoleChangeTarget, error) {
	if request.OrganizationID.IsZero() || request.OrganizationID.Prefix() != "ten" ||
		request.MembershipID.IsZero() || request.MembershipID.Prefix() != "mem" ||
		request.ExpectedVersion < 1 || !persistedMerchantRole(request.RequestedRole) ||
		request.RequestedRole == "merchant_security_admin" {
		return identity.MembershipRoleChangeTarget{}, identity.ErrValidationFailed
	}
	var (
		tenantIDText, membershipIDText, principalIDText string
		role, membershipStatus, principalStatus         string
		organizationStatus, population                  string
		version                                         int64
		createdAt                                       time.Time
	)
	err := transaction.QueryRow(ctx, `
SELECT membership.tenant_id, membership.membership_id, membership.principal_id,
       membership.role_id, membership.status, membership.version, membership.created_at,
       principal.status, principal.principal_type, organization.status
FROM atlas_identity.memberships AS membership
JOIN atlas_identity.principals AS principal
  ON principal.principal_id = membership.principal_id
JOIN atlas_identity.organizations AS organization
  ON organization.tenant_id = membership.tenant_id
WHERE membership.tenant_id = $1
  AND membership.membership_id = $2
FOR SHARE OF membership, principal, organization`,
		request.OrganizationID.String(), request.MembershipID.String(),
	).Scan(
		&tenantIDText, &membershipIDText, &principalIDText, &role, &membershipStatus,
		&version, &createdAt, &principalStatus, &population, &organizationStatus,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.MembershipRoleChangeTarget{}, identity.ErrMembershipNotFound
	}
	if err != nil {
		return identity.MembershipRoleChangeTarget{}, identity.ErrIdentityUnavailable
	}
	if membershipStatus != "active" || principalStatus != "active" || population != "merchant" ||
		organizationStatus != "active" {
		return identity.MembershipRoleChangeTarget{}, identity.ErrMembershipNotFound
	}
	if version != request.ExpectedVersion {
		return identity.MembershipRoleChangeTarget{}, identity.ErrMembershipPreconditionFailed
	}
	if !validApprovedAdministratorTransition(role, request.RequestedRole) {
		return identity.MembershipRoleChangeTarget{}, identity.ErrValidationFailed
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.MembershipRoleChangeTarget{}, identity.ErrIdentityUnavailable
	}
	membershipID, err := identifier.Parse(membershipIDText)
	if err != nil {
		return identity.MembershipRoleChangeTarget{}, identity.ErrIdentityUnavailable
	}
	principalID, err := identifier.Parse(principalIDText)
	if err != nil {
		return identity.MembershipRoleChangeTarget{}, identity.ErrIdentityUnavailable
	}
	return identity.MembershipRoleChangeTarget{
		OrganizationID: tenantID, MembershipID: membershipID, PrincipalID: principalID,
		BeforeRole: role, RequestedRole: request.RequestedRole,
		Version: version, CreatedAt: createdAt.UTC(),
	}, nil
}

func (boundary *ApprovalBoundary) ExecuteApprovedMembershipRoleChange(
	ctx context.Context,
	transaction identity.ApprovalTransaction,
	command identity.ExecuteApprovedMembershipRoleChangeCommand,
) (identity.ExecuteApprovedMembershipRoleChangeResult, error) {
	if command.ApprovalID.IsZero() || command.ApprovalID.Prefix() != "apr" ||
		command.RoleChangeID.IsZero() || command.RoleChangeID.Prefix() != "mrc" ||
		command.Executor.SessionID.IsZero() || command.Executor.PrincipalID.IsZero() ||
		command.Now.IsZero() || command.Now.Location() != time.UTC {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, identity.ErrValidationFailed
	}
	target, err := boundary.lockMembershipRoleChangeTarget(ctx, transaction, command.Target)
	if err != nil {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, err
	}
	updated, err := scanOrganizationMember(transaction.QueryRow(ctx, `
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
		command.Target.OrganizationID.String(), command.Target.MembershipID.String(),
		command.Target.ExpectedVersion, command.Target.RequestedRole, command.Now,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, identity.ErrMembershipPreconditionFailed
	}
	if err != nil {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, identity.ErrIdentityUnavailable
	}
	if _, err := transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $3, version = version + 1
WHERE principal_id = $1
  AND tenant_id = $2
  AND status = 'active'`,
		updated.PrincipalID.String(), command.Target.OrganizationID.String(), command.Now,
	); err != nil {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, identity.ErrIdentityUnavailable
	}

	event := command.AuditEvent
	event.TargetID = command.Target.MembershipID.String()
	event.ApprovalID = command.ApprovalID
	event.SafeBeforeReference = "membership-role:" + target.BeforeRole
	event.SafeAfterReference = "membership-role:" + command.Target.RequestedRole
	if err := boundary.recorder.Record(ctx, transaction, event); err != nil {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, identity.ErrIdentityUnavailable
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.membership_role_changes (
    role_change_id, tenant_id, membership_id, target_principal_id, population,
    actor_principal_id, actor_session_id, idempotency_key_sha256, request_sha256,
    expected_membership_version, before_role_id, after_role_id,
    result_membership_version, membership_created_at,
    decision_id, audit_event_id, created_at, approval_id
) VALUES (
    $1, $2, $3, $4, 'merchant',
    $5, $6, $7, $8,
    $9, $10, $11,
    $12, $13,
    $14, $15, $16, $17
)
ON CONFLICT (approval_id) WHERE approval_id IS NOT NULL DO NOTHING`,
		command.RoleChangeID.String(), command.Target.OrganizationID.String(),
		command.Target.MembershipID.String(), updated.PrincipalID.String(),
		command.Executor.PrincipalID.String(), command.Executor.SessionID.String(),
		command.IdempotencyDigest[:], command.PayloadDigest[:], command.Target.ExpectedVersion,
		target.BeforeRole, command.Target.RequestedRole, updated.Version, updated.CreatedAt,
		event.DecisionID.String(), event.AuditEventID.String(), command.Now, command.ApprovalID.String(),
	)
	if err != nil {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, identity.ErrIdentityUnavailable
	}
	if inserted.RowsAffected() != 1 {
		return identity.ExecuteApprovedMembershipRoleChangeResult{}, identity.ErrMembershipPreconditionFailed
	}
	target.RequestedRole = updated.Role
	return identity.ExecuteApprovedMembershipRoleChangeResult{
		Target: target, ResultVersion: updated.Version,
	}, nil
}

func (boundary *ApprovalBoundary) lockMembershipRoleChangeTarget(
	ctx context.Context,
	transaction identity.ApprovalTransaction,
	request identity.MembershipRoleChangeTargetRequest,
) (identity.MembershipRoleChangeTarget, error) {
	if request.OrganizationID.IsZero() || request.MembershipID.IsZero() ||
		request.ExpectedVersion < 1 || request.RequestedRole == "merchant_security_admin" {
		return identity.MembershipRoleChangeTarget{}, identity.ErrValidationFailed
	}
	var principalIDText, role, status string
	var version int64
	var createdAt time.Time
	err := transaction.QueryRow(ctx, `
SELECT principal_id, role_id, status, version, created_at
FROM atlas_identity.memberships
WHERE tenant_id = $1 AND membership_id = $2
FOR UPDATE`, request.OrganizationID.String(), request.MembershipID.String()).Scan(
		&principalIDText, &role, &status, &version, &createdAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.MembershipRoleChangeTarget{}, identity.ErrMembershipNotFound
	}
	if err != nil {
		return identity.MembershipRoleChangeTarget{}, identity.ErrIdentityUnavailable
	}
	if status != "active" {
		return identity.MembershipRoleChangeTarget{}, identity.ErrMembershipNotFound
	}
	if version != request.ExpectedVersion {
		return identity.MembershipRoleChangeTarget{}, identity.ErrMembershipPreconditionFailed
	}
	if !validApprovedAdministratorTransition(role, request.RequestedRole) {
		return identity.MembershipRoleChangeTarget{}, identity.ErrValidationFailed
	}
	principalID, err := identifier.Parse(principalIDText)
	if err != nil {
		return identity.MembershipRoleChangeTarget{}, identity.ErrIdentityUnavailable
	}
	return identity.MembershipRoleChangeTarget{
		OrganizationID: request.OrganizationID, MembershipID: request.MembershipID,
		PrincipalID: principalID, BeforeRole: role, RequestedRole: request.RequestedRole,
		Version: version, CreatedAt: createdAt.UTC(),
	}, nil
}

func validApprovedAdministratorTransition(before, after string) bool {
	if before == after || before == "merchant_security_admin" || after == "merchant_security_admin" {
		return false
	}
	if !persistedMerchantRole(before) || !persistedMerchantRole(after) {
		return false
	}
	return before == "merchant_admin" || after == "merchant_admin"
}
