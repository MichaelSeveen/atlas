package architecture

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase01ClosurePolicyIsCompleteAndHonest(t *testing.T) {
	root := repositoryRoot(t)
	contents, err := os.ReadFile(filepath.Join(root, "docs", "engineering", "phase-01-closure-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var policy struct {
		SchemaVersion int    `json:"schema_version"`
		Phase         string `json:"phase"`
		ClosureSlice  string `json:"closure_slice"`
		Requirements  []struct {
			ID              string   `json:"id"`
			Status          string   `json:"status"`
			OwnerSlice      string   `json:"owner_slice"`
			Evidence        string   `json:"evidence"`
			AcceptanceTests []string `json:"acceptance_tests"`
		} `json:"requirements"`
		SessionRotationBoundaries []struct{ Boundary, Result, Proof string } `json:"session_rotation_boundaries"`
		FreshStepUp               struct {
			FreshnessSeconds                 int                              `json:"freshness_seconds"`
			RevalidateAtCommit               bool                             `json:"revalidate_at_commit"`
			FutureActionRevalidationRequired bool                             `json:"future_action_revalidation_required"`
			Actions                          []struct{ Action, State string } `json:"actions"`
		} `json:"fresh_step_up"`
		Threats          []struct{ ID, Disposition, Proof string } `json:"threats"`
		AdversarialTests []struct{ ID, Result, Proof string }      `json:"adversarial_tests"`
		SkipTests        []struct {
			Number int
			Result string
			Proof  string
		} `json:"tests_most_agents_skip"`
		Decisions   map[string]string `json:"decisions"`
		Limitations []string          `json:"limitations"`
	}
	if err := json.Unmarshal(contents, &policy); err != nil {
		t.Fatal(err)
	}
	if policy.SchemaVersion != 1 || policy.Phase != "PHASE-01_IDENTITY_ACCESS_TENANCY" || policy.ClosureSlice != "P01-S09" {
		t.Fatal("Phase 01 closure policy identity is invalid")
	}

	requiredIDs := phase01RequirementIDs()
	seenRequirements := make(map[string]struct{}, len(policy.Requirements))
	for _, requirement := range policy.Requirements {
		if _, duplicate := seenRequirements[requirement.ID]; duplicate || requirement.Status != "Verified" ||
			requirement.OwnerSlice == "" || len(requirement.AcceptanceTests) == 0 {
			t.Fatalf("invalid or duplicate Phase 01 requirement disposition: %s", requirement.ID)
		}
		if _, required := requiredIDs[requirement.ID]; !required {
			t.Fatalf("unknown Phase 01 requirement disposition: %s", requirement.ID)
		}
		if !strings.HasPrefix(requirement.Evidence, "evidence/phase-01/") {
			t.Fatalf("requirement %s has unsafe evidence path", requirement.ID)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(requirement.Evidence))); err != nil {
			t.Fatalf("requirement %s evidence: %v", requirement.ID, err)
		}
		seenRequirements[requirement.ID] = struct{}{}
	}
	if len(seenRequirements) != len(requiredIDs) {
		t.Fatalf("Phase 01 requirement dispositions=%d, want %d", len(seenRequirements), len(requiredIDs))
	}
	traceability, err := os.Open(filepath.Join(root, "docs", "atlas-prd", "06-governance", "REQUIREMENTS_TRACEABILITY.csv"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = traceability.Close() })
	records, err := csv.NewReader(traceability).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 2 {
		t.Fatal("requirements traceability is empty")
	}
	headings := make(map[string]int, len(records[0]))
	for index, heading := range records[0] {
		headings[heading] = index
	}
	seenTraceability := make(map[string]struct{}, len(requiredIDs))
	for _, record := range records[1:] {
		if record[headings["Phase"]] != policy.Phase {
			continue
		}
		id := record[headings["Requirement ID"]]
		if _, required := requiredIDs[id]; !required || record[headings["Status"]] != "Verified" ||
			strings.TrimSpace(record[headings["Evidence link / notes"]]) == "" {
			t.Fatalf("Phase 01 traceability row is incomplete: %s", id)
		}
		seenTraceability[id] = struct{}{}
	}
	if len(seenTraceability) != len(requiredIDs) {
		t.Fatalf("verified Phase 01 traceability rows=%d, want %d", len(seenTraceability), len(requiredIDs))
	}

	exactStrings(t, "session rotation boundaries", policy.SessionRotationBoundaries,
		[]string{"login", "step-up", "privilege-change", "tenant-switch"},
		func(item struct{ Boundary, Result, Proof string }) (string, bool) {
			return item.Boundary, item.Result == "verified" && item.Proof != ""
		})
	if policy.FreshStepUp.FreshnessSeconds != 300 || !policy.FreshStepUp.RevalidateAtCommit ||
		!policy.FreshStepUp.FutureActionRevalidationRequired {
		t.Fatal("fresh step-up closure policy is incomplete")
	}
	wantActions := map[string]string{
		"payout": "reserved-fail-closed", "beneficiary_activation": "reserved-fail-closed",
		"contact_change": "reserved-fail-closed", "merchant_secret_creation": "implemented",
		"refund": "reserved-fail-closed", "restriction_removal": "reserved-fail-closed",
		"privileged_export": "reserved-fail-closed", "approval_decision": "implemented",
	}
	for _, action := range policy.FreshStepUp.Actions {
		if wantActions[action.Action] != action.State {
			t.Fatalf("unexpected step-up action disposition: %s=%s", action.Action, action.State)
		}
		delete(wantActions, action.Action)
	}
	if len(wantActions) != 0 {
		t.Fatalf("fresh step-up actions missing: %v", wantActions)
	}
	wantThreats := map[string]struct{}{}
	for _, number := range []int{5, 6, 7, 8, 18, 20, 23, 24, 28, 37, 38, 39, 40, 41, 44, 53, 56, 57, 58, 60} {
		wantThreats[fmt.Sprintf("THR-%03d", number)] = struct{}{}
	}
	for _, threat := range policy.Threats {
		if _, ok := wantThreats[threat.ID]; !ok || threat.Disposition != "controlled-open" || threat.Proof == "" {
			t.Fatalf("invalid Phase 01 threat disposition: %+v", threat)
		}
		delete(wantThreats, threat.ID)
	}
	if len(wantThreats) != 0 {
		t.Fatalf("Phase 01 threat dispositions missing: %v", wantThreats)
	}

	wantAdversarial := make(map[string]struct{}, 15)
	for number := 1; number <= 15; number++ {
		wantAdversarial[fmt.Sprintf("ADV-IAM-%03d", number)] = struct{}{}
	}
	for _, test := range policy.AdversarialTests {
		if _, ok := wantAdversarial[test.ID]; !ok || test.Proof == "" ||
			(test.Result != "verified" && test.Result != "fail-closed-absent-surface") {
			t.Fatalf("invalid adversarial disposition: %+v", test)
		}
		delete(wantAdversarial, test.ID)
	}
	if len(wantAdversarial) != 0 {
		t.Fatalf("Phase 01 adversarial dispositions missing: %v", wantAdversarial)
	}
	wantSkip := make(map[int]struct{}, 15)
	for number := 1; number <= 15; number++ {
		wantSkip[number] = struct{}{}
	}
	for _, test := range policy.SkipTests {
		if _, ok := wantSkip[test.Number]; !ok || test.Result != "verified" || test.Proof == "" {
			t.Fatalf("invalid most-agents-skip disposition: %+v", test)
		}
		delete(wantSkip, test.Number)
	}
	if len(wantSkip) != 0 {
		t.Fatalf("most-agents-skip dispositions missing: %v", wantSkip)
	}

	if !strings.Contains(policy.Decisions["P01-D13"], "openapi-typescript 7.13.0") ||
		!strings.Contains(policy.Decisions["administrator_membership_removal"], "fail-closed") ||
		policy.Decisions["financial_behavior"] != "absent" {
		t.Fatal("Phase 01 closure decisions are incomplete or unsafe")
	}
	limitations := strings.ToLower(strings.Join(policy.Limitations, " "))
	for _, required := range []string{"administrator membership removal remains fail-closed", "break-glass remains disabled", "no production", "financial-readiness"} {
		if !strings.Contains(limitations, required) {
			t.Fatalf("Phase 01 closure limitation is absent: %s", required)
		}
	}
}

func phase01RequirementIDs() map[string]struct{} {
	ids := make(map[string]struct{}, 30)
	for _, number := range []int{1, 2, 3, 4, 5, 6, 7, 10, 11, 12, 13, 14, 15, 20, 21, 22, 23, 24, 25, 26, 30, 31, 32, 33, 34, 40, 41, 42, 43, 44} {
		ids[fmt.Sprintf("IAM-%03d", number)] = struct{}{}
	}
	return ids
}

func exactStrings[T any](t *testing.T, label string, items []T, expected []string, identity func(T) (string, bool)) {
	t.Helper()
	want := make(map[string]struct{}, len(expected))
	for _, value := range expected {
		want[value] = struct{}{}
	}
	for _, item := range items {
		value, valid := identity(item)
		if _, ok := want[value]; !ok || !valid {
			t.Fatalf("invalid %s item: %s", label, value)
		}
		delete(want, value)
	}
	if len(want) != 0 {
		t.Fatalf("%s missing: %v", label, want)
	}
}
