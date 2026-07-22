package architecture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type s08Catalogue struct {
	SchemaVersion  int    `json:"schema_version"`
	EvidenceID     string `json:"evidence_id"`
	CreatedAt      string `json:"created_at"`
	SourceRevision string `json:"source_revision"`
	Sanitization   string `json:"sanitization"`
	Artifacts      []struct {
		EvidenceID string `json:"evidence_id"`
		Path       string `json:"path"`
		SHA256     string `json:"sha256"`
		Result     string `json:"result"`
	} `json:"artifacts"`
}

func TestS08EvidenceCatalogueIsClosedAndUntampered(t *testing.T) {
	root := repositoryRoot(t)
	acceptanceRoot := filepath.Join(root, "evidence", "phase-00", "acceptance")
	path := filepath.Join(acceptanceRoot, "S08-evidence-catalogue-postcommit.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(acceptanceRoot, "S08-evidence-catalogue-precommit.json")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var catalogue s08Catalogue
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalogue); err != nil {
		t.Fatal(err)
	}
	if catalogue.SchemaVersion != 1 || catalogue.EvidenceID != "EVD-P00-S08-001" {
		t.Fatal("S08 evidence catalogue identity is invalid")
	}
	validSource := regexp.MustCompile(`^(?:[0-9a-f]{40}|UNCOMMITTED_WORKTREE\(base=[0-9a-f]{40}\))$`)
	if !validSource.MatchString(catalogue.SourceRevision) {
		t.Fatalf("S08 evidence source is invalid: %q", catalogue.SourceRevision)
	}
	if len(catalogue.Artifacts) < 10 || !strings.Contains(catalogue.Sanitization, "no credentials") {
		t.Fatal("S08 catalogue coverage or sanitization statement is incomplete")
	}
	seenIDs := map[string]bool{}
	seenPaths := map[string]bool{}
	coverage := map[string]bool{}
	for _, artifact := range catalogue.Artifacts {
		if artifact.EvidenceID == "" || seenIDs[artifact.EvidenceID] || seenPaths[artifact.Path] {
			t.Fatal("S08 catalogue contains a blank or duplicate identity")
		}
		if filepath.IsAbs(artifact.Path) || strings.Contains(filepath.ToSlash(artifact.Path), "../") {
			t.Fatalf("S08 catalogue path is unsafe: %s", artifact.Path)
		}
		want, err := hex.DecodeString(artifact.SHA256)
		if err != nil || len(want) != sha256.Size {
			t.Fatalf("S08 catalogue digest is invalid: %s", artifact.Path)
		}
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(artifact.Path)))
		if err != nil {
			t.Fatal(err)
		}
		got := sha256.Sum256(body)
		if !strings.EqualFold(hex.EncodeToString(got[:]), artifact.SHA256) {
			t.Fatalf("S08 catalogue digest mismatch: %s", artifact.Path)
		}
		seenIDs[artifact.EvidenceID] = true
		seenPaths[artifact.Path] = true
		for slice := 1; slice <= 8; slice++ {
			if strings.Contains(artifact.EvidenceID, "S0"+strconv.Itoa(slice)) {
				coverage[artifact.EvidenceID] = true
			}
		}
	}
	for slice := 1; slice <= 8; slice++ {
		marker := "S0" + strconv.Itoa(slice)
		found := false
		for evidenceID := range coverage {
			found = found || strings.Contains(evidenceID, marker)
		}
		if !found {
			t.Errorf("S08 catalogue does not cover slice %s", marker)
		}
	}
}

func TestS08AcceptanceWiresFailureChecksAndHonestAbsences(t *testing.T) {
	root := repositoryRoot(t)
	acceptance := readText(t, filepath.Join(root, "docs", "engineering", "PHASE-00-ACCEPTANCE.md"))
	for testNumber := 1; testNumber <= 10; testNumber++ {
		if !strings.Contains(acceptance, "#"+strconv.Itoa(testNumber)) {
			t.Errorf("acceptance procedure omits skipped test #%d", testNumber)
		}
	}
	for _, required := range []string{"#4 claimed outbox", "NOT_APPLICABLE", "No outbox table", "#10 constrained-pool race", "Phase 00 is complete for the synthetic feature-free foundation scope", "FND-026", "FND-040", "FND-042"} {
		if !strings.Contains(acceptance, required) {
			t.Errorf("acceptance procedure omits %q", required)
		}
	}

	verifier := readText(t, filepath.Join(root, "scripts", "verify-s08.ps1"))
	for _, required := range []string{"verify-s07.ps1", "test-s08-evidence-integrity.ps1", "TestPhase00GateClosurePolicy", "s06.ps1", "s05.ps1", "test-s08-constrained-pool.ps1", "test-s08-clean-clone.ps1", "finally", "s08_phase_00_completion=PASS(scope=synthetic-feature-free;accepted=FND-026,FND-040,FND-042)"} {
		if !strings.Contains(verifier, required) {
			t.Errorf("S08 verifier omits %q", required)
		}
	}
	workflow := readText(t, filepath.Join(root, ".github", "workflows", "pr.yml"))
	if !strings.Contains(workflow, "test-s08-constrained-pool.ps1 -RequireRace") {
		t.Fatal("hosted PR integration lane does not require the constrained-pool race test")
	}
}

