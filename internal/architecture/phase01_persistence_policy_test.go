package architecture

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var phase01ProductTablePattern = regexp.MustCompile(`(?i)CREATE TABLE (atlas_(?:identity|audit))\.([a-z][a-z0-9_]*)`)

func TestPhase01PersistenceScopeAndAppendOnlyPolicies(t *testing.T) {
	root := repositoryRoot(t)
	identitySQL := readPhase01PolicyFile(t, root, "db/migrations/000003_phase_01_identity_persistence.sql")
	auditSQL := readPhase01PolicyFile(t, root, "db/migrations/000004_phase_01_audit_persistence.sql")
	sessionSQL := readPhase01PolicyFile(t, root, "db/migrations/000005_phase_01_oidc_sessions.sql")
	sessionGrantSQL := readPhase01PolicyFile(t, root, "db/migrations/000006_phase_01_session_authority_lock_grant.sql")
	adminRevocationSQL := readPhase01PolicyFile(t, root, "db/migrations/000008_phase_01_admin_session_revocation.sql")
	allSQL := identitySQL + "\n" + auditSQL + "\n" + sessionSQL + "\n" +
		sessionGrantSQL + "\n" + adminRevocationSQL

	tables := make([]string, 0)
	for _, match := range phase01ProductTablePattern.FindAllStringSubmatch(allSQL, -1) {
		tables = append(tables, strings.ToLower(match[1])+"."+strings.ToLower(match[2]))
	}
	sort.Strings(tables)
	wantTables := []string{
		"atlas_audit.audit_events",
		"atlas_identity.admin_session_revocation_requests",
		"atlas_identity.external_subjects",
		"atlas_identity.memberships",
		"atlas_identity.oidc_transactions",
		"atlas_identity.organizations",
		"atlas_identity.permission_catalogue",
		"atlas_identity.principal_roles",
		"atlas_identity.principals",
		"atlas_identity.role_catalogue",
		"atlas_identity.role_delegations",
		"atlas_identity.role_permissions",
		"atlas_identity.session_revocation_requests",
		"atlas_identity.sessions",
	}
	if !reflect.DeepEqual(tables, wantTables) {
		t.Fatalf("closed product-table inventory = %#v", tables)
	}
	for _, table := range wantTables {
		parts := strings.Split(table, ".")
		entry := regexp.MustCompile(`\(\s*'` + regexp.QuoteMeta(parts[0]) + `'\s*,\s*'` +
			regexp.QuoteMeta(parts[1]) + `'\s*,`)
		if !entry.MatchString(allSQL) {
			t.Errorf("%s lacks a data-scope registry entry", table)
		}
	}
	for _, required := range []string{
		"('atlas_identity', 'organizations', 'tenant', 'tenant_id', NULL)",
		"('atlas_identity', 'memberships', 'tenant', 'tenant_id', NULL)",
		"('atlas_identity', 'sessions', 'mixed', 'tenant_id'",
		"'atlas_identity',\n        'oidc_transactions',\n        'global'",
		"'atlas_identity',\n        'session_revocation_requests',\n        'global'",
		"'atlas_identity',\n    'admin_session_revocation_requests',\n    'global'",
		"('atlas_audit', 'audit_events', 'mixed', 'tenant_id'",
		"GRANT INSERT ON atlas_audit.audit_events TO atlas_api;",
		"GRANT USAGE ON SCHEMA atlas_identity TO atlas_api;",
		"GRANT USAGE ON SCHEMA atlas_audit TO atlas_api;",
		"GRANT SELECT, INSERT, UPDATE ON atlas_identity.oidc_transactions TO atlas_api;",
		"GRANT UPDATE (authorization_version) ON atlas_identity.principals TO atlas_api;",
		"GRANT SELECT, INSERT ON atlas_identity.admin_session_revocation_requests TO atlas_api;",
	} {
		if !strings.Contains(allSQL, required) {
			t.Errorf("persistence policy is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"GRANT SELECT ON atlas_audit.audit_events TO atlas_api",
		"GRANT UPDATE ON atlas_audit.audit_events",
		"GRANT DELETE ON atlas_audit.audit_events",
		"GRANT USAGE ON SCHEMA atlas_identity TO atlas_worker",
		"GRANT USAGE ON SCHEMA atlas_audit TO atlas_reporting_read",
		"atlas_operations.",
		"atlas_wallet.",
		"atlas_ledger.",
	} {
		if strings.Contains(allSQL, forbidden) {
			t.Errorf("persistence policy contains forbidden capability %q", forbidden)
		}
	}
}

func TestPhase01SeedManifestIsClosedAndChecksumBound(t *testing.T) {
	root := repositoryRoot(t)
	directory := filepath.Join(root, "db", "seeds")
	manifestContent := readPhase01PolicyFile(t, root, "db/seeds/MANIFEST.sha256")
	want := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(manifestContent), "\n") {
		parts := strings.SplitN(strings.TrimSuffix(line, "\r"), "  ./", 2)
		if len(parts) != 2 || len(parts[0]) != 64 || filepath.Base(parts[1]) != parts[1] {
			t.Fatalf("unsafe seed manifest entry %q", line)
		}
		want[parts[1]] = parts[0]
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	actual := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "MANIFEST.sha256" {
			continue
		}
		actual = append(actual, entry.Name())
		content, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(content)
		if want[entry.Name()] != hex.EncodeToString(digest[:]) {
			t.Errorf("seed artifact %s is not checksum-bound", entry.Name())
		}
	}
	sort.Strings(actual)
	expected := []string{
		"000001_phase_01_identity.json",
		"000002_phase_01_policy.json",
		"load-phase-01-identity.sql",
		"load-phase-01-policy.sql",
	}
	if !reflect.DeepEqual(actual, expected) || len(want) != len(expected) {
		t.Fatalf("closed seed inventory = %#v", actual)
	}
}

func TestPhase01TenantRepositorySignatureAndPredicateAreExplicit(t *testing.T) {
	root := repositoryRoot(t)
	source := readPhase01PolicyFile(t, root, "internal/identity/persistence/membership.go")
	for _, required := range []string{
		"tenant identity.TenantContext",
		"WHERE tenant_id = $1 AND membership_id = $2",
		"tenant.TenantID().String()",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("tenant repository policy is missing %q", required)
		}
	}
}

func readPhase01PolicyFile(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
