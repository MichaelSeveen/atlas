package oidcclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
	jose "github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	metricdata "go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAuthorizationCodeFlowValidatesDiscoveryPKCENonceAndTokenClaims(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	upstream := newOIDCTestServer(t, now)
	client := newTestClient(t, upstream, now)
	state := tokenCharacter('s')
	nonce := tokenCharacter('n')
	verifier := tokenCharacter('v')
	authorizationURL, err := client.AuthorizationURL(
		context.Background(), identity.PopulationCustomer,
		state, nonce, verifier, identity.TransactionLogin,
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if parsed.Scheme+"://"+parsed.Host != upstream.server.URL ||
		query.Get("state") != state || query.Get("nonce") != nonce ||
		query.Get("code_challenge_method") != "S256" ||
		query.Get("code_challenge") != oauth2.S256ChallengeFromVerifier(verifier) ||
		query.Get("acr_values") != "1" ||
		query.Get("redirect_uri") != upstream.server.URL+"/v1/auth/callback" {
		t.Fatalf("unsafe authorization URL: %s", authorizationURL)
	}

	upstream.setClaims(tokenClaims{
		issuer: upstream.issuer, audience: "atlas-bff-test", subject: "subject-1",
		nonce: nonce, acr: "1", issuedAt: now, expiresAt: now.Add(5 * time.Minute),
		authenticatedAt: now,
	})
	claims, err := client.Exchange(
		context.Background(), identity.PopulationCustomer, "synthetic-code-0001", verifier, upstream.issuer,
	)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Issuer != upstream.issuer || claims.Subject != "subject-1" ||
		claims.Nonce != nonce || claims.Assurance != identity.AssuranceBaseline ||
		claims.AuthenticatedAt != now {
		t.Fatalf("unexpected verified claims: %+v", claims)
	}
	if upstream.lastVerifier() != verifier {
		t.Fatal("token endpoint did not receive the original PKCE verifier")
	}
}

func TestStepUpAuthorizationRequestsFreshHigherAssurance(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	upstream := newOIDCTestServer(t, now)
	client := newTestClient(t, upstream, now)
	authorizationURL, err := client.AuthorizationURL(
		context.Background(), identity.PopulationCustomer,
		tokenCharacter('s'), tokenCharacter('n'), tokenCharacter('v'),
		identity.TransactionStepUp,
	)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("acr_values") != "2 3" ||
		query.Get("prompt") != "login" ||
		query.Get("max_age") != "0" {
		t.Fatalf("unsafe step-up authorization URL: %s", authorizationURL)
	}
}

func TestProviderMetricsExposeOnlyBoundedOperationPopulationAndOutcome(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	upstream := newOIDCTestServer(t, now)
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	client, err := NewWithMeter(
		testConfigs(upstream),
		clock.NewFixed(now),
		&http.Client{Timeout: 2 * time.Second},
		meterProvider.Meter("atlas-oidc-test"),
	)
	if err != nil {
		t.Fatal(err)
	}
	nonce := tokenCharacter('n')
	verifier := tokenCharacter('v')
	if _, err := client.AuthorizationURL(
		context.Background(), identity.PopulationCustomer,
		tokenCharacter('s'), nonce, verifier, identity.TransactionLogin,
	); err != nil {
		t.Fatal(err)
	}
	upstream.setClaims(tokenClaims{
		issuer: upstream.issuer, audience: "atlas-bff-test", subject: "subject-1",
		nonce: nonce, acr: "1", issuedAt: now, expiresAt: now.Add(5 * time.Minute),
		authenticatedAt: now,
	})
	if _, err := client.Exchange(
		context.Background(), identity.PopulationCustomer,
		"synthetic-code-metric-1", verifier, upstream.issuer,
	); err != nil {
		t.Fatal(err)
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, scope := range metrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			if metric.Name != "atlas.identity.provider.request.count" &&
				metric.Name != "atlas.identity.provider.request.duration" {
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
				t.Fatalf("unexpected provider metric aggregation %T", metric.Data)
			}
			if len(sets) != 2 {
				t.Fatalf("provider metric points=%d, want discovery and token", len(sets))
			}
			operations := make(map[string]bool)
			for _, set := range sets {
				values := make(map[string]string)
				for _, item := range set.ToSlice() {
					values[string(item.Key)] = item.Value.Emit()
				}
				if len(values) != 3 ||
					values["atlas.identity.population"] != "customer" ||
					values["atlas.outcome"] != "ok" ||
					(values["atlas.identity.provider.operation"] != "discovery" &&
						values["atlas.identity.provider.operation"] != "token") {
					t.Fatalf("unsafe provider metric attributes: %v", values)
				}
				operations[values["atlas.identity.provider.operation"]] = true
			}
			if !operations["discovery"] || !operations["token"] {
				t.Fatalf("provider operations missing: %v", operations)
			}
		}
	}
	if !seen["atlas.identity.provider.request.count"] ||
		!seen["atlas.identity.provider.request.duration"] {
		t.Fatalf("provider metrics missing: %v", seen)
	}
}

