package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
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
		"/v1/step-up/challenges", "/v1/me/active-organization", "/v1/organizations",
		"/v1/organizations/{organization_id}/members",
		"/v1/organizations/{organization_id}/members/{member_id}",
		"/v1/organizations/{organization_id}/invitations",
		"/v1/organization-invitations/{invitation_id}/authentication",
		"/v1/organization-invitations/{invitation_id}/acceptance",
	}
	if strings.Join(identityRoutes, ",") != strings.Join(want, ",") {
		t.Fatalf("identity route inventory=%v", identityRoutes)
	}
}

func TestOrganizationListAndTenantSwitchHTTPContract(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("O", 43)
	principalID, _ := identifier.Parse("usr_00000000000000000091")
	sessionID, _ := identifier.Parse("ses_00000000000000000090")
	currentTenant, _ := identifier.Parse("ten_00000000000000000090")
	targetTenant, _ := identifier.Parse("ten_00000000000000000091")
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		DisplayName: "Synthetic Merchant", Population: identity.PopulationMerchant,
		TenantID: currentTenant, Assurance: identity.AssuranceBaseline,
		AuthorizationVersion: 1, RotationVersion: 4,
		CreatedAt: testBuildTime, LastSeenAt: testBuildTime,
		IdleExpiresAt:     testBuildTime.Add(20 * time.Minute),
		AbsoluteExpiresAt: testBuildTime.Add(8 * time.Hour),
		Permissions:       []string{"organization.list", "organization.active.switch"},
	}
	organizationStore := store.organizationStore
	organizationStore.organizations = []identity.Organization{{
		OrganizationID: targetTenant, DisplayName: "Synthetic Merchant Two",
		PrincipalRole: "merchant_operator", MembershipVersion: 2,
	}}
	_, csrfToken, err := service.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
	})

	list := httptest.NewRequest(http.MethodGet, "/v1/organizations", nil)
	list.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	listResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusOK ||
		!strings.Contains(listResponse.Body.String(), targetTenant.String()) ||
		strings.Contains(listResponse.Body.String(), cookieValue) {
		t.Fatalf("organization list status=%d body=%s", listResponse.Code, listResponse.Body)
	}

	request := httptest.NewRequest(
		http.MethodPut,
		"/v1/me/active-organization",
		strings.NewReader(`{"organization_id":"`+targetTenant.String()+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(identity.CSRFHeaderName, csrfToken)
	request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	cookies := response.Result().Cookies()
	if response.Code != http.StatusOK ||
		len(cookies) != 1 ||
		cookies[0].Value == cookieValue ||
		response.Header().Get(identity.CSRFHeaderName) == csrfToken ||
		!strings.HasPrefix(response.Header().Get("X-Authorization-Decision-Id"), "dec_") ||
		!strings.Contains(response.Body.String(), `"active_tenant_id":"`+targetTenant.String()+`"`) {
		t.Fatalf("tenant switch status=%d headers=%v body=%s", response.Code, response.Header(), response.Body)
	}
	if _, _, err := service.Current(context.Background(), cookieValue); !errors.Is(err, identity.ErrAuthenticationRequired) {
		t.Fatalf("old session retained grace: %v", err)
	}
	if _, _, err := service.Current(context.Background(), cookies[0].Value); err != nil {
		t.Fatalf("rotated session is not usable: %v", err)
	}
}

func TestOrganizationMemberRoleChangeHTTPContractAndReplay(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("V", 43)
	principalID, _ := identifier.Parse("usr_00000000000000000101")
	sessionID, _ := identifier.Parse("ses_00000000000000000101")
	tenantID, _ := identifier.Parse("ten_00000000000000000101")
	membershipID, _ := identifier.Parse("mem_00000000000000000102")
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		DisplayName: "Synthetic Merchant Administrator", Population: identity.PopulationMerchant,
		TenantID: tenantID, Assurance: identity.AssuranceBaseline,
		AuthorizationVersion: 1, RotationVersion: 2,
		CreatedAt: testBuildTime, LastSeenAt: testBuildTime,
		IdleExpiresAt:     testBuildTime.Add(20 * time.Minute),
		AbsoluteExpiresAt: testBuildTime.Add(8 * time.Hour),
		Permissions:       []string{"organization.members.roles.update"},
	}
	_, csrfToken, err := service.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
		options.CORS = CORSConfig{AllowedOrigins: []string{"https://web.test.invalid"}, AllowCredentials: true}
	})
	path := "/v1/organizations/" + tenantID.String() + "/members/" + membershipID.String()
	requestBody := `{"role":"merchant_operator","purpose":"organization_administration"}`

	call := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(requestBody))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set(identity.CSRFHeaderName, csrfToken)
		request.Header.Set("Idempotency-Key", "http-role-change-0001")
		request.Header.Set("If-Match", "\"membership-v3\"")
		request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
		response := httptest.NewRecorder()
		app.Handler().ServeHTTP(response, request)
		return response
	}
	first := call()
	if first.Code != http.StatusOK || first.Header().Get("ETag") != "\"membership-v4\"" ||
		first.Header().Get("Idempotency-Replayed") != "false" ||
		first.Header().Get("X-Authorization-Decision-Id") == "" ||
		!strings.Contains(first.Body.String(), `"role":"merchant_operator"`) ||
		!strings.Contains(first.Body.String(), `"version":4`) {
		t.Fatalf("role change status=%d headers=%v body=%s", first.Code, first.Header(), first.Body)
	}
	second := call()
	if second.Code != http.StatusOK || second.Header().Get("Idempotency-Replayed") != "true" ||
		second.Header().Get("ETag") != first.Header().Get("ETag") ||
		second.Header().Get("X-Authorization-Decision-Id") != first.Header().Get("X-Authorization-Decision-Id") {
		t.Fatalf("role change replay status=%d headers=%v body=%s", second.Code, second.Header(), second.Body)
	}

	preflight := httptest.NewRequest(http.MethodOptions, path, nil)
	preflight.Header.Set("Origin", "https://web.test.invalid")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPatch)
	preflight.Header.Set(
		"Access-Control-Request-Headers",
		"Content-Type, Idempotency-Key, If-Match, X-Atlas-CSRF-Token",
	)
	preflightResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent ||
		preflightResponse.Header().Get("Access-Control-Allow-Methods") != http.MethodPatch ||
		preflightResponse.Header().Get("Access-Control-Allow-Headers") !=
			"Content-Type, Idempotency-Key, If-Match, X-Atlas-CSRF-Token" ||
		!strings.Contains(preflightResponse.Header().Get("Access-Control-Expose-Headers"), "ETag") {
		t.Fatalf("role change preflight status=%d headers=%v body=%s",
			preflightResponse.Code, preflightResponse.Header(), preflightResponse.Body)
	}
}

