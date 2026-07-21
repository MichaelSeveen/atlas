package telemetry

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type threatCoverage struct {
	Version int    `json:"version"`
	Model   string `json:"model"`
	Entries []struct {
		ID            string `json:"id"`
		Applicability string `json:"applicability"`
		Boundary      string `json:"boundary"`
		Controls      string `json:"controls"`
		Evidence      string `json:"evidence"`
		Owner         string `json:"owner"`
		Residual      string `json:"residual"`
	} `json:"entries"`
}

var threatIDPattern = regexp.MustCompile(`^THR-[0-9]{3}$`)

func TestPhase00ThreatCoverageLinksEveryCanonicalThreat(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	registerFile, err := os.Open(filepath.Join(root, "docs", "atlas-prd", "06-governance", "THREAT_REGISTER.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer registerFile.Close()
	rows, err := csv.NewReader(registerFile).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 61 || rows[0][0] != "Threat ID" || rows[0][9] != "Owner" {
		t.Fatal("canonical threat register shape changed")
	}
	canonical := make(map[string]string, 60)
	for _, row := range rows[1:] {
		canonical[row[0]] = row[9]
	}

	contents, err := os.ReadFile(filepath.Join(root, "docs", "security", "PHASE-00-THREAT-COVERAGE.json"))
	if err != nil {
		t.Fatal(err)
	}
	var coverage threatCoverage
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&coverage); err != nil {
		t.Fatal(err)
	}
	if coverage.Version != 1 || coverage.Model != "PHASE-00-THREAT-MODEL.md" || len(coverage.Entries) != len(canonical) {
		t.Fatal("Phase 00 threat coverage metadata is incomplete")
	}
	seen := make(map[string]struct{}, len(coverage.Entries))
	boundaries := map[string]struct{}{"TB-01": {}, "TB-02": {}, "TB-03": {}, "TB-04": {}, "TB-05": {}, "TB-06": {}, "future-boundary": {}}
	for _, entry := range coverage.Entries {
		owner, found := canonical[entry.ID]
		if !found || !threatIDPattern.MatchString(entry.ID) || owner != entry.Owner {
			t.Fatalf("threat source/owner link is invalid for %s", entry.ID)
		}
		if _, duplicate := seen[entry.ID]; duplicate {
			t.Fatalf("duplicate threat link %s", entry.ID)
		}
		switch entry.Applicability {
		case "active", "preserved", "future":
		default:
			t.Fatalf("invalid applicability for %s", entry.ID)
		}
		for _, boundary := range strings.Split(entry.Boundary, ",") {
			if _, found := boundaries[boundary]; !found {
				t.Fatalf("unknown boundary %q for %s", boundary, entry.ID)
			}
		}
		if entry.Controls == "" || entry.Evidence == "" || entry.Residual == "" {
			t.Fatalf("control/test/residual link is empty for %s", entry.ID)
		}
		if entry.Applicability == "active" && strings.HasPrefix(entry.Evidence, "future-gate:") {
			t.Fatalf("active threat %s has future-only evidence", entry.ID)
		}
		seen[entry.ID] = struct{}{}
	}
	for id := range canonical {
		if _, found := seen[id]; !found {
			t.Fatalf("canonical threat %s is not linked", id)
		}
	}
}
