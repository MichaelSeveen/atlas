package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var (
	ErrOrganizationNotFound = domainerror.New(
		domainerror.MustCode("ORGANIZATION_NOT_FOUND"),
		domainerror.KindNotFound,
		false,
	)
	ErrValidationFailed = domainerror.New(
		domainerror.MustCode("IDENTITY_VALIDATION_FAILED"),
		domainerror.KindInvalidArgument,
		false,
	)
	ErrInvitationNotFound = domainerror.New(
		domainerror.MustCode("INVITATION_NOT_FOUND"),
		domainerror.KindNotFound,
		false,
	)
	ErrMembershipNotFound = domainerror.New(
		domainerror.MustCode("MEMBERSHIP_NOT_FOUND"),
		domainerror.KindNotFound,
		false,
	)
	ErrMembershipPreconditionFailed = domainerror.New(
		domainerror.MustCode("MEMBERSHIP_PRECONDITION_FAILED"),
		domainerror.KindFailedPrecondition,
		false,
	)
	ErrMembershipApprovalRequired = domainerror.New(
		domainerror.MustCode("MEMBERSHIP_APPROVAL_REQUIRED"),
		domainerror.KindConflict,
		false,
	)
	ErrMembershipAdministratorRemovalUnavailable = domainerror.New(
		domainerror.MustCode("MEMBERSHIP_ADMINISTRATOR_REMOVAL_UNAVAILABLE"),
		domainerror.KindConflict,
		false,
	)
)

const invitationLifetime = 72 * time.Hour

// Organization is an organization membership visible to the current principal.
type Organization struct {
	OrganizationID    identifier.ID
	DisplayName       string
	PrincipalRole     string
	MembershipVersion int64
}

// OrganizationMember is the authorized, non-sensitive membership representation.
type OrganizationMember struct {
	MembershipID   identifier.ID
	OrganizationID identifier.ID
	PrincipalID    identifier.ID
	EmailHint      *string
	Role           string
	Status         string
	Version        int64
	CreatedAt      time.Time
	RevokedAt      *time.Time
}

// OrganizationMemberPage contains a bounded cursor page without a total count.
type OrganizationMemberPage struct {
	Members    []OrganizationMember
	NextCursor string
	HasMore    bool
}

// ListOrganizationMembersRequest is the cookie-authenticated member-list request.
type ListOrganizationMembersRequest struct {
	CookieValue      string
	OrganizationID   identifier.ID
	PageSize         string
	PageSizeProvided bool
	Cursor           string
	CursorProvided   bool
	CorrelationID    identifier.ID
}

// ListOrganizationMembersCommand carries explicit tenant and authorization context.
type ListOrganizationMembersCommand struct {
	Actor            Session
	OrganizationID   identifier.ID
	PageSize         string
	PageSizeProvided bool
	Cursor           string
	CursorProvided   bool
	Now              time.Time
	AuditEvent       audit.Event
}

// ListOrganizationMembersResult returns the page and its authorization decision.
type ListOrganizationMembersResult struct {
	Page       OrganizationMemberPage
	DecisionID identifier.ID
}

// SwitchOrganizationCommand carries every value needed for an atomic tenant-context rotation.
type SwitchOrganizationCommand struct {
	Actor          Session
	OrganizationID identifier.ID
	NewSessionID   identifier.ID
	VerifierDigest [32]byte
	Now            time.Time
	IdleExpiresAt  time.Time
	AuditEvent     audit.Event
}

// SwitchOrganizationResult returns the rotated session and its authorization decision.
type SwitchOrganizationResult struct {
	CookieValue string
	CSRFToken   string
	Session     Session
	DecisionID  identifier.ID
}