func TestOrganizationMemberRevocationHTTPContractAndReplay(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("Y", 43)
	principalID, _ := identifier.Parse("usr_00000000000000000111")
	sessionID, _ := identifier.Parse("ses_00000000000000000111")
	tenantID, _ := identifier.Parse("ten_00000000000000000111")
	membershipID, _ := identifier.Parse("mem_00000000000000000112")
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		DisplayName: "Synthetic Merchant Security Administrator", Population: identity.PopulationMerchant,
		TenantID: tenantID, Assurance: identity.AssuranceBaseline,
		AuthorizationVersion: 1, RotationVersion: 2,
		CreatedAt: testBuildTime, LastSeenAt: testBuildTime,
		IdleExpiresAt:     testBuildTime.Add(20 * time.Minute),
		AbsoluteExpiresAt: testBuildTime.Add(8 * time.Hour),
		Permissions:       []string{"organization.members.remove"},
	}
	_, csrfToken, err := service.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
		options.CORS = CORSConfig{AllowedOrigins: []string{"https://web.test.invalid"}, AllowCredentials: true}
	})
	path := "/v1/organizations/" + tenantID.String() + "/members/" + membershipID.String()
	call := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodDelete, path, nil)
		request.Header.Set(identity.CSRFHeaderName, csrfToken)
		request.Header.Set("Idempotency-Key", "http-member-revoke-0001")
		request.Header.Set("If-Match", "\"membership-v3\"")
		request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
		response := httptest.NewRecorder()
		app.Handler().ServeHTTP(response, request)
		return response
	}
	first := call()
	if first.Code != http.StatusNoContent || first.Body.Len() != 0 ||
		first.Header().Get("Idempotency-Replayed") != "false" ||
		!strings.HasPrefix(first.Header().Get("X-Authorization-Decision-Id"), "dec_") {
		t.Fatalf("member revocation status=%d headers=%v body=%s", first.Code, first.Header(), first.Body)
	}
	second := call()
	if second.Code != http.StatusNoContent || second.Header().Get("Idempotency-Replayed") != "true" ||
		second.Header().Get("X-Authorization-Decision-Id") != first.Header().Get("X-Authorization-Decision-Id") {
		t.Fatalf("member revocation replay status=%d headers=%v body=%s", second.Code, second.Header(), second.Body)
	}

	withBody := httptest.NewRequest(http.MethodDelete, path, strings.NewReader(`{}`))
	withBody.Header.Set("Content-Type", "application/json")
	withBodyResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(withBodyResponse, withBody)
	if withBodyResponse.Code != http.StatusBadRequest {
		t.Fatalf("DELETE body status=%d body=%s", withBodyResponse.Code, withBodyResponse.Body)
	}

	preflight := httptest.NewRequest(http.MethodOptions, path, nil)
	preflight.Header.Set("Origin", "https://web.test.invalid")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodDelete)
	preflight.Header.Set("Access-Control-Request-Headers", "Idempotency-Key, If-Match, X-Atlas-CSRF-Token")
	preflightResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent ||
		preflightResponse.Header().Get("Access-Control-Allow-Methods") != http.MethodDelete ||
		preflightResponse.Header().Get("Access-Control-Allow-Headers") !=
			"Idempotency-Key, If-Match, X-Atlas-CSRF-Token" {
		t.Fatalf("member revocation preflight status=%d headers=%v body=%s",
			preflightResponse.Code, preflightResponse.Header(), preflightResponse.Body)
	}
}

