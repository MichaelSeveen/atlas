package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestOIDCLoginIsSingleUseNonceBoundAndCreatesANewOpaqueSession(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	provider := &fakeProvider{claims: ProviderClaims{
		Issuer:    "https://identity.test.invalid/realms/customer",
		Subject:   "00000000-0000-4000-8000-000000000101",
		Assurance: AssuranceBaseline, AuthenticatedAt: now,
	}}
	service := newTestService(t, store, provider, now)
	begin, err := service.BeginLogin(context.Background(), BeginLoginRequest{
		Population: PopulationCustomer, ReturnTo: "/customer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if begin.TransactionID.IsZero() || begin.Population != PopulationCustomer ||
		provider.state == "" || provider.nonce == "" || provider.pkce == "" {
		t.Fatal("login transaction did not contain state, nonce, PKCE, and identity")
	}
	provider.claims.Nonce = provider.nonce
	result, err := service.CompleteLogin(context.Background(), CompleteLoginRequest{
		State: provider.state, Code: "synthetic-code-0001",
		CorrelationID: mustTestID(t, "cor", 1), ClientLabel: "browser\r\ninjected",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CookieValue == provider.state || result.CookieValue == provider.nonce ||
		result.CookieValue == provider.pkce || len(result.CookieValue) != 43 {
		t.Fatal("callback reused protocol material as the application session")
	}
	if result.ReturnTo != "/customer" || result.Session.Assurance != AssuranceBaseline ||
		result.Session.ClientLabel != "browserinjected" {
		t.Fatalf("unexpected session result: %+v", result)
	}
	if store.created.VerifierDigest != sha256.Sum256([]byte(result.CookieValue)) {
		t.Fatal("store did not receive only the cookie verifier digest")
	}
	if _, err := service.CompleteLogin(context.Background(), CompleteLoginRequest{
		State: provider.state, Code: "synthetic-code-0001", CorrelationID: mustTestID(t, "cor", 2),
	}); !errors.Is(err, ErrOIDCTransactionInvalid) {
		t.Fatalf("callback replay error = %v, want OIDC transaction invalid", err)
	}
}

func TestMostAgentsSkip08SessionFixationAndStepUpRotation(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	oldCookie := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	oldDigest := sha256.Sum256([]byte(oldCookie))
	oldSession := store.session
	oldSession.RotationVersion = 7
	store.sessions[oldDigest] = oldSession
	provider := &fakeProvider{claims: ProviderClaims{
		Issuer:    "https://identity.test.invalid/realms/customer",
		Subject:   "00000000-0000-4000-8000-000000000101",
		Assurance: AssuranceBaseline, AuthenticatedAt: now,
	}}
	service := newTestService(t, store, provider, now)
	begin, err := service.BeginLogin(context.Background(), BeginLoginRequest{
		Population: PopulationCustomer, CookieValue: oldCookie,
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.transaction.ReplacedSessionID != oldSession.SessionID {
		t.Fatal("valid pre-login session was not marked for rotation")
	}
	provider.claims.Nonce = provider.nonce
	login, err := service.CompleteLogin(context.Background(), CompleteLoginRequest{
		State: provider.state, Code: "synthetic-code-0002", CorrelationID: mustTestID(t, "cor", 3),
	})
	if err != nil {
		t.Fatal(err)
	}
	if login.CookieValue == oldCookie {
		t.Fatal("fixed pre-login cookie survived the callback")
	}

	store.sessions[sha256.Sum256([]byte(login.CookieValue))] = login.Session
	csrf, err := service.csrf.Token(login.Session.SessionID, login.Session.RotationVersion)
	if err != nil {
		t.Fatal(err)
	}
	provider.claims.Assurance = AssuranceSteppedUp
	step, err := service.BeginStepUp(context.Background(), BeginStepUpRequest{
		CookieValue: login.CookieValue, CSRFToken: csrf,
		Action: "identity.approval.decide", IdempotencyKey: "step-up-rotation-0001",
		CorrelationID: mustTestID(t, "cor", 31),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.transaction.Kind != TransactionStepUp ||
		store.transaction.ReplacedSessionID != login.Session.SessionID ||
		store.transaction.PrincipalID != login.Session.PrincipalID {
		t.Fatal("step-up transaction was not bound to the current principal and session")
	}
	provider.claims.Nonce = provider.nonce
	rotated, err := service.CompleteLogin(context.Background(), CompleteLoginRequest{
		State: provider.state, Code: "synthetic-code-0003", CorrelationID: mustTestID(t, "cor", 4),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rotated.CookieValue == login.CookieValue || store.created.Kind != TransactionStepUp {
		t.Fatal("step-up did not rotate the opaque application session")
	}
	if begin.TransactionID == step.TransactionID {
		t.Fatal("login and step-up reused a transaction identity")
	}
}

func TestCSRFAndRevocationFailClosed(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	cookie := "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	digest := sha256.Sum256([]byte(cookie))
	store.sessions[digest] = store.session
	service := newTestService(t, store, &fakeProvider{}, now)
	if err := service.Logout(
		context.Background(), cookie, "wrong-token", mustTestID(t, "cor", 5),
	); !errors.Is(err, ErrCSRFValidationFailed) {
		t.Fatalf("wrong CSRF error = %v", err)
	}
	if store.revocations != 0 {
		t.Fatal("CSRF failure reached the mutation boundary")
	}
	_, csrf, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Logout(
		context.Background(), cookie, csrf, mustTestID(t, "cor", 6),
	); err != nil {
		t.Fatal(err)
	}
	if store.revocations != 1 {
		t.Fatal("valid logout did not revoke exactly once")
	}
}

func TestProviderOutagePreservesExistingLowRiskSessionAndDeniesNewAuthentication(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	cookie := "OOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO"
	store.sessions[sha256.Sum256([]byte(cookie))] = store.session
	provider := &fakeProvider{err: ErrProviderUnavailable}
	service := newTestService(t, store, provider, now)

	current, csrf, err := service.Current(context.Background(), cookie)
	if err != nil || current.SessionID != store.session.SessionID || len(csrf) != 43 {
		t.Fatalf("existing low-risk session failed during provider outage: session=%+v err=%v", current, err)
	}
	if _, err := service.BeginLogin(context.Background(), BeginLoginRequest{
		Population: PopulationCustomer,
	}); !errors.Is(err, ErrIdentityUnavailable) {
		t.Fatalf("new login did not fail closed during provider outage: %v", err)
	}
	if _, err := service.BeginStepUp(context.Background(), BeginStepUpRequest{
		CookieValue:    cookie,
		CSRFToken:      csrf,
		Action:         "identity.approval.decide",
		IdempotencyKey: "provider-outage-0001",
		CorrelationID:  mustTestID(t, "cor", 61),
	}); !errors.Is(err, ErrIdentityUnavailable) {
		t.Fatalf("new step-up did not fail closed during provider outage: %v", err)
	}
}

func TestStepUpIdempotencyReplaysExactlyAndRejectsChangedAction(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	cookie := "IIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIII"
	store.sessions[sha256.Sum256([]byte(cookie))] = store.session
	provider := &fakeProvider{}
	service := newTestService(t, store, provider, now)
	_, csrf, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	request := BeginStepUpRequest{
		CookieValue: cookie, CSRFToken: csrf, Action: "identity.approval.decide",
		IdempotencyKey: "stable-step-up-key-0001",
		CorrelationID:  mustTestID(t, "cor", 62),
	}
	first, err := service.BeginStepUp(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	provider.err = ErrProviderUnavailable
	replay, err := service.BeginStepUp(context.Background(), request)
	if err != nil {
		t.Fatalf("stored replay depended on provider availability: %v", err)
	}
	if replay != first || provider.authorizationCalls != 1 {
		t.Fatalf("replay=%+v first=%+v authorization_calls=%d", replay, first, provider.authorizationCalls)
	}
	request.Action = "identity.approval.execute"
	if _, err := service.BeginStepUp(context.Background(), request); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed request error = %v, want idempotency conflict", err)
	}
}

func TestStepUpRejectsStaleHigherAssuranceAuthentication(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	cookie := "FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
	store.sessions[sha256.Sum256([]byte(cookie))] = store.session
	provider := &fakeProvider{claims: ProviderClaims{
		Issuer:    "https://identity.test.invalid/realms/customer",
		Subject:   "00000000-0000-4000-8000-000000000101",
		Assurance: AssuranceSteppedUp, AuthenticatedAt: now.Add(-stepUpFreshness),
	}}
	service := newTestService(t, store, provider, now)
	_, csrf, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.BeginStepUp(context.Background(), BeginStepUpRequest{
		CookieValue: cookie, CSRFToken: csrf, Action: "identity.approval.decide",
		IdempotencyKey: "stale-authentication-0001", CorrelationID: mustTestID(t, "cor", 63),
	})
	if err != nil {
		t.Fatal(err)
	}
	provider.claims.Nonce = provider.nonce
	_, err = service.CompleteLogin(context.Background(), CompleteLoginRequest{
		State: provider.state, Code: "synthetic-code-0005",
		CorrelationID: mustTestID(t, "cor", 64),
	})
	if !errors.Is(err, ErrOIDCTransactionInvalid) {
		t.Fatalf("stale step-up authentication error = %v", err)
	}
}

func TestWorkforceAndStepUpAssuranceAreMandatory(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	provider := &fakeProvider{claims: ProviderClaims{
		Issuer:    "https://identity.test.invalid/realms/workforce",
		Subject:   "00000000-0000-4000-8000-000000000301",
		Assurance: AssuranceBaseline, AuthenticatedAt: now,
	}}
	service := newTestService(t, store, provider, now)
	_, err := service.BeginLogin(context.Background(), BeginLoginRequest{
		Population: PopulationWorkforce,
	})
	if err != nil {
		t.Fatal(err)
	}
	provider.claims.Nonce = provider.nonce
	_, err = service.CompleteLogin(context.Background(), CompleteLoginRequest{
		State: provider.state, Code: "synthetic-code-0004", CorrelationID: mustTestID(t, "cor", 7),
	})
	if !errors.Is(err, ErrOIDCTransactionInvalid) {
		t.Fatalf("baseline workforce login error = %v", err)
	}
}

func TestAdministratorRevocationBuildsClosedPurposeBoundCommand(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := newFakeSessionStore(t, now)
	store.session = Session{
		SessionID: mustTestID(t, "ses", 70), PrincipalID: mustTestID(t, "usr", 71),
		PrincipalType: "workforce", DisplayName: "Synthetic Platform Administrator",
		Population: PopulationWorkforce, Assurance: AssurancePhishingResistant,
		AuthorizationVersion: 1, RotationVersion: 2,
		CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(10 * time.Minute),
		AbsoluteExpiresAt: now.Add(time.Hour), StepUpAction: "identity.session.admin_revoke",
		StepUpVerifiedAt: now,
		Permissions:      []string{"identity.sessions.revoke_admin"},
	}
	cookie := strings.Repeat("S", 43)
	store.sessions[sha256.Sum256([]byte(cookie))] = store.session
	service := newTestService(t, store, &fakeProvider{}, now)
	_, csrf, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	target := mustTestID(t, "ses", 73)
	result, err := service.RevokeForSecurity(context.Background(), AdminRevokeSessionRequest{
		CookieValue: cookie, CSRFToken: csrf, TargetSessionID: target,
		Purpose: "security_review", Reason: "suspected_account_takeover",
		IdempotencyKey: "admin-revocation-key-0001",
		CorrelationID:  mustTestID(t, "cor", 74),
	})
	if err != nil || result.DecisionID != store.adminResult.DecisionID {
		t.Fatalf("administrator revocation result=%+v err=%v", result, err)
	}
	command := store.adminCommand
	if command.Actor.SessionID != store.session.SessionID ||
		command.TargetSessionID != target ||
		command.Purpose != "security_review" ||
		command.Reason != "suspected_account_takeover" ||
		command.IdempotencyDigest == ([32]byte{}) ||
		command.RequestDigest == ([32]byte{}) ||
		command.AuditEvent.Action != "identity.session.admin_revoke" ||
		command.AuditEvent.DecisionID != result.DecisionID {
		t.Fatalf("unsafe administrator revocation command: %+v", command)
	}

	before := store.revocations
	_, err = service.RevokeForSecurity(context.Background(), AdminRevokeSessionRequest{
		CookieValue: cookie, CSRFToken: csrf, TargetSessionID: target,
		Purpose: "self_service", Reason: "free form reason",
		IdempotencyKey: "admin-revocation-key-0002",
		CorrelationID:  mustTestID(t, "cor", 75),
	})
	if !errors.Is(err, ErrInputInvalid) || store.revocations != before {
		t.Fatalf("open purpose/reason reached store: err=%v revocations=%d", err, store.revocations)
	}
}

func TestVersionedTransactionEncryptionAndCSRFTamperResistance(t *testing.T) {
	keyOne := bytes.Repeat([]byte{1}, 32)
	keyTwo := bytes.Repeat([]byte{2}, 32)
	cryptor, err := NewAESGCMCryptor([]VersionedKey{
		{Version: 1, Material: keyOne}, {Version: 2, Material: keyTwo},
	}, bytes.NewReader(bytes.Repeat([]byte{3}, 64)))
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, version, err := cryptor.Encrypt([]byte("pkce-verifier"))
	if err != nil || version != 2 {
		t.Fatalf("encrypt version=%d err=%v", version, err)
	}
	plaintext, err := cryptor.Decrypt(ciphertext, version)
	if err != nil || string(plaintext) != "pkce-verifier" {
		t.Fatalf("decrypt=%q err=%v", plaintext, err)
	}
	ciphertext[len(ciphertext)-1] ^= 1
	if _, err := cryptor.Decrypt(ciphertext, version); err == nil {
		t.Fatal("tampered PKCE ciphertext was accepted")
	}
	csrf, err := NewHMACCSRFProtector(bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatal(err)
	}
	sessionID := mustTestID(t, "ses", 8)
	first, _ := csrf.Token(sessionID, 1)
	rotated, _ := csrf.Token(sessionID, 2)
	if first == rotated || len(first) != 43 || len(rotated) != 43 {
		t.Fatal("CSRF token was not bound to the session rotation version")
	}
}

type fakeProvider struct {
	state              string
	nonce              string
	pkce               string
	kind               TransactionKind
	claims             ProviderClaims
	err                error
	authorizationCalls int
}

func (provider *fakeProvider) AuthorizationURL(
	_ context.Context,
	_ Population,
	state string,
	nonce string,
	pkce string,
	kind TransactionKind,
) (string, error) {
	provider.authorizationCalls++
	if provider.err != nil {
		return "", provider.err
	}
	provider.state, provider.nonce, provider.pkce, provider.kind = state, nonce, pkce, kind
	return "https://identity.test.invalid/authorize?state=" + state, nil
}

func (provider *fakeProvider) Exchange(
	_ context.Context,
	_ Population,
	_ string,
	pkce string,
	_ string,
) (ProviderClaims, error) {
	if provider.err != nil {
		return ProviderClaims{}, provider.err
	}
	if pkce != provider.pkce {
		return ProviderClaims{}, ErrProviderInvalid
	}
	return provider.claims, nil
}

type passthroughCryptor struct{}

func (passthroughCryptor) Encrypt(value []byte) ([]byte, uint64, error) {
	return append([]byte(nil), value...), 1, nil
}

func (passthroughCryptor) Decrypt(value []byte, version uint64) ([]byte, error) {
	if version != 1 {
		return nil, errors.New("wrong key version")
	}
	return append([]byte(nil), value...), nil
}

type fakeSessionStore struct {
	t               *testing.T
	now             time.Time
	transaction     OIDCTransaction
	transactionUsed bool
	created         CreateSessionCommand
	session         Session
	sessions        map[[32]byte]Session
	revocations     int
	adminCommand    AdminRevocationCommand
	adminResult     AdminRevocationResult
	adminErr        error
	stepUps         map[[32]byte]fakeStepUpClaim
}

type fakeStepUpClaim struct {
	requestID     identifier.ID
	requestDigest [32]byte
	lifecycle     string
	transaction   OIDCTransaction
	url           string
	retainedUntil time.Time
}

func newFakeSessionStore(t *testing.T, now time.Time) *fakeSessionStore {
	return &fakeSessionStore{
		t: t, now: now, sessions: make(map[[32]byte]Session),
		stepUps: make(map[[32]byte]fakeStepUpClaim),
		session: Session{
			SessionID: mustTestID(t, "ses", 50), PrincipalID: mustTestID(t, "usr", 51),
			PrincipalType: "customer", DisplayName: "Synthetic Customer",
			Population: PopulationCustomer, TenantID: mustTestID(t, "ten", 52),
			Assurance: AssuranceBaseline, AuthorizationVersion: 1, RotationVersion: 1,
			CreatedAt: now, LastSeenAt: now, IdleExpiresAt: now.Add(30 * time.Minute),
			AbsoluteExpiresAt: now.Add(12 * time.Hour),
			Permissions:       []string{"identity.me.read", "identity.sessions.revoke_self"},
		},
	}
}

func (store *fakeSessionStore) PutOIDCTransaction(_ context.Context, transaction OIDCTransaction) error {
	store.transaction = transaction
	store.transactionUsed = false
	return nil
}

func (store *fakeSessionStore) TakeOIDCTransaction(
	_ context.Context,
	state [32]byte,
	now time.Time,
) (OIDCTransaction, error) {
	if store.transactionUsed || state != store.transaction.StateDigest || !now.Before(store.transaction.ExpiresAt) {
		return OIDCTransaction{}, ErrOIDCTransactionInvalid
	}
	store.transactionUsed = true
	return store.transaction, nil
}

func (store *fakeSessionStore) ClaimStepUp(
	_ context.Context,
	command StepUpClaimCommand,
) (StepUpClaimResult, error) {
	claim, found := store.stepUps[command.IdempotencyScope]
	if found && command.Now.Before(claim.retainedUntil) {
		if claim.requestDigest != command.RequestDigest {
			return StepUpClaimResult{}, ErrIdempotencyConflict
		}
		if claim.lifecycle == "completed" {
			return StepUpClaimResult{
				ChallengeRequestID: claim.requestID, Replay: true,
				TransactionID:    claim.transaction.TransactionID,
				AuthorizationURL: claim.url, ExpiresAt: claim.transaction.ExpiresAt,
			}, nil
		}
	}
	claim = fakeStepUpClaim{
		requestID: command.ChallengeRequestID, requestDigest: command.RequestDigest,
		lifecycle: "processing", retainedUntil: command.RetainedUntil,
	}
	store.stepUps[command.IdempotencyScope] = claim
	return StepUpClaimResult{ChallengeRequestID: claim.requestID, Owner: true}, nil
}

func (store *fakeSessionStore) CompleteStepUp(
	_ context.Context,
	command CompleteStepUpCommand,
) error {
	claim, found := store.stepUps[command.IdempotencyScope]
	if !found || claim.requestID != command.ChallengeRequestID || claim.lifecycle != "processing" {
		return ErrIdentityUnavailable
	}
	claim.lifecycle = "completed"
	claim.transaction = command.Transaction
	claim.url = command.AuthorizationURL
	store.stepUps[command.IdempotencyScope] = claim
	store.transaction = command.Transaction
	store.transactionUsed = false
	return nil
}

func (store *fakeSessionStore) FailStepUp(
	_ context.Context,
	requestID identifier.ID,
	scope [32]byte,
	_ time.Time,
) error {
	claim, found := store.stepUps[scope]
	if found && claim.requestID == requestID && claim.lifecycle == "processing" {
		claim.lifecycle = "failed-retryable"
		store.stepUps[scope] = claim
	}
	return nil
}

func (store *fakeSessionStore) CreateSession(_ context.Context, command CreateSessionCommand) (Session, error) {
	store.created = command
	session := store.session
	session.SessionID = command.SessionID
	session.Population = command.Population
	session.Assurance = command.Assurance
	session.CreatedAt = command.AuthorizationAt
	session.LastSeenAt = command.AuthorizationAt
	session.IdleExpiresAt = command.IdleExpiresAt
	session.AbsoluteExpiresAt = command.AbsoluteExpiresAt
	session.StepUpAction = command.StepUpAction
	session.StepUpVerifiedAt = command.StepUpVerifiedAt
	session.ClientLabel = command.ClientLabel
	if command.Kind == TransactionStepUp {
		session.RotationVersion++
	}
	store.sessions[command.VerifierDigest] = session
	return session, nil
}

func (store *fakeSessionStore) Authenticate(
	_ context.Context,
	digest [32]byte,
	_ time.Time,
	_ time.Duration,
) (Session, error) {
	session, found := store.sessions[digest]
	if !found {
		return Session{}, ErrAuthenticationRequired
	}
	return session, nil
}

func (store *fakeSessionStore) ListSessions(context.Context, Session, time.Time) ([]SessionSummary, error) {
	return nil, nil
}

func (store *fakeSessionStore) RevokeCurrent(context.Context, Session, time.Time, audit.Event) error {
	store.revocations++
	return nil
}

func (store *fakeSessionStore) RevokeOne(context.Context, RevocationCommand) (RevocationResult, error) {
	store.revocations++
	return RevocationResult{}, nil
}

func (store *fakeSessionStore) RevokeAll(context.Context, RevocationCommand) (RevocationResult, error) {
	store.revocations++
	return RevocationResult{}, nil
}

func (store *fakeSessionStore) RevokeForSecurity(
	_ context.Context,
	command AdminRevocationCommand,
) (AdminRevocationResult, error) {
	store.revocations++
	store.adminCommand = command
	if store.adminResult.DecisionID.IsZero() {
		store.adminResult.DecisionID = command.AuditEvent.DecisionID
	}
	return store.adminResult, store.adminErr
}

func newTestService(
	t *testing.T,
	store Store,
	provider Provider,
	now time.Time,
) *Service {
	t.Helper()
	csrf, err := NewHMACCSRFProtector(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	counter := 0
	entropy := make([]byte, 4096)
	for index := range entropy {
		entropy[index] = byte(index % 251)
	}
	service, err := NewService(ServiceOptions{
		Store: store, Provider: provider, Cryptor: passthroughCryptor{}, CSRF: csrf,
		Clock: clock.NewFixed(now),
		NewID: func(prefix string) (identifier.ID, error) {
			counter++
			return identifier.Parse(fmt.Sprintf("%s_%020d", prefix, counter))
		},
		Entropy: bytes.NewReader(entropy),
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func mustTestID(t *testing.T, prefix string, value int) identifier.ID {
	t.Helper()
	id, err := identifier.Parse(fmt.Sprintf("%s_%020d", prefix, value))
	if err != nil {
		t.Fatal(err)
	}
	return id
}
