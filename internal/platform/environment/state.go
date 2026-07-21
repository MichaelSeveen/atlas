package environment

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const runtimeEnvironmentFile = "runtime.env"

func Prepare(name Name, configDirectory, stateRoot string, now time.Time) (string, error) {
	if name != Local && name != Test {
		return "", errors.New("only local and test environments may be prepared on a developer machine")
	}
	configPath := filepath.Join(configDirectory, string(name)+".json")
	if _, err := Load(configPath, now); err != nil {
		return "", err
	}
	target, err := containedEnvironmentTarget(stateRoot, name)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		return "", fmt.Errorf("create environment state: %w", err)
	}
	runtimePath := filepath.Join(target, runtimeEnvironmentFile)
	if content, err := os.ReadFile(runtimePath); err == nil {
		values, err := parseRuntimeEnvironment(content)
		if err != nil {
			return "", err
		}
		changed, err := completeRuntimeEnvironment(name, values)
		if err != nil {
			return "", err
		}
		if changed {
			if err := writeRuntimeEnvironment(runtimePath, values); err != nil {
				return "", err
			}
		}
		if err := validateRuntimeValues(values); err != nil {
			return "", err
		}
		return runtimePath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect runtime environment: %w", err)
	}

	values := make(map[string]string)
	if _, err := completeRuntimeEnvironment(name, values); err != nil {
		return "", err
	}
	if err := writeRuntimeEnvironment(runtimePath, values); err != nil {
		return "", err
	}
	return runtimePath, nil
}

func completeRuntimeEnvironment(name Name, values map[string]string) (bool, error) {
	changed := false
	deterministic := map[string]string{
		"ATLAS_ENVIRONMENT":     string(name),
		"ATLAS_KEYCLOAK_ADMIN":  "atlas_" + string(name) + "_admin",
		"ATLAS_MINIO_ROOT_USER": "atlas_" + string(name) + "_objects",
		"ATLAS_POSTGRES_DB":     "atlas_" + strings.ReplaceAll(string(name), "-", "_"),
		// ATLAS_POSTGRES_USER remains the cluster bootstrap identity for
		// compatibility with S04 developer state. Application processes receive
		// only their dedicated non-superuser credentials below.
		"ATLAS_POSTGRES_USER":             "atlas_" + strings.ReplaceAll(string(name), "-", "_") + "_api",
		"ATLAS_POSTGRES_API_USER":         "atlas_api",
		"ATLAS_POSTGRES_BACKUP_USER":      "atlas_backup",
		"ATLAS_POSTGRES_BREAK_GLASS_USER": "atlas_break_glass",
		"ATLAS_POSTGRES_MIGRATION_USER":   "atlas_migration",
		"ATLAS_POSTGRES_REPORTING_USER":   "atlas_reporting_read",
		"ATLAS_POSTGRES_WORKER_USER":      "atlas_worker",
	}
	for key, value := range deterministic {
		if existing, found := values[key]; found {
			if existing != value {
				return false, fmt.Errorf("runtime environment key %s does not match its resolved environment", key)
			}
			continue
		}
		values[key] = value
		changed = true
	}
	for _, key := range []string{
		"ATLAS_KEYCLOAK_ADMIN_PASSWORD", "ATLAS_MINIO_ROOT_PASSWORD", "ATLAS_NATS_TOKEN",
		"ATLAS_POSTGRES_API_PASSWORD", "ATLAS_POSTGRES_BACKUP_PASSWORD", "ATLAS_POSTGRES_BREAK_GLASS_PASSWORD",
		"ATLAS_POSTGRES_MIGRATION_PASSWORD", "ATLAS_POSTGRES_PASSWORD", "ATLAS_POSTGRES_REPORTING_PASSWORD",
		"ATLAS_POSTGRES_WORKER_PASSWORD", "ATLAS_REDIS_PASSWORD",
	} {
		if _, found := values[key]; found {
			continue
		}
		value := randomSecret()
		if value == "" {
			return false, fmt.Errorf("generate runtime value %s", key)
		}
		values[key] = value
		changed = true
	}
	return changed, nil
}

