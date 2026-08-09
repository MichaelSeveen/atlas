package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type httpCredentialMutation struct {
	requestDigest [32]byte
	result        identity.CredentialMutationResult
}

type httpStoredCredential struct {
	credential identity.APICredential
	verifier   [32]byte
}

type httpCredentialStore struct {
	credentials map[identifier.ID]httpStoredCredential
	mutations   map[[32]byte]httpCredentialMutation
	createCalls int
}

func newHTTPCredentialStore() *httpCredentialStore {
	return &httpCredentialStore{
		credentials: make(map[identifier.ID]httpStoredCredential),
		mutations:   make(map[[32]byte]httpCredentialMutation),
	}
}

func (store *httpCredentialStore) ListCredentials(
	_ context.Context,
	command identity.ListCredentialsCommand,
) ([]identity.APICredential, error) {
	result := make([]identity.APICredential, 0)
	for _, stored := range store.credentials {
		if stored.credential.OrganizationID == command.Actor.TenantID &&
			stored.credential.Environment == command.Environment {
			result = append(result, stored.credential)
		}
	}
	return result, nil
}

func (store *httpCredentialStore) CreateCredential(
	_ context.Context,
	command identity.CreateCredentialCommand,
) (identity.CredentialMutationResult, error) {
	store.createCalls++
	if existing, found := store.mutations[command.IdempotencyDigest]; found {
		if existing.requestDigest != command.RequestDigest {
			return identity.CredentialMutationResult{DecisionID: existing.result.DecisionID, Replay: true},
				identity.ErrIdempotencyConflict
		}
		replay := existing.result
		replay.Replay = true
		return replay, nil
	}
	result := identity.CredentialMutationResult{
		Credential: command.Credential, DecisionID: command.AuditEvent.DecisionID,
	}
	store.credentials[command.Credential.CredentialID] = httpStoredCredential{
		credential: command.Credential, verifier: command.Verifier,
	}
	store.mutations[command.IdempotencyDigest] = httpCredentialMutation{
		requestDigest: command.RequestDigest, result: result,
	}
	return result, nil
}

func (store *httpCredentialStore) RotateCredential(
	_ context.Context,
	command identity.RotateCredentialCommand,
) (identity.CredentialMutationResult, error) {
	if existing, found := store.mutations[command.IdempotencyDigest]; found {
		if existing.requestDigest != command.RequestDigest {
			return identity.CredentialMutationResult{DecisionID: existing.result.DecisionID, Replay: true},
				identity.ErrIdempotencyConflict
		}
		replay := existing.result
		replay.Replay = true
		return replay, nil
	}
	source, found := store.credentials[command.SourceCredentialID]
	if !found || source.credential.Status != identity.APICredentialActive ||
		source.credential.Environment != command.Environment {
		return identity.CredentialMutationResult{DecisionID: command.AuditEvent.DecisionID}, identity.ErrCredentialConflict
	}
	replacement := source.credential
	replacement.CredentialID = command.ReplacementID
	replacement.SecretHint = command.ReplacementSecretHint
	replacement.Status = identity.APICredentialActive
	replacement.Version = 1
	replacement.ExpiresAt = command.Now.Add(identity.CredentialDefaultExpiry)
	replacement.CreatedAt = command.Now
	replacement.OverlapEndsAt = time.Time{}
	replacement.PreviousCredentialID = source.credential.CredentialID
	replacement.ReplacementCredentialID = identifier.ID{}
	source.credential.Status = identity.APICredentialRotating
	source.credential.OverlapEndsAt = command.Now.Add(identity.CredentialRotationOverlap)
	source.credential.ReplacementCredentialID = replacement.CredentialID
	store.credentials[source.credential.CredentialID] = source
	store.credentials[replacement.CredentialID] = httpStoredCredential{
		credential: replacement, verifier: command.ReplacementVerifier,
	}
	result := identity.CredentialMutationResult{
		Credential: replacement, DecisionID: command.AuditEvent.DecisionID,
	}
	store.mutations[command.IdempotencyDigest] = httpCredentialMutation{
		requestDigest: command.RequestDigest, result: result,
	}
	return result, nil
}

func (store *httpCredentialStore) RevokeCredential(
	_ context.Context,
	command identity.RevokeCredentialCommand,
) (identity.CredentialMutationResult, error) {
	if existing, found := store.mutations[command.IdempotencyDigest]; found {
		if existing.requestDigest != command.RequestDigest {
			return identity.CredentialMutationResult{DecisionID: existing.result.DecisionID, Replay: true},
				identity.ErrIdempotencyConflict
		}
		replay := existing.result
		replay.Replay = true
		return replay, nil
	}
	stored, found := store.credentials[command.CredentialID]
	if !found || stored.credential.Environment != command.Environment {
		return identity.CredentialMutationResult{DecisionID: command.AuditEvent.DecisionID}, identity.ErrCredentialNotFound
	}
	stored.credential.Status = identity.APICredentialRevoked
	stored.credential.RevokedAt = command.Now
	stored.credential.Version++
	store.credentials[command.CredentialID] = stored
	result := identity.CredentialMutationResult{
		Credential: stored.credential, DecisionID: command.AuditEvent.DecisionID,
	}
	store.mutations[command.IdempotencyDigest] = httpCredentialMutation{
		requestDigest: command.RequestDigest, result: result,
	}
	return result, nil
}

