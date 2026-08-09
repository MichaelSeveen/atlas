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
)

type fakeCredentialStore struct {
	createCalls int
	create      CreateCredentialCommand
	createReply CredentialMutationResult
	createErr   error
	authCalls   int
	auth        AuthenticateCredentialCommand
	authReply   CredentialAuthentication
	authErr     error
}

func (*fakeCredentialStore) ListCredentials(context.Context, ListCredentialsCommand) ([]APICredential, error) {
	return nil, nil
}

func (store *fakeCredentialStore) CreateCredential(
	_ context.Context,
	command CreateCredentialCommand,
) (CredentialMutationResult, error) {
	store.createCalls++
	store.create = command
	if store.createReply.Credential.CredentialID.IsZero() {
		store.createReply.Credential = command.Credential
		store.createReply.DecisionID = command.AuditEvent.DecisionID
	}
	return store.createReply, store.createErr
}

func (*fakeCredentialStore) RotateCredential(context.Context, RotateCredentialCommand) (CredentialMutationResult, error) {
	return CredentialMutationResult{}, nil
}

func (*fakeCredentialStore) RevokeCredential(context.Context, RevokeCredentialCommand) (CredentialMutationResult, error) {
	return CredentialMutationResult{}, nil
}

func (store *fakeCredentialStore) AuthenticateCredential(
	_ context.Context,
	command AuthenticateCredentialCommand,
) (CredentialAuthentication, error) {
	store.authCalls++
	store.auth = command
	return store.authReply, store.authErr
}

type fakeCredentialLimiter struct {
	decision CredentialRateDecision
	err      error
	calls    int
	subject  CredentialRateSubject
}

func (limiter *fakeCredentialLimiter) Allow(
	_ context.Context,
	subject CredentialRateSubject,
	_ time.Time,
) (CredentialRateDecision, error) {
	limiter.calls++
	limiter.subject = subject
	return limiter.decision, limiter.err
}

type failingEntropy struct{}

func (failingEntropy) Read([]byte) (int, error) { return 0, errors.New("entropy unavailable") }