func writeRuntimeEnvironment(runtimePath string, values map[string]string) error {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var content strings.Builder
	for _, key := range keys {
		content.WriteString(key)
		content.WriteByte('=')
		content.WriteString(values[key])
		content.WriteByte('\n')
	}
	temporary := runtimePath + ".new"
	if err := os.WriteFile(temporary, []byte(content.String()), 0o600); err != nil {
		return fmt.Errorf("write temporary runtime environment: %w", err)
	}
	if err := os.Rename(temporary, runtimePath); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("publish runtime environment: %w", err)
	}
	return nil
}

func randomSecret() string {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(random)
}

func validateRuntimeEnvironment(content []byte) error {
	values, err := parseRuntimeEnvironment(content)
	if err != nil {
		return err
	}
	return validateRuntimeValues(values)
}

func parseRuntimeEnvironment(content []byte) (map[string]string, error) {
	if len(content) == 0 || len(content) > 64<<10 {
		return nil, errors.New("runtime environment file has unsafe size")
	}
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" || value == "" {
			return nil, errors.New("runtime environment file is malformed")
		}
		if _, duplicate := values[key]; duplicate {
			return nil, errors.New("runtime environment file has duplicate key")
		}
		values[key] = value
	}
	return values, nil
}

func validateRuntimeValues(values map[string]string) error {
	for _, key := range []string{
		"ATLAS_ENVIRONMENT", "ATLAS_KEYCLOAK_ADMIN", "ATLAS_KEYCLOAK_ADMIN_PASSWORD",
		"ATLAS_MINIO_ROOT_PASSWORD", "ATLAS_MINIO_ROOT_USER", "ATLAS_NATS_TOKEN",
		"ATLAS_POSTGRES_API_PASSWORD", "ATLAS_POSTGRES_API_USER", "ATLAS_POSTGRES_BACKUP_PASSWORD",
		"ATLAS_POSTGRES_BACKUP_USER", "ATLAS_POSTGRES_BREAK_GLASS_PASSWORD", "ATLAS_POSTGRES_BREAK_GLASS_USER",
		"ATLAS_POSTGRES_DB", "ATLAS_POSTGRES_MIGRATION_PASSWORD", "ATLAS_POSTGRES_MIGRATION_USER",
		"ATLAS_POSTGRES_PASSWORD", "ATLAS_POSTGRES_REPORTING_PASSWORD", "ATLAS_POSTGRES_REPORTING_USER",
		"ATLAS_POSTGRES_USER", "ATLAS_POSTGRES_WORKER_PASSWORD", "ATLAS_POSTGRES_WORKER_USER",
		"ATLAS_REDIS_PASSWORD",
	} {
		if _, found := values[key]; !found {
			return fmt.Errorf("runtime environment key %s is absent", key)
		}
	}
	return nil
}

func Reset(name Name, stateRoot, confirmation string) (string, error) {
	if name != Local && name != Test {
		return "", errors.New("reset is forbidden for staging and production-reference")
	}
	want := ResetConfirmation(name)
	if confirmation != want {
		return "", errors.New("reset confirmation does not match the resolved environment")
	}
	target, err := containedEnvironmentTarget(stateRoot, name)
	if err != nil {
		return "", err
	}
	if err := os.RemoveAll(target); err != nil {
		return "", fmt.Errorf("reset environment state: %w", err)
	}
	return target, nil
}

func ResetConfirmation(name Name) string {
	return "RESET ATLAS " + strings.ToUpper(string(name))
}

func containedEnvironmentTarget(stateRoot string, name Name) (string, error) {
	if strings.TrimSpace(stateRoot) == "" {
		return "", errors.New("environment state root is required")
	}
	root, err := filepath.Abs(stateRoot)
	if err != nil {
		return "", errors.New("resolve environment state root")
	}
	target, err := filepath.Abs(filepath.Join(root, string(name)))
	if err != nil {
		return "", errors.New("resolve environment target")
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == "" || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("environment target escapes its state root")
	}
	if filepath.Base(target) != string(name) {
		return "", errors.New("environment target name changed during resolution")
	}
	return target, nil
}
