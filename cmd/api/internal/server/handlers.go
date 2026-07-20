package server

import (
	"encoding/json"
	"net/http"
)

var foundationRoutes = []string{"/health/live", "/health/ready", "/version"}

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
	if request.URL.RawQuery != "" {
		a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
		return
	}

	known := false
	for _, route := range foundationRoutes {
		if request.URL.Path == route {
			known = true
			break
		}
	}
	if !known {
		a.writeProblem(response, request, http.StatusNotFound, "route-not-found", "Not found", "ROUTE_NOT_FOUND", false)
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
