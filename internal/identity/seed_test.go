package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCanonicalPhase01IdentitySeed(t *testing.T) {
	root := identityRepositoryRoot(t)
	manifest, digest, err := LoadSeedManifest(
		filepath.Join(root, "db", "seeds", "000001_phase_01_identity.json"),
		filepath.Join(root, "docs", "atlas-prd", "03-contracts", "identity-access-policy.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SeedID != "atlas-phase01-identity-v1" ||
		digest != phase01IdentitySeedV1SHA256 {
		t.Fatalf("identity seed release identity drifted: id=%s digest=%s", manifest.SeedID, digest)
	}

	policyContent, err := os.ReadFile(filepath.Join(root, "docs", "atlas-prd", "03-contracts", "identity-access-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	updateContent, err := os.ReadFile(filepath.Join(root, "db", "seeds", "000002_phase_01_policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var update struct {
		SchemaVersion        int    `json:"schema_version"`
		SeedID               string `json:"seed_id"`
		PredecessorSeedID    string `json:"predecessor_seed_id"`
		PreviousPolicySHA256 string `json:"previous_policy_sha256"`
		PolicySHA256         string `json:"policy_sha256"`
	}
	if err := json.Unmarshal(updateContent, &update); err != nil {
		t.Fatal(err)
	}
	policyDigest := sha256.Sum256(policyContent)
	if update.SchemaVersion != 1 ||
		update.SeedID != "atlas-phase01-identity-policy-v2" ||
		update.PredecessorSeedID != manifest.SeedID ||
		update.PreviousPolicySHA256 != manifest.PolicySHA256 ||
		update.PolicySHA256 != hex.EncodeToString(policyDigest[:]) {
		t.Fatalf("identity policy seed chain drifted: %#v", update)
	}
}

func TestIdentitySeedPolicyAndSubjectMutationsAreRejected(t *testing.T) {
	root := identityRepositoryRoot(t)
	seedPath := filepath.Join(root, "db", "seeds", "000001_phase_01_identity.json")
	policyPath := filepath.Join(root, "docs", "atlas-prd", "03-contracts", "identity-access-policy.json")
	content, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutated := range map[string]string{
		"policy checksum":       strings.Replace(string(content), `"policy_sha256": "f`, `"policy_sha256": "0`, 1),
		"cross population":      strings.Replace(string(content), `"population": "customer",`, `"population": "merchant",`, 1),
		"live recovery session": strings.Replace(string(content), `"status": "revoked",`, `"status": "active",`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "seed.json")
			if err := os.WriteFile(path, []byte(mutated), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, _, err := LoadSeedManifest(path, policyPath); err == nil {
				t.Fatal("mutated identity seed was accepted")
			}
		})
	}
}

func identityRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve identity package location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
}
