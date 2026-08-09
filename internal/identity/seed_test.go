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
	if update.SchemaVersion != 1 ||
		update.SeedID != "atlas-phase01-identity-policy-v2" ||
		update.PredecessorSeedID != manifest.SeedID ||
		update.PreviousPolicySHA256 != manifest.PolicySHA256 ||
		update.PolicySHA256 != "8c5085e94e6006b232f28974ebb6aa251452be18647f9863dd4155ce43c7f8cf" {
		t.Fatalf("identity policy seed chain drifted: %#v", update)
	}
	v3Content, err := os.ReadFile(filepath.Join(root, "db", "seeds", "000003_phase_01_policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v3 struct {
		SchemaVersion        int    `json:"schema_version"`
		SeedID               string `json:"seed_id"`
		PredecessorSeedID    string `json:"predecessor_seed_id"`
		PreviousPolicySHA256 string `json:"previous_policy_sha256"`
		PolicySHA256         string `json:"policy_sha256"`
	}
	if err := json.Unmarshal(v3Content, &v3); err != nil {
		t.Fatal(err)
	}
	if v3.SchemaVersion != 1 ||
		v3.SeedID != "atlas-phase01-identity-policy-v3" ||
		v3.PredecessorSeedID != update.SeedID ||
		v3.PreviousPolicySHA256 != update.PolicySHA256 ||
		v3.PolicySHA256 != "2acd97d4467eed25c0991331e5283b303df3fd52d0f4c9d5f6851353db64c2d1" {
		t.Fatalf("identity policy seed v3 chain drifted: %#v", v3)
	}
	v4Content, err := os.ReadFile(filepath.Join(root, "db", "seeds", "000004_phase_01_policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v4 struct {
		SchemaVersion        int    `json:"schema_version"`
		SeedID               string `json:"seed_id"`
		PredecessorSeedID    string `json:"predecessor_seed_id"`
		PreviousPolicySHA256 string `json:"previous_policy_sha256"`
		PolicySHA256         string `json:"policy_sha256"`
	}
	if err := json.Unmarshal(v4Content, &v4); err != nil {
		t.Fatal(err)
	}
	policyDigest := sha256.Sum256(policyContent)
	if v4.SchemaVersion != 1 ||
		v4.SeedID != "atlas-phase01-identity-policy-v4" ||
		v4.PredecessorSeedID != v3.SeedID ||
		v4.PreviousPolicySHA256 != v3.PolicySHA256 ||
		v4.PolicySHA256 != hex.EncodeToString(policyDigest[:]) {
		t.Fatalf("identity policy seed v4 chain drifted: %#v", v4)
	}
	v5Content, err := os.ReadFile(filepath.Join(root, "db", "seeds", "000005_phase_01_acceptance_personas.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v5 struct {
		SchemaVersion     int    `json:"schema_version"`
		SeedID            string `json:"seed_id"`
		PredecessorSeedID string `json:"predecessor_seed_id"`
		PolicySHA256      string `json:"policy_sha256"`
		Principals        []struct {
			RoleID   string `json:"role_id"`
			Subject  string `json:"subject"`
			Username string `json:"username"`
		} `json:"principals"`
	}
	if err := json.Unmarshal(v5Content, &v5); err != nil {
		t.Fatal(err)
	}
	roles := make(map[string]bool, len(v5.Principals))
	subjects := make(map[string]bool, len(v5.Principals))
	usernames := make(map[string]string, len(v5.Principals))
	for _, principal := range v5.Principals {
		roles[principal.RoleID] = true
		subjects[principal.Subject] = true
		usernames[principal.Username] = principal.Subject
	}
	if v5.SchemaVersion != 1 ||
		v5.SeedID != "atlas-phase01-acceptance-personas-v5" ||
		v5.PredecessorSeedID != v4.SeedID ||
		v5.PolicySHA256 != v4.PolicySHA256 ||
		len(v5.Principals) != 3 || len(subjects) != 3 ||
		!roles["support"] || !roles["risk_analyst"] || !roles["finance_operator"] {
		t.Fatalf("identity acceptance persona seed v5 drifted: %#v", v5)
	}
	realmContent, err := os.ReadFile(filepath.Join(root, "deploy", "local", "keycloak", "atlas-workforce-local-realm.json"))
	if err != nil {
		t.Fatal(err)
	}
	var realm struct {
		Users []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"users"`
	}
	if err := json.Unmarshal(realmContent, &realm); err != nil {
		t.Fatal(err)
	}
	for username, subject := range usernames {
		matched := false
		for _, user := range realm.Users {
			if user.Username == username && user.ID == subject {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("acceptance persona %s is not subject-bound in the workforce realm", username)
		}
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

func TestIdentitySeedOrganizationNameProjectionMutationsAreRejected(t *testing.T) {
	root := identityRepositoryRoot(t)
	manifest, _, err := LoadSeedManifest(
		filepath.Join(root, "db", "seeds", "000001_phase_01_identity.json"),
		filepath.Join(root, "docs", "atlas-prd", "03-contracts", "identity-access-policy.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*SeedManifest){
		"non-canonical normalized name": func(candidate *SeedManifest) {
			candidate.Organizations[0].NormalizedName = "NOT-CANONICAL"
		},
		"normalized collision": func(candidate *SeedManifest) {
			candidate.Organizations[1].DisplayName = candidate.Organizations[0].DisplayName
			candidate.Organizations[1].NormalizedName = candidate.Organizations[0].NormalizedName
		},
		"confusable skeleton collision": func(candidate *SeedManifest) {
			candidate.Organizations[1].ConfusableSkeleton = candidate.Organizations[0].ConfusableSkeleton
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := manifest
			candidate.Organizations = append([]SeedOrganization(nil), manifest.Organizations...)
			mutate(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("mutated organization name projection was accepted")
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
