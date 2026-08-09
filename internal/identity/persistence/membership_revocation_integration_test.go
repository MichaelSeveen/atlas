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

func TestMembershipRevocationRealPostgresConcurrencyStaleTabAndAuditRollback(t *testing.T) {
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
	for _, required := range []struct {
		table, privilege string
	}{
		{"atlas_identity.membership_revocations", "SELECT"},
		{"atlas_identity.membership_revocations", "INSERT"},
		{"atlas_identity.memberships", "UPDATE"},
		{"atlas_identity.sessions", "UPDATE"},
		{"atlas_audit.audit_events", "INSERT"},
	} {
		var allowed bool
		if err := apiPool.QueryRow(
			ctx, `SELECT has_table_privilege(current_user, $1, $2)`,
			required.table, required.privilege,
		).Scan(&allowed); err != nil {
			t.Fatal(err)
		}
		if !allowed {
			t.Fatalf("atlas_api lacks required %s privilege on %s", required.privilege, required.table)
		}
	}
	var replayUpdateAllowed bool
	if err := apiPool.QueryRow(
		ctx,
		`SELECT has_table_privilege(current_user, 'atlas_identity.membership_revocations', 'UPDATE')`,
	).Scan(&replayUpdateAllowed); err != nil {
		t.Fatal(err)
	}
	if replayUpdateAllowed {
		t.Fatal("atlas_api must not update immutable membership revocation replay records")
	}

	tenantID, err := identifier.Parse("ten_01JAT1AS00000000000002")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)
	actorPrincipalID := newIntegrationID(t, "usr")
	targetPrincipalID := newIntegrationID(t, "usr")
	adminTargetPrincipalID := newIntegrationID(t, "usr")
	raceTargetPrincipalID := newIntegrationID(t, "usr")
	failureTargetPrincipalID := newIntegrationID(t, "usr")
	actorMembershipID := newIntegrationID(t, "mem")
	targetMembershipID := newIntegrationID(t, "mem")
	adminTargetMembershipID := newIntegrationID(t, "mem")
	raceTargetMembershipID := newIntegrationID(t, "mem")
	failureTargetMembershipID := newIntegrationID(t, "mem")
	actorSessionID := newIntegrationID(t, "ses")
	targetSessionID := newIntegrationID(t, "ses")
	raceTargetSessionID := newIntegrationID(t, "ses")
	failureTargetSessionID := newIntegrationID(t, "ses")
	_, actorVerifier := integrationToken(t)
	_, targetVerifier := integrationToken(t)
	_, raceTargetVerifier := integrationToken(t)
	_, failureTargetVerifier := integrationToken(t)

	principalIDs := []string{
		actorPrincipalID.String(), targetPrincipalID.String(),
		adminTargetPrincipalID.String(), raceTargetPrincipalID.String(),
		failureTargetPrincipalID.String(),
	}
	membershipIDs := []string{
		actorMembershipID.String(), targetMembershipID.String(),
		adminTargetMembershipID.String(), raceTargetMembershipID.String(),
		failureTargetMembershipID.String(),
	}
	sessionIDs := []string{
		actorSessionID.String(), targetSessionID.String(),
		raceTargetSessionID.String(), failureTargetSessionID.String(),
	}
	auditIDs := make([]string, 0, 12)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.membership_revocations WHERE membership_id = ANY($1)`, membershipIDs)
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
		anchor := "syn_person_revoke_" + strings.ToLower(strings.TrimPrefix(principalID, "usr_"))
		if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', $2, $3, 'active', 1, 1, $4, $4)`,
			principalID, "Synthetic Revocation Principal "+string(rune('A'+index)), anchor, now,
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
		{raceTargetMembershipID, raceTargetPrincipalID, "merchant_viewer"},
		{failureTargetMembershipID, failureTargetPrincipalID, "merchant_operator"},
	} {
		if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, $2, $3, $4, 'merchant', 'active', 1, 1, $5, $5)`,
			fixture.membershipID.String(), tenantID.String(), fixture.principalID.String(), fixture.role, now,
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
		{raceTargetSessionID, raceTargetPrincipalID, raceTargetVerifier},
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
			fixture.verifier[:], now, now.Add(20*time.Minute), now.Add(8*time.Hour)); err != nil {
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
	newCommand := func(membershipID identifier.ID, idempotency string, expected int64) identity.RevokeOrganizationMemberCommand {
		auditID := newIntegrationID(t, "aud")
		auditIDs = append(auditIDs, auditID.String())
		return identity.RevokeOrganizationMemberCommand{
			Actor: actor, RevocationID: newIntegrationID(t, "mrv"),
			OrganizationID: tenantID, MembershipID: membershipID, ExpectedVersion: expected,
			IdempotencyDigest: sha256.Sum256([]byte(idempotency)),
			RequestDigest:     sha256.Sum256([]byte(membershipID.String() + "\n" + idempotency)),
			Now:               now.Add(time.Minute),
			AuditEvent: audit.Event{
				AuditEventID: auditID, ActorID: actorPrincipalID, ActorType: "merchant",
				TenantID: tenantID, SessionAssurance: "baseline",
				Action: "identity.organization.membership.revoke", TargetType: "membership",
				TargetID: membershipID.String(), DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
				ReasonCode:    "organization_membership_revoked",
				CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now.Add(time.Minute),
				SafeAfterReference: "membership-status:revoked",
			},
		}
	}

	command := newCommand(targetMembershipID, "member-revoke-live-0001", 1)
	type callResult struct {
		result identity.RevokeOrganizationMemberResult
		err    error
	}
	results := make(chan callResult, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, callErr := store.RevokeMember(ctx, command)
			results <- callResult{result: result, err: callErr}
		}()
	}
	wait.Wait()
	close(results)
	successes, replays := 0, 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent member revocation error=%v", result.err)
		}
		successes++
		if result.result.Replay {
			replays++
		}
		if result.result.Member.Status != "revoked" || result.result.Member.Version != 2 ||
			result.result.Member.RevokedAt == nil || result.result.DecisionID != command.AuditEvent.DecisionID {
			t.Fatalf("concurrent member revocation result=%+v", result.result)
		}
	}
	if successes != 2 || replays != 1 {
		t.Fatalf("member-revocation concurrency successes=%d replays=%d", successes, replays)
	}

	var membershipStatus, targetSessionStatus string
	var version, authorizationVersion, replayCount, successAuditCount int64
	var revokedAt *time.Time
	if err := migrationPool.QueryRow(ctx, `
