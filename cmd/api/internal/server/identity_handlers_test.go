package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	metricdata "go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestIdentityBFFLoginCookieCurrentPrincipalAndLogoutContract(t *testing.T) {
	service, store, provider := newHTTPIdentityService(t)
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
		options.CORS = CORSConfig{AllowedOrigins: []string{"https://web.test.invalid"}, AllowCredentials: true}
	})

	login := httptest.NewRequest(http.MethodGet, "/v1/auth/login?population=customer&return_to=%2Fcustomer", nil)
	login.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: strings.Repeat("A", 43)})
	loginResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusSeeOther {
		t.Fatalf("login status=%d body=%s", loginResponse.Code, loginResponse.Body)
	}
	authorizationURL, err := url.Parse(loginResponse.Header().Get("Location"))
	if err != nil || authorizationURL.Host != "identity.test.invalid" {
		t.Fatalf("unsafe authorization redirect: %s", loginResponse.Header().Get("Location"))
	}
	state := authorizationURL.Query().Get("state")
	if state == "" || provider.nonce == "" || provider.pkce == "" || store.transaction.TransactionID.IsZero() {
		t.Fatal("login handler did not create a complete one-time transaction")
	}
	provider.claims.Nonce = provider.nonce

	callback := httptest.NewRequest(
		http.MethodGet,
		"/v1/auth/callback?code=synthetic-code-0001&state="+url.QueryEscape(state)+
			"&iss="+url.QueryEscape("https://identity.test.invalid/realms/customer")+
			"&session_state=synthetic.session-state_01",
		nil,
	)
	callbackResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(callbackResponse, callback)
	if callbackResponse.Code != http.StatusSeeOther ||
		callbackResponse.Header().Get("Location") != "https://web.test.invalid/customer" {
		t.Fatalf("callback status=%d location=%s body=%s",
			callbackResponse.Code, callbackResponse.Header().Get("Location"), callbackResponse.Body)
	}
	if provider.authorizationIssuer != "https://identity.test.invalid/realms/customer" {
		t.Fatalf("authorization-response issuer was not bound to the provider: %q", provider.authorizationIssuer)
	}
	cookies := callbackResponse.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("callback cookies=%d", len(cookies))
	}
	sessionCookie := cookies[0]
	if sessionCookie.Name != identity.SessionCookieName || !sessionCookie.Secure ||
		!sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteLaxMode ||
		sessionCookie.Path != "/" || sessionCookie.Domain != "" ||
		sessionCookie.Value == state || sessionCookie.Value == provider.nonce ||
		sessionCookie.Value == provider.pkce {
		t.Fatalf("unsafe session cookie: %+v", sessionCookie)
	}

	current := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	current.AddCookie(sessionCookie)
	currentResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(currentResponse, current)
	if currentResponse.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", currentResponse.Code, currentResponse.Body)
	}
	csrfToken := currentResponse.Header().Get(identity.CSRFHeaderName)
	if len(csrfToken) != 43 ||
		strings.Contains(currentResponse.Body.String(), sessionCookie.Value) ||
		strings.Contains(currentResponse.Body.String(), "token") {
		t.Fatalf("unsafe current-principal response: headers=%v body=%s", currentResponse.Header(), currentResponse.Body)
	}

	stepUp := func(action string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodPost, "/v1/step-up/challenges",
			strings.NewReader(`{"action":"`+action+`"}`),
		)
		request.AddCookie(sessionCookie)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set(identity.CSRFHeaderName, csrfToken)
		request.Header.Set("Idempotency-Key", "http-step-up-replay-0001")
		result := httptest.NewRecorder()
		app.Handler().ServeHTTP(result, request)
		return result
	}
	firstStepUp := stepUp("identity.approval.decide")
	replayedStepUp := stepUp("identity.approval.decide")
	if firstStepUp.Code != http.StatusCreated ||
		replayedStepUp.Code != http.StatusCreated ||
		replayedStepUp.Body.String() != firstStepUp.Body.String() ||
		replayedStepUp.Header().Get("Location") != firstStepUp.Header().Get("Location") ||
		provider.authorizationCalls != 2 {
		t.Fatalf("step-up replay was not exact: first=%d/%s replay=%d/%s calls=%d",
			firstStepUp.Code, firstStepUp.Body, replayedStepUp.Code, replayedStepUp.Body,
			provider.authorizationCalls)
	}
	conflictStepUp := stepUp("identity.approval.execute")
	if conflictStepUp.Code != http.StatusConflict ||
		!strings.Contains(conflictStepUp.Body.String(), "IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_REQUEST") {
		t.Fatalf("step-up key mismatch status=%d body=%s", conflictStepUp.Code, conflictStepUp.Body)
	}

	missingCSRF := httptest.NewRequest(http.MethodPost, "/v1/logout", nil)
	missingCSRF.AddCookie(sessionCookie)
	missingCSRF.Header.Set("Origin", "https://web.test.invalid")
	missingCSRFResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(missingCSRFResponse, missingCSRF)
	if missingCSRFResponse.Code != http.StatusForbidden ||
		missingCSRFResponse.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		missingCSRFResponse.Header().Get("Access-Control-Allow-Origin") != "https://web.test.invalid" {
		t.Fatalf("CSRF/CORS error path is unsafe: status=%d headers=%v", missingCSRFResponse.Code, missingCSRFResponse.Header())
	}

	logout := httptest.NewRequest(http.MethodPost, "/v1/logout", nil)
	logout.AddCookie(sessionCookie)
	logout.Header.Set(identity.CSRFHeaderName, csrfToken)
	logoutResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent || store.revocations != 1 {
		t.Fatalf("logout status=%d revocations=%d body=%s", logoutResponse.Code, store.revocations, logoutResponse.Body)
	}
	cleared := logoutResponse.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != identity.SessionCookieName || cleared[0].MaxAge >= 0 {
		t.Fatalf("logout did not expire the host-bound cookie: %+v", cleared)
	}
}