func TestOrganizationMemberListHTTPPaginationAndConcealmentContract(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("W", 43)
	principalID, _ := identifier.Parse("usr_00000000000000000121")
	sessionID, _ := identifier.Parse("ses_00000000000000000120")
	tenantID, _ := identifier.Parse("ten_00000000000000000120")
	membershipID, _ := identifier.Parse("mem_00000000000000000120")
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		DisplayName: "Synthetic Merchant Viewer", Population: identity.PopulationMerchant,
		TenantID: tenantID, Assurance: identity.AssuranceBaseline,
		AuthorizationVersion: 1, RotationVersion: 1,
		CreatedAt: testBuildTime, LastSeenAt: testBuildTime,
		IdleExpiresAt:     testBuildTime.Add(20 * time.Minute),
		AbsoluteExpiresAt: testBuildTime.Add(8 * time.Hour),
		Permissions:       []string{"organization.members.read"},
	}
	store.organizationStore.members = identity.ListOrganizationMembersResult{
		Page: identity.OrganizationMemberPage{
			Members: []identity.OrganizationMember{{
				MembershipID: membershipID, OrganizationID: tenantID, PrincipalID: principalID,
				Role: "merchant_viewer", Status: "active", Version: 1, CreatedAt: testBuildTime,
			}},
			NextCursor: "djEKdGVuXzAwMDAwMDAwMDAwMDAwMDAwMTIwCnVzcl8wMDAwMDAwMDAwMDAwMDAwMDEyMQ",
			HasMore:    true,
		},
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
	})

	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/organizations/"+tenantID.String()+"/members?page_size=1",
		nil,
	)
	request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	request.Header.Set("X-Atlas-Purpose", "organization_administration")
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.HasPrefix(response.Header().Get("X-Authorization-Decision-Id"), "dec_") ||
		!strings.Contains(response.Body.String(), `"email_hint":null`) ||
		!strings.Contains(response.Body.String(), `"has_more":true`) ||
		!strings.Contains(response.Body.String(), `"next_cursor":"`) ||
		strings.Contains(response.Body.String(), cookieValue) {
		t.Fatalf("member list status=%d headers=%v body=%s", response.Code, response.Header(), response.Body)
	}
	if store.organizationStore.membersCommand.Purpose != "organization_administration" {
		t.Fatalf("member list purpose=%q", store.organizationStore.membersCommand.Purpose)
	}

	ambiguousPurpose := httptest.NewRequest(
		http.MethodGet,
		"/v1/organizations/"+tenantID.String()+"/members",
		nil,
	)
	ambiguousPurpose.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	ambiguousPurpose.Header.Add("X-Atlas-Purpose", "self_service")
	ambiguousPurpose.Header.Add("X-Atlas-Purpose", "organization_administration")
	ambiguousPurposeResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(ambiguousPurposeResponse, ambiguousPurpose)
	if ambiguousPurposeResponse.Code != http.StatusBadRequest {
		t.Fatalf("ambiguous purpose status=%d body=%s", ambiguousPurposeResponse.Code, ambiguousPurposeResponse.Body)
	}

	malformed := httptest.NewRequest(
		http.MethodGet,
		"/v1/organizations/"+tenantID.String()+"/members?total=true",
		nil,
	)
	malformed.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	malformedResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(malformedResponse, malformed)
	if malformedResponse.Code != http.StatusBadRequest {
		t.Fatalf("unknown member-list query status=%d body=%s", malformedResponse.Code, malformedResponse.Body)
	}

	store.organizationStore.membersErr = identity.ErrOrganizationNotFound
	callConcealed := func(organizationID string) (*httptest.ResponseRecorder, problemResponse) {
		t.Helper()
		request := httptest.NewRequest(
			http.MethodGet, "/v1/organizations/"+organizationID+"/members", nil,
		)
		request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
		response := httptest.NewRecorder()
		app.Handler().ServeHTTP(response, request)
		var problem problemResponse
		if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
			t.Fatalf("decode concealed member-list problem: %v", err)
		}
		return response, problem
	}
	knownForeignID := "ten_00000000000000000122"
	absentID := "ten_00000000000000000123"
	knownForeign, knownForeignProblem := callConcealed(knownForeignID)
	absent, absentProblem := callConcealed(absentID)
	if knownForeign.Code != http.StatusNotFound || absent.Code != http.StatusNotFound ||
		knownForeignProblem.Type != absentProblem.Type ||
		knownForeignProblem.Title != absentProblem.Title ||
		knownForeignProblem.Status != absentProblem.Status ||
		knownForeignProblem.Code != absentProblem.Code ||
		knownForeignProblem.Retryable != absentProblem.Retryable ||
		knownForeignProblem.Code != "NOT_FOUND_OR_CONCEALED" ||
		knownForeignProblem.RequestID == "" || absentProblem.RequestID == "" ||
		strings.Contains(knownForeign.Body.String(), knownForeignID) ||
		strings.Contains(absent.Body.String(), absentID) {
		t.Fatalf("concealment mismatch known=%d/%+v absent=%d/%+v",
			knownForeign.Code, knownForeignProblem, absent.Code, absentProblem)
	}
	for _, body := range []string{knownForeign.Body.String(), absent.Body.String()} {
		for _, forbidden := range []string{"data", "page", "next_cursor", "has_more", "total", "suggest"} {
			if strings.Contains(body, `"`+forbidden+`"`) {
				t.Fatalf("concealed member-list body exposes %q: %s", forbidden, body)
			}
		}
	}
}

