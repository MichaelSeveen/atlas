package architecture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
		"apps/web/Containerfile",
		"apps/web/bun.lock",
		"apps/web/package.json",
		"cmd/api/main.go",
		"cmd/api/internal/server/server.go",
		"cmd/dbctl/main.go",
		"cmd/envctl/main.go",
		"cmd/simulator/main.go",
		"cmd/worker/main.go",
		"contracts/asyncapi",
		"contracts/openapi",
		"contracts/provider-fixtures",
		"db/README.md",
		"db/migrations/MANIFEST.sha256",
		"db/roles/bootstrap.sql",
		"db/recovery/restore-entrypoint.sh",
		"deploy/local",
		"deploy/local/compose.yaml",
		"deploy/environments/local.json",
		"deploy/environments/test.json",
		"deploy/environments/staging.json",
		"deploy/environments/production-reference.json",
		"deploy/seeds/foundation.json",
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
		"internal/platform/environment",
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
		"docs/engineering/HTTP_FOUNDATION.md",
		"docs/runbooks/DATABASE_UNAVAILABLE.md",
		"scripts/test-s03-contract-canary.ps1",
		"scripts/verify-s03.ps1",
		"scripts/s04.ps1",
		"scripts/test-s04-config-canary.ps1",
		"scripts/test-s04-live.ps1",
		"scripts/test-s04-reset-canary.ps1",
		"scripts/test-s04-seed-canary.ps1",
		"scripts/verify-s04.ps1",
		"docs/engineering/LOCAL_ENVIRONMENT.md",
		"docs/atlas-prd/06-governance/adrs/0008-local-reference-platform.md",
		"docs/atlas-prd/06-governance/adrs/0009-react-bun-route-shells.md",
		"evidence/phase-00/environment/S04-environment-report.md",
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

func TestFrontendToolchainPolicy(t *testing.T) {
	root := repositoryRoot(t)
	manifestPath := filepath.Join(root, "apps", "web", "package.json")
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		PackageManager  string            `json:"packageManager"`
		Scripts         map[string]string `json:"scripts"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.PackageManager != "bun@1.3.0" {
		t.Errorf("frontend package manager must be exactly bun@1.3.0, got %q", manifest.PackageManager)
	}
	for dependency, version := range map[string]string{"react": "19.2.7", "react-dom": "19.2.7"} {
		if manifest.Dependencies[dependency] != version {
			t.Errorf("frontend dependency %s must be exactly %s", dependency, version)
		}
	}
	for dependency, version := range map[string]string{
		"@types/bun":       "1.3.0",
		"@types/react":     "19.2.17",
		"@types/react-dom": "19.2.3",
		"typescript":       "5.9.3",
	} {
		if manifest.DevDependencies[dependency] != version {
			t.Errorf("frontend development dependency %s must be exactly %s", dependency, version)
		}
	}
	if manifest.Scripts["typecheck"] != "tsc --noEmit" {
		t.Errorf("frontend typecheck must be exactly %q, got %q", "tsc --noEmit", manifest.Scripts["typecheck"])
	}
	for name, command := range manifest.Scripts {
		for _, forbidden := range []string{"node ", "npm ", "pnpm ", "yarn "} {
			if strings.Contains(strings.ToLower(command), forbidden) {
				t.Errorf("frontend script %s uses forbidden tool %q", name, forbidden)
			}
		}
	}
	for _, relative := range []string{
		"apps/web/package-lock.json", "apps/web/pnpm-lock.yaml", "apps/web/yarn.lock",
		"apps/web/.npmrc", "apps/web/.nvmrc",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("forbidden competing frontend artifact %q exists or cannot be checked: %v", relative, err)
		}
	}

	tsconfigPath := filepath.Join(root, "apps", "web", "tsconfig.json")
	tsconfigContent, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var tsconfig struct {
		CompilerOptions struct {
			Types []string `json:"types"`
		} `json:"compilerOptions"`
	}
	if err := json.Unmarshal(tsconfigContent, &tsconfig); err != nil {
		t.Fatal(err)
	}
	hasBunTypes := false
	for _, typeLibrary := range tsconfig.CompilerOptions.Types {
		switch strings.ToLower(typeLibrary) {
		case "bun":
			hasBunTypes = true
		case "node":
			t.Error("frontend TypeScript configuration must not load Node.js types")
		}
	}
	if !hasBunTypes {
		t.Error("frontend TypeScript configuration must explicitly load Bun types")
	}
}

func TestFrontendUsesFunctionComponentsOnly(t *testing.T) {
	root := repositoryRoot(t)
	violations, err := frontendClassDeclarations(root)
	if err != nil {
		t.Fatalf("scan frontend source: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("frontend source must use function components only; class declarations found in: %s", strings.Join(violations, ", "))
	}

	fixtureRoot := t.TempDir()
	fixturePath := filepath.Join(fixtureRoot, "apps", "web", "src", "unsafe.tsx")
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("create frontend policy fixture: %v", err)
	}
	fixture := `import React from "react";

class UnsafeBoundary extends React.Component {}
`
	if err := os.WriteFile(fixturePath, []byte(fixture), 0o600); err != nil {
		t.Fatalf("write frontend policy fixture: %v", err)
	}
	violations, err = frontendClassDeclarations(fixtureRoot)
	if err != nil {
		t.Fatalf("scan frontend policy fixture: %v", err)
	}
	if len(violations) != 1 || violations[0] != "apps/web/src/unsafe.tsx" {
		t.Fatalf("expected seeded class component to be rejected, got %v", violations)
	}
}

func frontendClassDeclarations(root string) ([]string, error) {
	classDeclaration := regexp.MustCompile(`\bclass\s+[A-Za-z_$][A-Za-z0-9_$]*`)
	sourceRoot := filepath.Join(root, "apps", "web", "src")
	violations := make([]string, 0)
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".ts" && extension != ".tsx" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !classDeclaration.Match(content) {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		violations = append(violations, filepath.ToSlash(relative))
		return nil
	})
	return violations, err
}

func TestNoRootPRDDuplicates(t *testing.T) {
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
		canonicalPath := filepath.Join(root, filepath.FromSlash(canonical))
		if _, err := os.Stat(canonicalPath); err != nil {
			t.Errorf("canonical PRD artifact %q: %v", canonical, err)
		}

		duplicatePath := filepath.Join(root, duplicate)
		if _, err := os.Stat(duplicatePath); err == nil {
			t.Errorf("non-authoritative root PRD copy %q reappeared; use %q", duplicate, canonical)
		} else if !os.IsNotExist(err) {
			t.Errorf("inspect forbidden root PRD copy %q: %v", duplicate, err)
		}
	}
}

func TestCanonicalPRDManifest(t *testing.T) {
	root := repositoryRoot(t)
	prdRoot := filepath.Join(root, "docs", "atlas-prd")
	manifestPath := filepath.Join(prdRoot, "MANIFEST.sha256")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read PRD manifest: %v", err)
	}

	seen := make(map[string]struct{})
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
		seen[strings.TrimPrefix(filepath.ToSlash(fields[1]), "./")] = struct{}{}
	}
	if err := filepath.WalkDir(prdRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || path == manifestPath {
			return nil
		}
		relative, err := filepath.Rel(prdRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, found := seen[relative]; !found {
			t.Errorf("canonical PRD file %q is absent from MANIFEST.sha256", relative)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
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