func (store *httpCredentialStore) AuthenticateCredential(
	_ context.Context,
	command identity.AuthenticateCredentialCommand,
) (identity.CredentialAuthentication, error) {
	stored, found := store.credentials[command.CredentialID]
	validStatus := found && (stored.credential.Status == identity.APICredentialActive ||
		(stored.credential.Status == identity.APICredentialRotating &&
			command.Now.Before(stored.credential.OverlapEndsAt)))
	if !found || subtle.ConstantTimeCompare(stored.verifier[:], command.Verifier[:]) != 1 ||
		stored.credential.Environment != command.Environment ||
		stored.credential.Audience != command.Audience ||
		len(stored.credential.Scopes) != 1 || stored.credential.Scopes[0] != command.Scope ||
		!validStatus || !command.Now.Before(stored.credential.ExpiresAt) {
		return identity.CredentialAuthentication{}, identity.ErrAuthenticationRequired
	}
	return identity.CredentialAuthentication{
		Credential: stored.credential, Permission: "identity.me.read", AnomalousNetwork: true,
	}, nil
}

type httpCredentialLimiter struct {
	decision identity.CredentialRateDecision
	err      error
}

func (limiter *httpCredentialLimiter) Allow(
	context.Context,
	identity.CredentialRateSubject,
	time.Time,
) (identity.CredentialRateDecision, error) {
	return limiter.decision, limiter.err
}

func TestAPICredentialHTTPSecretReplayMachineAuthAndRevocation(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("C", 43)
	tenantID, _ := identifier.Parse("ten_00000000000000000801")
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: mustHTTPID(t, "ses", 801), PrincipalID: mustHTTPID(t, "usr", 801),
		PrincipalType: "merchant", Population: identity.PopulationMerchant, TenantID: tenantID,
		Assurance: identity.AssurancePhishingResistant, AuthorizationVersion: 1, RotationVersion: 1,
		CreatedAt: testBuildTime, LastSeenAt: testBuildTime,
		IdleExpiresAt: testBuildTime.Add(time.Hour), AbsoluteExpiresAt: testBuildTime.Add(time.Hour),
	}
	_, csrfToken, err := service.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
		options.Meter = provider.Meter("atlas-credential-http-test")
	})
	body := `{"name":"Reconciliation reader","scopes":["identity:read"],"expires_in_days":90,"purpose":"credential_management"}`
	created := serveCredentialMutation(app, http.MethodPost, "/v1/api-credentials", body,
		cookieValue, csrfToken, "credential-http-create-0001")
	if created.Code != http.StatusCreated || created.Header().Get("Idempotency-Replayed") != "false" ||
		!strings.HasPrefix(created.Header().Get("Location"), "/v1/api-credentials/key_") ||
		!strings.HasPrefix(created.Header().Get("X-Authorization-Decision-Id"), "dec_") {
		t.Fatalf("create status=%d headers=%v body=%s", created.Code, created.Header(), created.Body)
	}
	var original apiCredentialCreatedResponse
	if err := json.Unmarshal(created.Body.Bytes(), &original); err != nil {
		t.Fatal(err)
	}
	secret, ok := original.Secret.(string)
	if !ok || !original.SecretDisclosed || !strings.HasPrefix(secret, identity.APICredentialScheme+" key_") {
		t.Fatalf("original secret response=%+v", original)
	}

	replayed := serveCredentialMutation(app, http.MethodPost, "/v1/api-credentials", body,
		cookieValue, csrfToken, "credential-http-create-0001")
	if replayed.Code != http.StatusCreated || replayed.Header().Get("Idempotency-Replayed") != "true" ||
		!strings.Contains(replayed.Body.String(), `"secret":null`) ||
		!strings.Contains(replayed.Body.String(), `"secret_disclosed":false`) {
		t.Fatalf("replay status=%d headers=%v body=%s", replayed.Code, replayed.Header(), replayed.Body)
	}

	machine := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	machine.RemoteAddr = "192.0.2.80:443"
	machine.Header.Set("Authorization", secret)
	machineResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(machineResponse, machine)
	if machineResponse.Code != http.StatusOK ||
		!strings.Contains(machineResponse.Body.String(), `"type":"machine"`) ||
		machineResponse.Header().Get(identity.CSRFHeaderName) != "" {
		t.Fatalf("machine response status=%d headers=%v body=%s", machineResponse.Code, machineResponse.Header(), machineResponse.Body)
	}

	ambiguous := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	ambiguous.RemoteAddr = "192.0.2.80:443"
	ambiguous.Header.Set("Authorization", secret)
	ambiguous.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	ambiguousResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(ambiguousResponse, ambiguous)
	if ambiguousResponse.Code != http.StatusUnauthorized {
		t.Fatalf("ambiguous authentication status=%d body=%s", ambiguousResponse.Code, ambiguousResponse.Body)
	}

	store.credentialLimiter.decision = identity.CredentialRateDecision{Allowed: false, Fallback: true}
	limited := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	limited.RemoteAddr = "192.0.2.80:443"
	limited.Header.Set("Authorization", secret)
	limitedResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(limitedResponse, limited)
	if limitedResponse.Code != http.StatusTooManyRequests || limitedResponse.Header().Get("Retry-After") != "60" {
		t.Fatalf("rate response status=%d headers=%v body=%s", limitedResponse.Code, limitedResponse.Header(), limitedResponse.Body)
	}
	store.credentialLimiter.decision = identity.CredentialRateDecision{Allowed: true}

	credentialID := original.Credential.ID
	revoked := serveCredentialMutation(app, http.MethodDelete, "/v1/api-credentials/"+credentialID, "",
		cookieValue, csrfToken, "credential-http-revoke-0001")
	if revoked.Code != http.StatusNoContent || revoked.Header().Get("Idempotency-Replayed") != "false" ||
		!strings.HasPrefix(revoked.Header().Get("X-Authorization-Decision-Id"), "dec_") {
		t.Fatalf("revoke status=%d headers=%v body=%s", revoked.Code, revoked.Header(), revoked.Body)
	}
	revokedMachine := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	revokedMachine.RemoteAddr = "192.0.2.80:443"
	revokedMachine.Header.Set("Authorization", secret)
	revokedResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(revokedResponse, revokedMachine)
	if revokedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked credential status=%d body=%s", revokedResponse.Code, revokedResponse.Body)
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, scope := range metrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			seen[metric.Name] = true
		}
	}
	for _, name := range []string{
		"atlas.identity.api_credential.authentication.count",
		"atlas.identity.api_credential.anomaly.count",
		"atlas.identity.api_credential.rate_rejection.count",
		"atlas.identity.api_credential.rate_fallback.count",
	} {
		if !seen[name] {
			t.Errorf("credential metric %s was not emitted", name)
		}
	}
}

