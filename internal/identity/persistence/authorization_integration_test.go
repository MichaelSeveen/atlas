package persistence

import (
	"context"
	"crypto/sha256"
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

func TestAuthorizationRealPostgresMultiReplicaRoleRestrictionAndAssuranceInvalidationWithoutCache(t *testing.T) {
	apiURL := os.Getenv("ATLAS_P01_DATABASE_URL")
	migrationURL := os.Getenv("ATLAS_P01_MIGRATION_DATABASE_URL")
	if apiURL == "" || migrationURL == "" {
		t.Skip("real Phase 01 database URLs are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	apiPoolA, err := pgxpool.New(ctx, apiURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(apiPoolA.Close)
	apiPoolB, err := pgxpool.New(ctx, apiURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(apiPoolB.Close)
	migrationPool, err := pgxpool.New(ctx, migrationURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(migrationPool.Close)
	storeA, err := NewOrganizationStore(apiPoolA, auditapplication.NewRecorder())
	if err != nil {
		t.Fatal(err)
	}
	storeB, err := NewOrganizationStore(apiPoolB, auditapplication.NewRecorder())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	tenantID := newIntegrationID(t, "ten")
	actorPrincipalID := newIntegrationID(t, "usr")
	listedPrincipalID := newIntegrationID(t, "usr")
	actorMembershipID := newIntegrationID(t, "mem")
	listedMembershipID := newIntegrationID(t, "mem")
	staleSessionID := newIntegrationID(t, "ses")
	currentSessionID := newIntegrationID(t, "ses")
	staleVerifier := sha256.Sum256([]byte("authorization-stale-" + staleSessionID.String()))
	currentVerifier := sha256.Sum256([]byte("authorization-current-" + currentSessionID.String()))
	auditIDs := make([]string, 0, 2)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if len(auditIDs) > 0 {
			_, _ = migrationPool.Exec(cleanupCtx,
				`DELETE FROM atlas_audit.audit_events WHERE audit_event_id = ANY($1)`, auditIDs)
		}
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.sessions WHERE session_id = ANY($1)`,
			[]string{staleSessionID.String(), currentSessionID.String()})
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.memberships WHERE membership_id = ANY($1)`,
			[]string{actorMembershipID.String(), listedMembershipID.String()})
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.principals WHERE principal_id = ANY($1)`,
			[]string{actorPrincipalID.String(), listedPrincipalID.String()})
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.organizations WHERE tenant_id = $1`, tenantID.String())
	})

	setup, err := migrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = setup.Rollback(ctx) }()
	uniqueName := "authorization-" + strings.ToLower(strings.TrimPrefix(tenantID.String(), "ten_"))
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.organizations (
    tenant_id, organization_type, display_name, normalized_name, confusable_skeleton,
    status, authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', 'Authorization Replica Fixture', $2, $2, 'active', 1, 1, $3, $3)`,
		tenantID.String(), uniqueName, now,
	); err != nil {
		t.Fatal(err)
	}
	for index, principalID := range []identifier.ID{actorPrincipalID, listedPrincipalID} {
		anchor := "syn_person_authorization_" + strings.ToLower(strings.TrimPrefix(principalID.String(), "usr_"))
		if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', $2, $3, 'active', 1, 1, $4, $4)`,
			principalID.String(), "Authorization Principal "+string(rune('A'+index)), anchor, now,
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, $2, $3, 'merchant_security_admin', 'merchant', 'active', 1, 1, $4, $4)`,
		actorMembershipID.String(), tenantID.String(), actorPrincipalID.String(), now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at, email_hint
) VALUES ($1, $2, $3, 'merchant_viewer', 'merchant', 'active', 1, 1, $4, $4, 'r***@example.invalid')`,
		listedMembershipID.String(), tenantID.String(), listedPrincipalID.String(), now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.sessions (
    session_id, principal_id, population, tenant_id, verifier_sha256, assurance,
    status, authorization_version, rotation_version, version, created_at,
    last_seen_at, idle_expires_at, absolute_expires_at
) VALUES ($1, $2, 'merchant', $3, $4, 'baseline', 'active', 1, 1, 1, $5, $5, $6, $7)`,
		staleSessionID.String(), actorPrincipalID.String(), tenantID.String(), staleVerifier[:],
		now, now.Add(20*time.Minute), now.Add(8*time.Hour),
	); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	staleActor := identity.Session{
		SessionID: staleSessionID, PrincipalID: actorPrincipalID, PrincipalType: "merchant",
		DisplayName: "Authorization Principal A", Population: identity.PopulationMerchant,
		TenantID: tenantID, Assurance: identity.AssuranceBaseline,
		AuthorizationVersion: 1, RotationVersion: 1,
		CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(20 * time.Minute),
		AbsoluteExpiresAt: now.Add(8 * time.Hour),
	}
	eventFor := func() audit.Event {
		auditID := newIntegrationID(t, "aud")
		auditIDs = append(auditIDs, auditID.String())
		return audit.Event{
			AuditEventID: auditID, ActorID: actorPrincipalID, ActorType: "merchant",
			TenantID: tenantID, SessionAssurance: "baseline",
			Action: "identity.organization.members.list", TargetType: "organization",
			TargetID: tenantID.String(), DecisionID: newIntegrationID(t, "dec"),
			Decision: "allowed", ReasonCode: "organization_members_listed",
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now,
			SafeAfterReference: "organization-members:masked",
		}
	}
	assertHint := func(store *OrganizationStore, actor identity.Session, wantHint bool) {
		t.Helper()
		event := eventFor()
		result, callErr := store.ListMembers(ctx, identity.ListOrganizationMembersCommand{
			Actor: actor, OrganizationID: tenantID, Purpose: "organization_administration",
			PageSize: "10", PageSizeProvided: true, Now: now.Add(3 * time.Minute), AuditEvent: event,
		})
		if callErr != nil || result.DecisionID != event.DecisionID || len(result.Page.Members) != 2 {
			t.Fatalf("replica member list=%+v err=%v", result, callErr)
		}
		foundHint := false
		for _, member := range result.Page.Members {
			if member.PrincipalID == listedPrincipalID && member.EmailHint != nil &&
				*member.EmailHint == "r***@example.invalid" {
				foundHint = true
			}
		}
		if foundHint != wantHint {
			t.Fatalf("replica masked-hint disclosure=%v want=%v members=%+v", foundHint, wantHint, result.Page.Members)
		}
	}
	assertHint(storeA, staleActor, true)
	assertHint(storeB, staleActor, true)
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.organizations
SET status = 'disabled', version = version + 1, updated_at = $2
WHERE tenant_id = $1`, tenantID.String(), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	for replica, store := range []*OrganizationStore{storeA, storeB} {
		_, callErr := store.ListMembers(ctx, identity.ListOrganizationMembersCommand{
			Actor: staleActor, OrganizationID: tenantID, Purpose: "organization_administration",
			PageSize: "10", PageSizeProvided: true, Now: now.Add(time.Minute), AuditEvent: eventFor(),
		})
		if !errors.Is(callErr, identity.ErrAuthenticationRequired) {
			t.Fatalf("replica %d retained authority for disabled resource: %v", replica, callErr)
		}
	}
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.organizations
SET status = 'active', version = version + 1, updated_at = $2
WHERE tenant_id = $1`, tenantID.String(), now.Add(90*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.sessions
SET assurance = 'phishing-resistant', version = version + 1
WHERE session_id = $1`, staleSessionID.String()); err != nil {
		t.Fatal(err)
	}
	for replica, store := range []*OrganizationStore{storeA, storeB} {
		_, callErr := store.ListMembers(ctx, identity.ListOrganizationMembersCommand{
			Actor: staleActor, OrganizationID: tenantID, Purpose: "organization_administration",
			PageSize: "10", PageSizeProvided: true, Now: now.Add(time.Minute), AuditEvent: eventFor(),
		})
		if !errors.Is(callErr, identity.ErrAuthenticationRequired) {
			t.Fatalf("replica %d retained stale session assurance: %v", replica, callErr)
		}
	}
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.sessions
SET assurance = 'baseline', version = version + 1
WHERE session_id = $1`, staleSessionID.String()); err != nil {
		t.Fatal(err)
	}

	change, err := migrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = change.Rollback(ctx) }()
	if _, err := change.Exec(ctx, `
