package server

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type principalResponse struct {
	ID                   string   `json:"id"`
	Type                 string   `json:"type"`
	DisplayName          string   `json:"display_name"`
	ActiveTenantID       any      `json:"active_tenant_id"`
	Assurance            string   `json:"assurance"`
	Permissions          []string `json:"permissions"`
	AuthorizationVersion int64    `json:"authorization_version"`
	SessionExpiresAt     string   `json:"session_expires_at"`
}

type sessionResponse struct {
	ID                string `json:"id"`
	Population        string `json:"population"`
	Assurance         string `json:"assurance"`
	Current           bool   `json:"current"`
	ClientLabel       any    `json:"client_label"`
	CreatedAt         string `json:"created_at"`
	LastSeenAt        string `json:"last_seen_at"`
	IdleExpiresAt     string `json:"idle_expires_at"`
	AbsoluteExpiresAt string `json:"absolute_expires_at"`
	RevokedAt         any    `json:"revoked_at"`
}

type revokeAllRequest struct {
	IncludeCurrent bool `json:"include_current"`
}

type stepUpRequest struct {
	Action string `json:"action"`
}

type stepUpResponse struct {
	ID                string `json:"id"`
	Action            string `json:"action"`
	RequiredAssurance string `json:"required_assurance"`
	AuthorizationURL  string `json:"authorization_url"`
	ExpiresAt         string `json:"expires_at"`
}

func (a *App) routeIdentity(response http.ResponseWriter, request *http.Request) {
	if !a.requireMethod(response, request) {
		return
	}
	if a.identity == nil {
		a.writeProblem(response, request, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "SERVICE_UNAVAILABLE", true)
		return
	}
	switch identityRoute(request.URL.Path) {
	case "/v1/auth/login":
		a.beginLogin(response, request)
	case "/v1/auth/callback":
		a.completeLogin(response, request)
	case "/v1/me":
		a.currentPrincipal(response, request)
	case "/v1/logout":
		a.logout(response, request)
	case "/v1/sessions":
		a.listSessions(response, request)
	case "/v1/sessions/{session_id}":
		a.revokeSession(response, request)
	case "/v1/sessions/revoke-all":
		a.revokeAllSessions(response, request)
	case "/v1/step-up/challenges":
		a.beginStepUp(response, request)
	default:
		a.writeProblem(response, request, http.StatusNotFound, "route-not-found", "Not found", "ROUTE_NOT_FOUND", false)
	}
}

func (a *App) beginLogin(response http.ResponseWriter, request *http.Request) {
	if requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	query, err := exactQuery(request.URL.RawQuery, "population", "return_to")
	if err != nil || len(query["population"]) != 1 || len(query) < 1 {
		a.malformed(response, request)
		return
	}
	population, err := identity.ParsePopulation(query.Get("population"))
	if err != nil {
		a.malformed(response, request)
		return
	}
	returnTo := ""
	if values, found := query["return_to"]; found {
		if len(values) != 1 {
			a.malformed(response, request)
			return
		}
		returnTo = values[0]
	}
	result, err := a.identity.BeginLogin(request.Context(), identity.BeginLoginRequest{
		Population: population, ReturnTo: returnTo, CookieValue: optionalSessionCookie(request),
	})
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	response.Header().Set("Location", result.AuthorizationURL)
	response.WriteHeader(http.StatusSeeOther)
}