func TestAPICredentialHTTPRejectsMassAssignmentAndDuplicateHeaders(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("D", 43)
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: mustHTTPID(t, "ses", 811), PrincipalID: mustHTTPID(t, "usr", 811),
		PrincipalType: "merchant", Population: identity.PopulationMerchant,
		TenantID: mustHTTPID(t, "ten", 811), Assurance: identity.AssurancePhishingResistant,
		AuthorizationVersion: 1, RotationVersion: 1,
		IdleExpiresAt: testBuildTime.Add(time.Hour), AbsoluteExpiresAt: testBuildTime.Add(time.Hour),
	}
	_, csrfToken, err := service.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
	})
	body := `{"name":"Reader","scopes":["identity:read"],"expires_in_days":90,"purpose":"credential_management","status":"active"}`
	response := serveCredentialMutation(app, http.MethodPost, "/v1/api-credentials", body,
		cookieValue, csrfToken, "credential-http-mass-assignment")
	if response.Code != http.StatusBadRequest || store.credentialStore.createCalls != 0 {
		t.Fatalf("mass assignment reached store: status=%d calls=%d body=%s", response.Code, store.credentialStore.createCalls, response.Body)
	}

	request := credentialMutationRequest(http.MethodPost, "/v1/api-credentials",
		`{"name":"Reader","scopes":["identity:read"],"expires_in_days":90,"purpose":"credential_management"}`,
		cookieValue, csrfToken, "credential-http-duplicate")
	request.Header.Add("Idempotency-Key", "credential-http-duplicate-2")
	duplicate := httptest.NewRecorder()
	app.Handler().ServeHTTP(duplicate, request)
	if duplicate.Code != http.StatusBadRequest || store.credentialStore.createCalls != 0 {
		t.Fatalf("duplicate header reached store: status=%d calls=%d", duplicate.Code, store.credentialStore.createCalls)
	}
}

func serveCredentialMutation(
	app *App,
	method string,
	path string,
	body string,
	cookie string,
	csrf string,
	idempotency string,
) *httptest.ResponseRecorder {
	request := credentialMutationRequest(method, path, body, cookie, csrf, idempotency)
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	return response
}

func credentialMutationRequest(
	method string,
	path string,
	body string,
	cookie string,
	csrf string,
	idempotency string,
) *http.Request {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	if body == "" {
		request.Body = http.NoBody
		request.ContentLength = 0
	} else {
		request.Header.Set("Content-Type", "application/json")
	}
	request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookie})
	request.Header.Set(identity.CSRFHeaderName, csrf)
	request.Header.Set("Idempotency-Key", idempotency)
	return request
}

func mustHTTPID(t *testing.T, prefix string, value int) identifier.ID {
	t.Helper()
	id, err := identifier.Parse(fmt.Sprintf("%s_%020d", prefix, value))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

var _ identity.CredentialStore = (*httpCredentialStore)(nil)
var _ identity.CredentialRateLimiter = (*httpCredentialLimiter)(nil)
