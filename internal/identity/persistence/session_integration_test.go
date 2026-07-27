package persistence

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	auditapplication "github.com/MichaelSeveen/atlas/internal/audit/application"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestSessionStoreRealPostgresOIDCRevocationAndAuthorityInvalidation(t *testing.T) {
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
	defer apiPool.Close()
	migrationPool, err := pgxpool.New(ctx, migrationURL)
	if err != nil {
		t.Fatal(err)
	}
	defer migrationPool.Close()
	store, err := NewSessionStore(apiPool, auditapplication.NewRecorder())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	state := sha256.Sum256([]byte("real-postgres-oidc-state"))
	nonce := sha256.Sum256([]byte("real-postgres-oidc-nonce"))
	transactionID := newIntegrationID(t, "oid")
	transaction := identity.OIDCTransaction{
		TransactionID: transactionID, Kind: identity.TransactionLogin,
		Population: identity.PopulationCustomer, StateDigest: state, NonceDigest: nonce,
		EncryptedPKCE: make([]byte, 64), EncryptionKeyVersion: 1,
		ReturnTo: "/customer", CreatedAt: now, ExpiresAt: now.Add(5 * time.Minute),
	}
	if err := store.PutOIDCTransaction(ctx, transaction); err != nil {
		t.Fatal(err)
	}
	consumed, err := store.TakeOIDCTransaction(ctx, state, now.Add(time.Minute))
	if err != nil || consumed.TransactionID != transactionID {
		t.Fatalf("take transaction=%+v err=%v", consumed, err)
	}
	if _, err := store.TakeOIDCTransaction(ctx, state, now.Add(time.Minute)); !errors.Is(err, identity.ErrOIDCTransactionInvalid) {
		t.Fatalf("OIDC replay error=%v", err)
	}

	correlationIDs := []identifier.ID{newIntegrationID(t, "cor"), newIntegrationID(t, "cor")}
	firstCookie, firstDigest := integrationToken(t)
	_ = firstCookie
	first := createIntegrationSession(
		t, ctx, store, firstDigest, newIntegrationID(t, "ses"), correlationIDs[0], now,
	)
	authenticated, err := store.Authenticate(ctx, firstDigest, now.Add(time.Minute), 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.PrincipalID.String() != "usr_01JAT1AS00000000000001" ||
		authenticated.TenantID.String() != "ten_01JAT1AS00000000000001" ||
		len(authenticated.Permissions) == 0 {
		t.Fatalf("unexpected authenticated authority: %+v", authenticated)
	}

	stepUpScope := sha256.Sum256([]byte("real-postgres-step-up-scope"))
	stepUpRequest := sha256.Sum256([]byte("v1\naction=identity.approval.decide"))
	stepUpClaim := identity.StepUpClaimCommand{
		ChallengeRequestID: newIntegrationID(t, "idr"), Actor: authenticated,
		IdempotencyScope: stepUpScope, RequestDigest: stepUpRequest,
		Action: "identity.approval.decide", CorrelationID: newIntegrationID(t, "cor"),
		Now: now.Add(time.Minute), ProcessingExpiresAt: now.Add(90 * time.Second),
		RetainedUntil: now.Add(24 * time.Hour),
	}
	type claimResult struct {
		result identity.StepUpClaimResult
		err    error
	}
	claims := make(chan claimResult, 2)
	var claimWait sync.WaitGroup
	claimWait.Add(2)
	for range 2 {
		go func() {
			defer claimWait.Done()
			result, claimErr := store.ClaimStepUp(ctx, stepUpClaim)
			claims <- claimResult{result: result, err: claimErr}
		}()
	}
	claimWait.Wait()
	close(claims)
	ownerCount := 0
	inProgressCount := 0
	for claim := range claims {
		switch {
		case claim.err == nil && claim.result.Owner:
			ownerCount++
		case errors.Is(claim.err, identity.ErrIdempotencyInProgress):
			inProgressCount++
		default:
			t.Fatalf("concurrent step-up claim=%+v err=%v", claim.result, claim.err)
		}
	}
	if ownerCount != 1 || inProgressCount != 1 {
		t.Fatalf("concurrent step-up owner=%d in_progress=%d", ownerCount, inProgressCount)
	}
	reclaimedStepUp := stepUpClaim
	reclaimedStepUp.ChallengeRequestID = newIntegrationID(t, "idr")
	reclaimedStepUp.CorrelationID = newIntegrationID(t, "cor")
	reclaimedStepUp.Now = now.Add(2 * time.Minute)
	reclaimedStepUp.ProcessingExpiresAt = now.Add(3 * time.Minute)
	reclaimed, err := store.ClaimStepUp(ctx, reclaimedStepUp)
	if err != nil || !reclaimed.Owner ||
		reclaimed.ChallengeRequestID != reclaimedStepUp.ChallengeRequestID {
		t.Fatalf("reclaimed step-up=%+v err=%v", reclaimed, err)
	}
	staleTransaction := identity.OIDCTransaction{
		TransactionID: newIntegrationID(t, "oid"), Kind: identity.TransactionStepUp,
		Population:    identity.PopulationCustomer,
		StateDigest:   sha256.Sum256([]byte("real-postgres-stale-step-up-state")),
		NonceDigest:   sha256.Sum256([]byte("real-postgres-stale-step-up-nonce")),
		EncryptedPKCE: make([]byte, 64), EncryptionKeyVersion: 1,
		ReturnTo: "/customer", PrincipalID: authenticated.PrincipalID,
		ReplacedSessionID: authenticated.SessionID,
		RequestedAction:   "identity.approval.decide",
		CreatedAt:         now.Add(2 * time.Minute), ExpiresAt: now.Add(7 * time.Minute),
	}
	if err := store.CompleteStepUp(ctx, identity.CompleteStepUpCommand{
		ChallengeRequestID: stepUpClaim.ChallengeRequestID,
		IdempotencyScope:   stepUpScope, Transaction: staleTransaction,
		AuthorizationURL: "https://identity.test.invalid/authorize?stale=1",
		CompletedAt:      now.Add(2 * time.Minute),
	}); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("stale step-up owner completion error=%v", err)
	}
	stepUpTransactionID := newIntegrationID(t, "oid")
	stepUpTransaction := identity.OIDCTransaction{
		TransactionID: stepUpTransactionID, Kind: identity.TransactionStepUp,
		Population:    identity.PopulationCustomer,
		StateDigest:   sha256.Sum256([]byte("real-postgres-step-up-state")),
		NonceDigest:   sha256.Sum256([]byte("real-postgres-step-up-nonce")),
		EncryptedPKCE: make([]byte, 64), EncryptionKeyVersion: 1,
		ReturnTo: "/customer", PrincipalID: authenticated.PrincipalID,
		ReplacedSessionID: authenticated.SessionID,
		RequestedAction:   "identity.approval.decide",
		CreatedAt:         now.Add(time.Minute), ExpiresAt: now.Add(6 * time.Minute),
	}
	if err := store.CompleteStepUp(ctx, identity.CompleteStepUpCommand{
		ChallengeRequestID: reclaimedStepUp.ChallengeRequestID,
		IdempotencyScope:   stepUpScope, Transaction: stepUpTransaction,
		AuthorizationURL: "https://identity.test.invalid/authorize?step-up=1",
		CompletedAt:      now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	stepUpReplay, err := store.ClaimStepUp(ctx, stepUpClaim)
	if err != nil || !stepUpReplay.Replay ||
		stepUpReplay.TransactionID != stepUpTransactionID ||
		stepUpReplay.AuthorizationURL != "https://identity.test.invalid/authorize?step-up=1" {
		t.Fatalf("step-up replay=%+v err=%v", stepUpReplay, err)
	}
	changedStepUp := stepUpClaim
	changedStepUp.RequestDigest = sha256.Sum256([]byte("v1\naction=identity.approval.execute"))
	if _, err := store.ClaimStepUp(ctx, changedStepUp); !errors.Is(err, identity.ErrIdempotencyConflict) {
		t.Fatalf("changed step-up request error=%v", err)
	}

	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.memberships
SET authorization_version = authorization_version + 1, version = version + 1
WHERE membership_id = 'mem_01JAT1AS00000000000001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate(ctx, firstDigest, now.Add(2*time.Minute), 30*time.Minute); !errors.Is(err, identity.ErrAuthenticationRequired) {
		t.Fatalf("stale authority version error=%v", err)
	}
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.memberships
SET authorization_version = 1, version = version + 1
WHERE membership_id = 'mem_01JAT1AS00000000000001'`); err != nil {
		t.Fatal(err)
	}

	revokeEvent := integrationAuditEvent(t, first, first.SessionID, correlationIDs[0], now.Add(3*time.Minute), "revoke_one")
	result, err := store.RevokeOne(ctx, identity.RevocationCommand{
		Actor: first, TargetSessionID: first.SessionID,
		Now: now.Add(3 * time.Minute), AuditEvent: revokeEvent,
	})
	if err != nil || !result.CurrentRevoked {
		t.Fatalf("revoke one result=%+v err=%v", result, err)
	}
	if _, err := store.Authenticate(ctx, firstDigest, now.Add(4*time.Minute), 30*time.Minute); !errors.Is(err, identity.ErrSessionRevoked) {
		t.Fatalf("revoked session error=%v", err)
	}

	_, secondDigest := integrationToken(t)
	second := createIntegrationSession(
		t, ctx, store, secondDigest, newIntegrationID(t, "ses"), correlationIDs[1],
		now.Add(5*time.Minute),
	)
	idempotency := sha256.Sum256([]byte("real-postgres-revoke-all-key"))
	requestDigest := sha256.Sum256([]byte("include_current=true"))
	revokeAll := identity.RevocationCommand{
		Actor: second, IncludeCurrent: true, RevocationRequestID: newIntegrationID(t, "rev"),
		IdempotencyDigest: idempotency, RequestDigest: requestDigest,
		Now: now.Add(6 * time.Minute),
		AuditEvent: integrationAuditEvent(
			t, second, second.PrincipalID, correlationIDs[1], now.Add(6*time.Minute), "revoke_all",
		),
	}
	type concurrentResult struct {
		result identity.RevocationResult
		err    error
	}
	results := make(chan concurrentResult, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			revokeResult, revokeErr := store.RevokeAll(ctx, revokeAll)
			results <- concurrentResult{result: revokeResult, err: revokeErr}
		}()
	}
	wait.Wait()
	close(results)
	originalCount := 0
	replayCount := 0
	for concurrent := range results {
		if concurrent.err != nil || !concurrent.result.CurrentRevoked {
			t.Fatalf("concurrent revoke-all result=%+v err=%v", concurrent.result, concurrent.err)
		}
		if concurrent.result.Replay {
			replayCount++
		} else {
			originalCount++
		}
	}
	if originalCount != 1 || replayCount != 1 {
		t.Fatalf("concurrent revoke-all original=%d replay=%d", originalCount, replayCount)
	}
	replay, err := store.RevokeAll(ctx, revokeAll)
	if err != nil || !replay.Replay || !replay.CurrentRevoked {
		t.Fatalf("revoke-all replay result=%+v err=%v", replay, err)
	}
	revokeAll.RequestDigest = sha256.Sum256([]byte("include_current=false"))
	if _, err := store.RevokeAll(ctx, revokeAll); !errors.Is(err, identity.ErrSessionConflict) {
		t.Fatalf("changed idempotent request error=%v", err)
	}

	adminActorCorrelation := newIntegrationID(t, "cor")
	adminTargetCorrelation := newIntegrationID(t, "cor")
	adminDecisionCorrelation := newIntegrationID(t, "cor")
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.principal_roles
SET role_id = 'platform_administrator',
    authorization_version = 2,
    version = version + 1,
    updated_at = $1
WHERE principal_role_id = 'prr_01JAT1AS00000000000001'`,
		now.Add(7*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	_, adminActorDigest := integrationToken(t)
	adminActor := createWorkforceIntegrationSession(
		t, ctx, store, adminActorDigest, newIntegrationID(t, "ses"),
		adminActorCorrelation, now.Add(7*time.Minute), "identity.session.admin_revoke",
	)
	_, adminTargetDigest := integrationToken(t)
	adminTarget := createIntegrationSession(
		t, ctx, store, adminTargetDigest, newIntegrationID(t, "ses"),
		adminTargetCorrelation, now.Add(7*time.Minute),
	)
	adminKey := sha256.Sum256([]byte("real-postgres-admin-revocation-key"))
	adminRequestDigest := sha256.Sum256([]byte(
		"v1\ntarget=" + adminTarget.SessionID.String() +
			"\npurpose=security_review\nreason=compromised_session",
	))
	adminCommand := identity.AdminRevocationCommand{
		Actor: adminActor, TargetSessionID: adminTarget.SessionID,
		RevocationRequestID: newIntegrationID(t, "asr"),
		IdempotencyDigest:   adminKey, RequestDigest: adminRequestDigest,
		Purpose: "security_review", Reason: "compromised_session",
		Now: now.Add(8 * time.Minute),
		AuditEvent: adminIntegrationAuditEvent(
			t, adminActor, adminTarget.SessionID, adminDecisionCorrelation,
			now.Add(8*time.Minute), "compromised_session",
		),
	}
	type adminConcurrentResult struct {
		result identity.AdminRevocationResult
		err    error
	}
	adminResults := make(chan adminConcurrentResult, 2)
	var adminWait sync.WaitGroup
	adminWait.Add(2)
	for range 2 {
		go func() {
			defer adminWait.Done()
			result, revokeErr := store.RevokeForSecurity(ctx, adminCommand)
			adminResults <- adminConcurrentResult{result: result, err: revokeErr}
		}()
	}
	adminWait.Wait()
	close(adminResults)
	adminOriginalCount := 0
	adminReplayCount := 0
	var adminDecisionID identifier.ID
	for concurrent := range adminResults {
		if concurrent.err != nil || concurrent.result.DecisionID.IsZero() {
			t.Fatalf("concurrent administrator revocation result=%+v err=%v", concurrent.result, concurrent.err)
		}
		if adminDecisionID.IsZero() {
			adminDecisionID = concurrent.result.DecisionID
		} else if concurrent.result.DecisionID != adminDecisionID {
			t.Fatalf("administrator revocation decision drifted: %s != %s",
				concurrent.result.DecisionID, adminDecisionID)
		}
		if concurrent.result.Replay {
			adminReplayCount++
		} else {
			adminOriginalCount++
		}
	}
	if adminOriginalCount != 1 || adminReplayCount != 1 {
		t.Fatalf("concurrent administrator revocation original=%d replay=%d",
			adminOriginalCount, adminReplayCount)
	}
	if _, err := store.Authenticate(
		ctx, adminTargetDigest, now.Add(9*time.Minute), 30*time.Minute,
	); !errors.Is(err, identity.ErrSessionRevoked) {
		t.Fatalf("administrator-revoked target error=%v", err)
	}
	changedAdmin := adminCommand
	changedAdmin.RequestDigest = sha256.Sum256([]byte("changed-administrator-request"))
	if _, err := store.RevokeForSecurity(
		ctx, changedAdmin,
	); !errors.Is(err, identity.ErrIdempotencyConflict) {
		t.Fatalf("changed administrator idempotency request error=%v", err)
	}

	staleActorCorrelation := newIntegrationID(t, "cor")
	deniedCorrelation := newIntegrationID(t, "cor")
	_, staleActorDigest := integrationToken(t)
	staleActor := createWorkforceIntegrationSession(
		t, ctx, store, staleActorDigest, newIntegrationID(t, "ses"),
		staleActorCorrelation, now.Add(9*time.Minute), "",
	)
	deniedCommand := adminCommand
	deniedCommand.Actor = staleActor
	deniedCommand.RevocationRequestID = newIntegrationID(t, "asr")
	deniedCommand.IdempotencyDigest = sha256.Sum256([]byte("stale-admin-revocation-key"))
	deniedCommand.RequestDigest = sha256.Sum256([]byte("stale-admin-revocation-request"))
	deniedCommand.Now = now.Add(10 * time.Minute)
	deniedCommand.AuditEvent = adminIntegrationAuditEvent(
		t, staleActor, adminTarget.SessionID, deniedCorrelation,
		now.Add(10*time.Minute), "compromised_session",
	)
	denied, err := store.RevokeForSecurity(ctx, deniedCommand)
	if !errors.Is(err, identity.ErrStepUpRequired) || denied.DecisionID.IsZero() {
		t.Fatalf("stale administrator step-up result=%+v err=%v", denied, err)
	}
	deniedReplay, err := store.RevokeForSecurity(ctx, deniedCommand)
	if !errors.Is(err, identity.ErrStepUpRequired) || !deniedReplay.Replay ||
		deniedReplay.DecisionID != denied.DecisionID {
		t.Fatalf("stale administrator replay result=%+v err=%v", deniedReplay, err)
	}

	rollbackTargetCorrelation := newIntegrationID(t, "cor")
	failedDecisionCorrelation := newIntegrationID(t, "cor")
	_, rollbackTargetDigest := integrationToken(t)
	rollbackTarget := createIntegrationSession(
		t, ctx, store, rollbackTargetDigest, newIntegrationID(t, "ses"),
		rollbackTargetCorrelation, now.Add(10*time.Minute),
	)
	failingStore, err := NewSessionStore(apiPool, failingAuditRecorder{})
	if err != nil {
		t.Fatal(err)
	}
	failedCommand := adminCommand
	failedCommand.TargetSessionID = rollbackTarget.SessionID
	failedCommand.RevocationRequestID = newIntegrationID(t, "asr")
	failedCommand.IdempotencyDigest = sha256.Sum256([]byte("audit-outage-admin-revocation-key"))
	failedCommand.RequestDigest = sha256.Sum256([]byte("audit-outage-admin-revocation-request"))
	failedCommand.Now = now.Add(11 * time.Minute)
	failedCommand.AuditEvent = adminIntegrationAuditEvent(
		t, adminActor, rollbackTarget.SessionID, failedDecisionCorrelation,
		now.Add(11*time.Minute), "workforce_security_response",
	)
	if _, err := failingStore.RevokeForSecurity(
		ctx, failedCommand,
	); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("Audit-outage administrator revocation error=%v", err)
	}
	if _, err := store.Authenticate(
		ctx, rollbackTargetDigest, now.Add(11*time.Minute), 30*time.Minute,
	); err != nil {
		t.Fatalf("Audit-outage revocation committed target mutation: %v", err)
	}

	var auditCount int
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*)
FROM atlas_audit.audit_events
WHERE correlation_id = ANY($1)`,
		[]string{correlationIDs[0].String(), correlationIDs[1].String()},
	).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 4 {
		t.Fatalf("atomic session audit count=%d, want 4", auditCount)
	}
	var adminAuditCount int
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*)
FROM atlas_audit.audit_events
WHERE correlation_id = ANY($1)`,
		[]string{
			adminDecisionCorrelation.String(),
			deniedCorrelation.String(),
			failedDecisionCorrelation.String(),
		},
	).Scan(&adminAuditCount); err != nil {
		t.Fatal(err)
	}
	if adminAuditCount != 2 {
		t.Fatalf("administrator decision audit count=%d, want 2", adminAuditCount)
	}
	if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.principal_roles
