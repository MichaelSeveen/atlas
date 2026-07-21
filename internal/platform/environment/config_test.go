package environment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var validationTime = time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)

func TestCanonicalEnvironmentSet(t *testing.T) {
	root := repositoryRoot(t)
	configs := make([]Config, 0, len(Names()))
	for _, name := range Names() {
		config, err := Load(filepath.Join(root, "deploy", "environments", string(name)+".json"), validationTime)
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		configs = append(configs, config)
	}
	if err := ValidateSet(configs); err != nil {
		t.Fatal(err)
	}
}

func TestMostAgentsSkip05ProductionConfigurationRejectsDevelopmentKeysAndWildcardOrigins(t *testing.T) {
	base := loadCanonicalConfig(t, ProductionReference)
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "wildcard origin", mutate: func(config *Config) { config.AllowedOrigins = []string{"*"} }},
		{name: "development credential", mutate: func(config *Config) { config.CredentialRefs["signing"] = "secret://atlas/local/signing" }},
		{name: "mock mode", mutate: func(config *Config) { config.MockMode = true }},
		{name: "real service", mutate: func(config *Config) { config.Services[0].Address = "payments.example.com:443" }},
		{name: "non synthetic service", mutate: func(config *Config) { config.Services[0].Synthetic = false }},
		{name: "plaintext origin", mutate: func(config *Config) {
			config.AllowedOrigins = []string{"http://web.production-reference.atlas.invalid"}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := cloneConfig(base)
			test.mutate(&config)
			if err := config.Validate(validationTime); err == nil {
				t.Fatal("unsafe production-reference configuration was accepted")
			}
		})
	}
}

func TestLocalConfigurationRejectsEndpointOutsideFixedComposeTopology(t *testing.T) {
	config := loadCanonicalConfig(t, Local)
	config.Services[0].Address = "production:5432"
	if err := config.Validate(validationTime); err == nil {
		t.Fatal("local real-service canary was accepted")
	}
	config = loadCanonicalConfig(t, Local)
	config.Surfaces.API = "http://api:8080"
	if err := config.Validate(validationTime); err == nil {
		t.Fatal("local browser surface outside loopback was accepted")
	}
}

func TestConfigurationDecoderRejectsUnknownAndTrailingFields(t *testing.T) {
	for _, source := range []string{
		`{"version":1,"unknown":true}`,
		`{"version":1} {"version":1}`,
	} {
		if _, err := Decode(strings.NewReader(source), validationTime); err == nil {
			t.Fatal("unsafe configuration JSON was accepted")
		}
	}
}

func TestFeatureFlagMetadataFailsClosed(t *testing.T) {
	base := loadCanonicalConfig(t, Local)
	tests := []func(*FeatureFlag){
		func(flag *FeatureFlag) { flag.Owner = "" },
		func(flag *FeatureFlag) { flag.Expires = "2026-07-20" },
		func(flag *FeatureFlag) { flag.Risk = "unknown" },
		func(flag *FeatureFlag) { flag.Rollback = "" },
		func(flag *FeatureFlag) { flag.Risk = "high"; flag.Default = true },
	}
	for _, mutate := range tests {
		config := cloneConfig(base)
		mutate(&config.FeatureFlags[0])
		if err := config.Validate(validationTime); err == nil {
			t.Fatal("incomplete feature-flag metadata was accepted")
		}
	}
}

func TestFeatureFlagEvaluationIsConcurrentAndDefaultsOnSourceOutage(t *testing.T) {
	config := loadCanonicalConfig(t, Local)
	flags := NewFlagSet(config)
	outage := FlagSourceFunc(func(context.Context, string) (bool, error) {
		return false, context.DeadlineExceeded
	})
	value, err := flags.Enabled(context.Background(), "foundation.synthetic-shells", outage)
	if err != nil || !value {
		t.Fatalf("source outage did not use the safe configured default: value=%v err=%v", value, err)
	}
	if _, err := flags.Enabled(context.Background(), "unknown.flag", nil); err == nil {
		t.Fatal("unknown feature flag was accepted")
	}
	rollback, err := flags.RollbackBehavior("foundation.synthetic-shells")
	if err != nil || strings.TrimSpace(rollback) == "" {
		t.Fatal("feature flag rollback behavior is unavailable")
	}

	source := FlagSourceFunc(func(context.Context, string) (bool, error) { return false, nil })
	var wait sync.WaitGroup
	errorsFound := make(chan error, 64)
	for range 64 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := flags.Enabled(context.Background(), "foundation.synthetic-shells", source)
			if err != nil || value {
				errorsFound <- errors.New("concurrent flag evaluation changed result")
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
}

func TestCredentialFingerprintsAreUniqueAcrossPreparedEnvironments(t *testing.T) {
	root := repositoryRoot(t)
	stateRoot := filepath.Join(t.TempDir(), "atlas-environments")
	fingerprints := make(map[[32]byte]string)
	for _, name := range []Name{Local, Test} {
		path, err := Prepare(name, filepath.Join(root, "deploy", "environments"), stateRoot, validationTime)
		if err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
			key, value, _ := strings.Cut(line, "=")
			if !strings.Contains(key, "PASSWORD") && key != "ATLAS_NATS_TOKEN" {
				continue
			}
			fingerprint := sha256.Sum256([]byte(value))
			if owner, duplicate := fingerprints[fingerprint]; duplicate {
				t.Fatalf("credential fingerprint is shared by %s and %s/%s", owner, name, key)
			}
			fingerprints[fingerprint] = string(name) + "/" + key
		}
	}
	if len(fingerprints) != 22 {
		t.Fatalf("unexpected credential fingerprint inventory: %d", len(fingerprints))
	}
}

