package contractcompat

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCanonicalContractsLintAndCompareToThemselves(t *testing.T) {
	for _, name := range []string{"openapi.yaml", "asyncapi.yaml"} {
		path := filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", name)
		if err := Lint(path); err != nil {
			t.Fatalf("lint %s: %v", name, err)
		}
		if err := Compare(path, path); err != nil {
			t.Fatalf("compare %s: %v", name, err)
		}
	}
}

func TestBreakingContractCanariesFailClosed(t *testing.T) {
	cases := []struct {
		name        string
		file        string
		marker      string
		replacement string
	}{
		{name: "removed OpenAPI path", file: "openapi.yaml", marker: "  /health/live:\n", replacement: "  removedByS07Canary:\n"},
		{name: "removed AsyncAPI channel", file: "asyncapi.yaml", marker: "  ledgerJournalPosted:\n", replacement: "  removedByS07Canary:\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			original := filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", tc.file)
			content, err := os.ReadFile(original)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), tc.marker) {
				t.Fatalf("canary marker %q is absent", tc.marker)
			}
			candidate := filepath.Join(t.TempDir(), tc.file)
			mutated := strings.Replace(string(content), tc.marker, tc.replacement, 1)
			if err := os.WriteFile(candidate, []byte(mutated), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Compare(original, candidate); err == nil || !strings.Contains(err.Error(), "breaking removals") {
				t.Fatalf("expected breaking-removal rejection, got %v", err)
			}
		})
	}
}

func TestUnresolvedReferenceFailsClosed(t *testing.T) {
	original := filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", "openapi.yaml")
	content, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	candidate := filepath.Join(t.TempDir(), "openapi.yaml")
	mutated := strings.Replace(string(content), "#/components/schemas/Liveness", "#/components/schemas/S07Missing", 1)
	if err := os.WriteFile(candidate, []byte(mutated), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Lint(candidate); err == nil || !strings.Contains(err.Error(), "does not resolve") {
		t.Fatalf("expected unresolved-reference rejection, got %v", err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
}
