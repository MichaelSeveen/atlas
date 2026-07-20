package architecture

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestArchitectureBoundaries(t *testing.T) {
	root := repositoryRoot(t)
	modulePath := repositoryModulePath(t, root)

	violations, err := Check(root, modulePath)
	if err != nil {
		t.Fatalf("check architecture boundaries: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("architecture boundary violations:\n%s", formatViolations(violations))
	}
}

func TestBoundaryCheckerRejectsForbiddenImport(t *testing.T) {
	root := t.TempDir()
	fixturePath := filepath.Join(root, "internal", "transfer", "service.go")
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	fixture := `package transfer

import _ "github.com/MichaelSeveen/atlas/internal/ledger/persistence"
`
	if err := os.WriteFile(fixturePath, []byte(fixture), 0o600); err != nil {
		t.Fatalf("write forbidden-import fixture: %v", err)
	}

	violations, err := Check(root, "github.com/MichaelSeveen/atlas")
	if err != nil {
		t.Fatalf("check fixture: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected one forbidden-import violation, got %d:\n%s", len(violations), formatViolations(violations))
	}
	if violations[0].Rule != "cross-context import must target the context root or application API" {
		t.Fatalf("unexpected rule: %s", violations[0].Rule)
	}
	if violations[0].ImportPath != "github.com/MichaelSeveen/atlas/internal/ledger/persistence" {
		t.Fatalf("unexpected import path: %s", violations[0].ImportPath)
	}
}

func TestBoundaryCheckerRejectsUnregisteredModule(t *testing.T) {
	root := t.TempDir()
	fixturePath := filepath.Join(root, "internal", "debugtools", "debug.go")
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(fixturePath, []byte("package debugtools\n"), 0o600); err != nil {
		t.Fatalf("write unregistered-module fixture: %v", err)
	}

	violations, err := Check(root, "github.com/MichaelSeveen/atlas")
	if err != nil {
		t.Fatalf("check fixture: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected one unregistered-module violation, got %d:\n%s", len(violations), formatViolations(violations))
	}
	if violations[0].Rule != "unregistered top-level module or command debugtools" {
		t.Fatalf("unexpected rule: %s", violations[0].Rule)
	}
}

func TestImportRules(t *testing.T) {
	tests := []struct {
		name         string
		sourceModule string
		sourceKind   ownership
		importPath   string
		wantRule     string
	}{
		{
			name:         "same context private package",
			sourceModule: "transfer",
			sourceKind:   ownershipDomain,
			importPath:   "github.com/MichaelSeveen/atlas/internal/transfer/persistence",
		},
		{
			name:         "cross context root API",
			sourceModule: "transfer",
			sourceKind:   ownershipDomain,
			importPath:   "github.com/MichaelSeveen/atlas/internal/ledger",
		},
		{
			name:         "cross context application API",
			sourceModule: "transfer",
			sourceKind:   ownershipDomain,
			importPath:   "github.com/MichaelSeveen/atlas/internal/ledger/application/posting",
		},
		{
			name:         "cross context persistence",
			sourceModule: "transfer",
			sourceKind:   ownershipDomain,
			importPath:   "github.com/MichaelSeveen/atlas/internal/ledger/persistence",
			wantRule:     "cross-context import must target the context root or application API",
		},
		{
			name:         "command private package",
			sourceModule: "api",
			sourceKind:   ownershipCommand,
			importPath:   "github.com/MichaelSeveen/atlas/internal/identity/store",
			wantRule:     "cross-context import must target the context root or application API",
		},
		{
			name:         "platform to domain",
			sourceModule: "platform",
			sourceKind:   ownershipFoundation,
			importPath:   "github.com/MichaelSeveen/atlas/internal/ledger",
			wantRule:     "foundation code cannot depend on a domain context",
		},
		{
			name:         "shared models dumping ground",
			sourceModule: "transfer",
			sourceKind:   ownershipDomain,
			importPath:   "github.com/MichaelSeveen/atlas/internal/models",
			wantRule:     "shared domain dumping-ground import is forbidden",
		},
		{
			name:         "domain to architecture tooling",
			sourceModule: "transfer",
			sourceKind:   ownershipDomain,
			importPath:   "github.com/MichaelSeveen/atlas/internal/architecture",
			wantRule:     "domain code cannot import architecture tooling",
		},
		{
			name:         "standard library",
			sourceModule: "transfer",
			sourceKind:   ownershipDomain,
			importPath:   "context",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			violation, found := checkImport(
				"internal/"+test.sourceModule+"/source.go",
				test.sourceModule,
				test.sourceKind,
				test.importPath,
				"github.com/MichaelSeveen/atlas",
			)
			if test.wantRule == "" {
				if found {
					t.Fatalf("expected import to be allowed, got %s", violation.String())
				}
				return
			}
			if !found {
				t.Fatalf("expected rule %q to reject the import", test.wantRule)
			}
			if violation.Rule != test.wantRule {
				t.Fatalf("expected rule %q, got %q", test.wantRule, violation.Rule)
			}
		})
	}
}