SELECT status, version, authorization_version, revoked_at
FROM atlas_identity.memberships
WHERE membership_id = $1`, targetMembershipID.String()).Scan(
		&membershipStatus, &version, &authorizationVersion, &revokedAt,
	); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx,
		`SELECT status FROM atlas_identity.sessions WHERE session_id = $1`, targetSessionID.String(),
	).Scan(&targetSessionStatus); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.membership_revocations
WHERE membership_id = $1`, targetMembershipID.String()).Scan(&replayCount); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_audit.audit_events
WHERE audit_event_id = $1`, command.AuditEvent.AuditEventID.String()).Scan(&successAuditCount); err != nil {
		t.Fatal(err)
	}
	if membershipStatus != "revoked" || version != 2 || authorizationVersion != 2 || revokedAt == nil ||
		targetSessionStatus != "revoked" || replayCount != 1 || successAuditCount != 1 {
		t.Fatalf("committed status=%s version=%d authority=%d revoked=%v session=%s replay=%d audit=%d",
			membershipStatus, version, authorizationVersion, revokedAt, targetSessionStatus, replayCount, successAuditCount)
	}

	replayed, err := store.RevokeMember(ctx, command)
	if err != nil || !replayed.Replay || replayed.Member.Version != 2 ||
		replayed.DecisionID != command.AuditEvent.DecisionID {
		t.Fatalf("stable member-revocation replay=%+v err=%v", replayed, err)
	}
	conflict := newCommand(targetMembershipID, "member-revoke-live-0002", 1)
	conflict.IdempotencyDigest = command.IdempotencyDigest
	if _, err := store.RevokeMember(ctx, conflict); !errors.Is(err, identity.ErrIdempotencyConflict) {
		t.Fatalf("idempotency mismatch error=%v", err)
	}

	staleRoleChange := identity.UpdateOrganizationMemberRoleCommand{
		Actor: actor, RoleChangeID: newIntegrationID(t, "mrc"),
		OrganizationID: tenantID, MembershipID: targetMembershipID,
		Role: "merchant_operator", ExpectedVersion: 1,
		IdempotencyDigest: sha256.Sum256([]byte("removed-member-stale-tab-0001")),
		RequestDigest:     sha256.Sum256([]byte("removed-member-stale-tab-request")),
		Now:               now.Add(2 * time.Minute),
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), ActorID: actorPrincipalID, ActorType: "merchant",
			TenantID: tenantID, SessionAssurance: "baseline",
			Action: "identity.organization.membership.role.change", TargetType: "membership",
			TargetID: targetMembershipID.String(), DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
			ReasonCode: "organization_membership_role_changed", CorrelationID: newIntegrationID(t, "cor"),
			OccurredAt: now.Add(2 * time.Minute), SafeAfterReference: "membership-role:merchant_operator",
		},
	}
	auditIDs = append(auditIDs, staleRoleChange.AuditEvent.AuditEventID.String())
	if _, err := store.UpdateMemberRole(ctx, staleRoleChange); !errors.Is(err, identity.ErrMembershipNotFound) {
		t.Fatalf("removed member stale-tab mutation error=%v", err)
	}

	raceRevoke := newCommand(raceTargetMembershipID, "member-revoke-role-race-0001", 1)
	raceRoleChange := identity.UpdateOrganizationMemberRoleCommand{
		Actor: actor, RoleChangeID: newIntegrationID(t, "mrc"),
		OrganizationID: tenantID, MembershipID: raceTargetMembershipID,
		Role: "merchant_operator", ExpectedVersion: 1,
		IdempotencyDigest: sha256.Sum256([]byte("member-role-revoke-race-0001")),
		RequestDigest:     sha256.Sum256([]byte("member-role-revoke-race-request")),
		Now:               now.Add(3 * time.Minute),
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), ActorID: actorPrincipalID, ActorType: "merchant",
			TenantID: tenantID, SessionAssurance: "baseline",
			Action: "identity.organization.membership.role.change", TargetType: "membership",
			TargetID: raceTargetMembershipID.String(), DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
			ReasonCode: "organization_membership_role_changed", CorrelationID: newIntegrationID(t, "cor"),
			OccurredAt: now.Add(3 * time.Minute), SafeAfterReference: "membership-role:merchant_operator",
		},
	}
	auditIDs = append(auditIDs, raceRoleChange.AuditEvent.AuditEventID.String())
	type mutationRaceResult struct {
		mutation string
		err      error
	}
	raceStarted := make(chan struct{})
	raceResults := make(chan mutationRaceResult, 2)
	go func() {
		<-raceStarted
		_, revokeErr := store.RevokeMember(ctx, raceRevoke)
		raceResults <- mutationRaceResult{mutation: "revoke", err: revokeErr}
	}()
	go func() {
		<-raceStarted
		_, roleErr := store.UpdateMemberRole(ctx, raceRoleChange)
		raceResults <- mutationRaceResult{mutation: "role", err: roleErr}
	}()
	close(raceStarted)
	raceWinner := ""
	for range 2 {
		result := <-raceResults
		if result.err == nil {
			if raceWinner != "" {
				t.Fatalf("revocation/role race committed both mutations")
			}
			raceWinner = result.mutation
			continue
		}
		if !errors.Is(result.err, identity.ErrMembershipNotFound) &&
			!errors.Is(result.err, identity.ErrMembershipPreconditionFailed) {
			t.Fatalf("revocation/role race loser=%s err=%v", result.mutation, result.err)
		}
	}
	if raceWinner == "" {
		t.Fatal("revocation/role race committed no mutation")
	}
	var raceRole, raceSessionStatus string
	var raceVersion, raceAuthorizationVersion, raceRevocationCount, raceRoleChangeCount int64
	if err := migrationPool.QueryRow(ctx, `