func TestIdentityMutationCORSPreflightAndRouteInventory(t *testing.T) {
	service, _, _ := newHTTPIdentityService(t)
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
		options.CORS = CORSConfig{AllowedOrigins: []string{"https://web.test.invalid"}, AllowCredentials: true}
	})
	preflight := httptest.NewRequest(http.MethodOptions, "/v1/sessions/revoke-all", nil)
	preflight.Header.Set("Origin", "https://web.test.invalid")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflight.Header.Set(
		"Access-Control-Request-Headers",
		"Content-Type, Idempotency-Key, X-Atlas-CSRF-Token",
	)
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, preflight)
	if response.Code != http.StatusNoContent ||
		response.Header().Get("Access-Control-Allow-Methods") != http.MethodPost ||
		response.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		response.Header().Get("Access-Control-Allow-Headers") !=
			"Content-Type, Idempotency-Key, X-Atlas-CSRF-Token" {
		t.Fatalf("credentialed mutation preflight failed: status=%d headers=%v body=%s",
			response.Code, response.Header(), response.Body)
	}
	want := []string{
		"/v1/me", "/v1/auth/login", "/v1/auth/callback", "/v1/logout",
		"/v1/sessions", "/v1/sessions/{session_id}", "/v1/sessions/revoke-all",
		"/v1/security/sessions/{session_id}/revocations",
		"/v1/step-up/challenges",
	}
	if strings.Join(identityRoutes, ",") != strings.Join(want, ",") {
		t.Fatalf("identity route inventory=%v", identityRoutes)
	}
}