UPDATE atlas_identity.memberships
SET role_id = 'merchant_viewer', authorization_version = 2, version = version + 1, updated_at = $2
WHERE membership_id = $1`, actorMembershipID.String(), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := change.Exec(ctx, `
UPDATE atlas_identity.principals
SET authorization_version = 2, version = version + 1, updated_at = $2
WHERE principal_id = $1`, actorPrincipalID.String(), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := change.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $2, version = version + 1
WHERE session_id = $1`, staleSessionID.String(), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := change.Exec(ctx, `
INSERT INTO atlas_identity.sessions (
    session_id, principal_id, population, tenant_id, verifier_sha256, assurance,
    status, authorization_version, rotation_version, version, created_at,
    last_seen_at, idle_expires_at, absolute_expires_at
) VALUES ($1, $2, 'merchant', $3, $4, 'baseline', 'active', 2, 2, 1, $5, $5, $6, $7)`,
		currentSessionID.String(), actorPrincipalID.String(), tenantID.String(), currentVerifier[:],
		now.Add(2*time.Minute), now.Add(22*time.Minute), now.Add(8*time.Hour),
	); err != nil {
		t.Fatal(err)
	}
	if err := change.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	for replica, store := range []*OrganizationStore{storeA, storeB} {
		_, callErr := store.ListMembers(ctx, identity.ListOrganizationMembersCommand{
			Actor: staleActor, OrganizationID: tenantID, Purpose: "organization_administration",
			PageSize: "10", PageSizeProvided: true, Now: now.Add(3 * time.Minute), AuditEvent: eventFor(),
		})
		if !errors.Is(callErr, identity.ErrAuthenticationRequired) {
			t.Fatalf("replica %d retained stale role authority: %v", replica, callErr)
		}
	}
	currentActor := staleActor
	currentActor.SessionID = currentSessionID
	currentActor.AuthorizationVersion = 2
	currentActor.RotationVersion = 2
	currentActor.CreatedAt = now.Add(2 * time.Minute)
	currentActor.LastSeenAt = now.Add(2 * time.Minute)
	currentActor.IdleExpiresAt = now.Add(22 * time.Minute)
	assertHint(storeA, currentActor, false)
	assertHint(storeB, currentActor, false)
}