func (a *App) completeLogin(response http.ResponseWriter, request *http.Request) {
	if requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	query, err := exactQuery(request.URL.RawQuery, "code", "state", "iss", "session_state")
	if err != nil || len(query["code"]) != 1 || len(query["state"]) != 1 ||
		(query.Has("iss") && !validOIDCIssuer(query.Get("iss"))) ||
		(query.Has("session_state") && !validOIDCSessionState(query.Get("session_state"))) {
		clearSessionCookie(response)
		a.malformed(response, request)
		return
	}
	correlationID, ok := requestCorrelationID(request)
	if !ok {
		clearSessionCookie(response)
		a.writeProblem(response, request, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "SERVICE_UNAVAILABLE", true)
		return
	}
	result, err := a.identity.CompleteLogin(request.Context(), identity.CompleteLoginRequest{
		State: query.Get("state"), Code: query.Get("code"),
		AuthorizationIssuer: query.Get("iss"),
		CorrelationID:       correlationID, ClientLabel: "browser",
	})
	if err != nil {
		clearSessionCookie(response)
		a.writeIdentityError(response, request, err)
		return
	}
	setSessionCookie(response, result.CookieValue, result.Session.AbsoluteExpiresAt)
	response.Header().Set("Location", a.webOrigin+result.ReturnTo)
	response.WriteHeader(http.StatusSeeOther)
}

func (a *App) currentPrincipal(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	cookie, err := sessionCookie(request)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrAuthenticationRequired)
		return
	}
	session, csrfToken, err := a.identity.Current(request.Context(), cookie)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	response.Header().Set(identity.CSRFHeaderName, csrfToken)
	writeJSON(response, http.StatusOK, principalFromSession(session))
}

func (a *App) logout(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	cookie, _ := sessionCookie(request)
	csrfToken, ok := singleHeader(request.Header, identity.CSRFHeaderName)
	if cookie != "" && !ok {
		a.writeIdentityError(response, request, identity.ErrCSRFValidationFailed)
		return
	}
	correlationID, ok := requestCorrelationID(request)
	if !ok {
		a.writeIdentityError(response, request, identity.ErrIdentityUnavailable)
		return
	}
	if err := a.identity.Logout(request.Context(), cookie, csrfToken, correlationID); err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	clearSessionCookie(response)
	response.WriteHeader(http.StatusNoContent)
}

func (a *App) listSessions(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	cookie, err := sessionCookie(request)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrAuthenticationRequired)
		return
	}
	sessions, err := a.identity.Sessions(request.Context(), cookie)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	data := make([]sessionResponse, 0, len(sessions))
	for _, session := range sessions {
		data = append(data, sessionFromSummary(session))
	}
	writeJSON(response, http.StatusOK, struct {
		Data []sessionResponse `json:"data"`
	}{Data: data})
}

func (a *App) revokeSession(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	cookie, err := sessionCookie(request)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrAuthenticationRequired)
		return
	}
	csrfToken, ok := singleHeader(request.Header, identity.CSRFHeaderName)
	if !ok {
		a.writeIdentityError(response, request, identity.ErrCSRFValidationFailed)
		return
	}
	targetText := strings.TrimPrefix(request.URL.Path, "/v1/sessions/")
	target, err := identifier.Parse(targetText)
	if err != nil || target.Prefix() != "ses" {
		a.writeIdentityError(response, request, identity.ErrSessionNotFound)
		return
	}
	correlationID, ok := requestCorrelationID(request)
	if !ok {
		a.writeIdentityError(response, request, identity.ErrIdentityUnavailable)
		return
	}
	result, err := a.identity.RevokeOne(
		request.Context(), cookie, csrfToken, target, correlationID,
	)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	if result.CurrentRevoked {
		clearSessionCookie(response)
	}
	response.WriteHeader(http.StatusNoContent)
}

func (a *App) revokeAllSessions(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" {
		a.malformed(response, request)
		return
	}
	var body revokeAllRequest
	if err := decodeStrictJSON(request, &body); err != nil {
		a.malformed(response, request)
		return
	}
	cookie, err := sessionCookie(request)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrAuthenticationRequired)
		return
	}
	csrfToken, ok := singleHeader(request.Header, identity.CSRFHeaderName)
	if !ok {
		a.writeIdentityError(response, request, identity.ErrCSRFValidationFailed)
		return
	}
	idempotencyKey, ok := singleHeader(request.Header, "Idempotency-Key")
	if !ok {
		a.malformed(response, request)
		return
	}
	correlationID, ok := requestCorrelationID(request)
	if !ok {
		a.writeIdentityError(response, request, identity.ErrIdentityUnavailable)
		return
	}
	result, err := a.identity.RevokeAll(
		request.Context(), cookie, csrfToken, body.IncludeCurrent,
		idempotencyKey, correlationID,
	)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	if result.CurrentRevoked {
		clearSessionCookie(response)
	}
	response.WriteHeader(http.StatusNoContent)
}