SET role_id = 'operations',
    authorization_version = 1,
    version = 1,
    updated_at = '2026-07-26T00:00:00Z'
WHERE principal_role_id = 'prr_01JAT1AS00000000000001'`); err != nil {
		t.Fatal(err)
	}

	cleanupIntegrationState(
		t, ctx, migrationPool, []identifier.ID{transactionID, stepUpTransactionID},
		[]identifier.ID{
			first.SessionID, second.SessionID, adminActor.SessionID, adminTarget.SessionID,
			staleActor.SessionID, rollbackTarget.SessionID,
		},
		append(correlationIDs,
			adminActorCorrelation, adminTargetCorrelation, adminDecisionCorrelation,
			staleActorCorrelation, deniedCorrelation, rollbackTargetCorrelation,
			failedDecisionCorrelation,
		),
	)
}

type failingAuditRecorder struct{}

func (failingAuditRecorder) Record(context.Context, audit.Transaction, audit.Event) error {
	return errors.New("synthetic Audit outage")
}

func createIntegrationSession(
	t *testing.T,
	ctx context.Context,
	store *SessionStore,
	verifierDigest [32]byte,
	sessionID identifier.ID,
	correlationID identifier.ID,
	now time.Time,
) identity.Session {
	t.Helper()
	command := identity.CreateSessionCommand{
		Claims: identity.ProviderClaims{
			Issuer:  "http://keycloak:8080/realms/atlas-customer-local",
			Subject: "00000000-0000-4000-8000-000000000101",
			Nonce:   "not-persisted", Assurance: identity.AssuranceBaseline,
			AuthenticatedAt: now,
		},
		Population: identity.PopulationCustomer, Kind: identity.TransactionLogin,
		SessionID: sessionID, VerifierDigest: verifierDigest,
		Assurance: identity.AssuranceBaseline, AuthorizationAt: now,
		IdleExpiresAt: now.Add(30 * time.Minute), AbsoluteExpiresAt: now.Add(12 * time.Hour),
		ClientLabel: "integration-browser",
		AuditEvent: audit.Event{
			AuditEventID:     newIntegrationID(t, "aud"),
			SessionAssurance: "baseline", Action: "identity.session.login",
			TargetType: "session", TargetID: sessionID.String(),
			DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
			ReasonCode: "oidc_login", CorrelationID: correlationID, OccurredAt: now,
			GlobalScope: "identity-security",
		},
	}
	session, err := store.CreateSession(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func createWorkforceIntegrationSession(
	t *testing.T,
	ctx context.Context,
	store *SessionStore,
	verifierDigest [32]byte,
	sessionID identifier.ID,
	correlationID identifier.ID,
	now time.Time,
	stepUpAction string,
) identity.Session {
	t.Helper()
	kind := identity.TransactionLogin
	auditAction := "identity.session.login"
	auditReason := "oidc_login"
	stepUpVerifiedAt := time.Time{}
	if stepUpAction != "" {
		kind = identity.TransactionStepUp
		auditAction = "identity.session.step_up"
		auditReason = "oidc_step_up"
		stepUpVerifiedAt = now
	}
	command := identity.CreateSessionCommand{
		Claims: identity.ProviderClaims{
			Issuer:  "http://keycloak:8080/realms/atlas-workforce-local",
			Subject: "00000000-0000-4000-8000-000000000301",
			Nonce:   "not-persisted", Assurance: identity.AssurancePhishingResistant,
			AuthenticatedAt: now,
		},
		Population: identity.PopulationWorkforce, Kind: kind,
		SessionID: sessionID, VerifierDigest: verifierDigest,
		Assurance: identity.AssurancePhishingResistant, AuthorizationAt: now,
		StepUpAction: stepUpAction, StepUpVerifiedAt: stepUpVerifiedAt,
		IdleExpiresAt: now.Add(10 * time.Minute), AbsoluteExpiresAt: now.Add(time.Hour),
		ClientLabel: "integration-workforce-browser",
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), ActorType: "workforce",
			GlobalScope: "identity-security", SessionAssurance: "phishing-resistant",
			Action: auditAction, TargetType: "session", TargetID: sessionID.String(),
			DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
			ReasonCode: auditReason, CorrelationID: correlationID, OccurredAt: now,
		},
	}
	session, err := store.CreateSession(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func adminIntegrationAuditEvent(
	t *testing.T,
	actor identity.Session,
	target identifier.ID,
	correlationID identifier.ID,
	now time.Time,
	reason string,
) audit.Event {
	t.Helper()
	return audit.Event{
		AuditEventID: newIntegrationID(t, "aud"), ActorID: actor.PrincipalID,
		ActorType: actor.PrincipalType, GlobalScope: "identity-security",
		SessionAssurance: string(actor.Assurance), Action: "identity.session.admin_revoke",
		TargetType: "session", TargetID: target.String(),
		DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
		ReasonCode: reason, CorrelationID: correlationID, OccurredAt: now,
	}
}

func integrationAuditEvent(
	t *testing.T,
	actor identity.Session,
	target identifier.ID,
	correlationID identifier.ID,
	now time.Time,
	reason string,
) audit.Event {
	t.Helper()
	return audit.Event{
		AuditEventID: newIntegrationID(t, "aud"), ActorID: actor.PrincipalID,
		ActorType: actor.PrincipalType, TenantID: actor.TenantID,
		SessionAssurance: string(actor.Assurance), Action: "identity.session.revoke",
		TargetType: "session", TargetID: target.String(),
		DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
		ReasonCode: reason, CorrelationID: correlationID, OccurredAt: now,
	}
}

func integrationToken(t *testing.T) (string, [32]byte) {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	return string(raw), sha256.Sum256(raw)
}

func newIntegrationID(t *testing.T, prefix string) identifier.ID {
	t.Helper()
	value, err := identifier.New(prefix)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func cleanupIntegrationState(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	transactionIDs []identifier.ID,
	sessionIDs []identifier.ID,
	correlationIDs []identifier.ID,
) {
	t.Helper()
	sessionTexts := make([]string, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		sessionTexts = append(sessionTexts, sessionID.String())
	}
	correlationTexts := make([]string, 0, len(correlationIDs))
	for _, correlationID := range correlationIDs {
		correlationTexts = append(correlationTexts, correlationID.String())
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM atlas_identity.session_revocation_requests
WHERE principal_id = 'usr_01JAT1AS00000000000001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM atlas_identity.step_up_challenge_requests
WHERE principal_id = 'usr_01JAT1AS00000000000001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
DELETE FROM atlas_identity.admin_session_revocation_requests
WHERE actor_principal_id = 'usr_01JAT1AS00000000000003'`); err != nil {
		t.Fatal(err)
	}
	transactionTexts := make([]string, 0, len(transactionIDs))
	for _, transactionID := range transactionIDs {
		transactionTexts = append(transactionTexts, transactionID.String())
	}
	if _, err := pool.Exec(ctx, `DELETE FROM atlas_identity.oidc_transactions WHERE transaction_id = ANY($1)`, transactionTexts); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM atlas_identity.sessions WHERE session_id = ANY($1)`, sessionTexts); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM atlas_audit.audit_events WHERE correlation_id = ANY($1)`, correlationTexts); err != nil {
		t.Fatal(err)
	}
}
