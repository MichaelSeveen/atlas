package architecture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type phase00GatePolicy struct {
	SchemaVersion    int                       `json:"schema_version"`
	Phase            string                    `json:"phase"`
	ClosureScope     string                    `json:"closure_scope"`
	CompletionStatus string                    `json:"completion_status"`
	Requirements     []phase00GateRequirement  `json:"requirements"`
	GuardedArtifacts []phase00GuardedArtifact  `json:"guarded_artifacts"`
	GuardedDirs      []phase00GuardedDirectory `json:"guarded_directories"`
	ProhibitedClaims []string                  `json:"prohibited_claims"`
	Verification     string                    `json:"verification_command"`
}

type phase00GateRequirement struct {
	ID                   string   `json:"id"`
	Disposition          string   `json:"disposition"`
	Basis                []string `json:"basis"`
	Threats              []string `json:"threats"`
	DeferredSurfaces     []string `json:"deferred_surfaces"`
	RevalidationTriggers []string `json:"revalidation_triggers"`
}

type phase00GuardedArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type phase00GuardedDirectory struct {
	Path  string   `json:"path"`
	Files []string `json:"files"`
}

func TestPhase00GateClosurePolicy(t *testing.T) {
	root := repositoryRoot(t)
	policy := loadPhase00GatePolicy(t, root)
	if err := validatePhase00GatePolicy(root, policy); err != nil {
		t.Fatal(err)
	}

	t.Run("seeded missing requirement is rejected", func(t *testing.T) {
		seeded := clonePhase00GatePolicy(t, policy)
		seeded.Requirements = seeded.Requirements[1:]
		if err := validatePhase00GatePolicy(root, seeded); err == nil {
			t.Fatal("phase gate accepted a policy with a removed requirement")
		}
	})

	t.Run("seeded missing trigger is rejected", func(t *testing.T) {
		seeded := clonePhase00GatePolicy(t, policy)
		seeded.Requirements[3].RevalidationTriggers = nil
		if err := validatePhase00GatePolicy(root, seeded); err == nil {
			t.Fatal("phase gate accepted a scope decision without its revalidation triggers")
		}
	})

	t.Run("seeded artifact drift is rejected", func(t *testing.T) {
		seeded := clonePhase00GatePolicy(t, policy)
		seeded.GuardedArtifacts[0].SHA256 = strings.Repeat("0", sha256.Size*2)
		if err := validatePhase00GatePolicy(root, seeded); err == nil {
			t.Fatal("phase gate accepted a drifted guarded artifact")
		}
	})

	t.Run("seeded capability expansion is rejected", func(t *testing.T) {
		seeded := clonePhase00GatePolicy(t, policy)
		seeded.GuardedDirs[0].Files = append(seeded.GuardedDirs[0].Files, "job.go")
		if err := validatePhase00GatePolicy(root, seeded); err == nil {
			t.Fatal("phase gate accepted an unobserved guarded-directory expansion")
		}
	})
}

func loadPhase00GatePolicy(t *testing.T, root string) phase00GatePolicy {
	t.Helper()
	path := filepath.Join(root, "docs", "engineering", "phase-00-gate-policy.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	var policy phase00GatePolicy
	if err := decoder.Decode(&policy); err != nil {
		t.Fatal(err)
	}
	if err := rejectTrailingPolicyJSON(decoder); err != nil {
		t.Fatal(err)
	}
	return policy
}

func rejectTrailingPolicyJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("phase gate policy contains trailing JSON")
		}
		return fmt.Errorf("decode trailing phase gate policy: %w", err)
	}
	return nil
}

