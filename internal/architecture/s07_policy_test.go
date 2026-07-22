package architecture

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type actionLock struct {
	SchemaVersion int    `json:"schema_version"`
	ReviewedAt    string `json:"reviewed_at"`
	Actions       map[string]struct {
		Version string `json:"version"`
		SHA     string `json:"sha"`
	} `json:"actions"`
}

type imageLock struct {
	SchemaVersion int               `json:"schema_version"`
	ReviewedAt    string            `json:"reviewed_at"`
	Images        map[string]string `json:"images"`
}

type toolLock struct {
	SchemaVersion int    `json:"schema_version"`
	ReviewedAt    string `json:"reviewed_at"`
	Tools         map[string]struct {
		Version string `json:"version"`
		Windows struct {
			URL    string `json:"url"`
			SHA256 string `json:"sha256"`
		} `json:"windows_amd64"`
		Linux struct {
			URL    string `json:"url"`
			SHA256 string `json:"sha256"`
		} `json:"linux_amd64"`
	} `json:"tools"`
}

type goToolLock struct {
	SchemaVersion int               `json:"schema_version"`
	ReviewedAt    string            `json:"reviewed_at"`
	Tools         map[string]string `json:"tools"`
}

func TestS07ActionsAreImmutableAndLocked(t *testing.T) {
	root := repositoryRoot(t)
	var lock actionLock
	readJSON(t, filepath.Join(root, ".github", "actions-lock.json"), &lock)
	if lock.SchemaVersion != 1 || len(lock.Actions) == 0 {
		t.Fatal("action lock must use schema 1 and contain actions")
	}
	workflows, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(workflows) < 3 {
		t.Fatalf("expected PR, nightly, and release workflows: %v", err)
	}
	used := make(map[string]bool)
	for _, workflow := range workflows {
		content, err := os.ReadFile(workflow)
		if err != nil {
			t.Fatal(err)
		}
		var document yaml.Node
		if err := yaml.Unmarshal(content, &document); err != nil {
			t.Errorf("%s is not valid YAML: %v", filepath.Base(workflow), err)
		}
		if err := validateActionUses(string(content), lock, used); err != nil {
			t.Errorf("%s: %v", filepath.Base(workflow), err)
		}
	}
	for name := range lock.Actions {
		if !used[name] {
			t.Errorf("locked action %s is not used", name)
		}
	}

	seeded := "steps:\n  - uses: actions/checkout@v6\n"
	if err := validateActionUses(seeded, lock, map[string]bool{}); err == nil {
		t.Fatal("mutable action reference canary was accepted")
	}
}

func validateActionUses(source string, lock actionLock, used map[string]bool) error {
	usesPattern := regexp.MustCompile(`(?m)^\s*-?\s*uses:\s*([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(?:/[A-Za-z0-9_./-]+)?@([^\s#]+)`)
	shaPattern := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, match := range usesPattern.FindAllStringSubmatch(source, -1) {
		name, ref := match[1], match[2]
		entry, ok := lock.Actions[name]
		if !ok {
			return fmt.Errorf("action %s is not in actions-lock.json", name)
		}
		if !shaPattern.MatchString(ref) || ref != entry.SHA {
			return fmt.Errorf("action %s must use locked SHA %s, got %s", name, entry.SHA, ref)
		}
		if !regexp.MustCompile(`^v\d+\.\d+\.\d+$`).MatchString(entry.Version) {
			return fmt.Errorf("action %s has invalid reviewed version %q", name, entry.Version)
		}
		used[name] = true
	}
	return nil
}

func TestS07ExternalImagesUseReviewedDigests(t *testing.T) {
	root := repositoryRoot(t)
	var lock imageLock
	readJSON(t, filepath.Join(root, "deploy", "images.lock.json"), &lock)
	if lock.SchemaVersion != 1 || len(lock.Images) != 9 {
		t.Fatalf("image lock must have schema 1 and nine reviewed images, got %d", len(lock.Images))
	}
	files := []string{
		filepath.Join(root, "deploy", "local", "compose.yaml"),
		filepath.Join(root, "deploy", "local", "Containerfile.backend"),
		filepath.Join(root, "apps", "web", "Containerfile"),
	}
	var combined strings.Builder
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		combined.Write(content)
	}
	for image, digest := range lock.Images {
		if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(digest) {
			t.Errorf("invalid digest for %s", image)
		}
		if !strings.Contains(combined.String(), image+"@"+digest) {
			t.Errorf("reviewed image %s@%s is not used", image, digest)
		}
	}
	for _, line := range strings.Split(combined.String(), "\n") {
		trimmed := strings.TrimSpace(line)
		if (strings.HasPrefix(trimmed, "FROM ") || strings.HasPrefix(trimmed, "image: ")) &&
			(strings.Contains(trimmed, "docker.io/") || strings.Contains(trimmed, "quay.io/")) &&
			!strings.Contains(trimmed, "@sha256:") {
			t.Errorf("mutable external image reference: %s", trimmed)
		}
	}
}

