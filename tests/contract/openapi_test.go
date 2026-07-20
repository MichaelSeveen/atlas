package contract_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const contractVersion = "2026-07-20"

func TestOpenAPIFoundationOperations(t *testing.T) {
	source := readOpenAPI(t)
	for _, fragment := range []string{
		"openapi: 3.1.1",
		"version: " + contractVersion,
		"  /health/live:",
		"operationId: getLiveness",
		"  /health/ready:",
		"operationId: getReadiness",
		"  /version:",
		"operationId: getVersion",
		"source_revision:",
		"contract_version:",
		"build_time:",
	} {
		if !strings.Contains(source, fragment) {
			t.Errorf("canonical OpenAPI is missing %q", fragment)
		}
	}

	for _, operation := range []struct {
		path        string
		operationID string
		responses   []string
	}{
		{path: "/health/live", operationID: "getLiveness", responses: []string{"'200'", "'400'", "'413'", "'415'", "'500'"}},
		{path: "/health/ready", operationID: "getReadiness", responses: []string{"'200'", "'400'", "'413'", "'415'", "'500'", "'503'"}},
		{path: "/version", operationID: "getVersion", responses: []string{"'200'", "'400'", "'413'", "'415'", "'500'"}},
	} {
		block := pathBlock(t, source, operation.path)
		for _, fragment := range []string{
			"    get:",
			"      operationId: " + operation.operationID,
			"      security: []",
			"#/components/parameters/RequestId",
			"#/components/parameters/CorrelationId",
			"#/components/parameters/Traceparent",
			"X-Request-Id:",
			"X-Correlation-Id:",
			"traceparent:",
			"Cache-Control:",
		} {
			if !strings.Contains(block, fragment) {
				t.Errorf("%s operation is missing %q", operation.path, fragment)
			}
		}
		for _, status := range operation.responses {
			if !strings.Contains(block, "        "+status+":") {
				t.Errorf("%s operation is missing response %s", operation.path, status)
			}
		}
	}
}

func TestOpenAPIFoundationSchemasAreClosedAndSafe(t *testing.T) {
	source := readOpenAPI(t)
	for _, schema := range []string{"Liveness", "Readiness", "VersionInfo"} {
		block := componentBlock(t, source, schema)
		if !strings.Contains(block, "additionalProperties: false") {
			t.Errorf("%s schema permits undeclared fields", schema)
		}
	}
	for _, fragment := range []string{
		"const: alive",
		"const: ready",
		"pattern: '^(development|[0-9a-f]{7,64})$'",
		"pattern: '^\\d{4}-\\d{2}-\\d{2}$'",
		"description: Returns only source revision, canonical contract version, and UTC build time.",
		"description: Fails closed when required dependencies or migration state cannot be verified; never returns topology or migration detail.",
	} {
		if !strings.Contains(source, fragment) {
			t.Errorf("foundation contract is missing safety rule %q", fragment)
		}
	}
}

func TestContractOpaqueIDExamplesMatchNormativePattern(t *testing.T) {
	for _, path := range []string{openAPIPath(t), filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", "asyncapi.yaml")} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		candidates := regexp.MustCompile(`\b[a-z]{2,8}_[0-9A-Z]{20,64}\b`).FindAllString(string(content), -1)
		for _, candidate := range candidates {
			if _, err := identifier.Parse(candidate); err != nil {
				t.Errorf("%s contains opaque-ID example %q that violates the normative pattern", filepath.Base(path), candidate)
			}
		}
	}
}

func TestCanonicalContractHasNoMutableDuplicate(t *testing.T) {
	root := repositoryRoot(t)
	for _, relative := range []string{
		"openapi.yaml",
		"asyncapi.yaml",
		"contracts/openapi/openapi.yaml",
		"contracts/asyncapi/asyncapi.yaml",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err == nil {
			t.Errorf("non-canonical mutable contract exists at %s", relative)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
}

func TestFoundationErrorCodesAreCatalogued(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", "ERROR_CATALOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, code := range []string{
		"REQUEST_MALFORMED",
		"REQUEST_TOO_LARGE",
		"UNSUPPORTED_MEDIA_TYPE",
		"ROUTE_NOT_FOUND",
		"METHOD_NOT_ALLOWED",
		"CORS_ORIGIN_DENIED",
		"INTERNAL_ERROR",
		"DEPENDENCY_DEGRADED",
	} {
		if !strings.Contains(source, "`"+code+"`") {
			t.Errorf("error code %s is not catalogued", code)
		}
	}
}

func pathBlock(t *testing.T, source, path string) string {
	t.Helper()
	marker := "\n  " + path + ":\n"
	start := strings.Index(source, marker)
	if start < 0 {
		t.Fatalf("path %s not found", path)
	}
	remaining := source[start+len(marker):]
	end := strings.Index(remaining, "\n  /")
	if end < 0 {
		end = strings.Index(remaining, "\ncomponents:")
	}
	if end < 0 {
		t.Fatalf("end of path %s not found", path)
	}
	return remaining[:end]
}

func componentBlock(t *testing.T, source, name string) string {
	t.Helper()
	marker := "\n    " + name + ":\n"
	start := strings.Index(source, marker)
	if start < 0 {
		t.Fatalf("component %s not found", name)
	}
	remaining := source[start+len(marker):]
	lines := strings.Split(remaining, "\n")
	var block []string
	for _, line := range lines {
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") {
			break
		}
		block = append(block, line)
	}
	return strings.Join(block, "\n")
}

func readOpenAPI(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(openAPIPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(string(content), '\t') {
		t.Fatal("canonical OpenAPI contains tab indentation")
	}
	return string(content)
}

func openAPIPath(t *testing.T) string {
	t.Helper()
	if override := os.Getenv("ATLAS_OPENAPI_PATH"); override != "" {
		return override
	}
	return filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", "openapi.yaml")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
}
