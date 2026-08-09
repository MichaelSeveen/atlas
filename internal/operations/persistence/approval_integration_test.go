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
	identityapplication "github.com/MichaelSeveen/atlas/internal/identity/application"
	"github.com/MichaelSeveen/atlas/internal/operations"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type failingApprovalAuditRecorder struct{}

func (failingApprovalAuditRecorder) Record(context.Context, audit.Transaction, audit.Event) error {
	return errors.New("synthetic approval Audit outage")
}

type approvalIntegrationFixture struct {
	principalID  identifier.ID
	membershipID identifier.ID
	role         string
	sessionID    identifier.ID
	stepUpAction string
	actor        identity.Session
}

func TestApprovalRealPostgresSeparationReauthorizationIntegrityConcurrencyAndAuditRollback(t *testing.T) {
	apiURL := os.Getenv("ATLAS_P01_DATABASE_URL")
	migrationURL := os.Getenv("ATLAS_P01_MIGRATION_DATABASE_URL")
	if apiURL == "" || migrationURL == "" {
		t.Skip("real Phase 01 database URLs are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
	cleanupStaleApprovalIntegrationFixtures(t, migrationPool)
	recorder := auditapplication.NewRecorder()
	identityBoundary, err := identityapplication.NewApprovalBoundary(recorder)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(apiPool, recorder, identityBoundary)
	if err != nil {
		t.Fatal(err)
	}
	failingStore, err := NewStore(apiPool, failingApprovalAuditRecorder{}, identityBoundary)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 9, 13, 0, 0, 0, time.UTC)
	tenantID := newApprovalIntegrationID(t, "ten")
	maker := newApprovalFixture(t, tenantID, "merchant_admin", identity.ApprovalStepUpCreateAction, now)
	makerAlternate := newApprovalFixture(t, tenantID, "merchant_admin", identity.ApprovalStepUpDecideAction, now)
	makerAlternate.principalID = maker.principalID
	makerAlternate.membershipID = identifier.ID{}
	makerAlternate.actor.PrincipalID = maker.principalID
	checkerA := newApprovalFixture(t, tenantID, "merchant_security_admin", identity.ApprovalStepUpDecideAction, now)
	checkerB := newApprovalFixture(t, tenantID, "merchant_security_admin", identity.ApprovalStepUpDecideAction, now)
	executor := newApprovalFixture(t, tenantID, "merchant_admin", identity.ApprovalStepUpExecuteAction, now)
	targetSuccess := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	targetTamper := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	targetAudit := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	targetRace := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	targetConcurrent := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	targetCancelled := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	targetRejected := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	targetExpired := newApprovalFixture(t, tenantID, "merchant_viewer", "", now)
	fixtures := []*approvalIntegrationFixture{
		&maker, &makerAlternate, &checkerA, &checkerB, &executor, &targetSuccess,
		&targetTamper, &targetAudit, &targetRace, &targetConcurrent,
		&targetCancelled, &targetRejected, &targetExpired,
	}
	seedApprovalIntegrationFixtures(t, ctx, migrationPool, tenantID, now, fixtures)
	t.Cleanup(func() { cleanupApprovalIntegrationFixtures(t, migrationPool, tenantID) })

	primaryApprovalID := newApprovalIntegrationID(t, "apr")
	primaryCreate := newCreateApprovalCommand(
		t, maker.actor, targetSuccess.membershipID, primaryApprovalID, "approval-create-primary", now.Add(time.Minute),
	)
	created, err := store.Create(ctx, primaryCreate)
	if err != nil || created.Approval.Status != operations.ApprovalPending || created.Approval.Version != 1 {
		t.Fatalf("create result=%+v err=%v", created, err)
	}
	replayedCreate, err := store.Create(ctx, primaryCreate)
	if err != nil || !replayedCreate.Replay || replayedCreate.DecisionID != created.DecisionID ||
		replayedCreate.Approval.ApprovalID != primaryApprovalID {
		t.Fatalf("create replay=%+v err=%v", replayedCreate, err)
	}

	selfDecision := newDecideApprovalCommand(
		t, makerAlternate.actor, primaryApprovalID, 1, "approve", "approval-decide-self", now.Add(2*time.Minute),
	)
	if _, err := store.Decide(ctx, selfDecision); !errors.Is(err, operations.ErrApprovalNotAuthorized) {
		t.Fatalf("alternate-session self-approval error=%v", err)
	}
	checkerDecision := newDecideApprovalCommand(
		t, checkerA.actor, primaryApprovalID, 1, "approve", "approval-decide-primary", now.Add(2*time.Minute),
	)
	approved, err := store.Decide(ctx, checkerDecision)
	if err != nil || approved.Approval.Status != operations.ApprovalApproved || approved.Approval.Version != 2 ||
		approved.Approval.DeciderPrincipalID != checkerA.principalID {
		t.Fatalf("decision result=%+v err=%v", approved, err)
	}
	replayedDecision, err := store.Decide(ctx, checkerDecision)
	if err != nil || !replayedDecision.Replay || replayedDecision.DecisionID != approved.DecisionID {
		t.Fatalf("decision replay=%+v err=%v", replayedDecision, err)
	}

	primaryExecution := newExecuteApprovalCommand(
		t, executor.actor, primaryApprovalID, targetSuccess.membershipID, 2,
		"approval-execute-primary", now.Add(3*time.Minute),
	)
	setApprovalFixtureRole(t, ctx, migrationPool, maker.membershipID, "merchant_viewer", now.Add(150*time.Second))
	if _, err := store.Execute(ctx, primaryExecution); !errors.Is(err, operations.ErrApprovalNotAuthorized) {
		t.Fatalf("maker permission downgrade error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, primaryApprovalID, targetSuccess.membershipID,
		operations.ApprovalApproved, "merchant_viewer", 1)
	setApprovalFixtureRole(t, ctx, migrationPool, maker.membershipID, "merchant_admin", now.Add(160*time.Second))
	refreshApprovalFixtureAuthority(t, ctx, migrationPool, &maker)

	setApprovalFixtureRole(t, ctx, migrationPool, checkerA.membershipID, "merchant_viewer", now.Add(170*time.Second))
	primaryExecution = newExecuteApprovalCommand(
		t, executor.actor, primaryApprovalID, targetSuccess.membershipID, 2,
		"approval-execute-primary", now.Add(3*time.Minute),
	)
	if _, err := store.Execute(ctx, primaryExecution); !errors.Is(err, operations.ErrApprovalNotAuthorized) {
		t.Fatalf("checker permission downgrade error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, primaryApprovalID, targetSuccess.membershipID,
		operations.ApprovalApproved, "merchant_viewer", 1)
	setApprovalFixtureRole(t, ctx, migrationPool, checkerA.membershipID, "merchant_security_admin", now.Add(180*time.Second))
	refreshApprovalFixtureAuthority(t, ctx, migrationPool, &checkerA)

	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.sessions
SET step_up_verified_at = $2, version = version + 1
WHERE session_id = $1`, executor.sessionID.String(), now.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	primaryExecution = newExecuteApprovalCommand(
		t, executor.actor, primaryApprovalID, targetSuccess.membershipID, 2,
		"approval-execute-primary", now.Add(3*time.Minute),
	)
	if _, err := store.Execute(ctx, primaryExecution); !errors.Is(err, operations.ErrApprovalStepUpRequired) {
		t.Fatalf("expired execution step-up error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, primaryApprovalID, targetSuccess.membershipID,
		operations.ApprovalApproved, "merchant_viewer", 1)
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.sessions
SET step_up_verified_at = $2, version = version + 1
WHERE session_id = $1`, executor.sessionID.String(), now); err != nil {
		t.Fatal(err)
	}

	primaryExecution = newExecuteApprovalCommand(
		t, executor.actor, primaryApprovalID, targetSuccess.membershipID, 2,
		"approval-execute-primary", now.Add(3*time.Minute),
	)
	executed, err := store.Execute(ctx, primaryExecution)
	if err != nil || executed.Approval.Status != operations.ApprovalExecuted || executed.Approval.Version != 3 {
		t.Fatalf("execution result=%+v err=%v", executed, err)
	}
	replayExecution := newExecuteApprovalCommand(
		t, executor.actor, primaryApprovalID, targetSuccess.membershipID, 2,
		"approval-execute-primary", now.Add(3*time.Minute),
	)
	replayedExecution, err := store.Execute(ctx, replayExecution)
	if err != nil || !replayedExecution.Replay || replayedExecution.DecisionID != executed.DecisionID ||
		replayedExecution.Approval.Status != operations.ApprovalExecuted {
		t.Fatalf("execution replay=%+v err=%v", replayedExecution, err)
	}
	assertExecutedApprovalEffects(t, ctx, migrationPool, primaryApprovalID, targetSuccess, primaryCreate.PayloadDigest)

	tamperApprovalID := createAndApproveIntegrationApproval(
		t, ctx, store, maker.actor, checkerA.actor, targetTamper.membershipID,
		"approval-tamper", now.Add(4*time.Minute),
	)
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_operations.approvals
SET payload_canonical = payload_canonical || decode('20', 'hex')
WHERE tenant_id = $1 AND approval_id = $2`, tenantID.String(), tamperApprovalID.String()); err != nil {
		t.Fatal(err)
	}
	tamperExecution := newExecuteApprovalCommand(
		t, executor.actor, tamperApprovalID, targetTamper.membershipID, 2,
		"approval-execute-tamper", now.Add(5*time.Minute),
	)
	if _, err := store.Execute(ctx, tamperExecution); !errors.Is(err, operations.ErrApprovalConflict) {
		t.Fatalf("tampered payload execution error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, tamperApprovalID, targetTamper.membershipID,
		operations.ApprovalSuperseded, "merchant_viewer", 1)

	raceApprovalID := createAndApproveIntegrationApproval(
		t, ctx, store, maker.actor, checkerA.actor, targetRace.membershipID,
		"approval-race", now.Add(4*time.Minute),
	)
	setApprovalFixtureRole(t, ctx, migrationPool, targetRace.membershipID, "merchant_operator", now.Add(5*time.Minute))
	raceExecution := newExecuteApprovalCommand(
		t, executor.actor, raceApprovalID, targetRace.membershipID, 2,
		"approval-execute-race", now.Add(5*time.Minute),
	)
	if _, err := store.Execute(ctx, raceExecution); !errors.Is(err, operations.ErrApprovalConflict) {
		t.Fatalf("target-version race execution error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, raceApprovalID, targetRace.membershipID,
		operations.ApprovalSuperseded, "merchant_operator", 2)

	auditApprovalID := createAndApproveIntegrationApproval(
		t, ctx, store, maker.actor, checkerA.actor, targetAudit.membershipID,
		"approval-audit", now.Add(4*time.Minute),
	)
	auditExecution := newExecuteApprovalCommand(
		t, executor.actor, auditApprovalID, targetAudit.membershipID, 2,
		"approval-execute-audit", now.Add(5*time.Minute),
	)
	if _, err := failingStore.Execute(ctx, auditExecution); !errors.Is(err, operations.ErrApprovalUnavailable) {
		t.Fatalf("Audit outage execution error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, auditApprovalID, targetAudit.membershipID,
		operations.ApprovalApproved, "merchant_viewer", 1)
	var targetAuditCount, failedExecutionCount int64
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_audit.audit_events WHERE audit_event_id = $1`,
		auditExecution.TargetAuditEvent.AuditEventID.String()).Scan(&targetAuditCount); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_operations.approval_executions WHERE approval_id = $1`,
		auditApprovalID.String()).Scan(&failedExecutionCount); err != nil {
		t.Fatal(err)
	}
	if targetAuditCount != 0 || failedExecutionCount != 0 {
		t.Fatalf("Audit outage leaked target audit=%d execution=%d", targetAuditCount, failedExecutionCount)
	}
	if _, err := store.Execute(ctx, auditExecution); err != nil {
		t.Fatalf("safe retry after Audit outage: %v", err)
	}
	assertExecutedApprovalEffects(t, ctx, migrationPool, auditApprovalID, targetAudit,
		newCreateApprovalCommand(t, maker.actor, targetAudit.membershipID, auditApprovalID,
			"approval-create-audit", now.Add(4*time.Minute)).PayloadDigest)

	concurrentApprovalID := newApprovalIntegrationID(t, "apr")
	concurrentCreate := newCreateApprovalCommand(
		t, maker.actor, targetConcurrent.membershipID, concurrentApprovalID,
		"approval-create-concurrent", now.Add(4*time.Minute),
	)
	if _, err := store.Create(ctx, concurrentCreate); err != nil {
		t.Fatal(err)
	}
	decisionCommands := []operations.DecideApprovalCommand{
		newDecideApprovalCommand(t, checkerA.actor, concurrentApprovalID, 1, "approve", "approval-checker-a", now.Add(5*time.Minute)),
		newDecideApprovalCommand(t, checkerB.actor, concurrentApprovalID, 1, "approve", "approval-checker-b", now.Add(5*time.Minute)),
	}
	type decisionResult struct {
		result operations.ApprovalResult
		err    error
	}
	results := make(chan decisionResult, 2)
	var wait sync.WaitGroup
	for _, command := range decisionCommands {
		command := command
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, callErr := store.Decide(ctx, command)
			results <- decisionResult{result: result, err: callErr}
		}()
	}
	wait.Wait()
	close(results)
	successes, conflicts := 0, 0
	for call := range results {
		switch {
		case call.err == nil:
			successes++
		case errors.Is(call.err, operations.ErrApprovalConflict):
			conflicts++
		default:
			t.Fatalf("simultaneous checker result=%+v err=%v", call.result, call.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("simultaneous checker successes=%d conflicts=%d", successes, conflicts)
	}
	var decisionCount int64
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_operations.approval_decisions WHERE approval_id = $1`,
		concurrentApprovalID.String()).Scan(&decisionCount); err != nil {
		t.Fatal(err)
	}
	if decisionCount != 1 {
		t.Fatalf("simultaneous checker decision rows=%d", decisionCount)
	}

	cancelledApprovalID := newApprovalIntegrationID(t, "apr")
	if _, err := store.Create(ctx, newCreateApprovalCommand(
		t, maker.actor, targetCancelled.membershipID, cancelledApprovalID,
		"approval-cancel-create", now.Add(2*time.Minute),
	)); err != nil {
		t.Fatalf("create cancellable approval: %v", err)
	}
	cancelCommand := newCancelApprovalCommand(
		t, maker.actor, cancelledApprovalID, 1, "approval-cancel", now.Add(3*time.Minute),
	)
	cancelled, err := store.Cancel(ctx, cancelCommand)
	if err != nil || cancelled.Approval.Status != operations.ApprovalCancelled || cancelled.Approval.Version != 2 {
		t.Fatalf("cancel result=%+v err=%v", cancelled, err)
	}
	cancelReplay, err := store.Cancel(ctx, cancelCommand)
	if err != nil || !cancelReplay.Replay || cancelReplay.DecisionID != cancelled.DecisionID {
		t.Fatalf("cancel replay=%+v err=%v", cancelReplay, err)
	}
	if _, err := store.Execute(ctx, newExecuteApprovalCommand(
		t, executor.actor, cancelledApprovalID, targetCancelled.membershipID, 2,
		"approval-execute-cancelled", now.Add(4*time.Minute),
	)); !errors.Is(err, operations.ErrApprovalConflict) {
		t.Fatalf("cancelled approval execution error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, cancelledApprovalID, targetCancelled.membershipID,
		operations.ApprovalCancelled, "merchant_viewer", 1)

	rejectedApprovalID := newApprovalIntegrationID(t, "apr")
	if _, err := store.Create(ctx, newCreateApprovalCommand(
		t, maker.actor, targetRejected.membershipID, rejectedApprovalID,
		"approval-reject-create", now.Add(2*time.Minute),
	)); err != nil {
		t.Fatalf("create rejectable approval: %v", err)
	}
	rejected, err := store.Decide(ctx, newDecideApprovalCommand(
		t, checkerA.actor, rejectedApprovalID, 1, "reject",
		"approval-reject", now.Add(3*time.Minute),
	))
	if err != nil || rejected.Approval.Status != operations.ApprovalRejected || rejected.Approval.Version != 2 {
		t.Fatalf("reject result=%+v err=%v", rejected, err)
	}
	if _, err := store.Execute(ctx, newExecuteApprovalCommand(
		t, executor.actor, rejectedApprovalID, targetRejected.membershipID, 2,
		"approval-execute-rejected", now.Add(4*time.Minute),
	)); !errors.Is(err, operations.ErrApprovalConflict) {
		t.Fatalf("rejected approval execution error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, rejectedApprovalID, targetRejected.membershipID,
		operations.ApprovalRejected, "merchant_viewer", 1)

	expiredApprovalID := newApprovalIntegrationID(t, "apr")
	expiringCreate := newCreateApprovalCommand(
		t, maker.actor, targetExpired.membershipID, expiredApprovalID,
		"approval-expire-create", now.Add(2*time.Minute),
	)
	expiringCreate.ExpiresAt = now.Add(3 * time.Minute)
	if _, err := store.Create(ctx, expiringCreate); err != nil {
		t.Fatalf("create expiring approval: %v", err)
	}
	if _, err := store.Execute(ctx, newExecuteApprovalCommand(
		t, executor.actor, expiredApprovalID, targetExpired.membershipID, 1,
		"approval-execute-expired", now.Add(4*time.Minute),
	)); !errors.Is(err, operations.ErrApprovalConflict) {
		t.Fatalf("expired approval execution error=%v", err)
	}
	assertApprovalStatusAndTarget(t, ctx, migrationPool, expiredApprovalID, targetExpired.membershipID,
		operations.ApprovalExpired, "merchant_viewer", 1)
}

func newApprovalFixture(
	t *testing.T,
	tenantID identifier.ID,
	role, stepUpAction string,
	now time.Time,
) approvalIntegrationFixture {
	t.Helper()
	principalID := newApprovalIntegrationID(t, "usr")
	membershipID := newApprovalIntegrationID(t, "mem")
	sessionID := newApprovalIntegrationID(t, "ses")
	assurance := identity.AssuranceBaseline
	if stepUpAction != "" {
		assurance = identity.AssurancePhishingResistant
	}
	return approvalIntegrationFixture{
		principalID: principalID, membershipID: membershipID, role: role,
		sessionID: sessionID, stepUpAction: stepUpAction,
		actor: identity.Session{
			SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
			Population: identity.PopulationMerchant, TenantID: tenantID, Assurance: assurance,
			AuthorizationVersion: 1, RotationVersion: 1, CreatedAt: now, LastSeenAt: now,
			IdleExpiresAt: now.Add(30 * time.Minute), AbsoluteExpiresAt: now.Add(8 * time.Hour),
		},
	}
}

func seedApprovalIntegrationFixtures(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID identifier.ID,
	now time.Time,
	fixtures []*approvalIntegrationFixture,
) {
	t.Helper()
	transaction, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	uniqueName := "approval-" + strings.ToLower(strings.TrimPrefix(tenantID.String(), "ten_"))
	if _, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.organizations (
    tenant_id, organization_type, display_name, normalized_name, confusable_skeleton,
    status, authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', 'Approval Integration Fixture', $2, $2, 'active', 1, 1, $3, $3)`,
		tenantID.String(), uniqueName, now); err != nil {
		t.Fatal(err)
	}
	seenPrincipals := make(map[identifier.ID]struct{})
	for index, fixture := range fixtures {
		if _, seen := seenPrincipals[fixture.principalID]; !seen {
			seenPrincipals[fixture.principalID] = struct{}{}
			anchor := "syn_person_approval_" + strings.ToLower(strings.TrimPrefix(fixture.principalID.String(), "usr_"))
			if _, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', $2, $3, 'active', 1, 1, $4, $4)`,
				fixture.principalID.String(), "Approval Principal "+string(rune('A'+index)), anchor, now); err != nil {
				t.Fatal(err)
			}
		}
		if !fixture.membershipID.IsZero() {
			if _, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, $2, $3, $4, 'merchant', 'active', 1, 1, $5, $5)`,
				fixture.membershipID.String(), tenantID.String(), fixture.principalID.String(), fixture.role, now); err != nil {
				t.Fatal(err)
			}
		}
		verifier := sha256.Sum256([]byte("approval-verifier-" + fixture.sessionID.String()))
		var stepUpAction any
		var stepUpVerifiedAt any
		if fixture.stepUpAction != "" {
			stepUpAction = fixture.stepUpAction
			stepUpVerifiedAt = now
		}
		if _, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.sessions (
    session_id, principal_id, population, tenant_id, verifier_sha256, assurance,
    status, authorization_version, rotation_version, version, created_at,
    last_seen_at, idle_expires_at, absolute_expires_at, step_up_action, step_up_verified_at
) VALUES ($1, $2, 'merchant', $3, $4, $5, 'active', 1, 1, 1, $6, $6, $7, $8, $9, $10)`,
			fixture.sessionID.String(), fixture.principalID.String(), tenantID.String(), verifier[:],
			string(fixture.actor.Assurance), now, now.Add(30*time.Minute), now.Add(8*time.Hour),
			stepUpAction, stepUpVerifiedAt); err != nil {
			t.Fatal(err)
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func cleanupApprovalIntegrationFixtures(t *testing.T, pool *pgxpool.Pool, tenantID identifier.ID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	statements := []string{
		`DELETE FROM atlas_identity.membership_role_changes WHERE tenant_id = $1`,
		`DELETE FROM atlas_operations.approval_executions WHERE tenant_id = $1`,
		`DELETE FROM atlas_operations.approval_cancellations WHERE tenant_id = $1`,
		`DELETE FROM atlas_operations.approval_decisions WHERE tenant_id = $1`,
		`DELETE FROM atlas_operations.approval_requests WHERE tenant_id = $1`,
		`DELETE FROM atlas_audit.audit_events WHERE tenant_id = $1`,
		`DELETE FROM atlas_operations.approvals WHERE tenant_id = $1`,
		`DELETE FROM atlas_identity.sessions WHERE tenant_id = $1`,
		`WITH removed AS (
            DELETE FROM atlas_identity.memberships WHERE tenant_id = $1 RETURNING principal_id
        ) DELETE FROM atlas_identity.principals WHERE principal_id IN (SELECT principal_id FROM removed)`,
		`DELETE FROM atlas_identity.organizations WHERE tenant_id = $1`,
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement, tenantID.String()); err != nil {
			t.Errorf("cleanup %q: %v", statement, err)
		}
	}
}

func cleanupStaleApprovalIntegrationFixtures(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rows, err := pool.Query(ctx, `
SELECT tenant_id
FROM atlas_identity.organizations
WHERE display_name = 'Approval Integration Fixture'
  AND normalized_name LIKE 'approval-%'`)
	if err != nil {
		t.Fatal(err)
	}
	var tenantIDs []identifier.ID
	for rows.Next() {
		var tenantIDText string
		if err := rows.Scan(&tenantIDText); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		tenantID, err := identifier.Parse(tenantIDText)
		if err != nil || tenantID.Prefix() != "ten" {
			rows.Close()
			t.Fatalf("invalid stale approval fixture tenant: %q", tenantIDText)
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	for _, tenantID := range tenantIDs {
		cleanupApprovalIntegrationFixtures(t, pool, tenantID)
	}
}

func newCreateApprovalCommand(
	t *testing.T,
	actor identity.Session,
	targetMembershipID, approvalID identifier.ID,
	idempotency string,
	now time.Time,
) operations.CreateApprovalCommand {
	t.Helper()
	payload := operations.MembershipRoleChangePayload{
		OrganizationID: actor.TenantID, MembershipID: targetMembershipID,
		ExpectedMembershipVersion: 1, RequestedRole: "merchant_admin",
		Purpose: "organization_administration",
	}
	canonical, payloadDigest, err := operations.CanonicalMembershipRoleChangePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	requestDigest := sha256.Sum256(append([]byte(operations.ActionMembershipChangeAdmin+"\n"), canonical...))
	return operations.CreateApprovalCommand{
		Actor: actor, ApprovalID: approvalID, ActionType: operations.ActionMembershipChangeAdmin,
		Payload: payload, CanonicalPayload: canonical, PayloadDigest: payloadDigest,
		IdempotencyDigest: sha256.Sum256([]byte(idempotency)), RequestDigest: requestDigest,
		Now: now, ExpiresAt: now.Add(24 * time.Hour),
		AuditEvent: newApprovalAuditEvent(t, actor, approvalID,
			"operations.approval.create", "approval_created", now),
	}
}

func newDecideApprovalCommand(
	t *testing.T,
	actor identity.Session,
	approvalID identifier.ID,
	version int64,
	decision, idempotency string,
	now time.Time,
) operations.DecideApprovalCommand {
	t.Helper()
	return operations.DecideApprovalCommand{
		Actor: actor, DecisionRecordID: newApprovalIntegrationID(t, "apd"),
		ApprovalID: approvalID, ExpectedVersion: version, Decision: decision,
		Purpose: "approval_review", Reason: "independent synthetic review",
		IdempotencyDigest: sha256.Sum256([]byte(idempotency)),
		RequestDigest:     sha256.Sum256([]byte(decision + "\n" + idempotency)),
		Now:               now,
		AuditEvent: newApprovalAuditEvent(t, actor, approvalID,
			"operations.approval.decide", "approval_decided", now),
	}
}

func newExecuteApprovalCommand(
	t *testing.T,
	actor identity.Session,
	approvalID, targetMembershipID identifier.ID,
	version int64,
	idempotency string,
	now time.Time,
) operations.ExecuteApprovalCommand {
	t.Helper()
	targetEvent := newApprovalAuditEvent(t, actor, approvalID,
		"identity.organization.membership.role.change", "approved_membership_role_changed", now)
	targetEvent.TargetType = "membership"
	targetEvent.TargetID = targetMembershipID.String()
	return operations.ExecuteApprovalCommand{
		Actor: actor, ExecutionRecordID: newApprovalIntegrationID(t, "aex"),
		RoleChangeID: newApprovalIntegrationID(t, "mrc"), ApprovalID: approvalID,
		ExpectedVersion: version, IdempotencyDigest: sha256.Sum256([]byte(idempotency)),
		RequestDigest: sha256.Sum256([]byte(approvalID.String() + "\n" + idempotency)),
		Now:           now,
		AuditEvent: newApprovalAuditEvent(t, actor, approvalID,
			"operations.approval.execute", "approval_executed", now),
		TargetAuditEvent: targetEvent,
	}
}

func newCancelApprovalCommand(
	t *testing.T,
	actor identity.Session,
	approvalID identifier.ID,
	version int64,
	idempotency string,
	now time.Time,
) operations.CancelApprovalCommand {
	t.Helper()
	return operations.CancelApprovalCommand{
		Actor: actor, CancellationRecordID: newApprovalIntegrationID(t, "apc"),
		ApprovalID: approvalID, ExpectedVersion: version,
		Reason:            "synthetic maker cancellation",
		IdempotencyDigest: sha256.Sum256([]byte(idempotency)),
		RequestDigest:     sha256.Sum256([]byte(approvalID.String() + "\n" + idempotency)),
		Now:               now,
		AuditEvent: newApprovalAuditEvent(t, actor, approvalID,
			"operations.approval.cancel", "approval_cancelled", now),
	}
}

func newApprovalAuditEvent(
	t *testing.T,
	actor identity.Session,
	approvalID identifier.ID,
	action, reason string,
	now time.Time,
) audit.Event {
	t.Helper()
	return audit.Event{
		AuditEventID: newApprovalIntegrationID(t, "aud"), ActorID: actor.PrincipalID,
		ActorType: actor.PrincipalType, TenantID: actor.TenantID,
		SessionAssurance: string(actor.Assurance), Action: action,
		TargetType: "approval", TargetID: approvalID.String(),
		DecisionID: newApprovalIntegrationID(t, "dec"), Decision: "executed",
		ReasonCode: reason, CorrelationID: newApprovalIntegrationID(t, "cor"),
		ApprovalID: approvalID, OccurredAt: now,
	}
}

func createAndApproveIntegrationApproval(
	t *testing.T,
	ctx context.Context,
	store *Store,
	maker, checker identity.Session,
	targetMembershipID identifier.ID,
	idempotencyPrefix string,
	now time.Time,
) identifier.ID {
	t.Helper()
	approvalID := newApprovalIntegrationID(t, "apr")
	if _, err := store.Create(ctx, newCreateApprovalCommand(
		t, maker, targetMembershipID, approvalID, idempotencyPrefix+"-create", now,
	)); err != nil {
		t.Fatalf("create %s: %v", idempotencyPrefix, err)
	}
	if _, err := store.Decide(ctx, newDecideApprovalCommand(
		t, checker, approvalID, 1, "approve", idempotencyPrefix+"-decide", now.Add(time.Minute),
	)); err != nil {
		t.Fatalf("decide %s: %v", idempotencyPrefix, err)
	}
	return approvalID
}

func setApprovalFixtureRole(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	membershipID identifier.ID,
	role string,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
UPDATE atlas_identity.memberships
SET role_id = $2, authorization_version = authorization_version + 1,
    version = version + 1, updated_at = $3
WHERE membership_id = $1`, membershipID.String(), role, now); err != nil {
		t.Fatal(err)
	}
}

func refreshApprovalFixtureAuthority(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture *approvalIntegrationFixture,
) {
	t.Helper()
	var authorizationVersion int64
	if err := pool.QueryRow(ctx, `
SELECT authorization_version
FROM atlas_identity.memberships
WHERE membership_id = $1`, fixture.membershipID.String()).Scan(&authorizationVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
UPDATE atlas_identity.sessions
SET authorization_version = $2, version = version + 1
WHERE session_id = $1`, fixture.sessionID.String(), authorizationVersion); err != nil {
		t.Fatal(err)
	}
	fixture.actor.AuthorizationVersion = authorizationVersion
}

func assertApprovalStatusAndTarget(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	approvalID, membershipID identifier.ID,
	wantStatus operations.ApprovalStatus,
	wantRole string,
	wantVersion int64,
) {
	t.Helper()
	var status, role string
	var version int64
	if err := pool.QueryRow(ctx,
		`SELECT status FROM atlas_operations.approvals WHERE approval_id = $1`, approvalID.String(),
	).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT role_id, version FROM atlas_identity.memberships WHERE membership_id = $1`, membershipID.String(),
	).Scan(&role, &version); err != nil {
		t.Fatal(err)
	}
	if status != string(wantStatus) || role != wantRole || version != wantVersion {
		t.Fatalf("approval=%s target role=%s version=%d want=%s/%s/%d",
			status, role, version, wantStatus, wantRole, wantVersion)
	}
}

func assertExecutedApprovalEffects(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	approvalID identifier.ID,
	target approvalIntegrationFixture,
	payloadDigest [32]byte,
) {
	t.Helper()
	assertApprovalStatusAndTarget(t, ctx, pool, approvalID, target.membershipID,
		operations.ApprovalExecuted, "merchant_admin", 2)
	var sessionStatus string
	var executionCount int64
	var historyDigest []byte
	if err := pool.QueryRow(ctx,
		`SELECT status FROM atlas_identity.sessions WHERE session_id = $1`, target.sessionID.String(),
	).Scan(&sessionStatus); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM atlas_operations.approval_executions WHERE approval_id = $1`, approvalID.String(),
	).Scan(&executionCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
SELECT request_sha256
FROM atlas_identity.membership_role_changes
WHERE approval_id = $1`, approvalID.String()).Scan(&historyDigest); err != nil {
		t.Fatal(err)
	}
	if sessionStatus != "revoked" || executionCount != 1 ||
		string(historyDigest) != string(payloadDigest[:]) {
		t.Fatalf("execution effects session=%s executions=%d digest=%x want=%x",
			sessionStatus, executionCount, historyDigest, payloadDigest)
	}
}

func newApprovalIntegrationID(t *testing.T, prefix string) identifier.ID {
	t.Helper()
	id, err := identifier.New(prefix)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