func TestAdministratorSessionRevocationHTTPContract(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("T", 43)
	principalID, _ := identifier.Parse("usr_00000000000000000071")
	sessionID, _ := identifier.Parse("ses_00000000000000000070")
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: sessionID, PrincipalID: principalID,
		PrincipalType: "workforce", DisplayName: "Synthetic Platform Administrator",
		Population: identity.PopulationWorkforce, Assurance: identity.AssurancePhishingResistant,
		AuthorizationVersion: 1, RotationVersion: 2,
		CreatedAt: testBuildTime, LastSeenAt: testBuildTime,
		IdleExpiresAt:     testBuildTime.Add(10 * time.Minute),
		AbsoluteExpiresAt: testBuildTime.Add(time.Hour),
		StepUpAction:      "identity.session.admin_revoke", StepUpVerifiedAt: testBuildTime,
		Permissions: []string{"identity.sessions.revoke_admin"},
	}
	_, csrf, err := service.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
	})
	target := "ses_00000000000000000073"
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/security/sessions/"+target+"/revocations",
		strings.NewReader(`{"purpose":"security_review","reason":"compromised_session"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(identity.CSRFHeaderName, csrf)
	request.Header.Set("Idempotency-Key", "http-admin-revocation-0001")
	request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNoContent ||
		response.Header().Get("Idempotency-Replayed") != "false" ||
		!strings.HasPrefix(response.Header().Get("X-Authorization-Decision-Id"), "dec_") ||
		store.revocations != 1 {
		t.Fatalf("administrator revocation status=%d headers=%v revocations=%d body=%s",
			response.Code, response.Header(), store.revocations, response.Body)
	}
}

func TestIdentityOperationMetricsUseOnlyBoundedOutcomeAttributes(t *testing.T) {
	service, _, _ := newHTTPIdentityService(t)
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
		options.Meter = meterProvider.Meter("atlas-identity-test")
	})
	response := perform(
		app.Handler(),
		http.MethodGet,
		"/v1/auth/login?population=customer&return_to=%2Fcustomer",
		nil,
	)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("login status=%d body=%s", response.Code, response.Body)
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, scope := range metrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name != "atlas.identity.operation.count" &&
				metric.Name != "atlas.identity.operation.duration" {
				continue
			}
			seen[metric.Name] = true
			var sets []attribute.Set
			switch data := metric.Data.(type) {
			case metricdata.Sum[int64]:
				for _, point := range data.DataPoints {
					sets = append(sets, point.Attributes)
				}
			case metricdata.Histogram[float64]:
				for _, point := range data.DataPoints {
					sets = append(sets, point.Attributes)
				}
			default:
				t.Fatalf("unexpected identity metric aggregation %T", metric.Data)
			}
			if len(sets) != 1 {
				t.Fatalf("identity metric points=%d, want 1", len(sets))
			}
			want := map[string]string{
				"atlas.identity.operation":  "login",
				"atlas.outcome":             "ok",
				"http.response.status_code": "303",
			}
			for _, item := range sets[0].ToSlice() {
				expected, allowed := want[string(item.Key)]
				if !allowed || item.Value.Emit() != expected {
					t.Fatalf("unsafe identity metric attribute %s=%s", item.Key, item.Value.Emit())
				}
				delete(want, string(item.Key))
			}
			if len(want) != 0 {
				t.Fatalf("identity metric attributes missing: %v", want)
			}
		}
	}
	if !seen["atlas.identity.operation.count"] || !seen["atlas.identity.operation.duration"] {
		t.Fatalf("identity operation metrics missing: %v", seen)
	}
}

func TestIdentityHandlersRejectUnknownJSONFieldsDuplicateCookiesAndCallbackReplay(t *testing.T) {
	service, _, provider := newHTTPIdentityService(t)
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
	})
	login := perform(app.Handler(), http.MethodGet, "/v1/auth/login?population=customer", nil)
	location, _ := url.Parse(login.Header().Get("Location"))
	state := location.Query().Get("state")
	provider.claims.Nonce = provider.nonce
	callbackTarget := "/v1/auth/callback?code=synthetic-code-0002&state=" + url.QueryEscape(state)
	first := perform(app.Handler(), http.MethodGet, callbackTarget, nil)
	if first.Code != http.StatusSeeOther {
		t.Fatalf("first callback status=%d body=%s", first.Code, first.Body)
	}
	replay := perform(app.Handler(), http.MethodGet, callbackTarget, nil)
	if replay.Code != http.StatusUnauthorized || !strings.Contains(replay.Body.String(), "OIDC_TRANSACTION_INVALID") {
		t.Fatalf("callback replay status=%d body=%s", replay.Code, replay.Body)
	}

	badBody := httptest.NewRequest(
		http.MethodPost, "/v1/sessions/revoke-all",
		strings.NewReader(`{"include_current":false,"unexpected":true}`),
	)
	badBody.Header.Set("Content-Type", "application/json")
	badBody.AddCookie(first.Result().Cookies()[0])
	badBodyResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(badBodyResponse, badBody)
	if badBodyResponse.Code != http.StatusBadRequest {
		t.Fatalf("unknown JSON field status=%d body=%s", badBodyResponse.Code, badBodyResponse.Body)
	}

	duplicate := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	duplicate.Header.Add("Cookie", identity.SessionCookieName+"="+strings.Repeat("A", 43))
	duplicate.Header.Add("Cookie", identity.SessionCookieName+"="+strings.Repeat("B", 43))
	duplicateResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(duplicateResponse, duplicate)
	if duplicateResponse.Code != http.StatusUnauthorized {
		t.Fatalf("duplicate session cookie status=%d body=%s", duplicateResponse.Code, duplicateResponse.Body)
	}
}

type httpIdentityProvider struct {
	state               string
	nonce               string
	pkce                string
	authorizationIssuer string
	claims              identity.ProviderClaims
	authorizationCalls  int
}

func (provider *httpIdentityProvider) AuthorizationURL(
	_ context.Context,
	_ identity.Population,
	state, nonce, pkce string,
	_ identity.TransactionKind,
) (string, error) {
	provider.authorizationCalls++
	provider.state, provider.nonce, provider.pkce = state, nonce, pkce
	return "https://identity.test.invalid/authorize?state=" + url.QueryEscape(state), nil
}

func (provider *httpIdentityProvider) Exchange(
	_ context.Context,
	_ identity.Population,
	_ string,
	pkce string,
	authorizationIssuer string,
) (identity.ProviderClaims, error) {
	if pkce != provider.pkce {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	provider.authorizationIssuer = authorizationIssuer
	return provider.claims, nil
}

type httpCryptor struct{}

func (httpCryptor) Encrypt(value []byte) ([]byte, uint64, error) {
	return append([]byte(nil), value...), 1, nil
}

func (httpCryptor) Decrypt(value []byte, version uint64) ([]byte, error) {
	if version != 1 {
		return nil, errors.New("wrong version")
	}
	return append([]byte(nil), value...), nil
}

type httpIdentityStore struct {
	transaction     identity.OIDCTransaction
	transactionUsed bool
	sessions        map[[32]byte]identity.Session
	revocations     int
	stepUps         map[[32]byte]httpStepUpClaim
}

type httpStepUpClaim struct {
	requestID     identifier.ID
	requestDigest [32]byte
	transaction   identity.OIDCTransaction
	url           string
	completed     bool
}

func (store *httpIdentityStore) PutOIDCTransaction(_ context.Context, transaction identity.OIDCTransaction) error {
	store.transaction = transaction
	store.transactionUsed = false
	return nil
}

func (store *httpIdentityStore) TakeOIDCTransaction(
	_ context.Context,
	state [32]byte,
	now time.Time,
) (identity.OIDCTransaction, error) {
	if store.transactionUsed || state != store.transaction.StateDigest || !now.Before(store.transaction.ExpiresAt) {
		return identity.OIDCTransaction{}, identity.ErrOIDCTransactionInvalid
	}
	store.transactionUsed = true
	return store.transaction, nil
}

func (store *httpIdentityStore) ClaimStepUp(
	_ context.Context,
	command identity.StepUpClaimCommand,
) (identity.StepUpClaimResult, error) {
	claim, found := store.stepUps[command.IdempotencyScope]
	if found {
		if claim.requestDigest != command.RequestDigest {
			return identity.StepUpClaimResult{}, identity.ErrIdempotencyConflict
		}
		if claim.completed {
			return identity.StepUpClaimResult{
				ChallengeRequestID: claim.requestID, Replay: true,
				TransactionID:    claim.transaction.TransactionID,
				AuthorizationURL: claim.url, ExpiresAt: claim.transaction.ExpiresAt,
			}, nil
		}
		return identity.StepUpClaimResult{}, identity.ErrIdempotencyInProgress
	}
	store.stepUps[command.IdempotencyScope] = httpStepUpClaim{
		requestID: command.ChallengeRequestID, requestDigest: command.RequestDigest,
	}
	return identity.StepUpClaimResult{
		ChallengeRequestID: command.ChallengeRequestID, Owner: true,
	}, nil
}

func (store *httpIdentityStore) CompleteStepUp(
	_ context.Context,
	command identity.CompleteStepUpCommand,
) error {
	claim := store.stepUps[command.IdempotencyScope]
	claim.completed = true
	claim.transaction = command.Transaction
	claim.url = command.AuthorizationURL
	store.stepUps[command.IdempotencyScope] = claim
	store.transaction = command.Transaction
	store.transactionUsed = false
	return nil
}

func (store *httpIdentityStore) FailStepUp(
	context.Context,
	identifier.ID,
	[32]byte,
	time.Time,
) error {
	return nil
}

func (store *httpIdentityStore) CreateSession(
	_ context.Context,
	command identity.CreateSessionCommand,
) (identity.Session, error) {
	principalID, _ := identifier.Parse("usr_00000000000000000001")
	tenantID, _ := identifier.Parse("ten_00000000000000000001")
	session := identity.Session{
		SessionID: command.SessionID, PrincipalID: principalID,
		PrincipalType: "customer", DisplayName: "Synthetic Customer",
		Population: command.Population, TenantID: tenantID, Assurance: command.Assurance,
		AuthorizationVersion: 1, RotationVersion: 1,
		CreatedAt: command.AuthorizationAt, LastSeenAt: command.AuthorizationAt,
		IdleExpiresAt: command.IdleExpiresAt, AbsoluteExpiresAt: command.AbsoluteExpiresAt,
		ClientLabel: command.ClientLabel,
		Permissions: []string{"identity.me.read", "identity.sessions.revoke_self"},
	}
	store.sessions[command.VerifierDigest] = session
	return session, nil
}

func (store *httpIdentityStore) Authenticate(
	_ context.Context,
	digest [32]byte,
	_ time.Time,
	_ time.Duration,
) (identity.Session, error) {
	session, found := store.sessions[digest]
	if !found {
		return identity.Session{}, identity.ErrAuthenticationRequired
	}
	return session, nil
}

func (store *httpIdentityStore) ListSessions(
	context.Context,
	identity.Session,
	time.Time,
) ([]identity.SessionSummary, error) {
	return nil, nil
}

func (store *httpIdentityStore) RevokeCurrent(
	context.Context,
	identity.Session,
	time.Time,
	audit.Event,
) error {
	store.revocations++
	return nil
}

func (store *httpIdentityStore) RevokeOne(
	context.Context,
	identity.RevocationCommand,
) (identity.RevocationResult, error) {
	store.revocations++
	return identity.RevocationResult{}, nil
}

func (store *httpIdentityStore) RevokeAll(
	context.Context,
	identity.RevocationCommand,
) (identity.RevocationResult, error) {
	store.revocations++
	return identity.RevocationResult{}, nil
}

func (store *httpIdentityStore) RevokeForSecurity(
	_ context.Context,
	command identity.AdminRevocationCommand,
) (identity.AdminRevocationResult, error) {
	store.revocations++
	return identity.AdminRevocationResult{DecisionID: command.AuditEvent.DecisionID}, nil
}

func newHTTPIdentityService(t *testing.T) (*identity.Service, *httpIdentityStore, *httpIdentityProvider) {
	t.Helper()
	store := &httpIdentityStore{
		sessions: make(map[[32]byte]identity.Session),
		stepUps:  make(map[[32]byte]httpStepUpClaim),
	}
	provider := &httpIdentityProvider{claims: identity.ProviderClaims{
		Issuer:          "https://identity.test.invalid/realms/customer",
		Subject:         "00000000-0000-4000-8000-000000000101",
		Assurance:       identity.AssuranceBaseline,
		AuthenticatedAt: testBuildTime,
	}}
	csrf, err := identity.NewHMACCSRFProtector(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	counter := 0
	service, err := identity.NewService(identity.ServiceOptions{
		Store: store, Provider: provider, Cryptor: httpCryptor{}, CSRF: csrf,
		Clock: clock.NewFixed(testBuildTime), Entropy: &sequentialReader{next: 40},
		NewID: func(prefix string) (identifier.ID, error) {
			counter++
			return identifier.Parse(fmt.Sprintf("%s_%020d", prefix, counter))
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, store, provider
}