func (a *App) beginStepUp(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" {
		a.malformed(response, request)
		return
	}
	var body stepUpRequest
	if err := decodeStrictJSON(request, &body); err != nil {
		a.malformed(response, request)
		return
	}
	cookie, err := sessionCookie(request)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrAuthenticationRequired)
		return
	}
	csrfToken, ok := singleHeader(request.Header, identity.CSRFHeaderName)
	if !ok {
		a.writeIdentityError(response, request, identity.ErrCSRFValidationFailed)
		return
	}
	result, err := a.identity.BeginStepUp(request.Context(), identity.BeginStepUpRequest{
		CookieValue: cookie, CSRFToken: csrfToken, Action: body.Action,
	})
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	requiredAssurance := "stepped_up"
	if result.Population == identity.PopulationWorkforce {
		requiredAssurance = "phishing_resistant"
	}
	response.Header().Set("Location", result.AuthorizationURL)
	writeJSON(response, http.StatusCreated, stepUpResponse{
		ID: result.TransactionID.String(), Action: body.Action,
		RequiredAssurance: requiredAssurance, AuthorizationURL: result.AuthorizationURL,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
	})
}

func principalFromSession(session identity.Session) principalResponse {
	var tenant any
	if !session.TenantID.IsZero() {
		tenant = session.TenantID.String()
	}
	principalType := session.PrincipalType
	if principalType == "merchant" {
		principalType = "merchant_user"
	}
	return principalResponse{
		ID: session.PrincipalID.String(), Type: principalType,
		DisplayName: session.DisplayName, ActiveTenantID: tenant,
		Assurance:            session.Assurance.ContractValue(),
		Permissions:          append([]string(nil), session.Permissions...),
		AuthorizationVersion: session.AuthorizationVersion,
		SessionExpiresAt:     session.AbsoluteExpiresAt.Format(time.RFC3339),
	}
}

func sessionFromSummary(session identity.SessionSummary) sessionResponse {
	var clientLabel any
	if session.ClientLabel != "" {
		clientLabel = session.ClientLabel
	}
	var revokedAt any
	if !session.RevokedAt.IsZero() {
		revokedAt = session.RevokedAt.Format(time.RFC3339)
	}
	return sessionResponse{
		ID: session.SessionID.String(), Population: string(session.Population),
		Assurance: session.Assurance.ContractValue(), Current: session.Current,
		ClientLabel: clientLabel, CreatedAt: session.CreatedAt.Format(time.RFC3339),
		LastSeenAt:        session.LastSeenAt.Format(time.RFC3339),
		IdleExpiresAt:     session.IdleExpiresAt.Format(time.RFC3339),
		AbsoluteExpiresAt: session.AbsoluteExpiresAt.Format(time.RFC3339),
		RevokedAt:         revokedAt,
	}
}

