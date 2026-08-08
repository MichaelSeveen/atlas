package persistence

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	auditapplication "github.com/MichaelSeveen/atlas/internal/audit/application"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestOrganizationStoreRealPostgresListAndZeroGraceSwitch(t *testing.T) {
	apiURL := os.Getenv("ATLAS_P01_DATABASE_URL")
	migrationURL := os.Getenv("ATLAS_P01_MIGRATION_DATABASE_URL")
	if apiURL == "" || migrationURL == "" {
		t.Skip("real Phase 01 database URLs are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	apiPool, err := pgxpool.New(ctx, apiURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(apiPool.Close)
	migrationPool, err := pgxpool.New(ctx, migrationURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(migrationPool.Close)
	recorder := auditapplication.NewRecorder()
	sessionStore, err := NewSessionStore(apiPool, recorder)
	if err != nil {
		t.Fatal(err)
	}
	organizationStore, err := NewOrganizationStore(apiPool, recorder)
	if err != nil {
		t.Fatal(err)
	}

	randomTenant := newIntegrationID(t, "ten").String()
	targetTenant, err := identifier.Parse(
		"ten_Z" + strings.TrimPrefix(randomTenant, "ten_")[1:],
	)
	if err != nil {
		t.Fatal(err)
	}
	targetMembership := newIntegrationID(t, "mem")
	listedPrincipal := newIntegrationID(t, "usr")
	listedMembership := newIntegrationID(t, "mem")
	oldSessionID := newIntegrationID(t, "ses")
	newSessionID := newIntegrationID(t, "ses")
	failedSessionID := newIntegrationID(t, "ses")
	loginAuditID := newIntegrationID(t, "aud")
	switchAuditID := newIntegrationID(t, "aud")
	memberListDenialAuditID := newIntegrationID(t, "aud")
	loginCorrelationID := newIntegrationID(t, "cor")
	switchCorrelationID := newIntegrationID(t, "cor")
	now := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)
	uniqueName := "synthetic-" + targetTenant.String()
	setup, err := migrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = setup.Rollback(ctx) }()
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.organizations (
    tenant_id, organization_type, display_name, normalized_name, confusable_skeleton,
    status, authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', 'Synthetic Merchant Two', $2, $2, 'active', 1, 1, $3, $3)`,
		targetTenant.String(), uniqueName, now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES (
    $2, $1, 'usr_01JAT1AS00000000000002', 'merchant_operator', 'merchant',
    'active', 1, 1, $3, $3
)`,
		targetTenant.String(), targetMembership.String(), now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', 'Synthetic Listed Merchant', $2, 'active', 1, 1, $3, $3)`,
		listedPrincipal.String(), "syn_person_"+strings.ToLower(strings.TrimPrefix(listedPrincipal.String(), "usr_")), now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, $2, $3, 'merchant_viewer', 'merchant', 'active', 1, 1, $4, $4)`,
		listedMembership.String(), targetTenant.String(), listedPrincipal.String(), now,
	); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = migrationPool.Exec(
			cleanupCtx,
			`DELETE FROM atlas_audit.audit_events WHERE audit_event_id = ANY($1)`,
			[]string{loginAuditID.String(), switchAuditID.String(), memberListDenialAuditID.String()},
		)
		_, _ = migrationPool.Exec(
			cleanupCtx,
			`DELETE FROM atlas_identity.sessions WHERE session_id = ANY($1)`,
			[]string{oldSessionID.String(), newSessionID.String(), failedSessionID.String()},
		)
		_, _ = migrationPool.Exec(
			cleanupCtx,
			`DELETE FROM atlas_identity.memberships WHERE membership_id = ANY($1)`,
			[]string{targetMembership.String(), listedMembership.String()},
		)
		_, _ = migrationPool.Exec(
			cleanupCtx,
			`DELETE FROM atlas_identity.principals WHERE principal_id = $1`,
			listedPrincipal.String(),
		)
		_, _ = migrationPool.Exec(
			cleanupCtx,
			`DELETE FROM atlas_identity.organizations WHERE tenant_id = $1`,
			targetTenant.String(),
		)
	})

	_, oldDigest := integrationToken(t)
	oldSession, err := sessionStore.CreateSession(ctx, identity.CreateSessionCommand{
		Claims: identity.ProviderClaims{
			Issuer:    "http://keycloak:8080/realms/atlas-merchant-local",
			Subject:   "00000000-0000-4000-8000-000000000201",
			Assurance: identity.AssuranceBaseline, AuthenticatedAt: now,
		},
		Population: identity.PopulationMerchant, Kind: identity.TransactionLogin,
		SessionID: oldSessionID, VerifierDigest: oldDigest,
		Assurance: identity.AssuranceBaseline, AuthorizationAt: now,
		IdleExpiresAt: now.Add(20 * time.Minute), AbsoluteExpiresAt: now.Add(8 * time.Hour),
		ClientLabel: "integration-merchant-browser",
		AuditEvent: audit.Event{
			AuditEventID: loginAuditID, SessionAssurance: "baseline",
			Action: "identity.session.login", TargetType: "session",
			TargetID: oldSessionID.String(), DecisionID: newIntegrationID(t, "dec"),
			Decision: "executed", ReasonCode: "oidc_login",
			CorrelationID: loginCorrelationID, OccurredAt: now,
			GlobalScope: "identity-security",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	organizations, err := organizationStore.ListOrganizations(ctx, oldSession)
	if err != nil || len(organizations) != 2 {
		t.Fatalf("organizations=%+v err=%v", organizations, err)
	}

	_, newDigest := integrationToken(t)
	switchEvent := audit.Event{
		AuditEventID: switchAuditID, ActorID: oldSession.PrincipalID,
		ActorType: "merchant", TenantID: targetTenant, SessionAssurance: "baseline",
		Action: "identity.organization.active.switch", TargetType: "organization",
		TargetID: targetTenant.String(), DecisionID: newIntegrationID(t, "dec"),
		Decision: "executed", ReasonCode: "active_organization_selected",
		CorrelationID: switchCorrelationID, OccurredAt: now.Add(time.Minute),
		SafeBeforeReference: "organization:" + oldSession.TenantID.String(),
		SafeAfterReference:  "organization:" + targetTenant.String(),
	}
	rotated, err := organizationStore.SwitchOrganization(ctx, identity.SwitchOrganizationCommand{
		Actor: oldSession, OrganizationID: targetTenant, NewSessionID: newSessionID,
		VerifierDigest: newDigest, Now: now.Add(time.Minute),
		IdleExpiresAt: now.Add(21 * time.Minute), AuditEvent: switchEvent,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rotated.TenantID != targetTenant ||
		rotated.RotationVersion != oldSession.RotationVersion+1 ||
		!rotated.AbsoluteExpiresAt.Equal(oldSession.AbsoluteExpiresAt) {
		t.Fatalf("rotated session=%+v", rotated)
	}
	if _, err := sessionStore.Authenticate(ctx, oldDigest, now.Add(2*time.Minute), 20*time.Minute); !errors.Is(err, identity.ErrSessionRevoked) {
		t.Fatalf("old session retained grace: %v", err)
	}
	authenticated, err := sessionStore.Authenticate(ctx, newDigest, now.Add(2*time.Minute), 20*time.Minute)
	if err != nil || authenticated.TenantID != targetTenant {
		t.Fatalf("new session=%+v err=%v", authenticated, err)
	}
	memberListEvent := audit.Event{
		AuditEventID: newIntegrationID(t, "aud"), ActorID: authenticated.PrincipalID,
		ActorType: "merchant", TenantID: targetTenant, SessionAssurance: "baseline",
		Action: "identity.organization.members.list", TargetType: "organization",
		TargetID: targetTenant.String(), DecisionID: newIntegrationID(t, "dec"),
		Decision: "allowed", ReasonCode: "organization_members_listed",
		CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now.Add(2 * time.Minute),
		SafeAfterReference: "organization-members:masked",
	}
	firstPage, err := organizationStore.ListMembers(ctx, identity.ListOrganizationMembersCommand{
		Actor: authenticated, OrganizationID: targetTenant, PageSize: "1", PageSizeProvided: true,
		Now: now.Add(2 * time.Minute), AuditEvent: memberListEvent,
	})
	if err != nil || len(firstPage.Page.Members) != 1 || !firstPage.Page.HasMore ||
		firstPage.Page.NextCursor == "" || firstPage.DecisionID != memberListEvent.DecisionID {
		t.Fatalf("first member page=%+v err=%v", firstPage, err)
	}
	secondListEvent := memberListEvent
	secondListEvent.AuditEventID = newIntegrationID(t, "aud")
	secondListEvent.DecisionID = newIntegrationID(t, "dec")
	secondListEvent.CorrelationID = newIntegrationID(t, "cor")
	secondPage, err := organizationStore.ListMembers(ctx, identity.ListOrganizationMembersCommand{
		Actor: authenticated, OrganizationID: targetTenant, PageSize: "1", PageSizeProvided: true,
		Cursor: firstPage.Page.NextCursor, CursorProvided: true, Now: now.Add(2 * time.Minute),
		AuditEvent: secondListEvent,
	})
	if err != nil || len(secondPage.Page.Members) != 1 || secondPage.Page.HasMore ||
		secondPage.Page.NextCursor != "" {
		t.Fatalf("second member page=%+v err=%v", secondPage, err)
	}
	listed := append(firstPage.Page.Members, secondPage.Page.Members...)
	if listed[0].PrincipalID == listed[1].PrincipalID {
		t.Fatal("member pagination repeated a principal")
	}
	for _, member := range listed {
		if member.OrganizationID != targetTenant || member.EmailHint != nil {
			t.Fatalf("cross-tenant or sensitive member field: %+v", member)
		}
	}
	denialEvent := memberListEvent
	denialEvent.AuditEventID = memberListDenialAuditID
	denialEvent.DecisionID = newIntegrationID(t, "dec")
	denialEvent.CorrelationID = newIntegrationID(t, "cor")
	denialEvent.TargetID = oldSession.TenantID.String()
	denied, err := organizationStore.ListMembers(ctx, identity.ListOrganizationMembersCommand{
		Actor: authenticated, OrganizationID: oldSession.TenantID,
		PageSize: "1", PageSizeProvided: true,
		Cursor: "malformed-cursor", CursorProvided: true,
		Now: now.Add(2 * time.Minute), AuditEvent: denialEvent,
	})
	if !errors.Is(err, identity.ErrOrganizationNotFound) || denied.DecisionID != denialEvent.DecisionID {
		t.Fatalf("cross-tenant member list=%+v err=%v", denied, err)
	}
	var memberListDenialCount int
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_audit.audit_events WHERE audit_event_id = $1`,
		memberListDenialAuditID.String(),
	).Scan(&memberListDenialCount); err != nil || memberListDenialCount != 1 {
		t.Fatalf("member-list denial audit count=%d err=%v", memberListDenialCount, err)
	}
	_, failedDigest := integrationToken(t)
	failingStore, err := NewOrganizationStore(apiPool, failingAuditRecorder{})
	if err != nil {
		t.Fatal(err)
	}
	failedAudit := switchEvent
	failedAudit.AuditEventID = newIntegrationID(t, "aud")
	failedAudit.DecisionID = newIntegrationID(t, "dec")
	failedAudit.TargetID = oldSession.TenantID.String()
	failedAudit.TenantID = oldSession.TenantID
	failedAudit.CorrelationID = newIntegrationID(t, "cor")
	failedAudit.OccurredAt = now.Add(3 * time.Minute)
	failedAudit.SafeBeforeReference = "organization:" + targetTenant.String()
	failedAudit.SafeAfterReference = "organization:" + oldSession.TenantID.String()
	if _, err := failingStore.SwitchOrganization(ctx, identity.SwitchOrganizationCommand{
		Actor: authenticated, OrganizationID: oldSession.TenantID, NewSessionID: failedSessionID,
		VerifierDigest: failedDigest, Now: now.Add(3 * time.Minute),
		IdleExpiresAt: now.Add(23 * time.Minute), AuditEvent: failedAudit,
	}); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("Audit outage switch error=%v", err)
	}
	if _, err := sessionStore.Authenticate(ctx, newDigest, now.Add(4*time.Minute), 20*time.Minute); err != nil {
		t.Fatalf("Audit outage revoked the current session: %v", err)
	}
	if _, err := sessionStore.Authenticate(ctx, failedDigest, now.Add(4*time.Minute), 20*time.Minute); !errors.Is(err, identity.ErrAuthenticationRequired) {
		t.Fatalf("Audit outage committed a replacement session: %v", err)
	}
	var auditCount int
	if err := migrationPool.QueryRow(
		ctx,
		`SELECT count(*) FROM atlas_audit.audit_events WHERE audit_event_id = $1`,
		switchAuditID.String(),
	).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("switch audit count=%d err=%v", auditCount, err)
	}
}