func TestRepositoryLayout(t *testing.T) {
	root := repositoryRoot(t)
	if modulePath := repositoryModulePath(t, root); modulePath != "github.com/MichaelSeveen/atlas" {
		t.Errorf("S01 module path must match the configured origin identity, got %q", modulePath)
	}
	required := []string{
		".git/HEAD",
		".go-version",
		"AGENTS.md",
		"README.md",
		"apps/web",
		"cmd/api/main.go",
		"cmd/simulator/main.go",
		"cmd/worker/main.go",
		"contracts/asyncapi",
		"contracts/openapi",
		"contracts/provider-fixtures",
		"deploy/local",
		"deploy/production-reference",
		"deploy/staging",
		"docs/adr",
		"docs/atlas-prd/README.md",
		"docs/engineering",
		"docs/evidence",
		"docs/runbooks",
		"docs/threat-models",
		"evidence/phase-00/architecture",
		"go.mod",
		"internal/architecture",
		"internal/platform/actor",
		"internal/platform/clock",
		"internal/platform/correlation",
		"internal/platform/domainerror",
		"internal/platform/identifier",
		"internal/platform/money",
		"internal/audit",
		"internal/customer",
		"internal/identity",
		"internal/ledger",
		"internal/operations",
		"internal/payment",
		"internal/platform",
		"internal/provider",
		"internal/reconciliation",
		"internal/reporting",
		"internal/risk",
		"internal/settlement",
		"internal/transfer",
		"internal/wallet",
		"migrations",
		"scripts/verify-s01.ps1",
		"scripts/verify-s02.ps1",
		"scripts/test-s02-mutation.ps1",
		"docs/engineering/PLATFORM_PRIMITIVES.md",
		"tests/architecture",
		"tests/chaos",
		"tests/contract",
		"tests/integration",
		"tests/performance",
		"tests/security",
	}

	for _, relative := range required {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			t.Errorf("required repository path %q: %v", relative, err)
		}
	}

	for _, forbidden := range []string{"internal/common", "internal/models", "internal/shared"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(forbidden))); !os.IsNotExist(err) {
			t.Errorf("forbidden shared domain path %q exists or cannot be checked: %v", forbidden, err)
		}
	}

}

func TestCanonicalPRDDuplicates(t *testing.T) {
	root := repositoryRoot(t)
	pairs := map[string]string{
		"00_PRODUCT_CHARTER.md":                    "docs/atlas-prd/00-master/00_PRODUCT_CHARTER.md",
		"00_SYSTEM_ARCHITECTURE.md":                "docs/atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md",
		"01_SECURITY_AND_TRUST_MODEL.md":           "docs/atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md",
		"02_DATA_ARCHITECTURE_AND_LEDGER_MODEL.md": "docs/atlas-prd/01-architecture/02_DATA_ARCHITECTURE_AND_LEDGER_MODEL.md",
		"ADVERSARIAL_TEST_CATALOG.md":              "docs/atlas-prd/04-testing/ADVERSARIAL_TEST_CATALOG.md",
		"CONTENT_CALENDAR.md":                      "docs/atlas-prd/05-content/CONTENT_CALENDAR.md",
		"PHASE-03_LEDGER_CORE.md":                  "docs/atlas-prd/02-phases/PHASE-03_LEDGER_CORE.md",
		"REQUIREMENTS_TRACEABILITY.csv":            "docs/atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv",
		"THREAT_REGISTER.csv":                      "docs/atlas-prd/06-governance/THREAT_REGISTER.csv",
		"asyncapi.yaml":                            "docs/atlas-prd/03-contracts/asyncapi.yaml",
		"openapi.yaml":                             "docs/atlas-prd/03-contracts/openapi.yaml",
	}

	for duplicate, canonical := range pairs {
		duplicatePath := filepath.Join(root, duplicate)
		if _, err := os.Stat(duplicatePath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Errorf("inspect duplicate %q: %v", duplicate, err)
			continue
		}

		duplicateHash := fileSHA256(t, duplicatePath)
		canonicalHash := fileSHA256(t, filepath.Join(root, filepath.FromSlash(canonical)))
		if duplicateHash != canonicalHash {
			t.Errorf("non-authoritative root copy %q drifted from %q", duplicate, canonical)
		}
	}
}

func TestCanonicalPRDManifest(t *testing.T) {
	root := repositoryRoot(t)
	manifestPath := filepath.Join(root, "docs", "atlas-prd", "MANIFEST.sha256")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read PRD manifest: %v", err)
	}

	for lineNumber, line := range strings.Split(strings.TrimSpace(string(manifest)), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 || len(fields[0]) != sha256.Size*2 || !strings.HasPrefix(fields[1], "./") {
			t.Fatalf("malformed PRD manifest line %d: %q", lineNumber+1, line)
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			t.Fatalf("invalid hash on PRD manifest line %d: %v", lineNumber+1, err)
		}
		actual := fileSHA256(t, filepath.Join(root, "docs", "atlas-prd", filepath.FromSlash(strings.TrimPrefix(fields[1], "./"))))
		if !strings.EqualFold(actual, fields[0]) {
			t.Errorf("PRD manifest mismatch for %s: expected %s, observed %s", fields[1], fields[0], actual)
		}
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
}

func repositoryModulePath(t *testing.T, root string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	t.Fatal("go.mod has no module directive")
	return ""
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}

func formatViolations(violations []Violation) string {
	formatted := make([]string, 0, len(violations))
	for _, violation := range violations {
		formatted = append(formatted, violation.String())
	}
	return strings.Join(formatted, "\n")
}