func TestAPICredentialSecretIs256BitShownOnceAndReplayIsRedacted(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	service, credentials, cookie, csrf := newCredentialTestService(t, now)
	request := CreateAPICredentialRequest{
		CookieValue: cookie, CSRFToken: csrf, IdempotencyKey: "credential-create-0001",
		Name: "Reconciliation reader", Scopes: []string{APICredentialScopeIdentityRead},
		ExpiresInDays: 90, Purpose: CredentialPurposeManagement,
		CorrelationID: mustTestID(t, "cor", 8801),
	}
	created, err := service.CreateAPICredential(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	credentialID, verifier, err := parseAPICredential(created.Secret)
	if err != nil || credentialID != created.Credential.CredentialID ||
		verifier != credentials.create.Verifier || !created.SecretDisclosed ||
		len(strings.TrimPrefix(strings.SplitN(created.Secret, ".", 2)[0], APICredentialScheme+" ")) == 0 {
		t.Fatalf("unsafe original secret response: result=%+v err=%v", created, err)
	}
	secretText := strings.SplitN(created.Secret, ".", 2)[1]
	if len(secretText) != 43 || credentials.create.Credential.SecretHint != secretText[len(secretText)-8:] {
		t.Fatalf("secret is not canonical 256-bit base64url: %q", secretText)
	}
	if credentials.create.Credential.Environment != "local" ||
		credentials.create.Credential.Audience != APICredentialAudience ||
		credentials.create.Credential.ExpiresAt.Sub(now) != CredentialDefaultExpiry {
		t.Fatalf("credential binding is incomplete: %+v", credentials.create.Credential)
	}
	if strings.Contains(fmt.Sprintf("%+v", credentials.create), created.Secret) {
		t.Fatal("plaintext authorization value crossed the persistence boundary")
	}

	credentials.createReply = CredentialMutationResult{
		Credential: credentials.create.Credential,
		DecisionID: credentials.create.AuditEvent.DecisionID,
		Replay:     true,
	}
	replayed, err := service.CreateAPICredential(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replay || replayed.SecretDisclosed || replayed.Secret != "" {
		t.Fatalf("idempotent replay redisclosed secret: %+v", replayed)
	}
}

func TestAPICredentialEntropyFailureLeavesNoDurableMutation(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	service, credentials, cookie, csrf := newCredentialTestService(t, now)
	service.entropy = failingEntropy{}
	_, err := service.CreateAPICredential(context.Background(), CreateAPICredentialRequest{
		CookieValue: cookie, CSRFToken: csrf, IdempotencyKey: "credential-create-entropy",
		Name: "Reader", Scopes: []string{APICredentialScopeIdentityRead}, ExpiresInDays: 1,
		Purpose: CredentialPurposeManagement, CorrelationID: mustTestID(t, "cor", 8802),
	})
	if !errors.Is(err, ErrIdentityUnavailable) || credentials.createCalls != 0 {
		t.Fatalf("entropy failure reached store: calls=%d err=%v", credentials.createCalls, err)
	}
}

func TestAPICredentialAuthenticationUsesPostgresFactsAndConservativeLimiter(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	service, credentials, _, _ := newCredentialTestService(t, now)
	credentialID := mustTestID(t, "key", 8803)
	secret := bytes.Repeat([]byte{42}, 32)
	secretText := encodeRawURL(secret)
	authorization := APICredentialScheme + " " + credentialID.String() + "." + secretText
	credentials.authReply = CredentialAuthentication{
		Credential: APICredential{
			CredentialID: credentialID, OrganizationID: mustTestID(t, "ten", 8804),
			Name: "Machine reader", Version: 7, ExpiresAt: now.Add(time.Hour),
		},
		Permission: "identity.me.read", AnomalousNetwork: true,
	}
	limiter := service.credentialLimiter.(*fakeCredentialLimiter)
	limiter.decision = CredentialRateDecision{Allowed: true, Fallback: true}
	principal, err := service.AuthenticateAPICredential(
		context.Background(), authorization, "[2001:db8::1]:443",
	)
	if err != nil {
		t.Fatal(err)
	}
	if credentials.auth.Verifier != sha256.Sum256([]byte(secretText)) ||
		credentials.auth.Environment != "local" || credentials.auth.Audience != APICredentialAudience ||
		credentials.auth.Scope != APICredentialScopeIdentityRead ||
		principal.TenantID != credentials.authReply.Credential.OrganizationID ||
		!principal.AnomalousNetwork || !principal.RateFallback || limiter.calls != 1 {
		t.Fatalf("machine authorization lost a required binding: principal=%+v command=%+v", principal, credentials.auth)
	}
	if limiter.subject.NetworkSignal == ([32]byte{}) ||
		bytes.Contains(limiter.subject.NetworkSignal[:], []byte("2001:db8::1")) {
		t.Fatal("rate limiter received a missing or raw network address")
	}
	limiter.decision = CredentialRateDecision{Allowed: false, Fallback: true}
	if _, err := service.AuthenticateAPICredential(
		context.Background(), authorization, "192.0.2.10:443",
	); !errors.Is(err, ErrCredentialRateLimited) {
		t.Fatalf("conservative limiter denial error=%v", err)
	}
}

func TestParseAPICredentialRejectsMalformedAndDowngradeForms(t *testing.T) {
	credentialID := mustTestID(t, "key", 8805).String()
	secret := encodeRawURL(bytes.Repeat([]byte{7}, 32))
	valid := APICredentialScheme + " " + credentialID + "." + secret
	if _, _, err := parseAPICredential(valid); err != nil {
		t.Fatalf("valid credential rejected: %v", err)
	}
	for _, value := range []string{
		"Bearer " + credentialID + "." + secret,
		"atlaskey " + credentialID + "." + secret,
		APICredentialScheme + "  " + credentialID + "." + secret,
		valid + " ",
		APICredentialScheme + " " + strings.Replace(credentialID, "key_", "usr_", 1) + "." + secret,
		APICredentialScheme + " " + credentialID + "." + secret + "=",
		APICredentialScheme + " " + credentialID + "." + secret[:42],
		APICredentialScheme + " " + credentialID + "." + secret + ".extra",
	} {
		if _, _, err := parseAPICredential(value); err == nil {
			t.Fatalf("malformed or downgrade form accepted: %q", value)
		}
	}
}

func FuzzParseAPICredentialFailsClosed(f *testing.F) {
	f.Add("Bearer key_00000000000000008805.secret")
	f.Add("AtlasKey key_00000000000000008805.not-base64!")
	f.Fuzz(func(t *testing.T, value string) {
		credentialID, _, err := parseAPICredential(value)
		if err == nil && credentialID.Prefix() != "key" {
			t.Fatalf("parser accepted non-key identifier from %q", value)
		}
	})
}

func newCredentialTestService(
	t *testing.T,
	now time.Time,
) (*Service, *fakeCredentialStore, string, string) {
	t.Helper()
	store := newFakeSessionStore(t, now)
	store.session.Population = PopulationMerchant
	store.session.PrincipalType = "merchant"
	store.session.Assurance = AssurancePhishingResistant
	store.session.Permissions = []string{
		CredentialActionRead, CredentialActionCreate, CredentialActionRotate, CredentialActionRevoke,
	}
	cookie := strings.Repeat("K", 43)
	store.sessions[sha256.Sum256([]byte(cookie))] = store.session
	service := newTestService(t, store, &fakeProvider{}, now)
	credentials := &fakeCredentialStore{}
	service.credentials = credentials
	service.credentialLimiter = &fakeCredentialLimiter{}
	service.credentialEnvironment = "local"
	service.credentialNetworkSignalKey = bytes.Repeat([]byte{71}, 32)
	_, csrf, err := service.Current(context.Background(), cookie)
	if err != nil {
		t.Fatal(err)
	}
	return service, credentials, cookie, csrf
}

func encodeRawURL(value []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var result strings.Builder
	for index := 0; index < len(value); index += 3 {
		remaining := len(value) - index
		chunk := uint32(value[index]) << 16
		if remaining > 1 {
			chunk |= uint32(value[index+1]) << 8
		}
		if remaining > 2 {
			chunk |= uint32(value[index+2])
		}
		result.WriteByte(alphabet[(chunk>>18)&63])
		result.WriteByte(alphabet[(chunk>>12)&63])
		if remaining > 1 {
			result.WriteByte(alphabet[(chunk>>6)&63])
		}
		if remaining > 2 {
			result.WriteByte(alphabet[chunk&63])
		}
	}
	return result.String()
}

var _ CredentialStore = (*fakeCredentialStore)(nil)
var _ CredentialRateLimiter = (*fakeCredentialLimiter)(nil)
