package identity

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type fakeOrganizationStore struct {
	organizations     []Organization
	membersCommand    ListOrganizationMembersCommand
	membersResult     ListOrganizationMembersResult
	membersErr        error
	command           SwitchOrganizationCommand
	result            Session
	err               error
	invitationCommand CreateInvitationCommand
	invitationResult  CreateInvitationStoreResult
	invitationErr     error
	acceptanceCommand AcceptInvitationCommand
	acceptanceResult  AcceptInvitationStoreResult
	acceptanceErr     error
	roleChangeCommand UpdateOrganizationMemberRoleCommand
	roleChangeResult  UpdateOrganizationMemberRoleResult
	roleChangeErr     error
	revocationCommand RevokeOrganizationMemberCommand
	revocationResult  RevokeOrganizationMemberResult
	revocationErr     error
}

func (store *fakeOrganizationStore) RevokeMember(
	_ context.Context,
	command RevokeOrganizationMemberCommand,
) (RevokeOrganizationMemberResult, error) {
	store.revocationCommand = command
	result := store.revocationResult
	if result.Member.MembershipID.IsZero() {
		revokedAt := command.Now
		result.Member = OrganizationMember{
			MembershipID: command.MembershipID, OrganizationID: command.OrganizationID,
			PrincipalID: command.Actor.PrincipalID, Role: "merchant_viewer",
			Status: "revoked", Version: command.ExpectedVersion + 1,
			CreatedAt: command.Now, RevokedAt: &revokedAt,
		}
	}
	if result.DecisionID.IsZero() {
		result.DecisionID = command.AuditEvent.DecisionID
	}
	return result, store.revocationErr
}

func (store *fakeOrganizationStore) UpdateMemberRole(
	_ context.Context,
	command UpdateOrganizationMemberRoleCommand,
) (UpdateOrganizationMemberRoleResult, error) {
	store.roleChangeCommand = command
	result := store.roleChangeResult
	if result.Member.MembershipID.IsZero() {
		result.Member = OrganizationMember{
			MembershipID: command.MembershipID, OrganizationID: command.OrganizationID,
			PrincipalID: command.Actor.PrincipalID, Role: command.Role,
			Status: "active", Version: command.ExpectedVersion + 1, CreatedAt: command.Now,
		}
	}
	if result.DecisionID.IsZero() {
		result.DecisionID = command.AuditEvent.DecisionID
	}
	return result, store.roleChangeErr
}

func (store *fakeOrganizationStore) AcceptInvitation(
	_ context.Context,
	command AcceptInvitationCommand,
) (AcceptInvitationStoreResult, error) {
	store.acceptanceCommand = command
	result := store.acceptanceResult
	if result.Member.MembershipID.IsZero() {
		result.Member = OrganizationMember{
			MembershipID: command.MembershipID, OrganizationID: command.Actor.TenantID,
			PrincipalID: command.Actor.PrincipalID, Role: "merchant_viewer",
			Status: "active", Version: 1, CreatedAt: command.Now,
		}
	}
	if result.Session.SessionID.IsZero() {
		result.Session = command.Actor
		result.Session.SessionID = command.NewSessionID
		result.Session.RotationVersion++
		result.Session.InvitationAcceptanceOnly = false
		result.Session.InvitationID = identifier.ID{}
		result.Session.IdleExpiresAt = command.IdleExpiresAt
		result.Session.AbsoluteExpiresAt = command.AbsoluteExpiresAt
	}
	if result.DecisionID.IsZero() {
		result.DecisionID = command.AuditEvent.DecisionID
	}
	return result, store.acceptanceErr
}

func (store *fakeOrganizationStore) ListMembers(
	_ context.Context,
	command ListOrganizationMembersCommand,
) (ListOrganizationMembersResult, error) {
	store.membersCommand = command
	result := store.membersResult
	if result.DecisionID.IsZero() {
		result.DecisionID = command.AuditEvent.DecisionID
	}
	return result, store.membersErr
}

