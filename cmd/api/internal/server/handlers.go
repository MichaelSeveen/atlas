package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

var foundationRoutes = []string{"/health/live", "/health/ready", "/version"}
var identityRoutes = []string{
	"/v1/me",
	"/v1/auth/login",
	"/v1/auth/callback",
	"/v1/logout",
	"/v1/sessions",
	"/v1/sessions/{session_id}",
	"/v1/sessions/revoke-all",
	"/v1/step-up/challenges",
}

type statusResponse struct {
	Status string `json:"status"`
}

type versionResponse struct {
	SourceRevision  string `json:"source_revision"`
	ContractVersion string `json:"contract_version"`
	BuildTime       string `json:"build_time"`
}

type problemResponse struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	Status        int    `json:"status"`
	Code          string `json:"code"`
	RequestID     string `json:"request_id"`
	CorrelationID string `json:"correlation_id"`
	Retryable     bool   `json:"retryable"`
}

func (a *App) route(response http.ResponseWriter, request *http.Request) {
	operational := false
	for _, route := range foundationRoutes {
		if request.URL.Path == route {
			operational = true
			break
		}
	}
	if !operational {
		if identityRoute(request.URL.Path) == "" {
			a.writeProblem(response, request, http.StatusNotFound, "route-not-found", "Not found", "ROUTE_NOT_FOUND", false)
			return
		}
		a.routeIdentity(response, request)
		return
	}
	if request.URL.RawQuery != "" {
		a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
		return
	}
	if requestHasBody(request) {
		a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
		return
	}
	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		a.writeProblem(response, request, http.StatusMethodNotAllowed, "method-not-allowed", "Method not allowed", "METHOD_NOT_ALLOWED", false)
		return
	}

	switch request.URL.Path {
	case "/health/live":
		writeJSON(response, http.StatusOK, statusResponse{Status: "alive"})
	case "/health/ready":
		if a.readinessState(request.Context()).Ready() {
			writeJSON(response, http.StatusOK, statusResponse{Status: "ready"})
			return
		}
		a.writeProblem(response, request, http.StatusServiceUnavailable, "dependency-degraded", "Service unavailable", "DEPENDENCY_DEGRADED", true)
	case "/version":
		writeJSON(response, http.StatusOK, versionResponse{
			SourceRevision:  a.build.SourceRevision,
			ContractVersion: a.build.ContractVersion,
			BuildTime:       a.build.BuildTime.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

func requestHasBody(request *http.Request) bool {
	return request.Body != nil && request.Body != http.NoBody && request.ContentLength != 0
}

func identityRoute(path string) string {
	for _, route := range identityRoutes {
		if path == route {
			return route
		}
	}
	const prefix = "/v1/sessions/"
	if strings.HasPrefix(path, prefix) {
		identifier := strings.TrimPrefix(path, prefix)
		if identifier != "" && !strings.Contains(identifier, "/") && identifier != "revoke-all" {
			return "/v1/sessions/{session_id}"
		}
	}
	return ""
}

func allowedMethods(path string) []string {
	switch identityRoute(path) {
	case "/v1/me", "/v1/auth/login", "/v1/auth/callback", "/v1/sessions":
		return []string{http.MethodGet}
	case "/v1/logout", "/v1/sessions/revoke-all", "/v1/step-up/challenges":
		return []string{http.MethodPost}
	case "/v1/sessions/{session_id}":
		return []string{http.MethodDelete}
	case "":
		for _, route := range foundationRoutes {
			if path == route {
				return []string{http.MethodGet}
			}
		}
	}
	return nil
}

func methodAllowed(method string, allowed []string) bool {
	for _, candidate := range allowed {
		if method == candidate {
			return true
		}
	}
	return false
}

func (a *App) requireMethod(response http.ResponseWriter, request *http.Request) bool {
	allowed := allowedMethods(request.URL.Path)
	if methodAllowed(request.Method, allowed) {
		return true
	}
	if len(allowed) == 0 {
		a.writeProblem(response, request, http.StatusNotFound, "route-not-found", "Not found", "ROUTE_NOT_FOUND", false)
		return false
	}
	response.Header().Set("Allow", strings.Join(allowed, ", "))
	a.writeProblem(response, request, http.StatusMethodNotAllowed, "method-not-allowed", "Method not allowed", "METHOD_NOT_ALLOWED", false)
	return false
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}

func (a *App) writeProblem(response http.ResponseWriter, request *http.Request, status int, slug, title, code string, retryable bool) {
	requestID := ""
	correlationID := ""
	if state, found := requestContextFrom(request.Context()); found {
		requestID = state.correlation.RequestID().String()
		correlationID = state.correlation.CorrelationID().String()
	}
	response.Header().Set("Content-Type", "application/problem+json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(problemResponse{
		Type:          "https://atlas.example/problems/" + slug,
		Title:         title,
		Status:        status,
		Code:          code,
		RequestID:     requestID,
		CorrelationID: correlationID,
		Retryable:     retryable,
	})
}
