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
		if err := validateRuntimeEnvironment(content); err != nil {
			return "", err
		}
		return runtimePath, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect runtime environment: %w", err)
	}

	values := map[string]string{
		"ATLAS_ENVIRONMENT":             string(name),
		"ATLAS_KEYCLOAK_ADMIN":          "atlas_" + string(name) + "_admin",
		"ATLAS_KEYCLOAK_ADMIN_PASSWORD": randomSecret(),
		"ATLAS_MINIO_ROOT_PASSWORD":     randomSecret(),
		"ATLAS_MINIO_ROOT_USER":         "atlas_" + string(name) + "_objects",
		"ATLAS_NATS_TOKEN":              randomSecret(),
		"ATLAS_POSTGRES_DB":             "atlas_" + strings.ReplaceAll(string(name), "-", "_"),
		"ATLAS_POSTGRES_PASSWORD":       randomSecret(),
		"ATLAS_POSTGRES_USER":           "atlas_" + strings.ReplaceAll(string(name), "-", "_") + "_api",
		"ATLAS_REDIS_PASSWORD":          randomSecret(),
	}
	for key, value := range values {
		if value == "" {
			return "", fmt.Errorf("generate runtime value %s", key)
		}
	}
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
		return "", fmt.Errorf("write temporary runtime environment: %w", err)
	}
	if err := os.Rename(temporary, runtimePath); err != nil {
		_ = os.Remove(temporary)
		return "", fmt.Errorf("publish runtime environment: %w", err)
	}
	return runtimePath, nil
}

func randomSecret() string {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(random)
}

func validateRuntimeEnvironment(content []byte) error {
	if len(content) == 0 || len(content) > 64<<10 {
		return errors.New("runtime environment file has unsafe size")
	}
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" || value == "" {
			return errors.New("runtime environment file is malformed")
		}
		if _, duplicate := values[key]; duplicate {
			return errors.New("runtime environment file has duplicate key")
		}
		values[key] = value
	}
	for _, key := range []string{
		"ATLAS_ENVIRONMENT", "ATLAS_KEYCLOAK_ADMIN", "ATLAS_KEYCLOAK_ADMIN_PASSWORD",
		"ATLAS_MINIO_ROOT_PASSWORD", "ATLAS_MINIO_ROOT_USER", "ATLAS_NATS_TOKEN",
		"ATLAS_POSTGRES_DB", "ATLAS_POSTGRES_PASSWORD", "ATLAS_POSTGRES_USER", "ATLAS_REDIS_PASSWORD",
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