func TestS07WebRuntimeDependenciesAndCanaryArePinned(t *testing.T) {
	root := repositoryRoot(t)
	containerfile, err := os.ReadFile(filepath.Join(root, "apps", "web", "Containerfile"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(containerfile)
	for _, pin := range []string{"libgcc=15.2.0-r2", "libstdc++=15.2.0-r2"} {
		if !strings.Contains(source, pin) {
			t.Errorf("web runtime dependency is not exactly pinned: %s", pin)
		}
	}
	supplyCheck, err := os.ReadFile(filepath.Join(root, "scripts", "test-s07-supply-chain.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, assertion := range []string{"'/usr/local/bin/bun'", "$bunVersion -ne '1.3.0'"} {
		if !strings.Contains(string(supplyCheck), assertion) {
			t.Errorf("web runtime execution canary is missing %s", assertion)
		}
	}
}

func TestS07ContainerInvokedShellScriptsAreExecutableInGit(t *testing.T) {
	root := repositoryRoot(t)
	paths := []string{
		"db/recovery/postgres-entrypoint.sh",
		"db/recovery/restore-entrypoint.sh",
		"db/tools/apply-migrations.sh",
	}
	arguments := append([]string{"-C", root, "ls-files", "--stage", "--"}, paths...)
	output, err := exec.Command("git", arguments...).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect Git executable modes: %v: %s", err, output)
	}
	observed := make(map[string]string, len(paths))
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		metadata, path, found := strings.Cut(line, "\t")
		if !found {
			continue
		}
		fields := strings.Fields(metadata)
		if len(fields) > 0 {
			observed[strings.TrimSpace(path)] = fields[0]
		}
	}
	for _, path := range paths {
		if observed[path] != "100755" {
			t.Errorf("container-invoked shell script %s must be Git mode 100755, got %q", path, observed[path])
		}
	}
}

func TestS07DownloadedToolsAreVersionAndHashLocked(t *testing.T) {
	root := repositoryRoot(t)
	var lock toolLock
	readJSON(t, filepath.Join(root, "tools", "supply-chain.lock.json"), &lock)
	want := []string{"cosign", "gitleaks", "gosec", "grype", "syft"}
	var got []string
	for name, tool := range lock.Tools {
		got = append(got, name)
		if tool.Version == "" {
			t.Errorf("%s version is empty", name)
		}
		for platform, artifact := range map[string]struct{ URL, SHA string }{
			"windows_amd64": {tool.Windows.URL, tool.Windows.SHA256},
			"linux_amd64":   {tool.Linux.URL, tool.Linux.SHA256},
		} {
			if !strings.HasPrefix(artifact.URL, "https://github.com/") {
				t.Errorf("%s %s URL is not an approved HTTPS GitHub release", name, platform)
			}
			if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(artifact.SHA) {
				t.Errorf("%s %s checksum is not SHA-256", name, platform)
			}
		}
	}
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("tool lock is not closed: got %v want %v", got, want)
	}
}

func TestS07GoSecurityToolsAreExactlyVersioned(t *testing.T) {
	var lock goToolLock
	readJSON(t, filepath.Join(repositoryRoot(t), "tools", "go-tools.lock.json"), &lock)
	want := map[string]string{"govulncheck": "golang.org/x/vuln/cmd/govulncheck@v1.1.4"}
	if lock.SchemaVersion != 1 || len(lock.Tools) != len(want) {
		t.Fatal("Go security tool lock must be closed and use schema 1")
	}
	for name, module := range want {
		if lock.Tools[name] != module {
			t.Errorf("%s is not exactly locked: got %q want %q", name, lock.Tools[name], module)
		}
	}
}

func TestS07CodeOwnershipCoversSensitiveBoundaries(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github", "CODEOWNERS"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, boundary := range []string{
		"* @MichaelSeveen",
		"/internal/ledger/",
		"/internal/authorization/",
		"/internal/platform/secret/",
		"/db/migrations/",
		"/.github/",
		"/deploy/",
		"/docs/atlas-prd/03-contracts/",
	} {
		if !strings.Contains(source, boundary) {
			t.Errorf("CODEOWNERS missing %s", boundary)
		}
	}
}

func TestS07DependabotCoversEveryDependencyEcosystem(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github", "dependabot.yml"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, ecosystem := range []string{"gomod", "bun", "github-actions", "docker"} {
		if !strings.Contains(source, "package-ecosystem: "+ecosystem) {
			t.Errorf("Dependabot missing %s", ecosystem)
		}
	}
	for _, forbidden := range []string{"package-ecosystem: npm", "package-ecosystem: yarn"} {
		if strings.Contains(source, forbidden) {
			t.Errorf("forbidden frontend ecosystem configured: %s", forbidden)
		}
	}
}

func readJSON(t *testing.T, path string, target any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