func (store *fakeOrganizationStore) CreateInvitation(
	_ context.Context,
	command CreateInvitationCommand,
) (CreateInvitationStoreResult, error) {
	store.invitationCommand = command
	result := store.invitationResult
	if result.Invitation.InvitationID.IsZero() {
		result.Invitation = OrganizationInvitation{
			InvitationID: command.InvitationID, OrganizationID: command.OrganizationID,
			EmailHint: command.EmailHint, Role: command.Role, Status: "pending",
			ExpiresAt: command.ExpiresAt, CreatedAt: command.Now,
		}
	}
	if result.DecisionID.IsZero() {
		result.DecisionID = command.AuditEvent.DecisionID
	}
	return result, store.invitationErr
}

func (store *fakeOrganizationStore) ListOrganizations(
	context.Context,
	Session,
) ([]Organization, error) {
	return append([]Organization(nil), store.organizations...), store.err
}

func (store *fakeOrganizationStore) SwitchOrganization(
	_ context.Context,
	command SwitchOrganizationCommand,
) (Session, error) {
	store.command = command
	result := store.result
	result.SessionID = command.NewSessionID
	result.TenantID = command.OrganizationID
	result.RotationVersion = command.Actor.RotationVersion + 1
	result.CreatedAt = command.Now
	result.LastSeenAt = command.Now
	result.IdleExpiresAt = command.IdleExpiresAt
	result.AbsoluteExpiresAt = command.Actor.AbsoluteExpiresAt
	return result, store.err
}