// OrganizationInvitation is the non-secret invitation representation returned by the API.
type OrganizationInvitation struct {
	InvitationID   identifier.ID
	OrganizationID identifier.ID
	EmailHint      string
	Role           string
	Status         string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// CreateInvitationRequest is the cookie-authenticated invitation issuance request.
type CreateInvitationRequest struct {
	CookieValue    string
	CSRFToken      string
	OrganizationID identifier.ID
	Email          string
	Role           string
	IdempotencyKey string
	CorrelationID  identifier.ID
}

// CreateInvitationCommand carries only digests, masked recipient data, and authority context.
type CreateInvitationCommand struct {
	Actor             Session
	InvitationID      identifier.ID
	OrganizationID    identifier.ID
	EmailDigest       [32]byte
	EmailHint         string
	Role              string
	TokenDigest       [32]byte
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	ExpiresAt         time.Time
	AuditEvent        audit.Event
}

// CreateInvitationStoreResult is the durable issuance or its redacted replay.
type CreateInvitationStoreResult struct {
	Invitation OrganizationInvitation
	DecisionID identifier.ID
	Replay     bool
}

// CreateInvitationResult includes one-time material only for the transaction that inserted it.
type CreateInvitationResult struct {
	Invitation      OrganizationInvitation
	AcceptanceToken string
	SecretDisclosed bool
	DecisionID      identifier.ID
	Replay          bool
}

// AcceptInvitationRequest is the invitation-bound, cookie-authenticated command.
type AcceptInvitationRequest struct {
	CookieValue     string
	CSRFToken       string
	InvitationID    identifier.ID
	AcceptanceToken string
	IdempotencyKey  string
	CorrelationID   identifier.ID
}

// AcceptInvitationCommand carries the verified bootstrap identity and all
// material required for atomic membership creation and session rotation.
type AcceptInvitationCommand struct {
	Actor                    Session
	InvitationID             identifier.ID
	TokenDigest              [32]byte
	MembershipID             identifier.ID
	NewSessionID             identifier.ID
	NewSessionVerifierDigest [32]byte
	IdempotencyDigest        [32]byte
	RequestDigest            [32]byte
	Now                      time.Time
	IdleExpiresAt            time.Time
	AbsoluteExpiresAt        time.Time
	AuditEvent               audit.Event
}

// AcceptInvitationStoreResult returns the committed membership, rotated
// session, authorization decision, and replay posture.
type AcceptInvitationStoreResult struct {
	Member     OrganizationMember
	Session    Session
	DecisionID identifier.ID
	Replay     bool
}

// AcceptInvitationResult includes the new cookie only on the committing call.
type AcceptInvitationResult struct {
	Member      OrganizationMember
	Session     Session
	CookieValue string
	CSRFToken   string
	DecisionID  identifier.ID
	Replay      bool
}

// UpdateOrganizationMemberRoleRequest is the cookie-authenticated direct role-change request.
type UpdateOrganizationMemberRoleRequest struct {
	CookieValue    string
	CSRFToken      string
	OrganizationID identifier.ID
	MembershipID   identifier.ID
	Role           string
	Purpose        string
	IfMatch        string
	IdempotencyKey string
	CorrelationID  identifier.ID
}

// UpdateOrganizationMemberRoleCommand carries the locked authority, precondition, and replay state.
type UpdateOrganizationMemberRoleCommand struct {
	Actor             Session
	RoleChangeID      identifier.ID
	OrganizationID    identifier.ID
	MembershipID      identifier.ID
	Role              string
	ExpectedVersion   int64
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	AuditEvent        audit.Event
}

// UpdateOrganizationMemberRoleResult is the committed direct mutation or its durable replay.
type UpdateOrganizationMemberRoleResult struct {
	Member     OrganizationMember
	DecisionID identifier.ID
	Replay     bool
}

// RevokeOrganizationMemberRequest is the cookie-authenticated direct membership-revocation request.
type RevokeOrganizationMemberRequest struct {
	CookieValue    string
	CSRFToken      string
	OrganizationID identifier.ID
	MembershipID   identifier.ID
	IfMatch        string
	IdempotencyKey string
	CorrelationID  identifier.ID
}

// RevokeOrganizationMemberCommand carries locked authority, precondition, and durable replay state.
type RevokeOrganizationMemberCommand struct {
	Actor             Session
	RevocationID      identifier.ID
	OrganizationID    identifier.ID
	MembershipID      identifier.ID
	ExpectedVersion   int64
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	AuditEvent        audit.Event
}

// RevokeOrganizationMemberResult is the committed revocation or its durable replay.
type RevokeOrganizationMemberResult struct {
	Member     OrganizationMember
	DecisionID identifier.ID
	Replay     bool
}

// OrganizationStore is the Identity-owned persistence boundary for organization context.
type OrganizationStore interface {
	ListOrganizations(context.Context, Session) ([]Organization, error)
	ListMembers(context.Context, ListOrganizationMembersCommand) (ListOrganizationMembersResult, error)
	SwitchOrganization(context.Context, SwitchOrganizationCommand) (Session, error)
	CreateInvitation(context.Context, CreateInvitationCommand) (CreateInvitationStoreResult, error)
	AcceptInvitation(context.Context, AcceptInvitationCommand) (AcceptInvitationStoreResult, error)
	UpdateMemberRole(context.Context, UpdateOrganizationMemberRoleCommand) (UpdateOrganizationMemberRoleResult, error)
	RevokeMember(context.Context, RevokeOrganizationMemberCommand) (RevokeOrganizationMemberResult, error)
}

// UpdateOrganizationMemberRole executes only the direct viewer/operator lane. Administrator
// transitions remain fail-closed until the Operations-owned approval workflow exists.
func (service *Service) UpdateOrganizationMemberRole(
	ctx context.Context,
	request UpdateOrganizationMemberRoleRequest,
) (UpdateOrganizationMemberRoleResult, error) {
	actor, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return UpdateOrganizationMemberRoleResult{}, err
	}
	if actor.InvitationAcceptanceOnly {
		return UpdateOrganizationMemberRoleResult{}, ErrActionNotAuthorized
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return UpdateOrganizationMemberRoleResult{}, ErrCSRFValidationFailed
	}
	if service.organizations == nil {
		return UpdateOrganizationMemberRoleResult{}, ErrIdentityUnavailable
	}
	if actor.Population != PopulationMerchant {
		return UpdateOrganizationMemberRoleResult{}, ErrActionNotAuthorized
	}
	if request.OrganizationID.IsZero() || request.OrganizationID.Prefix() != "ten" ||
		request.MembershipID.IsZero() || request.MembershipID.Prefix() != "mem" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" ||
		!validIdempotencyKey(request.IdempotencyKey) {
		return UpdateOrganizationMemberRoleResult{}, ErrInputInvalid
	}
	if request.Purpose != "organization_administration" || !validMerchantRole(request.Role) {
		return UpdateOrganizationMemberRoleResult{}, ErrValidationFailed
	}
	expectedVersion, err := parseOrganizationMemberETag(request.IfMatch)
	if err != nil {
		return UpdateOrganizationMemberRoleResult{}, ErrInputInvalid
	}
	roleChangeID, err := service.generatedID("mrc")
	if err != nil {
		return UpdateOrganizationMemberRoleResult{}, ErrIdentityUnavailable
	}
	auditID, err := service.generatedID("aud")
	if err != nil {
		return UpdateOrganizationMemberRoleResult{}, ErrIdentityUnavailable
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return UpdateOrganizationMemberRoleResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	idempotencyDigest := sha256.Sum256([]byte(request.IdempotencyKey))
	requestDigest := sha256.Sum256([]byte(
		"v1\norganization=" + request.OrganizationID.String() +
			"\nmembership=" + request.MembershipID.String() +
			"\nexpected_version=" + strconv.FormatInt(expectedVersion, 10) +
			"\nrole=" + request.Role + "\npurpose=" + request.Purpose,
	))
	return service.organizations.UpdateMemberRole(ctx, UpdateOrganizationMemberRoleCommand{
		Actor: actor, RoleChangeID: roleChangeID, OrganizationID: request.OrganizationID,
		MembershipID: request.MembershipID, Role: request.Role, ExpectedVersion: expectedVersion,
		IdempotencyDigest: idempotencyDigest, RequestDigest: requestDigest, Now: now,
		AuditEvent: audit.Event{
			AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
			TenantID: actor.TenantID, SessionAssurance: string(actor.Assurance),
			Action: "identity.organization.membership.role.change", TargetType: "membership",
			TargetID: request.MembershipID.String(), DecisionID: decisionID, Decision: "executed",
			ReasonCode: "organization_membership_role_changed", CorrelationID: request.CorrelationID,
			OccurredAt: now, SafeAfterReference: "membership-role:" + request.Role,
		},
	})
}

// RevokeOrganizationMember executes only the direct viewer/operator lane.
// Administrator removal remains fail-closed until its exact step-up and
// last-administrator execution policy is implemented.
func (service *Service) RevokeOrganizationMember(
	ctx context.Context,
	request RevokeOrganizationMemberRequest,
) (RevokeOrganizationMemberResult, error) {
	actor, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return RevokeOrganizationMemberResult{}, err
	}
	if actor.InvitationAcceptanceOnly {
		return RevokeOrganizationMemberResult{}, ErrActionNotAuthorized
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return RevokeOrganizationMemberResult{}, ErrCSRFValidationFailed
	}
	if service.organizations == nil {
		return RevokeOrganizationMemberResult{}, ErrIdentityUnavailable
	}
	if actor.Population != PopulationMerchant {
		return RevokeOrganizationMemberResult{}, ErrActionNotAuthorized
	}
	if request.OrganizationID.IsZero() || request.OrganizationID.Prefix() != "ten" ||
		request.MembershipID.IsZero() || request.MembershipID.Prefix() != "mem" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" ||
		!validIdempotencyKey(request.IdempotencyKey) {
		return RevokeOrganizationMemberResult{}, ErrInputInvalid
	}
	expectedVersion, err := parseOrganizationMemberETag(request.IfMatch)
	if err != nil {
		return RevokeOrganizationMemberResult{}, ErrInputInvalid
	}
	revocationID, err := service.generatedID("mrv")
	if err != nil {
		return RevokeOrganizationMemberResult{}, ErrIdentityUnavailable
	}
	auditID, err := service.generatedID("aud")
	if err != nil {
		return RevokeOrganizationMemberResult{}, ErrIdentityUnavailable
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return RevokeOrganizationMemberResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	idempotencyDigest := sha256.Sum256([]byte(request.IdempotencyKey))
	requestDigest := sha256.Sum256([]byte(
		"v1\norganization=" + request.OrganizationID.String() +
			"\nmembership=" + request.MembershipID.String() +
			"\nexpected_version=" + strconv.FormatInt(expectedVersion, 10),
	))
	return service.organizations.RevokeMember(ctx, RevokeOrganizationMemberCommand{
		Actor: actor, RevocationID: revocationID, OrganizationID: request.OrganizationID,
		MembershipID: request.MembershipID, ExpectedVersion: expectedVersion,
		IdempotencyDigest: idempotencyDigest, RequestDigest: requestDigest, Now: now,
		AuditEvent: audit.Event{
			AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
			TenantID: actor.TenantID, SessionAssurance: string(actor.Assurance),
			Action: "identity.organization.membership.revoke", TargetType: "membership",
			TargetID: request.MembershipID.String(), DecisionID: decisionID, Decision: "executed",
			ReasonCode: "organization_membership_revoked", CorrelationID: request.CorrelationID,
			OccurredAt: now, SafeAfterReference: "membership-status:revoked",
		},
	})
}

// OrganizationMemberETag returns the exact strong validator for a membership version.
func OrganizationMemberETag(version int64) string {
	if version < 1 {
		return ""
	}
	return "\"membership-v" + strconv.FormatInt(version, 10) + "\""
}

func parseOrganizationMemberETag(value string) (int64, error) {
	if len(value) < len("\"membership-v1\"") || len(value) > 200 ||
		!strings.HasPrefix(value, "\"membership-v") || !strings.HasSuffix(value, "\"") {
		return 0, ErrInputInvalid
	}
	number := strings.TrimSuffix(strings.TrimPrefix(value, "\"membership-v"), "\"")
	if number == "" || (len(number) > 1 && number[0] == '0') {
		return 0, ErrInputInvalid
	}
	version, err := strconv.ParseInt(number, 10, 64)
	if err != nil || version < 1 || OrganizationMemberETag(version) != value {
		return 0, ErrInputInvalid
	}
	return version, nil
}

// OrganizationMembers returns an authorized page without totals or sensitive member fields.
func (service *Service) OrganizationMembers(
	ctx context.Context,
	request ListOrganizationMembersRequest,
) (ListOrganizationMembersResult, error) {
	actor, _, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return ListOrganizationMembersResult{}, err
	}
	if actor.InvitationAcceptanceOnly {
		return ListOrganizationMembersResult{}, ErrActionNotAuthorized
	}
	if service.organizations == nil {
		return ListOrganizationMembersResult{}, ErrIdentityUnavailable
	}
	if request.OrganizationID.IsZero() || request.OrganizationID.Prefix() != "ten" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return ListOrganizationMembersResult{}, ErrInputInvalid
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return ListOrganizationMembersResult{}, ErrIdentityUnavailable
	}
	if actor.Population != PopulationMerchant {
		return ListOrganizationMembersResult{DecisionID: decisionID}, ErrActionNotAuthorized
	}
	auditID, err := service.generatedID("aud")
	if err != nil {
		return ListOrganizationMembersResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	return service.organizations.ListMembers(ctx, ListOrganizationMembersCommand{
		Actor: actor, OrganizationID: request.OrganizationID,
		PageSize: request.PageSize, PageSizeProvided: request.PageSizeProvided,
		Cursor: request.Cursor, CursorProvided: request.CursorProvided, Now: now,
		AuditEvent: audit.Event{
			AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
			TenantID: actor.TenantID, SessionAssurance: string(actor.Assurance),
			Action: "identity.organization.members.list", TargetType: "organization",
			TargetID: request.OrganizationID.String(), DecisionID: decisionID, Decision: "allowed",
			ReasonCode: "organization_members_listed", CorrelationID: request.CorrelationID,
			OccurredAt: now, SafeAfterReference: "organization-members:masked",
		},
	})
}