func TestPrepareUpgradesLegacyS04RuntimeCredentialsWithoutRotatingThem(t *testing.T) {
	root := repositoryRoot(t)
	stateRoot := filepath.Join(t.TempDir(), "atlas-environments")
	target := filepath.Join(stateRoot, string(Local))
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	legacyPassword := "legacy-bootstrap-password"
	legacy := strings.Join([]string{
		"ATLAS_ENVIRONMENT=local",
		"ATLAS_KEYCLOAK_ADMIN=atlas_local_admin",
		"ATLAS_KEYCLOAK_ADMIN_PASSWORD=legacy-keycloak-password",
		"ATLAS_MINIO_ROOT_PASSWORD=legacy-minio-password",
		"ATLAS_MINIO_ROOT_USER=atlas_local_objects",
		"ATLAS_NATS_TOKEN=legacy-nats-token",
		"ATLAS_POSTGRES_DB=atlas_local",
		"ATLAS_POSTGRES_PASSWORD=" + legacyPassword,
		"ATLAS_POSTGRES_USER=atlas_local_api",
		"ATLAS_REDIS_PASSWORD=legacy-redis-password",
	}, "\n") + "\n"
	runtimePath := filepath.Join(target, runtimeEnvironmentFile)
	if err := os.WriteFile(runtimePath, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(Local, filepath.Join(root, "deploy", "environments"), stateRoot, validationTime)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(prepared)
	if err != nil {
		t.Fatal(err)
	}
	values, err := parseRuntimeEnvironment(content)
	if err != nil {
		t.Fatal(err)
	}
	if values["ATLAS_POSTGRES_PASSWORD"] != legacyPassword {
		t.Fatal("S04 bootstrap credential was rotated during the S05 state upgrade")
	}
	if err := validateRuntimeValues(values); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareIsIdempotentAndResetRequiresExactEnvironmentConfirmation(t *testing.T) {
	root := repositoryRoot(t)
	stateRoot := filepath.Join(t.TempDir(), "atlas-environments")
	runtimePath, err := Prepare(Local, filepath.Join(root, "deploy", "environments"), stateRoot, validationTime)
	if err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(runtimePath)
	if err != nil {
		t.Fatal(err)
	}
	secondPath, err := Prepare(Local, filepath.Join(root, "deploy", "environments"), stateRoot, validationTime)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("repeated prepare rotated local credentials unexpectedly")
	}
	if bytes.Contains(first, []byte("secret://")) || bytes.Contains(first, []byte("change_me")) {
		t.Fatal("runtime environment contains a placeholder credential")
	}

	if _, err := Reset(Local, stateRoot, "RESET ATLAS TEST"); err == nil {
		t.Fatal("wrong environment reset confirmation was accepted")
	}
	if _, err := os.Stat(runtimePath); err != nil {
		t.Fatal("failed reset removed environment state")
	}
	removed, err := Reset(Local, stateRoot, ResetConfirmation(Local))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(removed); !os.IsNotExist(err) {
		t.Fatalf("environment target remains after confirmed reset: %v", err)
	}
	if _, err := Reset(ProductionReference, stateRoot, ResetConfirmation(ProductionReference)); err == nil {
		t.Fatal("production-reference reset was accepted")
	}
}

func TestProbeSetFailsClosed(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ready := NewProbeSet(Config{Services: []Service{{Name: "test", Address: listener.Addr().String(), Synthetic: true}}})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if !ready.Ready(ctx) {
		t.Fatal("reachable dependency did not become ready")
	}
	unready := NewProbeSet(Config{Services: []Service{{Name: "test", Address: "127.0.0.1:1", Synthetic: true}}})
	if unready.Ready(ctx) {
		t.Fatal("unreachable dependency was reported ready")
	}
}

func TestCanonicalSeedManifestIsDeterministicAndFeatureFree(t *testing.T) {
	path := filepath.Join(repositoryRoot(t), "deploy", "seeds", "foundation.json")
	manifest, first, err := LoadSeedManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := LoadSeedManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || len(first) != 64 {
		t.Fatal("seed manifest digest is not deterministic")
	}
	if len(manifest.Tenants) != 2 || len(manifest.Users) != 3 || len(manifest.Accounts) != 2 || len(manifest.Scenarios) != 8 {
		t.Fatalf("unexpected foundation fixture inventory: %+v", manifest)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"balance", "posting", "journal", "payment", "wallet", "real service"} {
		if strings.Contains(strings.ToLower(string(source)), forbidden) {
			t.Fatalf("seed manifest introduced forbidden product/financial term %q", forbidden)
		}
	}
}

func loadCanonicalConfig(t *testing.T, name Name) Config {
	t.Helper()
	config, err := Load(filepath.Join(repositoryRoot(t), "deploy", "environments", string(name)+".json"), validationTime)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func cloneConfig(source Config) Config {
	cloned := source
	cloned.AllowedOrigins = append([]string(nil), source.AllowedOrigins...)
	cloned.Services = append([]Service(nil), source.Services...)
	cloned.FeatureFlags = append([]FeatureFlag(nil), source.FeatureFlags...)
	cloned.CredentialRefs = make(map[string]string, len(source.CredentialRefs))
	for key, value := range source.CredentialRefs {
		cloned.CredentialRefs[key] = value
	}
	return cloned
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve environment package location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(current), "..", "..", ".."))
}