func TestOrganizationInvitationHTTPSecretReplayContract(t *testing.T) {
	service, store, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("V", 43)
	principalID, _ := identifier.Parse("usr_00000000000000000111")
	sessionID, _ := identifier.Parse("ses_00000000000000000110")
	tenantID, _ := identifier.Parse("ten_00000000000000000110")
	store.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		DisplayName: "Synthetic Merchant Administrator", Population: identity.PopulationMerchant,
		TenantID: tenantID, Assurance: identity.AssuranceBaseline,
		AuthorizationVersion: 1, RotationVersion: 1,
		CreatedAt: testBuildTime, LastSeenAt: testBuildTime,
		IdleExpiresAt:     testBuildTime.Add(20 * time.Minute),
		AbsoluteExpiresAt: testBuildTime.Add(8 * time.Hour),
		Permissions:       []string{"organization.invitations.create"},
	}
	_, csrfToken, err := service.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
	})
	issue := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(
			http.MethodPost,
			"/v1/organizations/"+tenantID.String()+"/invitations",
			strings.NewReader(`{"email":"invitee@example.test","role":"merchant_viewer"}`),
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set(identity.CSRFHeaderName, csrfToken)
		request.Header.Set("Idempotency-Key", "http-invitation-create-0001")
		request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
		response := httptest.NewRecorder()
		app.Handler().ServeHTTP(response, request)
		return response
	}
	first := issue()
	replay := issue()
	if first.Code != http.StatusCreated || replay.Code != http.StatusCreated ||
		first.Header().Get("Idempotency-Replayed") != "false" ||
		replay.Header().Get("Idempotency-Replayed") != "true" ||
		!strings.HasPrefix(first.Header().Get("Location"), "/v1/organization-invitations/inv_") ||
		!strings.Contains(first.Body.String(), `"secret_disclosed":true`) ||
		!strings.Contains(replay.Body.String(), `"acceptance_token":null`) ||
		!strings.Contains(replay.Body.String(), `"secret_disclosed":false`) ||
		strings.Contains(first.Body.String(), "invitee@example.test") ||
		strings.Contains(replay.Body.String(), "invitee@example.test") {
		t.Fatalf("invitation first=%d/%v/%s replay=%d/%v/%s",
			first.Code, first.Header(), first.Body, replay.Code, replay.Header(), replay.Body)
	}
	if first.Header().Get("X-Authorization-Decision-Id") != replay.Header().Get("X-Authorization-Decision-Id") {
		t.Fatal("invitation replay changed the durable authorization decision")
	}
}