SELECT status, role_id, version, authorization_version
FROM atlas_identity.memberships WHERE membership_id = $1`, raceTargetMembershipID.String()).Scan(
		&membershipStatus, &raceRole, &raceVersion, &raceAuthorizationVersion,
	); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx,
		`SELECT status FROM atlas_identity.sessions WHERE session_id = $1`, raceTargetSessionID.String(),
	).Scan(&raceSessionStatus); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT
  (SELECT count(*) FROM atlas_identity.membership_revocations WHERE membership_id = $1),
  (SELECT count(*) FROM atlas_identity.membership_role_changes WHERE membership_id = $1)`,
		raceTargetMembershipID.String(),
	).Scan(&raceRevocationCount, &raceRoleChangeCount); err != nil {
		t.Fatal(err)
	}
	if raceVersion != 2 || raceAuthorizationVersion != 2 || raceSessionStatus != "revoked" ||
		raceRevocationCount+raceRoleChangeCount != 1 {
		t.Fatalf("revocation/role race winner=%s status=%s role=%s version=%d authority=%d session=%s revocations=%d role_changes=%d",
			raceWinner, membershipStatus, raceRole, raceVersion, raceAuthorizationVersion,
			raceSessionStatus, raceRevocationCount, raceRoleChangeCount)
	}
	if raceWinner == "revoke" && (membershipStatus != "revoked" || raceRole != "merchant_viewer" ||
		raceRevocationCount != 1 || raceRoleChangeCount != 0) {
		t.Fatalf("revocation race winner produced status=%s role=%s revocations=%d role_changes=%d",
			membershipStatus, raceRole, raceRevocationCount, raceRoleChangeCount)
	}
	if raceWinner == "role" && (membershipStatus != "active" || raceRole != "merchant_operator" ||
		raceRevocationCount != 0 || raceRoleChangeCount != 1) {
		t.Fatalf("role-change race winner produced status=%s role=%s revocations=%d role_changes=%d",
			membershipStatus, raceRole, raceRevocationCount, raceRoleChangeCount)
	}

	adminCommand := newCommand(adminTargetMembershipID, "member-revoke-live-0004", 1)
	if _, err := store.RevokeMember(ctx, adminCommand); !errors.Is(err, identity.ErrMembershipAdministratorRemovalUnavailable) {
		t.Fatalf("administrator removal error=%v", err)
	}
	if err := migrationPool.QueryRow(ctx,
		`SELECT status FROM atlas_identity.memberships WHERE membership_id = $1`, adminTargetMembershipID.String(),
	).Scan(&membershipStatus); err != nil || membershipStatus != "active" {
		t.Fatalf("administrator target status=%s err=%v", membershipStatus, err)
	}

	failure := newCommand(failureTargetMembershipID, "member-revoke-live-0005", 1)
	if _, err := failingStore.RevokeMember(ctx, failure); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("Audit outage error=%v", err)
	}
	var failureSessionStatus string
	var failureVersion, failureReplayCount int64
	if err := migrationPool.QueryRow(ctx, `
SELECT status, version FROM atlas_identity.memberships WHERE membership_id = $1`,
		failureTargetMembershipID.String()).Scan(&membershipStatus, &failureVersion); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx,
		`SELECT status FROM atlas_identity.sessions WHERE session_id = $1`, failureTargetSessionID.String(),
	).Scan(&failureSessionStatus); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.membership_revocations WHERE membership_id = $1`,
		failureTargetMembershipID.String()).Scan(&failureReplayCount); err != nil {
		t.Fatal(err)
	}
	if membershipStatus != "active" || failureVersion != 1 ||
		failureSessionStatus != "active" || failureReplayCount != 0 {
		t.Fatalf("Audit rollback status=%s version=%d session=%s replay=%d",
			membershipStatus, failureVersion, failureSessionStatus, failureReplayCount)
	}
}
