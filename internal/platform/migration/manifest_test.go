package migration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCanonicalReleasedMigrations(t *testing.T) {
	migrations, err := Load(canonicalDirectory(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) != CurrentVersion || migrations[len(migrations)-1].Checksum != CurrentChecksum {
		t.Fatal("canonical migration set is not bound to its current version")
	}
	for _, migration := range migrations {
		if !migration.Risk.Released {
			t.Fatalf("migration %d is not released", migration.Version)
		}
	}
}

func TestReleasedMigrationMutationAndDeletionAreRejected(t *testing.T) {
	directory := copyCanonicalDirectory(t)
	path := filepath.Join(directory, "000001_foundation_control_schema.sql")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\n-- seeded mutation\n"); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(directory); err == nil {
		t.Fatal("changed released migration passed checksum verification")
	}

	directory = copyCanonicalDirectory(t)
	if err := os.Remove(filepath.Join(directory, "000002_recovery_probe.metadata.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(directory); err == nil {
		t.Fatal("deleted released migration metadata was accepted")
	}
}

func TestMigrationRiskAndSQLPoliciesFailClosed(t *testing.T) {
	metadata := RiskMetadata{Version: 1, Name: "unsafe_migration", Released: true, LockTimeoutMS: 500, StatementTimeoutMS: 5000}
	if err := metadata.validate(); err == nil {
		t.Fatal("incomplete migration risk metadata was accepted")
	}
	for _, source := range []string{
		"BEGIN; CREATE TABLE atlas_foundation.unsafe(id integer); COMMIT;",
		"ALTER SYSTEM SET log_statement = 'all';",
		"CREATE TABLE wallet(id integer);",
	} {
		if err := validateSQL([]byte(source)); err == nil {
			t.Fatalf("unsafe migration SQL was accepted: %s", source)
		}
	}
}

func copyCanonicalDirectory(t *testing.T) string {
	t.Helper()
	source := canonicalDirectory(t)
	target := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, entry.Name()), content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return target
}

func canonicalDirectory(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve migration package location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", "..", "db", "migrations"))
}

func TestManifestRejectsUnsafePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "MANIFEST.sha256")
	line := strings.Repeat("0", 64) + "  ./../outside.sql\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readManifest(path); err == nil {
		t.Fatal("manifest path traversal was accepted")
	}
}