// Organizations lists the merchant organizations that are currently eligible for this principal.
func (service *Service) Organizations(ctx context.Context, cookieValue string) ([]Organization, error) {
	session, _, err := service.Current(ctx, cookieValue)
	if err != nil {
		return nil, err
	}
	if session.InvitationAcceptanceOnly {
		return nil, ErrActionNotAuthorized
	}
	if session.Population != PopulationMerchant {
		return []Organization{}, nil
	}
	if service.organizations == nil {
		return nil, ErrIdentityUnavailable
	}
	return service.organizations.ListOrganizations(ctx, session)
}

// SwitchActiveOrganization atomically revokes the old session and creates a new tenant-bound session.
func (service *Service) SwitchActiveOrganization(
	ctx context.Context,
	cookieValue string,
	csrfToken string,
	organizationID identifier.ID,
	correlationID identifier.ID,
) (SwitchOrganizationResult, error) {
	session, expectedCSRF, err := service.Current(ctx, cookieValue)
	if err != nil {
		return SwitchOrganizationResult{}, err
	}
	if session.InvitationAcceptanceOnly {
		return SwitchOrganizationResult{}, ErrActionNotAuthorized
	}
	if !constantTimeStringEqual(expectedCSRF, csrfToken) {
		return SwitchOrganizationResult{}, ErrCSRFValidationFailed
	}
	if service.organizations == nil {
		return SwitchOrganizationResult{}, ErrIdentityUnavailable
	}
	if session.Population != PopulationMerchant {
		return SwitchOrganizationResult{}, ErrActionNotAuthorized
	}
	if organizationID.IsZero() || organizationID.Prefix() != "ten" ||
		correlationID.IsZero() || correlationID.Prefix() != "cor" {
		return SwitchOrganizationResult{}, ErrInputInvalid
	}
	newSessionID, err := service.generatedID("ses")
	if err != nil {
		return SwitchOrganizationResult{}, ErrIdentityUnavailable
	}
	cookie, verifierDigest, err := randomToken(service.entropy)
	if err != nil {
		return SwitchOrganizationResult{}, ErrIdentityUnavailable
	}
	auditID, err := service.generatedID("aud")
	if err != nil {
		return SwitchOrganizationResult{}, ErrIdentityUnavailable
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return SwitchOrganizationResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	idleExpiresAt := now.Add(service.sessionPolicies[PopulationMerchant].Idle)
	if idleExpiresAt.After(session.AbsoluteExpiresAt) {
		idleExpiresAt = session.AbsoluteExpiresAt
	}
	rotated, err := service.organizations.SwitchOrganization(ctx, SwitchOrganizationCommand{
		Actor: session, OrganizationID: organizationID, NewSessionID: newSessionID,
		VerifierDigest: verifierDigest, Now: now, IdleExpiresAt: idleExpiresAt,
		AuditEvent: audit.Event{
			AuditEventID: auditID, ActorID: session.PrincipalID, ActorType: session.PrincipalType,
			TenantID: organizationID, SessionAssurance: string(session.Assurance),
			Action: "identity.organization.active.switch", TargetType: "organization",
			TargetID: organizationID.String(), DecisionID: decisionID, Decision: "executed",
			ReasonCode: "active_organization_selected", CorrelationID: correlationID,
			OccurredAt:          now,
			SafeBeforeReference: "organization:" + session.TenantID.String(),
			SafeAfterReference:  "organization:" + organizationID.String(),
		},
	})
	if err != nil {
		return SwitchOrganizationResult{}, err
	}
	rotatedCSRF, err := service.csrf.Token(rotated.SessionID, rotated.RotationVersion)
	if err != nil {
		return SwitchOrganizationResult{}, ErrIdentityUnavailable
	}
	return SwitchOrganizationResult{
		CookieValue: cookie, CSRFToken: rotatedCSRF, Session: rotated, DecisionID: decisionID,
	}, nil
}

// CreateOrganizationInvitation issues hash-only invitation material with redacted replay.
func (service *Service) CreateOrganizationInvitation(
	ctx context.Context,
	request CreateInvitationRequest,
) (CreateInvitationResult, error) {
	actor, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return CreateInvitationResult{}, err
	}
	if actor.InvitationAcceptanceOnly {
		return CreateInvitationResult{}, ErrActionNotAuthorized
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return CreateInvitationResult{}, ErrCSRFValidationFailed
	}
	if service.organizations == nil {
		return CreateInvitationResult{}, ErrIdentityUnavailable
	}
	if actor.Population != PopulationMerchant {
		return CreateInvitationResult{}, ErrActionNotAuthorized
	}
	if request.OrganizationID.IsZero() || request.OrganizationID.Prefix() != "ten" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" ||
		!validIdempotencyKey(request.IdempotencyKey) {
		return CreateInvitationResult{}, ErrInputInvalid
	}
	emailDigest, emailHint, err := invitationEmail(request.Email)
	if err != nil || !validMerchantRole(request.Role) {
		return CreateInvitationResult{}, ErrValidationFailed
	}
	invitationID, err := service.generatedID("inv")
	if err != nil {
		return CreateInvitationResult{}, ErrIdentityUnavailable
	}
	token, tokenDigest, err := randomToken(service.entropy)
	if err != nil {
		return CreateInvitationResult{}, ErrIdentityUnavailable
	}
	auditID, err := service.generatedID("aud")
	if err != nil {
		return CreateInvitationResult{}, ErrIdentityUnavailable
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return CreateInvitationResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	idempotencyDigest := sha256.Sum256([]byte(request.IdempotencyKey))
	requestDigest := sha256.Sum256([]byte(
		"v1\nemail_sha256=" + hex.EncodeToString(emailDigest[:]) + "\nrole=" + request.Role,
	))
	stored, err := service.organizations.CreateInvitation(ctx, CreateInvitationCommand{
		Actor: actor, InvitationID: invitationID, OrganizationID: request.OrganizationID,
		EmailDigest: emailDigest, EmailHint: emailHint, Role: request.Role,
		TokenDigest: tokenDigest, IdempotencyDigest: idempotencyDigest,
		RequestDigest: requestDigest, Now: now, ExpiresAt: now.Add(invitationLifetime),
		AuditEvent: audit.Event{
			AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
			TenantID: actor.TenantID, SessionAssurance: string(actor.Assurance),
			Action: "identity.organization.invitation.create", TargetType: "invitation",
			TargetID: invitationID.String(), DecisionID: decisionID, Decision: "executed",
			ReasonCode: "organization_invitation_created", CorrelationID: request.CorrelationID,
			OccurredAt: now, SafeAfterReference: "invitation:pending",
		},
	})
	if err != nil {
		return CreateInvitationResult{DecisionID: stored.DecisionID, Replay: stored.Replay}, err
	}
	result := CreateInvitationResult{
		Invitation: stored.Invitation, DecisionID: stored.DecisionID, Replay: stored.Replay,
	}
	if !stored.Replay {
		result.AcceptanceToken = token
		result.SecretDisclosed = true
	}
	return result, nil
}

// AcceptOrganizationInvitation consumes an invitation only for the verified
// recipient and rotates the acceptance-only session into tenant authority.
func (service *Service) AcceptOrganizationInvitation(
	ctx context.Context,
	request AcceptInvitationRequest,
) (AcceptInvitationResult, error) {
	actor, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return AcceptInvitationResult{}, err
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return AcceptInvitationResult{}, ErrCSRFValidationFailed
	}
	if service.organizations == nil {
		return AcceptInvitationResult{}, ErrIdentityUnavailable
	}
	if actor.Population != PopulationMerchant || actor.VerifiedEmailDigest == ([32]byte{}) {
		return AcceptInvitationResult{}, ErrActionNotAuthorized
	}
	if request.InvitationID.IsZero() || request.InvitationID.Prefix() != "inv" ||
		!validProtocolToken(request.AcceptanceToken) ||
		!validIdempotencyKey(request.IdempotencyKey) ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return AcceptInvitationResult{}, ErrInputInvalid
	}
	if actor.InvitationAcceptanceOnly && actor.InvitationID != request.InvitationID {
		return AcceptInvitationResult{}, ErrInvitationNotFound
	}
	membershipID, err := service.generatedID("mem")
	if err != nil {
		return AcceptInvitationResult{}, ErrIdentityUnavailable
	}
	newSessionID, err := service.generatedID("ses")
	if err != nil {
		return AcceptInvitationResult{}, ErrIdentityUnavailable
	}
	cookieValue, verifierDigest, err := randomToken(service.entropy)
	if err != nil {
		return AcceptInvitationResult{}, ErrIdentityUnavailable
	}
	auditID, err := service.generatedID("aud")
	if err != nil {
		return AcceptInvitationResult{}, ErrIdentityUnavailable
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return AcceptInvitationResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	tokenDigest := sha256.Sum256([]byte(request.AcceptanceToken))
	idempotencyDigest := sha256.Sum256([]byte(request.IdempotencyKey))
	requestDigest := sha256.Sum256([]byte(
		"v1\ninvitation=" + request.InvitationID.String() +
			"\ntoken_sha256=" + hex.EncodeToString(tokenDigest[:]),
	))
	policy := service.sessionPolicies[PopulationMerchant]
	stored, err := service.organizations.AcceptInvitation(ctx, AcceptInvitationCommand{
		Actor: actor, InvitationID: request.InvitationID, TokenDigest: tokenDigest,
		MembershipID: membershipID, NewSessionID: newSessionID,
		NewSessionVerifierDigest: verifierDigest,
		IdempotencyDigest:        idempotencyDigest, RequestDigest: requestDigest,
		Now: now, IdleExpiresAt: now.Add(policy.Idle), AbsoluteExpiresAt: now.Add(policy.Absolute),
		AuditEvent: audit.Event{
			AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
			SessionAssurance: string(actor.Assurance), Action: "identity.organization.invitation.accept",
			TargetType: "invitation", TargetID: request.InvitationID.String(),
			DecisionID: decisionID, Decision: "executed",
			ReasonCode: "organization_invitation_accepted", CorrelationID: request.CorrelationID,
			OccurredAt: now, SafeBeforeReference: "invitation:pending",
			SafeAfterReference: "invitation:accepted",
		},
	})
	if err != nil {
		return AcceptInvitationResult{DecisionID: stored.DecisionID, Replay: stored.Replay}, err
	}
	result := AcceptInvitationResult{
		Member: stored.Member, Session: stored.Session,
		DecisionID: stored.DecisionID, Replay: stored.Replay,
	}
	if !stored.Replay {
		result.CookieValue = cookieValue
	}
	result.CSRFToken, err = service.csrf.Token(stored.Session.SessionID, stored.Session.RotationVersion)
	if err != nil {
		return AcceptInvitationResult{}, ErrIdentityUnavailable
	}
	return result, nil
}

func invitationEmail(value string) ([32]byte, string, error) {
	if value != strings.TrimSpace(value) || len(value) < 3 || len(value) > 254 ||
		strings.Count(value, "@") != 1 {
		return [32]byte{}, "", ErrValidationFailed
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return [32]byte{}, "", ErrValidationFailed
	}
	local, domain, found := strings.Cut(value, "@")
	if !found || local == "" || domain == "" {
		return [32]byte{}, "", ErrValidationFailed
	}
	normalized := local + "@" + strings.ToLower(domain)
	runes := []rune(local)
	hint := string(runes[0]) + "***@" + strings.ToLower(domain)
	return sha256.Sum256([]byte(normalized)), hint, nil
}

func validMerchantRole(role string) bool {
	switch role {
	case "merchant_viewer", "merchant_operator", "merchant_admin", "merchant_security_admin":
		return true
	default:
		return false
	}
}
