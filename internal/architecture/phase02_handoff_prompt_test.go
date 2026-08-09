package architecture

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPhase02KickoffPromptIsCompleteAndFailClosed(t *testing.T) {
	root := repositoryRoot(t)
	promptPath := filepath.Join(root, "docs", "engineering", "PHASE-02-KICKOFF-PROMPT.md")
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	prompt := string(promptBytes)

	requiredIDs := []string{
		"CUS-001", "CUS-002", "CUS-003", "CUS-004", "CUS-005",
		"CUS-030", "CUS-031", "CUS-032", "CUS-033",
		"KYC-001", "KYC-002", "KYC-003", "KYC-004", "KYC-005", "KYC-006", "KYC-007", "KYC-008",
		"PRV-010", "PRV-011", "PRV-012", "PRV-013", "PRV-014",
		"PRV-020", "PRV-021", "PRV-022", "PRV-023",
	}
	for _, id := range requiredIDs {
		if !strings.Contains(prompt, "`"+id) {
			t.Errorf("Phase 02 kickoff prompt omits requirement %s", id)
		}
	}

	requiredOperations := []string{
		"GET /v1/customers/{customer_id}",
		"PATCH /v1/customers/{customer_id}",
		"GET /v1/customers/{customer_id}/contact-points",
		"POST /v1/customers/{customer_id}/contact-change-requests",
		"GET /v1/kyc/cases/{case_id}",
		"POST /v1/kyc/cases",
		"POST /v1/kyc/cases/{case_id}/submissions",
		"POST /v1/kyc/cases/{case_id}/decisions",
		"GET /v1/privacy/notices/current",
		"POST /v1/privacy/acknowledgements",
		"GET /v1/customers/{customer_id}/restrictions",
		"POST /v1/customers/{customer_id}/restriction-requests",
	}
	for _, operation := range requiredOperations {
		if !strings.Contains(prompt, operation) {
			t.Errorf("Phase 02 kickoff prompt omits proposed operation %s", operation)
		}
	}

	requiredStatements := []string{
		"P02-S01 — canonical audit, decision inventory, and execution planning",
		"Do not implement Customer/KYC/privacy runtime behavior in this first slice",
		"All 26 traceability rows remain `Planned`",
		"Absence is a contract/decision gap to close in P02-S02, not permission to invent endpoints now",
		"Do not invent a callback path or event",
		"Future privileged and financial actions remain reserved fail-closed",
		"The current worker and simulator boundaries are intentionally inert",
		"keep all 26 Phase 02 requirements Planned",
		"Do not push unless explicitly asked",
		"Do not begin P02-S02 or product implementation in the same slice",
		"scripts/verify-p01.ps1",
		"scripts/verify-p02-s01.ps1",
		"evidence/phase-02/architecture/",
		"067a31607dc173f02cb628db1f380f88ab2d411f",
		"ea901dba10821ef8ce9ff3b8f04a34f908d6261a",
	}
	for _, statement := range requiredStatements {
		if !strings.Contains(prompt, statement) {
			t.Errorf("Phase 02 kickoff prompt omits required boundary %q", statement)
		}
	}

	traceabilityPath := filepath.Join(root, "docs", "atlas-prd", "06-governance", "REQUIREMENTS_TRACEABILITY.csv")
	traceabilityFile, err := os.Open(traceabilityPath)
	if err != nil {
		t.Fatal(err)
	}
	defer traceabilityFile.Close()
	records, err := csv.NewReader(traceabilityFile).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) < 2 {
		t.Fatal("requirements traceability is empty")
	}
	header := make(map[string]int, len(records[0]))
	for index, name := range records[0] {
		header[name] = index
	}
	for _, name := range []string{"Requirement ID", "Phase", "Status"} {
		if _, ok := header[name]; !ok {
			t.Fatalf("requirements traceability omits %q column", name)
		}
	}

	expected := make(map[string]struct{}, len(requiredIDs))
	for _, id := range requiredIDs {
		expected[id] = struct{}{}
	}
	observed := make(map[string]struct{}, len(requiredIDs))
	for _, record := range records[1:] {
		if record[header["Phase"]] != "PHASE-02_CUSTOMER_KYC_PRIVACY" {
			continue
		}
		id := record[header["Requirement ID"]]
		if _, ok := expected[id]; !ok {
			t.Errorf("unexpected Phase 02 requirement in traceability: %s", id)
		}
		if status := record[header["Status"]]; status != "Planned" {
			t.Errorf("Phase 02 kickoff requires %s to remain Planned, observed %s", id, status)
		}
		observed[id] = struct{}{}
	}
	if len(observed) != len(expected) {
		t.Errorf("Phase 02 traceability set mismatch: expected %d, observed %d", len(expected), len(observed))
	}

	openAPIBytes, err := os.ReadFile(filepath.Join(root, "docs", "atlas-prd", "03-contracts", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	openAPI := string(openAPIBytes)
	uniquePaths := []string{
		"/v1/customers/{customer_id}",
		"/v1/customers/{customer_id}/contact-points",
		"/v1/customers/{customer_id}/contact-change-requests",
		"/v1/kyc/cases/{case_id}",
		"/v1/kyc/cases",
		"/v1/kyc/cases/{case_id}/submissions",
		"/v1/kyc/cases/{case_id}/decisions",
		"/v1/privacy/notices/current",
		"/v1/privacy/acknowledgements",
		"/v1/customers/{customer_id}/restrictions",
		"/v1/customers/{customer_id}/restriction-requests",
	}
	for _, path := range uniquePaths {
		if strings.Contains(openAPI, "  "+path+":") {
			t.Errorf("Phase 02 kickoff prompt is stale because OpenAPI already contains %s", path)
		}
	}
}