func TestOIDCTokenConfusionTimingAndKeyRotationAreRejectedOrRevalidated(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	upstream := newOIDCTestServer(t, now)
	client := newTestClient(t, upstream, now)
	verifier := tokenCharacter('p')
	base := tokenClaims{
		issuer: upstream.issuer, audience: "atlas-bff-test", subject: "subject-1",
		nonce: tokenCharacter('n'), acr: "2", issuedAt: now,
		expiresAt: now.Add(5 * time.Minute), authenticatedAt: now,
	}
	tests := []struct {
		name   string
		mutate func(*tokenClaims)
	}{
		{name: "wrong issuer", mutate: func(claims *tokenClaims) { claims.issuer = "https://wrong.invalid" }},
		{name: "wrong audience", mutate: func(claims *tokenClaims) { claims.audience = "other-client" }},
		{name: "expired outside skew", mutate: func(claims *tokenClaims) { claims.expiresAt = now.Add(-2 * time.Minute) }},
		{name: "future issued at", mutate: func(claims *tokenClaims) { claims.issuedAt = now.Add(2 * time.Minute) }},
		{name: "future not before", mutate: func(claims *tokenClaims) { claims.notBefore = now.Add(2 * time.Minute) }},
		{name: "unknown assurance", mutate: func(claims *tokenClaims) { claims.acr = "urn:attacker:downgrade" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claims := base
			test.mutate(&claims)
			upstream.setClaims(claims)
			if _, err := client.Exchange(
				context.Background(), identity.PopulationCustomer,
				"synthetic-code-"+test.name, verifier, upstream.issuer,
			); err == nil {
				t.Fatal("invalid OIDC token was accepted")
			}
		})
	}

	upstream.setClaims(base)
	if _, err := client.Exchange(
		context.Background(), identity.PopulationCustomer, "synthetic-code-valid-1", verifier, upstream.issuer,
	); err != nil {
		t.Fatalf("initial key failed: %v", err)
	}
	upstream.rotateKey(t)
	if _, err := client.Exchange(
		context.Background(), identity.PopulationCustomer, "synthetic-code-valid-2", verifier, upstream.issuer,
	); err != nil {
		t.Fatalf("rotated allow-listed key was not refreshed: %v", err)
	}
}

func TestPublicIssuerFormIsVerifiedThenCanonicalizedToTheInternalSubjectIssuer(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	upstream := newOIDCTestServer(t, now)
	client := newTestClient(t, upstream, now)
	config := client.configs[identity.PopulationCustomer]
	config.PublicOrigin = "https://identity.browser.test.invalid"
	client.configs[identity.PopulationCustomer] = config
	publicIssuer := config.PublicOrigin + "/realms/customer"
	nonce := tokenCharacter('n')
	verifier := tokenCharacter('v')
	upstream.setClaims(tokenClaims{
		issuer: publicIssuer, audience: "atlas-bff-test", subject: "subject-1",
		nonce: nonce, acr: "1", issuedAt: now, expiresAt: now.Add(5 * time.Minute),
		authenticatedAt: now,
	})

	claims, err := client.Exchange(
		context.Background(),
		identity.PopulationCustomer,
		"synthetic-code-public-issuer",
		verifier,
		publicIssuer,
	)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Issuer != upstream.issuer {
		t.Fatalf("verified public issuer was not canonicalized: %q", claims.Issuer)
	}
	if _, err := client.Exchange(
		context.Background(),
		identity.PopulationCustomer,
		"synthetic-code-attacker-issuer",
		verifier,
		"https://attacker.invalid/realms/customer",
	); err == nil {
		t.Fatal("unconfigured authorization-response issuer was accepted")
	}
}

func TestConfigurationRejectsAlgorithmAndEndpointEscapes(t *testing.T) {
	config := PopulationConfig{
		Population:          identity.PopulationCustomer,
		Issuer:              "https://identity.test.invalid/realms/customer",
		PublicOrigin:        "https://identity.test.invalid",
		ClientID:            "atlas-bff-test",
		RedirectURL:         "https://api.test.invalid/v1/auth/callback",
		SupportedAlgorithms: []string{"RS256"},
	}
	tests := []func(*PopulationConfig){
		func(value *PopulationConfig) { value.SupportedAlgorithms = []string{"none"} },
		func(value *PopulationConfig) { value.RedirectURL = "https://attacker.invalid/callback" },
		func(value *PopulationConfig) { value.PublicOrigin = "javascript:alert(1)" },
		func(value *PopulationConfig) { value.Issuer = "https://identity.test.invalid/realms/customer/" },
	}
	for _, mutate := range tests {
		mutated := config
		mutate(&mutated)
		configs := []PopulationConfig{mutated, validConfig(identity.PopulationMerchant), validConfig(identity.PopulationWorkforce)}
		if _, err := New(configs, nil, nil); err == nil {
			t.Fatal("unsafe OIDC configuration was accepted")
		}
	}
}

type tokenClaims struct {
	issuer          string
	audience        string
	subject         string
	nonce           string
	acr             string
	issuedAt        time.Time
	expiresAt       time.Time
	notBefore       time.Time
	authenticatedAt time.Time
}

