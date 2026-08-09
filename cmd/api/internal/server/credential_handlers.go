package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type apiCredentialResponse struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	SecretHint     string   `json:"secret_hint"`
	Scopes         []string `json:"scopes"`
	Status         string   `json:"status"`
	ExpiresAt      string   `json:"expires_at"`
	OverlapEndsAt  any      `json:"overlap_ends_at"`
	LastUsedAt     any      `json:"last_used_at"`
	CreatedAt      string   `json:"created_at"`
	RevokedAt      any      `json:"revoked_at"`
}

type apiCredentialListResponse struct {
	Data []apiCredentialResponse `json:"data"`
}

type createAPICredentialRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays int      `json:"expires_in_days"`
	Purpose       string   `json:"purpose"`
}

type apiCredentialCreatedResponse struct {
	Credential      apiCredentialResponse `json:"credential"`
	Secret          any                   `json:"secret"`
	SecretDisclosed bool                  `json:"secret_disclosed"`
}

func (a *App) routeCredential(response http.ResponseWriter, request *http.Request) {
	if !a.requireMethod(response, request) {
		return
	}
	if a.identity == nil {
		a.writeProblem(response, request, http.StatusServiceUnavailable,
			"service-unavailable", "Service unavailable", "SERVICE_UNAVAILABLE", true)
		return
	}
	switch credentialRoute(request.URL.Path) {
	case "/v1/api-credentials":
		if request.Method == http.MethodPost {
			a.createAPICredential(response, request)
			return
		}
		a.listAPICredentials(response, request)
	case "/v1/api-credentials/{credential_id}/rotate":
		a.rotateAPICredential(response, request)
	case "/v1/api-credentials/{credential_id}":
		a.revokeAPICredential(response, request)
	default:
		a.writeProblem(response, request, http.StatusNotFound,
			"route-not-found", "Not found", "ROUTE_NOT_FOUND", false)
	}
}

func (a *App) listAPICredentials(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	cookie, err := credentialManagementCookie(request)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrAuthenticationRequired)
		return
	}
	credentials, err := a.identity.ListAPICredentials(request.Context(), cookie)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	data := make([]apiCredentialResponse, 0, len(credentials))
	for _, credential := range credentials {
		data = append(data, apiCredentialFromDomain(credential))
	}
	writeJSON(response, http.StatusOK, apiCredentialListResponse{Data: data})
}

func (a *App) createAPICredential(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" {
		a.malformed(response, request)
		return
	}
	var body createAPICredentialRequest
	if err := decodeStrictJSON(request, &body); err != nil {
		a.malformed(response, request)
		return
	}
	cookie, csrfToken, idempotencyKey, correlationID, ok := credentialMutationHeaders(request)
	if !ok {
		a.malformed(response, request)
		return
	}
	result, err := a.identity.CreateAPICredential(request.Context(), identity.CreateAPICredentialRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		Name: body.Name, Scopes: body.Scopes, ExpiresInDays: body.ExpiresInDays,
		Purpose: body.Purpose, CorrelationID: correlationID,
	})
	writeCredentialDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	response.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replay))
	response.Header().Set("Location", "/v1/api-credentials/"+result.Credential.CredentialID.String())
	writeJSON(response, http.StatusCreated, apiCredentialCreatedFromDomain(result))
}

func (a *App) rotateAPICredential(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	credentialID, err := credentialIDFromPath(request.URL.Path, true)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrCredentialNotFound)
		return
	}
	cookie, csrfToken, idempotencyKey, correlationID, ok := credentialMutationHeaders(request)
	if !ok {
		a.malformed(response, request)
		return
	}
	result, err := a.identity.RotateAPICredential(request.Context(), identity.RotateAPICredentialRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		CredentialID: credentialID, CorrelationID: correlationID,
	})
	writeCredentialDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	response.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replay))
	writeJSON(response, http.StatusOK, apiCredentialCreatedFromDomain(result))
}

func (a *App) revokeAPICredential(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	credentialID, err := credentialIDFromPath(request.URL.Path, false)
	if err != nil {
		a.writeIdentityError(response, request, identity.ErrCredentialNotFound)
		return
	}
	cookie, csrfToken, idempotencyKey, correlationID, ok := credentialMutationHeaders(request)
	if !ok {
		a.malformed(response, request)
		return
	}
	result, err := a.identity.RevokeAPICredential(request.Context(), identity.RevokeAPICredentialRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		CredentialID: credentialID, CorrelationID: correlationID,
	})
	writeCredentialDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeIdentityError(response, request, err)
		return
	}
	response.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replay))
	response.WriteHeader(http.StatusNoContent)
}

func credentialManagementCookie(request *http.Request) (string, error) {
	if len(request.Header.Values("Authorization")) != 0 {
		return "", errors.New("ambiguous authentication")
	}
	return sessionCookie(request)
}

func credentialMutationHeaders(request *http.Request) (
	string,
	string,
	string,
	identifier.ID,
	bool,
) {
	cookie, err := credentialManagementCookie(request)
	if err != nil {
		return "", "", "", identifier.ID{}, false
	}
	csrfToken, csrfOK := singleHeader(request.Header, identity.CSRFHeaderName)
	idempotencyKey, idempotencyOK := singleHeader(request.Header, "Idempotency-Key")
	correlationID, correlationOK := requestCorrelationID(request)
	return cookie, csrfToken, idempotencyKey, correlationID,
		csrfOK && idempotencyOK && correlationOK
}

func credentialIDFromPath(path string, rotation bool) (identifier.ID, error) {
	value := strings.TrimPrefix(path, "/v1/api-credentials/")
	if rotation {
		value = strings.TrimSuffix(value, "/rotate")
	}
	credentialID, err := identifier.Parse(value)
	if err != nil || credentialID.Prefix() != "key" {
		return identifier.ID{}, errors.New("credential identifier is invalid")
	}
	return credentialID, nil
}

func apiCredentialCreatedFromDomain(result identity.APICredentialCreated) apiCredentialCreatedResponse {
	var secret any
	if result.SecretDisclosed {
		secret = result.Secret
	}
	return apiCredentialCreatedResponse{
		Credential: apiCredentialFromDomain(result.Credential),
		Secret:     secret, SecretDisclosed: result.SecretDisclosed,
	}
}

func apiCredentialFromDomain(credential identity.APICredential) apiCredentialResponse {
	return apiCredentialResponse{
		ID: credential.CredentialID.String(), OrganizationID: credential.OrganizationID.String(),
		Name: credential.Name, SecretHint: credential.SecretHint,
		Scopes: append([]string(nil), credential.Scopes...), Status: string(credential.Status),
		ExpiresAt:     credential.ExpiresAt.Format(time.RFC3339),
		OverlapEndsAt: nullableCredentialTime(credential.OverlapEndsAt),
		LastUsedAt:    nullableCredentialTime(credential.LastUsedAt),
		CreatedAt:     credential.CreatedAt.Format(time.RFC3339),
		RevokedAt:     nullableCredentialTime(credential.RevokedAt),
	}
}

func nullableCredentialTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.Format(time.RFC3339)
}

func writeCredentialDecisionHeader(response http.ResponseWriter, decisionID identifier.ID) {
	if !decisionID.IsZero() {
		response.Header().Set("X-Authorization-Decision-Id", decisionID.String())
	}
}
