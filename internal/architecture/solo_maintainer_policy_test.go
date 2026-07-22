package architecture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type soloMaintainerPolicy struct {
	SchemaVersion           int      `json:"schema_version"`
	Mode                    string   `json:"mode"`
	Owner                   string   `json:"owner"`
	RequirementID           string   `json:"requirement_id"`
	RiskID                  string   `json:"risk_id"`
	IndependentReviewStatus string   `json:"independent_review_status"`
	SensitivePaths          []string `json:"sensitive_paths"`
	RequiredAttestations    []string `json:"required_attestations"`
	ProhibitedWhileActive   []string `json:"prohibited_while_active"`
	RevalidationTriggers    []string `json:"independent_review_revalidation_triggers"`
	CoolingOffHours         int      `json:"recommended_cooling_off_hours_for_sensitive_financial_semantics"`
}

func TestSoloMaintainerPolicyIsClosedScopedAndHonest(t *testing.T) {
	root := repositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, ".github", "solo-maintainer-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var policy soloMaintainerPolicy
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		t.Fatal(err)
	}
	if policy.SchemaVersion != 1 || policy.Mode != "solo-maintainer-synthetic-portfolio" || policy.Owner != "MichaelSeveen" {
		t.Fatal("solo-maintainer policy identity is invalid")
	}
	if policy.RequirementID != "FND-026" || policy.RiskID != "RSK-031" || policy.IndependentReviewStatus != "unavailable-not-claimed" {
		t.Fatal("solo-maintainer policy must bind its requirement/risk and refuse an independent-review claim")
	}
	if policy.CoolingOffHours < 24 {
		t.Fatal("sensitive financial semantics must retain at least 24-hour cooling-off guidance")
	}

	wantPaths := []string{
		".github/", "db/migrations/", "deploy/", "docs/atlas-prd/03-contracts/",
		"internal/authorization/", "internal/identity/", "internal/ledger/", "internal/platform/secret/",
		"scripts/verify-s07.ps1", "scripts/verify-s08.ps1", "tools/supply-chain.lock.json",
	}
	assertSameStrings(t, "sensitive paths", policy.SensitivePaths, wantPaths)
	if len(policy.RequiredAttestations) != 6 {
		t.Fatalf("expected six sensitive-change attestations, got %d", len(policy.RequiredAttestations))
	}
	for _, prohibited := range []string{"real-money", "real-customer-or-identity-data", "production-credentials", "production-financial-or-identity-providers", "production-ready-or-independently-reviewed-claim"} {
		if !contains(policy.ProhibitedWhileActive, prohibited) {
			t.Errorf("solo mode does not prohibit %s", prohibited)
		}
	}
	for _, trigger := range []string{"first-non-synthetic-deployment", "first-real-money-or-personal-data", "first-real-financial-or-identity-provider", "second-qualified-maintainer", "production-readiness-claim"} {
		if !contains(policy.RevalidationTriggers, trigger) {
			t.Errorf("independent-review trigger is missing %s", trigger)
		}
	}

	template, err := os.ReadFile(filepath.Join(root, ".github", "pull_request_template.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, attestation := range policy.RequiredAttestations {
		if !strings.Contains(string(template), "- [ ] "+attestation) {
			t.Errorf("pull request template omits unchecked attestation %s", attestation)
		}
	}

	workflow := readText(t, filepath.Join(root, ".github", "workflows", "pr.yml"))
	for _, marker := range []string{"test-solo-maintainer-governance.ps1", "ATLAS_PR_BODY", "ATLAS_BASE_SHA", "ATLAS_HEAD_SHA"} {
		if !strings.Contains(workflow, marker) {
			t.Errorf("PR workflow omits solo-governance marker %s", marker)
		}
	}
	script := readText(t, filepath.Join(root, "scripts", "test-solo-maintainer-governance.ps1"))
	for _, marker := range []string{"Sensitive-path seeded canary", "Incomplete-attestation seeded canary", "UNAVAILABLE_NOT_CLAIMED"} {
		if !strings.Contains(script, marker) {
			t.Errorf("solo-governance verifier omits %s", marker)
		}
	}
}

func assertSameStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)
	if strings.Join(gotCopy, "\n") != strings.Join(wantCopy, "\n") {
		t.Fatalf("%s mismatch:\n got %v\nwant %v", label, gotCopy, wantCopy)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
