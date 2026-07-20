package server

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
}

type corsPolicy struct {
	origins          map[string]struct{}
	allowCredentials bool
}

func newCORSPolicy(config CORSConfig) (corsPolicy, error) {
	policy := corsPolicy{
		origins:          make(map[string]struct{}, len(config.AllowedOrigins)),
		allowCredentials: config.AllowCredentials,
	}
	for _, origin := range config.AllowedOrigins {
		if origin == "*" {
			return corsPolicy{}, errors.New("wildcard CORS origin is forbidden")
		}
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") ||
			parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
			(parsed.Path != "" && parsed.Path != "/") {
			return corsPolicy{}, errors.New("CORS origin must be an exact HTTP origin")
		}
		canonical := parsed.Scheme + "://" + parsed.Host
		if canonical != origin {
			return corsPolicy{}, errors.New("CORS origin must use canonical origin form")
		}
		if _, duplicate := policy.origins[origin]; duplicate {
			return corsPolicy{}, errors.New("duplicate CORS origin")
		}
		policy.origins[origin] = struct{}{}
	}
	return policy, nil
}

func (a *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		originValues := request.Header.Values("Origin")
		if len(originValues) == 0 {
			next.ServeHTTP(response, request)
			return
		}
		origin := ""
		if len(originValues) == 1 {
			origin = originValues[0]
		}
		addVary(response.Header(), "Origin")
		_, allowed := a.cors.origins[origin]
		if !allowed {
			if request.Method == http.MethodOptions {
				a.writeProblem(response, request, http.StatusForbidden, "cors-origin-denied", "Origin denied", "CORS_ORIGIN_DENIED", false)
				return
			}
			next.ServeHTTP(response, request)
			return
		}

		response.Header().Set("Access-Control-Allow-Origin", origin)
		if a.cors.allowCredentials {
			response.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if request.Method != http.MethodOptions {
			next.ServeHTTP(response, request)
			return
		}
		requestedMethods := request.Header.Values("Access-Control-Request-Method")
		if telemetryRoute(request.URL.Path) == "unmatched" || len(requestedMethods) != 1 || requestedMethods[0] != http.MethodGet {
			a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
			return
		}
		requestedHeaderValues := request.Header.Values("Access-Control-Request-Headers")
		if len(requestedHeaderValues) > 1 {
			a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
			return
		}
		requestedHeaders := ""
		if len(requestedHeaderValues) == 1 {
			requestedHeaders = requestedHeaderValues[0]
		}
		requested, ok := allowedCORSHeaders(requestedHeaders)
		if !ok {
			a.writeProblem(response, request, http.StatusBadRequest, "request-malformed", "Malformed request", "REQUEST_MALFORMED", false)
			return
		}
		response.Header().Set("Access-Control-Allow-Methods", http.MethodGet)
		if len(requested) > 0 {
			response.Header().Set("Access-Control-Allow-Headers", strings.Join(requested, ", "))
		}
		response.WriteHeader(http.StatusNoContent)
	})
}

func allowedCORSHeaders(value string) ([]string, bool) {
	if strings.TrimSpace(value) == "" {
		return nil, true
	}
	allowlist := map[string]string{
		"traceparent":      "traceparent",
		"x-correlation-id": "X-Correlation-Id",
		"x-request-id":     "X-Request-Id",
	}
	seen := map[string]struct{}{}
	var headers []string
	for _, item := range strings.Split(value, ",") {
		normalized := strings.ToLower(strings.TrimSpace(item))
		canonical, allowed := allowlist[normalized]
		if !allowed {
			return nil, false
		}
		if _, duplicate := seen[normalized]; duplicate {
			continue
		}
		seen[normalized] = struct{}{}
		headers = append(headers, canonical)
	}
	sort.Strings(headers)
	return headers, true
}

func addVary(headers http.Header, value string) {
	for _, existing := range headers.Values("Vary") {
		for _, item := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(item), value) {
				return
			}
		}
	}
	headers.Add("Vary", value)
}