type oidcTestServer struct {
	server       *httptest.Server
	issuer       string
	redirectURL  string
	mu           sync.Mutex
	privateKey   *rsa.PrivateKey
	keyID        string
	claims       tokenClaims
	pkceVerifier string
}

func newOIDCTestServer(t *testing.T, now time.Time) *oidcTestServer {
	t.Helper()
	instance := &oidcTestServer{keyID: "test-key-1"}
	instance.privateKey = newRSAKey(t)
	instance.server = httptest.NewServer(http.HandlerFunc(instance.serveHTTP))
	instance.issuer = instance.server.URL + "/realms/customer"
	instance.redirectURL = instance.server.URL + "/v1/auth/callback"
	instance.claims = tokenClaims{
		issuer: instance.issuer, audience: "atlas-bff-test", subject: "subject-1",
		nonce: tokenCharacter('n'), acr: "1", issuedAt: now,
		expiresAt: now.Add(5 * time.Minute), authenticatedAt: now,
	}
	t.Cleanup(instance.server.Close)
	return instance
}

func (server *oidcTestServer) serveHTTP(response http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/realms/customer/.well-known/openid-configuration":
		writeTestJSON(response, map[string]any{
			"issuer":                                server.issuer,
			"authorization_endpoint":                server.issuer + "/protocol/openid-connect/auth",
			"token_endpoint":                        server.issuer + "/protocol/openid-connect/token",
			"jwks_uri":                              server.issuer + "/protocol/openid-connect/certs",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	case "/realms/customer/protocol/openid-connect/certs":
		server.mu.Lock()
		key := jose.JSONWebKey{
			Key: &server.privateKey.PublicKey, KeyID: server.keyID,
			Algorithm: coreoidc.RS256, Use: "sig",
		}
		server.mu.Unlock()
		writeTestJSON(response, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{key}})
	case "/realms/customer/protocol/openid-connect/token":
		if err := request.ParseForm(); err != nil {
			http.Error(response, "malformed", http.StatusBadRequest)
			return
		}
		server.mu.Lock()
		server.pkceVerifier = request.Form.Get("code_verifier")
		claims := server.claims
		key := server.privateKey
		keyID := server.keyID
		server.mu.Unlock()
		raw := map[string]any{
			"iss": claims.issuer, "aud": claims.audience, "sub": claims.subject,
			"nonce": claims.nonce, "acr": claims.acr,
			"iat": claims.issuedAt.Unix(), "exp": claims.expiresAt.Unix(),
			"auth_time": claims.authenticatedAt.Unix(),
		}
		if !claims.notBefore.IsZero() {
			raw["nbf"] = claims.notBefore.Unix()
		}
		encoded, _ := json.Marshal(raw)
		idToken := oidctest.SignIDToken(key, keyID, coreoidc.RS256, string(encoded))
		writeTestJSON(response, map[string]any{
			"access_token": "discarded-upstream-access-token",
			"token_type":   "Bearer", "expires_in": 300, "id_token": idToken,
		})
	default:
		http.NotFound(response, request)
	}
}

func (server *oidcTestServer) setClaims(claims tokenClaims) {
	server.mu.Lock()
	server.claims = claims
	server.mu.Unlock()
}

func (server *oidcTestServer) lastVerifier() string {
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.pkceVerifier
}

func (server *oidcTestServer) rotateKey(t *testing.T) {
	t.Helper()
	server.mu.Lock()
	server.privateKey = newRSAKey(t)
	server.keyID = "test-key-2"
	server.mu.Unlock()
}

func newTestClient(t *testing.T, upstream *oidcTestServer, now time.Time) *Client {
	t.Helper()
	client, err := New(testConfigs(upstream), clock.NewFixed(now), &http.Client{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func testConfigs(upstream *oidcTestServer) []PopulationConfig {
	configs := make([]PopulationConfig, 0, 3)
	for _, population := range []identity.Population{
		identity.PopulationCustomer, identity.PopulationMerchant, identity.PopulationWorkforce,
	} {
		configs = append(configs, PopulationConfig{
			Population: population, Issuer: upstream.issuer,
			PublicOrigin: upstream.server.URL, ClientID: "atlas-bff-test",
			RedirectURL: upstream.redirectURL, SupportedAlgorithms: []string{"RS256"},
		})
	}
	return configs
}

func validConfig(population identity.Population) PopulationConfig {
	return PopulationConfig{
		Population:          population,
		Issuer:              "https://identity.test.invalid/realms/" + string(population),
		PublicOrigin:        "https://identity.test.invalid",
		ClientID:            "atlas-bff-test",
		RedirectURL:         "https://api.test.invalid/v1/auth/callback",
		SupportedAlgorithms: []string{"RS256"},
	}
}

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func tokenCharacter(character byte) string {
	return string(bytesOf(character, 43))
}

func bytesOf(character byte, count int) []byte {
	value := make([]byte, count)
	for index := range value {
		value[index] = character
	}
	return value
}

func writeTestJSON(response http.ResponseWriter, value any) {
	response.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(response).Encode(value)
}