func TestOrganizationListAndSwitchRotateTenantContext(t *testing.T) {
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.DisplayName = "Synthetic Merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 70)
	sessionStore.session.IdleExpiresAt = now.Add(20 * time.Minute)
	sessionStore.session.AbsoluteExpiresAt = now.Add(8 * time.Hour)
	sessionStore.session.Permissions = []string{"organization.list", "organization.active.switch"}
	cookie := strings.Repeat("M", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session

	target := mustTestID(t, "ten", 71)
	organizations := &fakeOrganizationStore{
		organizations: []Organization{{
			OrganizationID: target, DisplayName: "Synthetic Merchant Two",
			PrincipalRole: "merchant_operator", MembershipVersion: 3,
		}},
		result: sessionStore.session,
	}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizations

	listed, err := service.Organizations(context.Background(), cookie)
	if err != nil || len(listed) != 1 || listed[0].OrganizationID != target {
		t.Fatalf("organizations=%+v err=%v", listed, err)
	}
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.SwitchActiveOrganization(
		context.Background(),
		cookie,
		csrfToken,
		target,
		mustTestID(t, "cor", 72),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.CookieValue == cookie || len(result.CookieValue) != 43 ||
		result.Session.TenantID != target ||
		result.Session.RotationVersion != sessionStore.session.RotationVersion+1 ||
		result.Session.AbsoluteExpiresAt != sessionStore.session.AbsoluteExpiresAt ||
		result.CSRFToken == csrfToken ||
		result.DecisionID.IsZero() {
		t.Fatalf("unsafe tenant rotation: %+v", result)
	}
	command := organizations.command
	if command.Actor.SessionID != sessionStore.session.SessionID ||
		command.OrganizationID != target ||
		command.AuditEvent.Action != "identity.organization.active.switch" ||
		command.AuditEvent.DecisionID != result.DecisionID ||
		command.AuditEvent.TenantID != target ||
		command.AuditEvent.SafeBeforeReference == command.AuditEvent.SafeAfterReference {
		t.Fatalf("incomplete switch command: %+v", command)
	}
}

func TestOrganizationSwitchRejectsCSRFAndNonMerchantPopulation(t *testing.T) {
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	cookie := strings.Repeat("N", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = &fakeOrganizationStore{}
	target := mustTestID(t, "ten", 81)
	correlationID := mustTestID(t, "cor", 82)

	if _, err := service.SwitchActiveOrganization(
		context.Background(), cookie, strings.Repeat("X", 43), target, correlationID,
	); err != ErrCSRFValidationFailed {
		t.Fatalf("wrong CSRF error=%v", err)
	}
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SwitchActiveOrganization(
		context.Background(), cookie, csrfToken, target, correlationID,
	); err != ErrActionNotAuthorized {
		t.Fatalf("customer tenant-switch error=%v", err)
	}
}

func TestOrganizationMembersCarriesExplicitTenantAndAuthorizationDecision(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 83)
	cookie := strings.Repeat("L", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	membershipID := mustTestID(t, "mem", 84)
	principalID := mustTestID(t, "usr", 85)
	organizationStore := &fakeOrganizationStore{membersResult: ListOrganizationMembersResult{
		Page: OrganizationMemberPage{Members: []OrganizationMember{{
			MembershipID: membershipID, OrganizationID: sessionStore.session.TenantID,
			PrincipalID: principalID, Role: "merchant_viewer", Status: "active",
			Version: 1, CreatedAt: now,
		}}},
	}}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	result, err := service.OrganizationMembers(context.Background(), ListOrganizationMembersRequest{
		CookieValue: cookie, OrganizationID: sessionStore.session.TenantID,
		PageSize: "25", PageSizeProvided: true, CorrelationID: mustTestID(t, "cor", 86),
	})
	if err != nil || result.DecisionID.IsZero() || len(result.Page.Members) != 1 {
		t.Fatalf("member page=%+v err=%v", result, err)
	}
	command := organizationStore.membersCommand
	if command.Actor.SessionID != sessionStore.session.SessionID ||
		command.OrganizationID != sessionStore.session.TenantID || command.PageSize != "25" ||
		!command.PageSizeProvided ||
		command.AuditEvent.DecisionID != result.DecisionID ||
		command.AuditEvent.Action != "identity.organization.members.list" ||
		command.AuditEvent.TenantID != sessionStore.session.TenantID {
		t.Fatalf("incomplete member-list command: %+v", command)
	}
}

func TestUpdateOrganizationMemberRoleBindsPreconditionReplayAndAudit(t *testing.T) {
	now := time.Date(2026, 8, 8, 14, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 84)
	sessionStore.session.Permissions = []string{"organization.members.roles.update"}
	cookie := strings.Repeat("R", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	organizationStore := &fakeOrganizationStore{}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	membershipID := mustTestID(t, "mem", 85)
	result, err := service.UpdateOrganizationMemberRole(
		context.Background(), UpdateOrganizationMemberRoleRequest{
			CookieValue: cookie, CSRFToken: csrfToken,
			OrganizationID: sessionStore.session.TenantID, MembershipID: membershipID,
			Role: "merchant_operator", Purpose: "organization_administration",
			IfMatch: OrganizationMemberETag(3), IdempotencyKey: "role-change-key-0001",
			CorrelationID: mustTestID(t, "cor", 86),
		},
	)
	if err != nil || result.Member.Version != 4 || result.Member.Role != "merchant_operator" ||
		result.DecisionID.IsZero() || OrganizationMemberETag(result.Member.Version) != "\"membership-v4\"" {
		t.Fatalf("role change result=%+v err=%v", result, err)
	}
	command := organizationStore.roleChangeCommand
	if command.Actor.SessionID != sessionStore.session.SessionID ||
		command.OrganizationID != sessionStore.session.TenantID ||
		command.MembershipID != membershipID || command.ExpectedVersion != 3 ||
		command.Role != "merchant_operator" || command.IdempotencyDigest == ([32]byte{}) ||
		command.RequestDigest == ([32]byte{}) || command.AuditEvent.Action !=
		"identity.organization.membership.role.change" ||
		command.AuditEvent.TargetID != membershipID.String() ||
		command.AuditEvent.DecisionID != result.DecisionID ||
		command.AuditEvent.SafeAfterReference != "membership-role:merchant_operator" {
		t.Fatalf("role-change command=%+v", command)
	}
}

func TestUpdateOrganizationMemberRoleRejectsProtocolAndScopeBeforeStore(t *testing.T) {
	now := time.Date(2026, 8, 8, 14, 30, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 87)
	cookie := strings.Repeat("S", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	organizationStore := &fakeOrganizationStore{}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	base := UpdateOrganizationMemberRoleRequest{
		CookieValue: cookie, CSRFToken: csrfToken,
		OrganizationID: sessionStore.session.TenantID,
		MembershipID:   mustTestID(t, "mem", 88), Role: "merchant_viewer",
		Purpose: "organization_administration", IfMatch: OrganizationMemberETag(1),
		IdempotencyKey: "role-change-key-0002", CorrelationID: mustTestID(t, "cor", 89),
	}
	invalidCSRF := base
	invalidCSRF.CSRFToken = strings.Repeat("X", 43)
	if _, err := service.UpdateOrganizationMemberRole(context.Background(), invalidCSRF); !errors.Is(err, ErrCSRFValidationFailed) {
		t.Fatalf("invalid CSRF error=%v", err)
	}
	invalidPurpose := base
	invalidPurpose.Purpose = "self_service"
	if _, err := service.UpdateOrganizationMemberRole(context.Background(), invalidPurpose); !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("invalid purpose error=%v", err)
	}
	for _, invalidETag := range []string{"", "*", "membership-v1", "W/\"membership-v1\"", "\"membership-v01\""} {
		request := base
		request.IfMatch = invalidETag
		if _, err := service.UpdateOrganizationMemberRole(context.Background(), request); !errors.Is(err, ErrInputInvalid) {
			t.Fatalf("invalid If-Match %q error=%v", invalidETag, err)
		}
	}
	if !organizationStore.roleChangeCommand.RoleChangeID.IsZero() {
		t.Fatal("invalid role change reached the persistence boundary")
	}
}

func TestRevokeOrganizationMemberBindsPreconditionReplayAndAudit(t *testing.T) {
	now := time.Date(2026, 8, 9, 9, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 130)
	sessionStore.session.Permissions = []string{"organization.members.remove"}
	cookie := strings.Repeat("T", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	organizationStore := &fakeOrganizationStore{}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	membershipID := mustTestID(t, "mem", 131)
	result, err := service.RevokeOrganizationMember(
		context.Background(), RevokeOrganizationMemberRequest{
			CookieValue: cookie, CSRFToken: csrfToken,
			OrganizationID: sessionStore.session.TenantID, MembershipID: membershipID,
			IfMatch: OrganizationMemberETag(4), IdempotencyKey: "member-revoke-key-0001",
			CorrelationID: mustTestID(t, "cor", 132),
		},
	)
	if err != nil || result.Member.Version != 5 || result.Member.Status != "revoked" ||
		result.Member.RevokedAt == nil || result.DecisionID.IsZero() {
		t.Fatalf("member revocation result=%+v err=%v", result, err)
	}
	command := organizationStore.revocationCommand
	if command.Actor.SessionID != sessionStore.session.SessionID ||
		command.OrganizationID != sessionStore.session.TenantID ||
		command.MembershipID != membershipID || command.ExpectedVersion != 4 ||
		command.IdempotencyDigest == ([32]byte{}) || command.RequestDigest == ([32]byte{}) ||
		command.AuditEvent.Action != "identity.organization.membership.revoke" ||
		command.AuditEvent.TargetID != membershipID.String() ||
		command.AuditEvent.DecisionID != result.DecisionID ||
		command.AuditEvent.SafeAfterReference != "membership-status:revoked" {
		t.Fatalf("member-revocation command=%+v", command)
	}
}

func TestRevokeOrganizationMemberRejectsProtocolBeforeStore(t *testing.T) {
	now := time.Date(2026, 8, 9, 9, 30, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 133)
	cookie := strings.Repeat("U", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	organizationStore := &fakeOrganizationStore{}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	base := RevokeOrganizationMemberRequest{
		CookieValue: cookie, CSRFToken: csrfToken,
		OrganizationID: sessionStore.session.TenantID,
		MembershipID:   mustTestID(t, "mem", 134),
		IfMatch:        OrganizationMemberETag(1),
		IdempotencyKey: "member-revoke-key-0002", CorrelationID: mustTestID(t, "cor", 135),
	}
	invalidCSRF := base
	invalidCSRF.CSRFToken = strings.Repeat("X", 43)
	if _, err := service.RevokeOrganizationMember(context.Background(), invalidCSRF); !errors.Is(err, ErrCSRFValidationFailed) {
		t.Fatalf("invalid CSRF error=%v", err)
	}
	for _, invalidETag := range []string{"", "*", "membership-v1", "W/\"membership-v1\"", "\"membership-v01\""} {
		request := base
		request.IfMatch = invalidETag
		if _, err := service.RevokeOrganizationMember(context.Background(), request); !errors.Is(err, ErrInputInvalid) {
			t.Fatalf("invalid If-Match %q error=%v", invalidETag, err)
		}
	}
	if !organizationStore.revocationCommand.RevocationID.IsZero() {
		t.Fatal("invalid membership revocation reached the persistence boundary")
	}
}

func TestCreateOrganizationInvitationDisclosesSecretOnceAndStoresOnlyDigests(t *testing.T) {
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 91)
	sessionStore.session.IdleExpiresAt = now.Add(20 * time.Minute)
	sessionStore.session.AbsoluteExpiresAt = now.Add(8 * time.Hour)
	cookie := strings.Repeat("I", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	organizationStore := &fakeOrganizationStore{}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	request := CreateInvitationRequest{
		CookieValue: cookie, CSRFToken: csrfToken,
		OrganizationID: sessionStore.session.TenantID,
		Email:          "Invitee@Example.test", Role: "merchant_viewer",
		IdempotencyKey: "invitation-create-0001", CorrelationID: mustTestID(t, "cor", 92),
	}
	first, err := service.CreateOrganizationInvitation(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	command := organizationStore.invitationCommand
	if !first.SecretDisclosed || len(first.AcceptanceToken) != 43 || first.Replay ||
		first.Invitation.EmailHint == request.Email || first.Invitation.EmailHint != "I***@example.test" ||
		command.ExpiresAt.Sub(command.Now) != 72*time.Hour ||
		command.AuditEvent.Action != "identity.organization.invitation.create" ||
		command.AuditEvent.TargetID != first.Invitation.InvitationID.String() ||
		sha256.Sum256([]byte(first.AcceptanceToken)) != command.TokenDigest {
		t.Fatalf("unsafe invitation issuance result=%+v command=%+v", first, command)
	}
	if command.EmailDigest == sha256.Sum256([]byte(request.Email)) {
		t.Fatal("email domain was not canonicalized before hashing")
	}
	organizationStore.invitationResult = CreateInvitationStoreResult{
		Invitation: first.Invitation, DecisionID: first.DecisionID, Replay: true,
	}
	replayed, err := service.CreateOrganizationInvitation(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replay || replayed.SecretDisclosed || replayed.AcceptanceToken != "" ||
		replayed.Invitation != first.Invitation || replayed.DecisionID != first.DecisionID {
		t.Fatalf("secret-producing replay was not redacted: %+v", replayed)
	}
}

func TestCreateOrganizationInvitationRejectsInvalidRecipientAndRole(t *testing.T) {
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	sessionStore.session.Population = PopulationMerchant
	sessionStore.session.PrincipalType = "merchant"
	sessionStore.session.TenantID = mustTestID(t, "ten", 101)
	cookie := strings.Repeat("J", 43)
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = sessionStore.session
	organizationStore := &fakeOrganizationStore{}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	request := CreateInvitationRequest{
		CookieValue: cookie, CSRFToken: csrfToken,
		OrganizationID: sessionStore.session.TenantID,
		Email:          "not an email", Role: "merchant_owner",
		IdempotencyKey: "invitation-create-0002", CorrelationID: mustTestID(t, "cor", 102),
	}
	if _, err := service.CreateOrganizationInvitation(context.Background(), request); !errors.Is(err, ErrValidationFailed) {
		t.Fatalf("invalid invitation error=%v", err)
	}
	if !organizationStore.invitationCommand.InvitationID.IsZero() {
		t.Fatal("invalid invitation reached persistence")
	}
}

func TestAcceptOrganizationInvitationRequiresBoundBootstrapAndRotatesAuthorityAtomically(t *testing.T) {
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	sessionStore := newFakeSessionStore(t, now)
	invitationID := mustTestID(t, "inv", 111)
	tenantID := mustTestID(t, "ten", 112)
	cookie := strings.Repeat("B", 43)
	bootstrap := sessionStore.session
	bootstrap.Population = PopulationMerchant
	bootstrap.PrincipalType = "merchant"
	bootstrap.TenantID = identifier.ID{}
	bootstrap.Permissions = []string{}
	bootstrap.InvitationID = invitationID
	bootstrap.VerifiedEmailDigest = sha256.Sum256([]byte("recipient@example.test"))
	bootstrap.InvitationAcceptanceOnly = true
	bootstrap.IdleExpiresAt = now.Add(invitationAcceptanceSessionLifetime)
	bootstrap.AbsoluteExpiresAt = bootstrap.IdleExpiresAt
	sessionStore.sessions[sha256.Sum256([]byte(cookie))] = bootstrap
	organizationStore := &fakeOrganizationStore{acceptanceResult: AcceptInvitationStoreResult{
		Member: OrganizationMember{
			MembershipID: mustTestID(t, "mem", 113), OrganizationID: tenantID,
			PrincipalID: bootstrap.PrincipalID, Role: "merchant_viewer",
			Status: "active", Version: 1, CreatedAt: now,
		},
	}}
	service := newTestService(t, sessionStore, &fakeProvider{}, now)
	service.organizations = organizationStore
	_, csrfToken, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	acceptanceToken := strings.Repeat("A", 43)
	result, err := service.AcceptOrganizationInvitation(context.Background(), AcceptInvitationRequest{
		CookieValue: cookie, CSRFToken: csrfToken, InvitationID: invitationID,
		AcceptanceToken: acceptanceToken, IdempotencyKey: "invitation-accept-0001",
		CorrelationID: mustTestID(t, "cor", 114),
	})
	if err != nil {
		t.Fatal(err)
	}
	command := organizationStore.acceptanceCommand
	if result.Replay || len(result.CookieValue) != 43 || result.CookieValue == cookie ||
		result.Member.OrganizationID != tenantID || result.DecisionID.IsZero() ||
		command.Actor.SessionID != bootstrap.SessionID ||
		command.InvitationID != invitationID ||
		command.TokenDigest != sha256.Sum256([]byte(acceptanceToken)) ||
		command.IdempotencyDigest != sha256.Sum256([]byte("invitation-accept-0001")) ||
		command.MembershipID.Prefix() != "mem" || command.NewSessionID.Prefix() != "ses" ||
		command.IdleExpiresAt.Sub(command.Now) != DefaultSessionPolicies[PopulationMerchant].Idle ||
		command.AbsoluteExpiresAt.Sub(command.Now) != DefaultSessionPolicies[PopulationMerchant].Absolute ||
		command.AuditEvent.Action != "identity.organization.invitation.accept" ||
		command.AuditEvent.TargetID != invitationID.String() {
		t.Fatalf("unsafe invitation acceptance result=%+v command=%+v", result, command)
	}

	wrongInvitation := mustTestID(t, "inv", 115)
	if _, err := service.AcceptOrganizationInvitation(context.Background(), AcceptInvitationRequest{
		CookieValue: cookie, CSRFToken: csrfToken, InvitationID: wrongInvitation,
		AcceptanceToken: acceptanceToken, IdempotencyKey: "invitation-accept-0002",
		CorrelationID: mustTestID(t, "cor", 116),
	}); !errors.Is(err, ErrInvitationNotFound) {
		t.Fatalf("cross-invitation bootstrap error=%v", err)
	}
}
