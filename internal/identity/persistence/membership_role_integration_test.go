package persistence

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	auditapplication "github.com/MichaelSeveen/atlas/internal/audit/application"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestMembershipRoleChangeRealPostgresConcurrencyAuthorityAndAuditRollback(t *testing.T) {
	apiURL := os.Getenv("ATLAS_P01_DATABASE_URL")
	migrationURL := os.Getenv("ATLAS_P01_MIGRATION_DATABASE_URL")
	if apiURL == "" || migrationURL == "" {
		t.Skip("real Phase 01 database URLs are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
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
	store, err := NewOrganizationStore(apiPool, auditapplication.NewRecorder())
	if err != nil {
		t.Fatal(err)
	}
	failingStore, err := NewOrganizationStore(apiPool, failingAuditRecorder{})
	if err != nil {
		t.Fatal(err)
	}

	tenantID, err := identifier.Parse("ten_01JAT1AS00000000000002")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 8, 15, 0, 0, 0, time.UTC)
	actorPrincipalID := newIntegrationID(t, "usr")
	targetPrincipalID := newIntegrationID(t, "usr")
	adminTargetPrincipalID := newIntegrationID(t, "usr")
	failureTargetPrincipalID := newIntegrationID(t, "usr")
	actorMembershipID := newIntegrationID(t, "mem")
	targetMembershipID := newIntegrationID(t, "mem")
	adminTargetMembershipID := newIntegrationID(t, "mem")
	failureTargetMembershipID := newIntegrationID(t, "mem")
	actorSessionID := newIntegrationID(t, "ses")
	targetSessionID := newIntegrationID(t, "ses")
	failureTargetSessionID := newIntegrationID(t, "ses")
	_, actorVerifier := integrationToken(t)
	_, targetVerifier := integrationToken(t)
	_, failureTargetVerifier := integrationToken(t)

	principalIDs := []string{
		actorPrincipalID.String(), targetPrincipalID.String(),
		adminTargetPrincipalID.String(), failureTargetPrincipalID.String(),
	}
	membershipIDs := []string{
		actorMembershipID.String(), targetMembershipID.String(),
		adminTargetMembershipID.String(), failureTargetMembershipID.String(),
	}
	sessionIDs := []string{
		actorSessionID.String(), targetSessionID.String(), failureTargetSessionID.String(),
	}
	auditIDs := make([]string, 0, 12)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.membership_role_changes WHERE membership_id = ANY($1)`, membershipIDs)
		if len(auditIDs) > 0 {
			_, _ = migrationPool.Exec(cleanupCtx,
				`DELETE FROM atlas_audit.audit_events WHERE audit_event_id = ANY($1)`, auditIDs)
		}
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.sessions WHERE session_id = ANY($1)`, sessionIDs)
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.memberships WHERE membership_id = ANY($1)`, membershipIDs)
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.principals WHERE principal_id = ANY($1)`, principalIDs)
	})

	setup, err := migrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = setup.Rollback(ctx) }()
	for index, principalID := range principalIDs {
		anchor := "syn_person_role_" + strings.ToLower(strings.TrimPrefix(principalID, "usr_"))
		if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', $2, $3, 'active', 1, 1, $4, $4)`,
			principalID, "Synthetic Role Principal "+string(rune('A'+index)), anchor, now,
		); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		membershipID identifier.ID
		principalID  identifier.ID
		role         string
	}{
		{actorMembershipID, actorPrincipalID, "merchant_security_admin"},
		{targetMembershipID, targetPrincipalID, "merchant_viewer"},
		{adminTargetMembershipID, adminTargetPrincipalID, "merchant_admin"},
		{failureTargetMembershipID, failureTargetPrincipalID, "merchant_viewer"},
	} {
		if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, $2, $3, $4, 'merchant', 'active', 1, 1, $5, $5)`,
			fixture.membershipID.String(), tenantID.String(), fixture.principalID.String(),
			fixture.role, now,
		); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		sessionID   identifier.ID
		principalID identifier.ID
		verifier    [32]byte
	}{
		{actorSessionID, actorPrincipalID, actorVerifier},
		{targetSessionID, targetPrincipalID, targetVerifier},
		{failureTargetSessionID, failureTargetPrincipalID, failureTargetVerifier},
	} {
		if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.sessions (
    session_id, principal_id, population, tenant_id, verifier_sha256, assurance,
    status, authorization_version, rotation_version, version, created_at,
    last_seen_at, idle_expires_at, absolute_expires_at
) VALUES (
    $1, $2, 'merchant', $3, $4, 'baseline',
    'active', 1, 1, 1, $5, $5, $6, $7
)`, fixture.sessionID.String(), fixture.principalID.String(), tenantID.String(),
			fixture.verifier[:], now, now.Add(20*time.Minute), now.Add(8*time.Hour),
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	actor := identity.Session{
		SessionID: actorSessionID, PrincipalID: actorPrincipalID,
		PrincipalType: "merchant", Population: identity.PopulationMerchant,
		TenantID: tenantID, Assurance: identity.AssuranceBaseline,
		AuthorizationVersion: 1, RotationVersion: 1,
		CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(20 * time.Minute),
		AbsoluteExpiresAt: now.Add(8 * time.Hour),
	}
	newCommand := func(membershipID identifier.ID, role, idempotency string, expected int64) identity.UpdateOrganizationMemberRoleCommand {
		auditID := newIntegrationID(t, "aud")
		auditIDs = append(auditIDs, auditID.String())
		decisionID := newIntegrationID(t, "dec")
		return identity.UpdateOrganizationMemberRoleCommand{
			Actor: actor, RoleChangeID: newIntegrationID(t, "mrc"),
			OrganizationID: tenantID, MembershipID: membershipID, Role: role,
			ExpectedVersion:   expected,
			IdempotencyDigest: sha256.Sum256([]byte(idempotency)),
			RequestDigest: sha256.Sum256([]byte(
				membershipID.String() + "\n" + role + "\n" + idempotency,
			)),
			Now: now.Add(time.Minute),
			AuditEvent: audit.Event{
				AuditEventID: auditID, ActorID: actorPrincipalID, ActorType: "merchant",
				TenantID: tenantID, SessionAssurance: "baseline",
				Action: "identity.organization.membership.role.change", TargetType: "membership",
				TargetID: membershipID.String(), DecisionID: decisionID, Decision: "executed",
				ReasonCode:    "organization_membership_role_changed",
				CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now.Add(time.Minute),
				SafeAfterReference: "membership-role:" + role,
			},
		}
	}

	command := newCommand(targetMembershipID, "merchant_operator", "role-change-live-0001", 1)
	type callResult struct {
		result identity.UpdateOrganizationMemberRoleResult
		err    error
	}
	results := make(chan callResult, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, callErr := store.UpdateMemberRole(ctx, command)
			results <- callResult{result: result, err: callErr}
		}()
	}
	wait.Wait()
	close(results)
	successes, replays := 0, 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent role change error=%v", result.err)
		}
		successes++
		if result.result.Replay {
			replays++
		}
		if result.result.Member.Role != "merchant_operator" || result.result.Member.Version != 2 ||
			result.result.DecisionID != command.AuditEvent.DecisionID {
			t.Fatalf("concurrent role change result=%+v", result.result)
		}
	}
	if successes != 2 || replays != 1 {
		t.Fatalf("role-change concurrency successes=%d replays=%d", successes, replays)
	}

	var role, targetSessionStatus string
	var version, authorizationVersion, replayCount, successAuditCount int64
	if err := migrationPool.QueryRow(ctx, `
SELECT role_id, version, authorization_version
FROM atlas_identity.memberships
WHERE membership_id = $1`, targetMembershipID.String()).Scan(&role, &version, &authorizationVersion); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx,
		`SELECT status FROM atlas_identity.sessions WHERE session_id = $1`, targetSessionID.String(),
	).Scan(&targetSessionStatus); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.membership_role_changes