func TestOrganizationInvitationAuthenticationAndAcceptanceHTTPContract(t *testing.T) {
	service, store, provider := newHTTPIdentityService(t)
	invitationID, _ := identifier.Parse("inv_00000000000000000131")
	tenantID, _ := identifier.Parse("ten_00000000000000000131")
	store.organizationStore.invitation = identity.CreateInvitationStoreResult{
		Invitation: identity.OrganizationInvitation{
			InvitationID: invitationID, OrganizationID: tenantID,
			EmailHint: "r***@example.test", Role: "merchant_viewer", Status: "pending",
			CreatedAt: testBuildTime, ExpiresAt: testBuildTime.Add(72 * time.Hour),
		},
	}
	provider.claims.Issuer = "https://identity.test.invalid/realms/merchant"
	provider.claims.Subject = "00000000-0000-4000-8000-000000000131"
	provider.claims.Email = "recipient@example.test"
	provider.claims.EmailVerified = true
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = service
		options.WebOrigin = "https://web.test.invalid"
	})
	acceptanceToken := strings.Repeat("R", 43)
	authentication := httptest.NewRequest(
		http.MethodPost,
		"/v1/organization-invitations/"+invitationID.String()+"/authentication",
		strings.NewReader(`{"acceptance_token":"`+acceptanceToken+`"}`),
	)
	authentication.Header.Set("Content-Type", "application/json")
	authenticationResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(authenticationResponse, authentication)
	if authenticationResponse.Code != http.StatusSeeOther ||
		strings.Contains(authenticationResponse.Header().Get("Location"), acceptanceToken) ||
		store.transaction.Kind != identity.TransactionInvitationAcceptance ||
		store.transaction.InvitationID != invitationID ||
		store.transaction.InvitationTokenDigest != sha256.Sum256([]byte(acceptanceToken)) {
		t.Fatalf("invitation authentication status=%d location=%s transaction=%+v body=%s",
			authenticationResponse.Code, authenticationResponse.Header().Get("Location"),
			store.transaction, authenticationResponse.Body)
	}
	authorizationURL, err := url.Parse(authenticationResponse.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	provider.claims.Nonce = provider.nonce
	callback := httptest.NewRequest(
		http.MethodGet,
		"/v1/auth/callback?code=synthetic-invitation-code-0001&state="+
			url.QueryEscape(authorizationURL.Query().Get("state")),
		nil,
	)
	callbackResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(callbackResponse, callback)
	if callbackResponse.Code != http.StatusSeeOther || len(callbackResponse.Result().Cookies()) != 1 {
		t.Fatalf("invitation callback status=%d headers=%v body=%s",
			callbackResponse.Code, callbackResponse.Header(), callbackResponse.Body)
	}
	bootstrapCookie := callbackResponse.Result().Cookies()[0]
	current := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	current.AddCookie(bootstrapCookie)
	currentResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(currentResponse, current)
	csrfToken := currentResponse.Header().Get(identity.CSRFHeaderName)
	if currentResponse.Code != http.StatusOK ||
		!strings.Contains(currentResponse.Body.String(), `"active_tenant_id":null`) ||
		!strings.Contains(currentResponse.Body.String(), `"permissions":[]`) || len(csrfToken) != 43 {
		t.Fatalf("bootstrap current status=%d headers=%v body=%s",
			currentResponse.Code, currentResponse.Header(), currentResponse.Body)
	}
	acceptance := httptest.NewRequest(
		http.MethodPost,
		"/v1/organization-invitations/"+invitationID.String()+"/acceptance",
		strings.NewReader(`{"acceptance_token":"`+acceptanceToken+`"}`),
	)
	acceptance.Header.Set("Content-Type", "application/json")
	acceptance.Header.Set(identity.CSRFHeaderName, csrfToken)
	acceptance.Header.Set("Idempotency-Key", "http-invitation-accept-0001")
	acceptance.AddCookie(bootstrapCookie)
	acceptanceResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(acceptanceResponse, acceptance)
	if acceptanceResponse.Code != http.StatusOK ||
		acceptanceResponse.Header().Get("Idempotency-Replayed") != "false" ||
		!strings.HasPrefix(acceptanceResponse.Header().Get("X-Authorization-Decision-Id"), "dec_") ||
		len(acceptanceResponse.Result().Cookies()) != 1 ||
		!strings.Contains(acceptanceResponse.Body.String(), `"organization_id":"`+tenantID.String()+`"`) ||
		strings.Contains(acceptanceResponse.Body.String(), acceptanceToken) {
		t.Fatalf("invitation acceptance status=%d headers=%v body=%s",
			acceptanceResponse.Code, acceptanceResponse.Header(), acceptanceResponse.Body)
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
	transaction       identity.OIDCTransaction
	transactionUsed   bool
	sessions          map[[32]byte]identity.Session
	revocations       int
	stepUps           map[[32]byte]httpStepUpClaim
	organizationStore *httpOrganizationStore
	credentialStore   *httpCredentialStore
	credentialLimiter *httpCredentialLimiter
}

type httpOrganizationStore struct {
	sessions       *httpIdentityStore
	organizations  []identity.Organization
	members        identity.ListOrganizationMembersResult
	membersErr     error
	membersCommand identity.ListOrganizationMembersCommand
	invitation     identity.CreateInvitationStoreResult
	acceptance     identity.AcceptInvitationStoreResult
	roleChange     identity.UpdateOrganizationMemberRoleResult
	revocation     identity.RevokeOrganizationMemberResult
}

func (store *httpOrganizationStore) RevokeMember(
	_ context.Context,
	command identity.RevokeOrganizationMemberCommand,
) (identity.RevokeOrganizationMemberResult, error) {
	if !store.revocation.Member.MembershipID.IsZero() {
		replayed := store.revocation
		replayed.Replay = true
		return replayed, nil
	}
	revokedAt := command.Now
	store.revocation = identity.RevokeOrganizationMemberResult{
		Member: identity.OrganizationMember{
			MembershipID: command.MembershipID, OrganizationID: command.OrganizationID,
			PrincipalID: command.Actor.PrincipalID, Role: "merchant_viewer",
			Status: "revoked", Version: command.ExpectedVersion + 1,
			CreatedAt: command.Now, RevokedAt: &revokedAt,
		},
		DecisionID: command.AuditEvent.DecisionID,
	}
	return store.revocation, nil
}

func (store *httpOrganizationStore) UpdateMemberRole(
	_ context.Context,
	command identity.UpdateOrganizationMemberRoleCommand,
) (identity.UpdateOrganizationMemberRoleResult, error) {
	if !store.roleChange.Member.MembershipID.IsZero() {
		replayed := store.roleChange
		replayed.Replay = true
		return replayed, nil
	}
	store.roleChange = identity.UpdateOrganizationMemberRoleResult{
		Member: identity.OrganizationMember{
			MembershipID: command.MembershipID, OrganizationID: command.OrganizationID,
			PrincipalID: command.Actor.PrincipalID, Role: command.Role,
			Status: "active", Version: command.ExpectedVersion + 1, CreatedAt: command.Now,
		},
		DecisionID: command.AuditEvent.DecisionID,
	}
	return store.roleChange, nil
}

func (store *httpOrganizationStore) AcceptInvitation(
	_ context.Context,
	command identity.AcceptInvitationCommand,
) (identity.AcceptInvitationStoreResult, error) {
	if !store.acceptance.Member.MembershipID.IsZero() {
		replayed := store.acceptance
		replayed.Session = command.Actor
		replayed.Replay = true
		return replayed, nil
	}
	for digest, session := range store.sessions.sessions {
		if session.SessionID == command.Actor.SessionID {
			delete(store.sessions.sessions, digest)
		}
	}
	session := command.Actor
	session.SessionID = command.NewSessionID
	session.TenantID = command.Actor.TenantID
	if session.TenantID.IsZero() && store.invitation.Invitation.InvitationID == command.InvitationID {
		session.TenantID = store.invitation.Invitation.OrganizationID
	}
	session.RotationVersion++
	session.CreatedAt = command.Now
	session.LastSeenAt = command.Now
	session.IdleExpiresAt = command.IdleExpiresAt
	session.AbsoluteExpiresAt = command.AbsoluteExpiresAt
	session.InvitationAcceptanceOnly = false
	session.InvitationID = identifier.ID{}
	store.sessions.sessions[command.NewSessionVerifierDigest] = session
	store.acceptance = identity.AcceptInvitationStoreResult{
		Member: identity.OrganizationMember{
			MembershipID: command.MembershipID, OrganizationID: session.TenantID,
			PrincipalID: command.Actor.PrincipalID, Role: "merchant_viewer",
			Status: "active", Version: 1, CreatedAt: command.Now,
		},
		Session: session, DecisionID: command.AuditEvent.DecisionID,
	}
	return store.acceptance, nil
}

func (store *httpOrganizationStore) ListMembers(
	_ context.Context,
	command identity.ListOrganizationMembersCommand,
) (identity.ListOrganizationMembersResult, error) {
	store.membersCommand = command
	result := store.members
	if result.DecisionID.IsZero() {
		result.DecisionID = command.AuditEvent.DecisionID
	}
	return result, store.membersErr
}

func (store *httpOrganizationStore) ListOrganizations(
	context.Context,
	identity.Session,
) ([]identity.Organization, error) {
	return append([]identity.Organization(nil), store.organizations...), nil
}

func (store *httpOrganizationStore) SwitchOrganization(
	_ context.Context,
	command identity.SwitchOrganizationCommand,
) (identity.Session, error) {
	for digest, session := range store.sessions.sessions {
		if session.SessionID == command.Actor.SessionID {
			delete(store.sessions.sessions, digest)
		}
	}
	session := command.Actor
	session.SessionID = command.NewSessionID
	session.TenantID = command.OrganizationID
	session.RotationVersion++
	session.CreatedAt = command.Now
	session.LastSeenAt = command.Now
	session.IdleExpiresAt = command.IdleExpiresAt
	session.StepUpAction = ""
	session.StepUpVerifiedAt = time.Time{}
	store.sessions.sessions[command.VerifierDigest] = session
	return session, nil
}

func (store *httpOrganizationStore) CreateInvitation(
	_ context.Context,
	command identity.CreateInvitationCommand,
) (identity.CreateInvitationStoreResult, error) {
	if !store.invitation.Invitation.InvitationID.IsZero() {
		replayed := store.invitation
		replayed.Replay = true
		return replayed, nil
	}
	store.invitation = identity.CreateInvitationStoreResult{
		Invitation: identity.OrganizationInvitation{
			InvitationID: command.InvitationID, OrganizationID: command.OrganizationID,
			EmailHint: command.EmailHint, Role: command.Role, Status: "pending",
			ExpiresAt: command.ExpiresAt, CreatedAt: command.Now,
		},
		DecisionID: command.AuditEvent.DecisionID,
	}
	return store.invitation, nil
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
	if command.Kind == identity.TransactionInvitationAcceptance {
		session.PrincipalID = command.ProvisionalPrincipalID
		session.PrincipalType = "merchant"
		session.DisplayName = "Synthetic Invitation Recipient"
		session.TenantID = identifier.ID{}
		session.Permissions = []string{}
		session.InvitationID = command.InvitationID
		session.VerifiedEmailDigest = command.VerifiedEmailDigest
		session.InvitationAcceptanceOnly = true
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
	organizationStore := &httpOrganizationStore{sessions: store}
	store.organizationStore = organizationStore
	credentialStore := newHTTPCredentialStore()
	credentialLimiter := &httpCredentialLimiter{decision: identity.CredentialRateDecision{Allowed: true}}
	store.credentialStore = credentialStore
	store.credentialLimiter = credentialLimiter
	service, err := identity.NewService(identity.ServiceOptions{
		Store: store, Organizations: organizationStore, Credentials: credentialStore,
		CredentialLimiter: credentialLimiter, CredentialEnvironment: "test",
		CredentialNetworkSignalKey: bytes.Repeat([]byte{61}, 32),
		Provider:                   provider, Cryptor: httpCryptor{}, CSRF: csrf,
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
