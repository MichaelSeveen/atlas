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

func TestAPICredentialRealPostgresOneTimeStateRotationRevocationAndAuditRollback(t *testing.T) {
	apiURL := os.Getenv("ATLAS_P01_DATABASE_URL")
	migrationURL := os.Getenv("ATLAS_P01_MIGRATION_DATABASE_URL")
	if apiURL == "" || migrationURL == "" {
		t.Skip("real Phase 01 database URLs are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	store, err := NewCredentialStore(apiPool, auditapplication.NewRecorder())
	if err != nil {
		t.Fatal(err)
	}
	failingStore, err := NewCredentialStore(apiPool, failingAuditRecorder{})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 9, 14, 0, 0, 0, time.UTC)
	tenantID := newIntegrationID(t, "ten")
	principalID := newIntegrationID(t, "usr")
	membershipID := newIntegrationID(t, "mem")
	sessionID := newIntegrationID(t, "ses")
	sessionVerifier := sha256.Sum256([]byte("credential-session-" + sessionID.String()))
	credentialIDs := []identifier.ID{
		newIntegrationID(t, "key"), newIntegrationID(t, "key"), newIntegrationID(t, "key"),
		newIntegrationID(t, "key"), newIntegrationID(t, "key"),
	}
	auditIDs := make([]string, 0, 8)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.api_credential_mutation_requests WHERE tenant_id = $1`, tenantID.String())
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.api_credentials WHERE tenant_id = $1`, tenantID.String())
		if len(auditIDs) > 0 {
			_, _ = migrationPool.Exec(cleanupCtx,
				`DELETE FROM atlas_audit.audit_events WHERE audit_event_id = ANY($1)`, auditIDs)
		}
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.sessions WHERE session_id = $1`, sessionID.String())
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.memberships WHERE membership_id = $1`, membershipID.String())
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.principals WHERE principal_id = $1`, principalID.String())
		_, _ = migrationPool.Exec(cleanupCtx,
			`DELETE FROM atlas_identity.organizations WHERE tenant_id = $1`, tenantID.String())
	})

	setup, err := migrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = setup.Rollback(ctx) }()
	uniqueName := "credential-" + strings.ToLower(strings.TrimPrefix(tenantID.String(), "ten_"))
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.organizations (
    tenant_id, organization_type, display_name, normalized_name, confusable_skeleton,
    status, authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', 'Credential Fixture', $2, $2, 'active', 1, 1, $3, $3)`,
		tenantID.String(), uniqueName, now); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', 'Credential Administrator', $2, 'active', 1, 1, $3, $3)`,
		principalID.String(), "syn_person_credential_"+strings.ToLower(strings.TrimPrefix(principalID.String(), "usr_")), now); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, $2, $3, 'merchant_security_admin', 'merchant', 'active', 1, 1, $4, $4)`,
		membershipID.String(), tenantID.String(), principalID.String(), now); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.sessions (
    session_id, principal_id, population, tenant_id, verifier_sha256, assurance,
    status, authorization_version, rotation_version, version, created_at,
    last_seen_at, idle_expires_at, absolute_expires_at, step_up_action, step_up_verified_at
) VALUES ($1, $2, 'merchant', $3, $4, 'phishing-resistant', 'active', 1, 1, 1,
          $5, $5, $6, $7, $8, $5)`,
		sessionID.String(), principalID.String(), tenantID.String(), sessionVerifier[:], now,
		now.Add(20*time.Minute), now.Add(8*time.Hour), identity.CredentialActionCreate); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	actor := identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		Population: identity.PopulationMerchant, TenantID: tenantID,
		Assurance:            identity.AssurancePhishingResistant,
		AuthorizationVersion: 1, RotationVersion: 1,
		CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(20 * time.Minute),
		AbsoluteExpiresAt: now.Add(8 * time.Hour),
	}
	newEvent := func(action, target, reason string, occurredAt time.Time) audit.Event {
		auditID := newIntegrationID(t, "aud")
		auditIDs = append(auditIDs, auditID.String())
		return audit.Event{
			AuditEventID: auditID, ActorID: principalID, ActorType: "merchant", TenantID: tenantID,
			SessionAssurance: string(identity.AssurancePhishingResistant), Action: action,
			TargetType: "api_credential", TargetID: target,
			DecisionID: newIntegrationID(t, "dec"), Decision: "executed", ReasonCode: reason,
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: occurredAt,
			SafeAfterReference: "credential-status:active",
		}
	}
	setStepUp := func(action string, at time.Time) {
		t.Helper()
		if _, err := migrationPool.Exec(ctx, `
UPDATE atlas_identity.sessions
SET step_up_action = $2, step_up_verified_at = $3, version = version + 1
WHERE session_id = $1`, sessionID.String(), action, at); err != nil {
			t.Fatal(err)
		}
	}

	originalSecret := strings.Repeat("original-secret-material-", 3)
	originalVerifier := sha256.Sum256([]byte(originalSecret))
	createEvent := newEvent(identity.CredentialActionCreate, credentialIDs[0].String(), "api_credential_created", now)
	createCommand := identity.CreateCredentialCommand{
		Actor: actor,
		Credential: identity.APICredential{
			CredentialID: credentialIDs[0], OrganizationID: tenantID, Name: "Reconciliation reader",
			SecretHint: "safehint", Scopes: []string{identity.APICredentialScopeIdentityRead},
			Status: identity.APICredentialActive, Environment: "test",
			Audience: identity.APICredentialAudience, Version: 1,
			ExpiresAt: now.Add(identity.CredentialDefaultExpiry), CreatedAt: now,
		},
		Verifier: originalVerifier, IdempotencyDigest: sha256.Sum256([]byte("credential-create-db-1")),
		RequestDigest: sha256.Sum256([]byte("credential-create-request-v1")),
		Now:           now, AuditEvent: createEvent,
	}
	created, err := store.CreateCredential(ctx, createCommand)
	if err != nil || created.Replay || created.Credential.CredentialID != credentialIDs[0] {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	replay, err := store.CreateCredential(ctx, createCommand)
	if err != nil || !replay.Replay || replay.Credential.CredentialID != credentialIDs[0] {
		t.Fatalf("create replay=%+v err=%v", replay, err)
	}
	changed := createCommand
	changed.RequestDigest = sha256.Sum256([]byte("credential-create-request-changed"))
	if _, err := store.CreateCredential(ctx, changed); !errors.Is(err, identity.ErrIdempotencyConflict) {
		t.Fatalf("changed idempotent request error=%v", err)
	}
	var storedDocument string
	if err := migrationPool.QueryRow(ctx, `
SELECT to_jsonb(credential)::text
FROM atlas_identity.api_credentials AS credential
WHERE credential_id = $1`, credentialIDs[0].String()).Scan(&storedDocument); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(storedDocument, originalSecret) || strings.Contains(storedDocument, "192.0.2.10") ||
		!strings.Contains(storedDocument, "secret_verifier_sha256") {
		t.Fatalf("credential storage contains forbidden or missing material: %s", storedDocument)
	}

	networkOne := sha256.Sum256([]byte("network-one"))
	networkTwo := sha256.Sum256([]byte("network-two"))
	authenticate := func(id identifier.ID, verifier [32]byte, at time.Time, environment, audience, scope string, network [32]byte) (identity.CredentialAuthentication, error) {
		return store.AuthenticateCredential(ctx, identity.AuthenticateCredentialCommand{
			CredentialID: id, Verifier: verifier, Environment: environment,
			Audience: audience, Scope: scope, NetworkSignal: network, Now: at,
		})
	}
	firstUse, err := authenticate(credentialIDs[0], originalVerifier, now.Add(time.Minute), "test",
		identity.APICredentialAudience, identity.APICredentialScopeIdentityRead, networkOne)
	if err != nil || firstUse.AnomalousNetwork {
		t.Fatalf("first use=%+v err=%v", firstUse, err)
	}
	secondUse, err := authenticate(credentialIDs[0], originalVerifier, now.Add(2*time.Minute), "test",
		identity.APICredentialAudience, identity.APICredentialScopeIdentityRead, networkTwo)
	if err != nil || !secondUse.AnomalousNetwork {
		t.Fatalf("network anomaly=%+v err=%v", secondUse, err)
	}
	for name, mutate := range map[string]func(*identity.AuthenticateCredentialCommand){
		"environment": func(command *identity.AuthenticateCredentialCommand) { command.Environment = "staging" },
		"audience":    func(command *identity.AuthenticateCredentialCommand) { command.Audience = "other" },
		"scope":       func(command *identity.AuthenticateCredentialCommand) { command.Scope = "money:write" },
		"verifier": func(command *identity.AuthenticateCredentialCommand) {
			command.Verifier = sha256.Sum256([]byte("wrong"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			command := identity.AuthenticateCredentialCommand{
				CredentialID: credentialIDs[0], Verifier: originalVerifier, Environment: "test",
				Audience: identity.APICredentialAudience, Scope: identity.APICredentialScopeIdentityRead,
				NetworkSignal: networkOne, Now: now.Add(3 * time.Minute),
			}
			mutate(&command)
			if _, err := store.AuthenticateCredential(ctx, command); !errors.Is(err, identity.ErrAuthenticationRequired) {
				t.Fatalf("wrong binding error=%v", err)
			}
		})
	}

	setStepUp(identity.CredentialActionRotate, now.Add(time.Minute))
	rotateCommands := make([]identity.RotateCredentialCommand, 2)
	rotateVerifiers := make([][32]byte, 2)
	for index := range rotateCommands {
		rotateVerifiers[index] = sha256.Sum256([]byte("replacement-" + credentialIDs[index+1].String()))
		event := newEvent(identity.CredentialActionRotate, credentialIDs[0].String(), "api_credential_rotated", now.Add(3*time.Minute))
		rotateCommands[index] = identity.RotateCredentialCommand{
			Actor: actor, SourceCredentialID: credentialIDs[0], ReplacementID: credentialIDs[index+1],
			ReplacementVerifier: rotateVerifiers[index], ReplacementSecretHint: "newhint0",
			Environment: "test", IdempotencyDigest: sha256.Sum256([]byte("credential-rotate-db-" + string(rune('1'+index)))),
			RequestDigest: sha256.Sum256([]byte("credential-rotate-request-" + string(rune('1'+index)))),
			Now:           now.Add(3 * time.Minute), AuditEvent: event,
		}
	}
	type rotateOutcome struct {
		index  int
		result identity.CredentialMutationResult
		err    error
	}
	outcomes := make(chan rotateOutcome, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	start := make(chan struct{})
	for index := range rotateCommands {
		go func(index int) {
			ready.Done()
			<-start
			result, callErr := store.RotateCredential(ctx, rotateCommands[index])
			outcomes <- rotateOutcome{index: index, result: result, err: callErr}
		}(index)
	}
	ready.Wait()
	close(start)
	first := <-outcomes
	second := <-outcomes
	close(outcomes)
	winner := first
	loser := second
	if first.err != nil {
		winner, loser = second, first
	}
	if winner.err != nil || !errors.Is(loser.err, identity.ErrCredentialConflict) {
		t.Fatalf("concurrent rotations winner=%+v loser=%+v", winner, loser)
	}
	var deniedAuditRows, losingMutationRows int
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_audit.audit_events
WHERE audit_event_id = $1 AND decision = 'denied' AND reason_code = 'credential_state_conflict'`,
		rotateCommands[loser.index].AuditEvent.AuditEventID.String()).Scan(&deniedAuditRows); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.api_credential_mutation_requests
WHERE tenant_id = $1 AND actor_principal_id = $2 AND operation = 'rotate'
  AND idempotency_key_sha256 = $3`, tenantID.String(), principalID.String(),
		rotateCommands[loser.index].IdempotencyDigest[:]).Scan(&losingMutationRows); err != nil {
		t.Fatal(err)
	}
	if deniedAuditRows != 1 || losingMutationRows != 0 {
		t.Fatalf("losing rotation audit=%d mutation=%d", deniedAuditRows, losingMutationRows)
	}
	replacementID := rotateCommands[winner.index].ReplacementID
	replacementVerifier := rotateVerifiers[winner.index]

	var inFlight sync.WaitGroup
	authErrors := make(chan error, 2)
	for _, candidate := range []struct {
		id       identifier.ID
		verifier [32]byte
	}{{credentialIDs[0], originalVerifier}, {replacementID, replacementVerifier}} {
		candidate := candidate
		inFlight.Add(1)
		go func() {
			defer inFlight.Done()
			_, callErr := authenticate(candidate.id, candidate.verifier, now.Add(4*time.Minute), "test",
				identity.APICredentialAudience, identity.APICredentialScopeIdentityRead, networkOne)
			authErrors <- callErr
		}()
	}
	inFlight.Wait()
	close(authErrors)
	for callErr := range authErrors {
		if callErr != nil {
			t.Fatalf("old/new overlap authentication error=%v", callErr)
		}
	}

	setStepUp(identity.CredentialActionRevoke, now.Add(time.Minute))
	revokeEvent := newEvent(identity.CredentialActionRevoke, credentialIDs[0].String(), "api_credential_revoked", now.Add(5*time.Minute))
	revoked, err := store.RevokeCredential(ctx, identity.RevokeCredentialCommand{
		Actor: actor, CredentialID: credentialIDs[0], Environment: "test",
		IdempotencyDigest: sha256.Sum256([]byte("credential-revoke-db-1")),
		RequestDigest:     sha256.Sum256([]byte("credential-revoke-request-1")),
		Now:               now.Add(5 * time.Minute), AuditEvent: revokeEvent,
	})
	if err != nil || revoked.Credential.Status != identity.APICredentialRevoked {
		t.Fatalf("revoke=%+v err=%v", revoked, err)
	}
	if _, err := authenticate(credentialIDs[0], originalVerifier, now.Add(5*time.Minute), "test",
		identity.APICredentialAudience, identity.APICredentialScopeIdentityRead, networkOne); !errors.Is(err, identity.ErrAuthenticationRequired) {
		t.Fatalf("revoked old credential error=%v", err)
	}
	if _, err := authenticate(replacementID, replacementVerifier, now.Add(5*time.Minute), "test",
		identity.APICredentialAudience, identity.APICredentialScopeIdentityRead, networkOne); err != nil {
		t.Fatalf("replacement credential failed after old revocation: %v", err)
	}
	if _, err := authenticate(replacementID, replacementVerifier, winner.result.Credential.ExpiresAt, "test",
		identity.APICredentialAudience, identity.APICredentialScopeIdentityRead, networkOne); !errors.Is(err, identity.ErrAuthenticationRequired) {
		t.Fatalf("expired replacement error=%v", err)
	}

	setStepUp(identity.CredentialActionCreate, now.Add(time.Minute))
	failureEvent := newEvent(identity.CredentialActionCreate, credentialIDs[3].String(), "api_credential_created", now.Add(6*time.Minute))
	failureCommand := createCommand
	failureCommand.Credential.CredentialID = credentialIDs[3]
	failureCommand.Credential.CreatedAt = now.Add(6 * time.Minute)
	failureCommand.Credential.ExpiresAt = now.Add(6*time.Minute + identity.CredentialDefaultExpiry)
	failureCommand.IdempotencyDigest = sha256.Sum256([]byte("credential-create-audit-failure"))
	failureCommand.RequestDigest = sha256.Sum256([]byte("credential-create-audit-failure-request"))
	failureCommand.Now = now.Add(6 * time.Minute)
	failureCommand.AuditEvent = failureEvent
	if _, err := failingStore.CreateCredential(ctx, failureCommand); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("Audit-outage create error=%v", err)
	}
	var failureRows int
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.api_credentials WHERE credential_id = $1`,
		credentialIDs[3].String()).Scan(&failureRows); err != nil {
		t.Fatal(err)
	}
	if failureRows != 0 {
		t.Fatal("Audit-outage create committed credential state")
	}
}