func (a *App) writeIdentityError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, identity.ErrOIDCTransactionInvalid):
		a.writeProblem(response, request, http.StatusUnauthorized, "oidc-transaction-invalid", "Authentication required", "OIDC_TRANSACTION_INVALID", false)
	case errors.Is(err, identity.ErrSessionExpired):
		clearSessionCookie(response)
		a.writeProblem(response, request, http.StatusUnauthorized, "session-expired", "Authentication required", "SESSION_EXPIRED", false)
	case errors.Is(err, identity.ErrSessionRevoked):
		clearSessionCookie(response)
		a.writeProblem(response, request, http.StatusUnauthorized, "session-revoked", "Authentication required", "SESSION_REVOKED", false)
	case errors.Is(err, identity.ErrAuthenticationRequired):
		a.writeProblem(response, request, http.StatusUnauthorized, "authentication-required", "Authentication required", "AUTHENTICATION_REQUIRED", false)
	case errors.Is(err, identity.ErrCSRFValidationFailed):
		a.writeProblem(response, request, http.StatusForbidden, "csrf-validation-failed", "Action not authorized", "CSRF_VALIDATION_FAILED", false)
	case errors.Is(err, identity.ErrActionNotAuthorized):
		a.writeProblem(response, request, http.StatusForbidden, "action-not-authorized", "Action not authorized", "ACTION_NOT_AUTHORIZED", false)
	case errors.Is(err, identity.ErrSessionNotFound):
		a.writeProblem(response, request, http.StatusNotFound, "not-found-or-concealed", "Not found", "NOT_FOUND_OR_CONCEALED", false)
	case errors.Is(err, identity.ErrSessionConflict):
		a.writeProblem(response, request, http.StatusConflict, "conflict", "Conflict", "CONFLICT", false)
	case errors.Is(err, identity.ErrInputInvalid):
		a.malformed(response, request)
	default:
		a.writeProblem(response, request, http.StatusServiceUnavailable, "service-unavailable", "Service unavailable", "SERVICE_UNAVAILABLE", true)
	}
}

func (a *App) malformed(response http.ResponseWriter, request *http.Request) {
	a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
}

func exactQuery(raw string, allowed ...string) (url.Values, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, err
	}
	allowlist := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowlist[key] = struct{}{}
	}
	for key, entries := range values {
		if _, ok := allowlist[key]; !ok || len(entries) != 1 || entries[0] == "" {
			return nil, errors.New("query is malformed")
		}
	}
	return values, nil
}

func validOIDCIssuer(value string) bool {
	if len(value) < 12 || len(value) > 2048 {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https") &&
		parsed.Host != "" &&
		parsed.Path != "" &&
		parsed.User == nil &&
		parsed.RawQuery == "" &&
		parsed.Fragment == "" &&
		parsed.String() == value
}

func validOIDCSessionState(value string) bool {
	if len(value) == 0 || len(value) > 512 {
		return false
	}
	for _, character := range value {
		if (character >= 'A' && character <= 'Z') ||
			(character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_' || character == '~' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func sessionCookie(request *http.Request) (string, error) {
	var value string
	for _, cookie := range request.Cookies() {
		if cookie.Name != identity.SessionCookieName {
			continue
		}
		if value != "" || cookie.Value == "" {
			return "", errors.New("session cookie is ambiguous")
		}
		value = cookie.Value
	}
	if value == "" {
		return "", errors.New("session cookie is absent")
	}
	return value, nil
}

func optionalSessionCookie(request *http.Request) string {
	value, err := sessionCookie(request)
	if err != nil {
		return ""
	}
	return value
}

func singleHeader(headers http.Header, name string) (string, bool) {
	values := headers.Values(name)
	if len(values) != 1 {
		return "", false
	}
	value := strings.TrimSpace(values[0])
	return value, value != ""
}

func requestCorrelationID(request *http.Request) (identifier.ID, bool) {
	state, found := requestContextFrom(request.Context())
	if !found {
		return identifier.ID{}, false
	}
	return state.correlation.CorrelationID(), true
}

func decodeStrictJSON(request *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" || !requestHasBody(request) {
		return errors.New("JSON request is malformed")
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("JSON request has trailing content")
	}
	return nil
}

func setSessionCookie(response http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(response, &http.Cookie{
		Name: identity.SessionCookieName, Value: value, Path: "/",
		Expires:  expires.UTC(),
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(response http.ResponseWriter) {
	http.SetCookie(response, &http.Cookie{
		Name: identity.SessionCookieName, Value: "", Path: "/",
		Expires: time.Unix(1, 0).UTC(), MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
}

func validateWebOrigin(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Scheme+"://"+parsed.Host != value {
		return errors.New("web origin must be a canonical exact HTTP origin")
	}
	return nil
}
