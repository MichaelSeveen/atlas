package architecture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type phase01EvidencePolicy struct {
	SchemaVersion int    `json:"schema_version"`
	Phase         string `json:"phase"`
	Decision      string `json:"decision"`
	Phase00       struct {
		Path             string `json:"path"`
		Mode             string `json:"mode"`
		RebindForPhase01 bool   `json:"rebind_for_phase_01"`
	} `json:"phase_00_catalogue"`
	Phase01 struct {
		PrecommitPath    string `json:"precommit_path"`
		PostcommitPath   string `json:"postcommit_path"`
		SourceIdentity   string `json:"source_identity"`
		ArtifactDigest   string `json:"artifact_digest"`
		HistoricalUpdate string `json:"historical_update"`
	} `json:"phase_01_catalogue"`
	RequiredCanaries   []string `json:"required_canaries"`
	ProhibitedShortcut []string `json:"prohibited_shortcuts"`
}

func TestPhase01EvidencePolicyPreservesPhase00AndBindsNewEvidence(t *testing.T) {
	root := repositoryRoot(t)
	path := filepath.Join(root, "docs", "engineering", "phase-01-evidence-policy.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var policy phase01EvidencePolicy
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&policy); err != nil {
		t.Fatal(err)
	}

	if policy.SchemaVersion != 1 ||
		policy.Phase != "PHASE-01_IDENTITY_ACCESS_TENANCY" ||
		policy.Decision != "ADR-0014" {
		t.Fatalf("invalid Phase 01 evidence-policy identity: %#v", policy)
	}
	if policy.Phase00.Mode != "historical-read-only" || policy.Phase00.RebindForPhase01 {
		t.Fatal("Phase 00 evidence may not be rebound for Phase 01 work")
	}
	if policy.Phase01.ArtifactDigest != "sha256" ||
		policy.Phase01.SourceIdentity != "40-hex-commit-or-UNCOMMITTED_WORKTREE(base=40-hex)" ||
		policy.Phase01.HistoricalUpdate != "add-new-version-never-delete-material-evidence" {
		t.Fatal("Phase 01 evidence identity/digest/history policy is incomplete")
	}
	for _, relative := range []string{
		policy.Phase00.Path,
		policy.Phase01.PrecommitPath,
		policy.Phase01.PostcommitPath,
	} {
		if filepath.IsAbs(relative) || strings.Contains(filepath.ToSlash(relative), "../") {
			t.Errorf("unsafe evidence path %q", relative)
		}
	}
	if !reflect.DeepEqual(policy.RequiredCanaries, []string{
		"artifact-tamper",
		"stale-source",
		"duplicate-evidence-id",
		"unsafe-artifact-path",
	}) {
		t.Errorf("required canaries = %#v", policy.RequiredCanaries)
	}

	phase00Content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(policy.Phase00.Path)))
	if err != nil {
		t.Fatal(err)
	}
	var phase00 struct {
		EvidenceID     string `json:"evidence_id"`
		SourceRevision string `json:"source_revision"`
	}
	if err := json.Unmarshal(phase00Content, &phase00); err != nil {
		t.Fatal(err)
	}
	if phase00.EvidenceID != "EVD-P00-S08-001" ||
		phase00.SourceRevision != "188578b96e5b2fe5dab27930a9e2e66f20d2ca12" {
		t.Fatalf("Phase 00 evidence identity was rewritten: %#v", phase00)
	}
}