WHERE membership_id = $1`, targetMembershipID.String()).Scan(&replayCount); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_audit.audit_events
WHERE audit_event_id = $1`, command.AuditEvent.AuditEventID.String()).Scan(&successAuditCount); err != nil {
		t.Fatal(err)
	}
	if role != "merchant_operator" || version != 2 || authorizationVersion != 2 ||
		targetSessionStatus != "revoked" || replayCount != 1 || successAuditCount != 1 {
		t.Fatalf("committed role=%s version=%d authority=%d session=%s replay=%d audit=%d",
			role, version, authorizationVersion, targetSessionStatus, replayCount, successAuditCount)
	}

	replayed, err := store.UpdateMemberRole(ctx, command)
	if err != nil || !replayed.Replay || replayed.Member.Version != 2 ||
		replayed.DecisionID != command.AuditEvent.DecisionID {
		t.Fatalf("stable role-change replay=%+v err=%v", replayed, err)
	}

	conflict := newCommand(targetMembershipID, "merchant_viewer", "role-change-live-0002", 2)
	conflict.IdempotencyDigest = command.IdempotencyDigest
	if _, err := store.UpdateMemberRole(ctx, conflict); !errors.Is(err, identity.ErrIdempotencyConflict) {
		t.Fatalf("idempotency mismatch error=%v", err)
	}

	stale := newCommand(targetMembershipID, "merchant_viewer", "role-change-live-0003", 1)
	if _, err := store.UpdateMemberRole(ctx, stale); !errors.Is(err, identity.ErrMembershipPreconditionFailed) {
		t.Fatalf("stale version error=%v", err)
	}

	undelegable := newCommand(targetMembershipID, "merchant_security_admin", "role-change-live-0004", 2)
	if _, err := store.UpdateMemberRole(ctx, undelegable); !errors.Is(err, identity.ErrActionNotAuthorized) {
		t.Fatalf("undelegable role error=%v", err)
	}

	adminTransition := newCommand(adminTargetMembershipID, "merchant_operator", "role-change-live-0005", 1)
	if _, err := store.UpdateMemberRole(ctx, adminTransition); !errors.Is(err, identity.ErrMembershipApprovalRequired) {
		t.Fatalf("administrator transition error=%v", err)
	}
	if err := migrationPool.QueryRow(ctx,
		`SELECT role_id FROM atlas_identity.memberships WHERE membership_id = $1`, adminTargetMembershipID.String(),
	).Scan(&role); err != nil || role != "merchant_admin" {
		t.Fatalf("administrator target role=%s err=%v", role, err)
	}

	failure := newCommand(failureTargetMembershipID, "merchant_operator", "role-change-live-0006", 1)
	if _, err := failingStore.UpdateMemberRole(ctx, failure); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("Audit outage error=%v", err)
	}
	var failureRole, failureSessionStatus string
	var failureVersion, failureReplayCount int64
	if err := migrationPool.QueryRow(ctx, `
SELECT role_id, version FROM atlas_identity.memberships WHERE membership_id = $1`,
		failureTargetMembershipID.String(),
	).Scan(&failureRole, &failureVersion); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx,
		`SELECT status FROM atlas_identity.sessions WHERE session_id = $1`, failureTargetSessionID.String(),
	).Scan(&failureSessionStatus); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.membership_role_changes WHERE membership_id = $1`,
		failureTargetMembershipID.String(),
	).Scan(&failureReplayCount); err != nil {
		t.Fatal(err)
	}
	if failureRole != "merchant_viewer" || failureVersion != 1 ||
		failureSessionStatus != "active" || failureReplayCount != 0 {
		t.Fatalf("Audit rollback role=%s version=%d session=%s replay=%d",
			failureRole, failureVersion, failureSessionStatus, failureReplayCount)
	}
}
