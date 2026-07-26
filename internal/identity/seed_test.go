package identity

import (
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
		digest != "e5a8ab37437edad69ed655e6589efffd824ca4b9151b6f9d9358632bf1f13d6c" {
		t.Fatalf("identity seed release identity drifted: id=%s digest=%s", manifest.SeedID, digest)
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