func validatePhase00GatePolicy(root string, policy phase00GatePolicy) error {
	if policy.SchemaVersion != 1 || policy.Phase != "PHASE-00_ENGINEERING_FOUNDATION" || policy.ClosureScope != "synthetic-feature-free-foundation" || policy.CompletionStatus != "complete-with-accepted-deviations" {
		return errors.New("phase gate policy identity is invalid")
	}
	if policy.Verification != "go test ./internal/architecture -run TestPhase00GateClosurePolicy -count=1" {
		return errors.New("phase gate verification command is not closed")
	}
	if len(policy.ProhibitedClaims) != 5 {
		return errors.New("phase gate prohibited-claim inventory is incomplete")
	}

	expected := map[string]struct {
		disposition string
		triggers    []string
	}{
		"FND-011": {"satisfied-current-scope", []string{"first-product-schema", "first-executable-provider-scenario"}},
		"FND-026": {"accepted-deviation", []string{"first-non-synthetic-deployment", "first-real-money-or-personal-data", "first-real-financial-or-identity-provider", "second-qualified-maintainer", "production-readiness-claim"}},
		"FND-031": {"satisfied-current-scope", []string{"first-staging-or-production-credential-material", "first-managed-secret-provider", "first-deployable-environment"}},
		"FND-040": {"accepted-scope-decision", []string{"first-worker-or-simulator-input", "first-implemented-event-or-consumer", "first-broker-stream"}},
		"FND-042": {"accepted-scope-decision", []string{"first-worker-job-or-retry", "first-queue-lag-source", "first-deployed-alert-backend"}},
		"FND-064": {"satisfied-current-scope", []string{"first-product-durable-state", "first-reference-deployment", "first-backup-encryption-or-key-custody"}},
	}
	if len(policy.Requirements) != len(expected) {
		return fmt.Errorf("phase gate requirement inventory has %d entries", len(policy.Requirements))
	}
	seen := make(map[string]bool, len(expected))
	for _, requirement := range policy.Requirements {
		want, found := expected[requirement.ID]
		if !found || seen[requirement.ID] || requirement.Disposition != want.disposition {
			return fmt.Errorf("phase gate disposition is invalid for %s", requirement.ID)
		}
		if len(requirement.Basis) < 3 || len(requirement.Threats) == 0 || len(requirement.DeferredSurfaces) == 0 || !sameStringSet(requirement.RevalidationTriggers, want.triggers) {
			return fmt.Errorf("phase gate evidence or trigger inventory is incomplete for %s", requirement.ID)
		}
		for _, path := range requirement.Basis {
			if err := requireContainedPath(root, path); err != nil {
				return fmt.Errorf("%s basis: %w", requirement.ID, err)
			}
		}
		for _, threat := range requirement.Threats {
			if len(threat) != 7 || !strings.HasPrefix(threat, "THR-") {
				return fmt.Errorf("%s has an invalid threat identity", requirement.ID)
			}
		}
		seen[requirement.ID] = true
	}

	if err := validateSoloReviewTriggers(root, policy.Requirements); err != nil {
		return err
	}
	if len(policy.GuardedArtifacts) != 10 || len(policy.GuardedDirs) != 5 {
		return errors.New("phase gate guarded inventory is incomplete")
	}
	seenPaths := map[string]bool{}
	for _, artifact := range policy.GuardedArtifacts {
		if seenPaths[artifact.Path] {
			return fmt.Errorf("duplicate guarded artifact %s", artifact.Path)
		}
		path, err := containedPath(root, artifact.Path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(content)
		if !strings.EqualFold(hex.EncodeToString(digest[:]), artifact.SHA256) {
			return fmt.Errorf("guarded artifact changed without Phase 00 revalidation: %s", artifact.Path)
		}
		seenPaths[artifact.Path] = true
	}
	for _, directory := range policy.GuardedDirs {
		path, err := containedPath(root, directory.Path)
		if err != nil {
			return err
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		actual := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() {
				return fmt.Errorf("guarded directory contains an unexpected subdirectory: %s", directory.Path)
			}
			actual = append(actual, entry.Name())
		}
		sort.Strings(actual)
		want := append([]string(nil), directory.Files...)
		sort.Strings(want)
		if !reflect.DeepEqual(actual, want) {
			return fmt.Errorf("guarded directory changed without Phase 00 revalidation: %s", directory.Path)
		}
	}
	return nil
}

func validateSoloReviewTriggers(root string, requirements []phase00GateRequirement) error {
	content, err := os.ReadFile(filepath.Join(root, ".github", "solo-maintainer-policy.json"))
	if err != nil {
		return err
	}
	var solo struct {
		Triggers []string `json:"independent_review_revalidation_triggers"`
	}
	if err := json.Unmarshal(content, &solo); err != nil {
		return err
	}
	for _, requirement := range requirements {
		if requirement.ID == "FND-026" && !sameStringSet(requirement.RevalidationTriggers, solo.Triggers) {
			return errors.New("FND-026 triggers diverge from the solo-maintainer policy")
		}
	}
	return nil
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	a := append([]string(nil), left...)
	b := append([]string(nil), right...)
	sort.Strings(a)
	sort.Strings(b)
	return reflect.DeepEqual(a, b)
}

func requireContainedPath(root, relative string) error {
	path, err := containedPath(root, relative)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return nil
}

func containedPath(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(filepath.ToSlash(relative), "../") {
		return "", fmt.Errorf("unsafe phase gate path %q", relative)
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return "", err
	}
	prefix := cleanRoot + string(filepath.Separator)
	if path != cleanRoot && !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("phase gate path escapes repository: %s", relative)
	}
	return path, nil
}

func clonePhase00GatePolicy(t *testing.T, source phase00GatePolicy) phase00GatePolicy {
	t.Helper()
	content, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var clone phase00GatePolicy
	if err := json.Unmarshal(content, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}