func TestReleasePublishesOnlyAfterFullLiveAcceptance(t *testing.T) {
	root := repositoryRoot(t)
	workflow := readText(t, filepath.Join(root, ".github", "workflows", "release.yml"))
	assertReleaseWorkflowClosed(t, workflow)

	t.Run("seeded missing live preflight is rejected", func(t *testing.T) {
		seeded := strings.Replace(workflow, "-Live -History", "-History", 1)
		if releaseWorkflowClosed(seeded) {
			t.Fatal("release policy accepted a seeded workflow without live acceptance")
		}
	})

	t.Run("seeded publish before preflight is rejected", func(t *testing.T) {
		seeded := strings.Replace(workflow, "- name: Authenticate to GHCR", "- name: Authenticate to GHCR\n        # seeded before-preflight marker", 1)
		seeded = strings.Replace(seeded, "      - name: Run release preflight", "      - name: Authenticate to GHCR\n        # seeded before-preflight marker\n      - name: Run release preflight", 1)
		if releaseWorkflowClosed(seeded) {
			t.Fatal("release policy accepted a seeded registry action before preflight")
		}
	})

	t.Run("seeded missing attestation verifier token is rejected", func(t *testing.T) {
		seeded := strings.Replace(workflow, "GH_TOKEN: ${{ github.token }}", "# seeded missing verifier token", 1)
		if releaseWorkflowClosed(seeded) {
			t.Fatal("release policy accepted an attestation verifier without the GitHub token binding")
		}
	})

	t.Run("seeded missing SBOM attestation verification is rejected", func(t *testing.T) {
		seeded := strings.Replace(workflow, "https://spdx.dev/Document/v2.3", "https://example.invalid/seeded-missing-spdx", 1)
		if releaseWorkflowClosed(seeded) {
			t.Fatal("release policy accepted incomplete attestation verification")
		}
	})
}

func TestS08CleanCloneKeepsItsGoModuleCacheRemovable(t *testing.T) {
	root := repositoryRoot(t)
	script := readText(t, filepath.Join(root, "scripts", "test-s08-clean-clone.ps1"))
	assertCleanCloneUsesWritableModuleCache(t, script)

	t.Run("seeded read-only module cache is rejected", func(t *testing.T) {
		seeded := strings.Replace(script, "'-modcacherw'", "'-modcache-readonly'", 1)
		if cleanCloneUsesWritableModuleCache(seeded) {
			t.Fatal("clean-clone policy accepted a seeded read-only Go module cache")
		}
	})
}

func assertCleanCloneUsesWritableModuleCache(t *testing.T, script string) {
	t.Helper()
	if !cleanCloneUsesWritableModuleCache(script) {
		t.Fatal("clean-clone verification does not keep its isolated Go module cache removable")
	}
}

func cleanCloneUsesWritableModuleCache(script string) bool {
	for _, required := range []string{
		"GetEnvironmentVariable('GOFLAGS', 'Process')",
		"'-modcacherw'",
		"$env:GOFLAGS",
		"Remove-Item Env:GOFLAGS",
	} {
		if !strings.Contains(script, required) {
			return false
		}
	}
	flag := strings.Index(script, "'-modcacherw'")
	nestedVerification := strings.Index(script, "scripts/verify-s08.ps1")
	cleanup := strings.Index(script, "Remove-Item -LiteralPath $resolvedClone -Recurse -Force")
	return flag >= 0 && nestedVerification > flag && cleanup > nestedVerification
}

func assertReleaseWorkflowClosed(t *testing.T, workflow string) {
	t.Helper()
	if !releaseWorkflowClosed(workflow) {
		t.Fatal("release workflow does not keep full live acceptance ahead of registry mutation")
	}
}

func releaseWorkflowClosed(workflow string) bool {
	for _, required := range []string{
		"github.ref == 'refs/heads/main'",
		"startsWith(github.ref, 'refs/tags/v')",
		"verify-s08.ps1 -Live -History -SupplyChain -CleanClone -ContainerRuntime docker",
		"GH_TOKEN: ${{ github.token }}",
		"--signer-workflow",
		"--source-digest",
		"--source-ref",
		"https://slsa.dev/provenance/v1",
		"https://spdx.dev/Document/v2.3",
	} {
		if !strings.Contains(workflow, required) {
			return false
		}
	}
	preflight := strings.Index(workflow, "- name: Run release preflight")
	authenticate := strings.Index(workflow, "- name: Authenticate to GHCR")
	firstPush := strings.Index(workflow, "push: true")
	verification := strings.Index(workflow, "- name: Verify signatures and provenance identities")
	verificationToken := strings.Index(workflow, "GH_TOKEN: ${{ github.token }}")
	attestationVerify := strings.Index(workflow, "gh attestation verify")
	return preflight >= 0 && authenticate > preflight && firstPush > authenticate &&
		verification > firstPush && verificationToken > verification && attestationVerify > verificationToken
}

func readText(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
