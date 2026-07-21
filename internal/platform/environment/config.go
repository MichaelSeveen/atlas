// Package environment owns the typed, synthetic-only Atlas environment contract.
package environment

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ConfigVersion          = 1
	RequiredMigrationState = "foundation-v2-required"
)

type Name string

const (
	Local               Name = "local"
	Test                Name = "test"
	Staging             Name = "staging"
	ProductionReference Name = "production-reference"
)

var allNames = []Name{Local, Test, Staging, ProductionReference}

type Service struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Synthetic bool   `json:"synthetic"`
}

type Surfaces struct {
	API      string `json:"api"`
	Web      string `json:"web"`
	Identity string `json:"identity"`
}

type FeatureFlag struct {
	Key      string `json:"key"`
	Owner    string `json:"owner"`
	Expires  string `json:"expires"`
	Default  bool   `json:"default"`
	Risk     string `json:"risk"`
	Rollback string `json:"rollback"`
}

type Config struct {
	Version        int               `json:"version"`
	Environment    Name              `json:"environment"`
	SyntheticData  bool              `json:"synthetic_data"`
	Banner         string            `json:"banner"`
	MockMode       bool              `json:"mock_mode"`
	MigrationState string            `json:"migration_state"`
	AllowedOrigins []string          `json:"allowed_origins"`
	Surfaces       Surfaces          `json:"surfaces"`
	Services       []Service         `json:"services"`
	CredentialRefs map[string]string `json:"credential_refs"`
	FeatureFlags   []FeatureFlag     `json:"feature_flags"`
}

var (
	serviceNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)
	flagKeyPattern      = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,63}$`)
	requiredServices    = []string{"broker", "identity-provider", "object-storage", "postgres", "redis", "telemetry"}
	requiredCredentials = []string{
		"broker", "database", "encryption", "identity", "merchant", "object-storage", "signing",
	}
)

var localServiceHosts = map[string]string{
	"broker":            "nats",
	"identity-provider": "keycloak",
	"object-storage":    "minio",
	"postgres":          "postgres",
	"redis":             "redis",
	"telemetry":         "otel-collector",
}

func Load(path string, now time.Time) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open environment configuration: %w", err)
	}
	defer file.Close()
	config, err := Decode(file, now)
	if err != nil {
		return Config{}, fmt.Errorf("decode environment configuration: %w", err)
	}
	return config, nil
}

func Decode(reader io.Reader, now time.Time) (Config, error) {
	decoder := json.NewDecoder(io.LimitReader(reader, 1<<20))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, err
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return Config{}, err
	}
	if err := config.Validate(now); err != nil {
		return Config{}, err
	}
	return config, nil
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("environment configuration has trailing JSON")
		}
		return err
	}
	return nil
}

func (c Config) Validate(now time.Time) error {
	if c.Version != ConfigVersion {
		return errors.New("unsupported environment configuration version")
	}
	if !validName(c.Environment) {
		return errors.New("unknown environment")
	}
	if !c.SyntheticData {
		return errors.New("portfolio environments must be synthetic")
	}
	if strings.TrimSpace(c.Banner) == "" || !strings.Contains(strings.ToUpper(c.Banner), "SYNTHETIC") {
		return errors.New("environment banner must visibly identify synthetic data")
	}
	if (c.Environment == Staging || c.Environment == ProductionReference) && c.MockMode {
		return errors.New("mock mode is forbidden outside local and test")
	}
	if c.MigrationState != RequiredMigrationState {
		return errors.New("unexpected migration-state policy")
	}
	if err := c.validateOrigins(); err != nil {
		return err
	}
	if err := c.validateSurfaces(); err != nil {
		return err
	}
	if err := c.validateServices(); err != nil {
		return err
	}
	if err := c.validateCredentialRefs(); err != nil {
		return err
	}
	if err := c.validateFeatureFlags(now); err != nil {
		return err
	}
	return nil
}

func (c Config) validateOrigins() error {
	if len(c.AllowedOrigins) == 0 {
		return errors.New("at least one exact browser origin is required")
	}
	seen := make(map[string]struct{}, len(c.AllowedOrigins))
	for _, origin := range c.AllowedOrigins {
		if origin == "*" {
			return errors.New("wildcard origin is forbidden")
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" ||
			(parsed.Path != "" && parsed.Path != "/") || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("allowed origin must be an exact HTTP origin")
		}
		if parsed.Scheme+"://"+parsed.Host != origin {
			return errors.New("allowed origin must use canonical origin form")
		}
		if c.Environment == ProductionReference && parsed.Scheme != "https" {
			return errors.New("production-reference origins require HTTPS")
		}
		if c.Environment == Local && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" {
			return errors.New("local origins must remain loopback-only")
		}
		if _, duplicate := seen[origin]; duplicate {
			return errors.New("duplicate allowed origin")
		}
		seen[origin] = struct{}{}
	}
	return nil
}

func (c Config) validateSurfaces() error {
	for name, value := range map[string]string{"api": c.Surfaces.API, "web": c.Surfaces.Web, "identity": c.Surfaces.Identity} {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("%s surface must be an absolute URL", name)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("%s surface must use HTTP(S)", name)
		}
		if c.Environment == ProductionReference && parsed.Scheme != "https" {
			return fmt.Errorf("%s production-reference surface requires HTTPS", name)
		}
		if c.Environment == Local && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" {
			return fmt.Errorf("%s local surface must remain loopback-only", name)
		}
		if err := validatePortfolioHost(c.Environment, parsed.Hostname()); err != nil {
			return fmt.Errorf("%s surface: %w", name, err)
		}
	}
	return nil
}

func (c Config) validateServices() error {
	observed := make(map[string]struct{}, len(c.Services))
	for _, service := range c.Services {
		if !serviceNamePattern.MatchString(service.Name) {
			return errors.New("invalid service name")
		}
		if _, duplicate := observed[service.Name]; duplicate {
			return errors.New("duplicate service")
		}
		observed[service.Name] = struct{}{}
		if !service.Synthetic {
			return fmt.Errorf("service %s is not synthetic", service.Name)
		}
		host, port, err := net.SplitHostPort(service.Address)
		if err != nil || host == "" || port == "" {
			return fmt.Errorf("service %s has invalid address", service.Name)
		}
		if err := validatePortfolioHost(c.Environment, host); err != nil {
			return fmt.Errorf("service %s: %w", service.Name, err)
		}
		if c.Environment == Local && localServiceHosts[service.Name] != host {
			return fmt.Errorf("service %s is outside its fixed local compose endpoint", service.Name)
		}
	}
	for _, required := range requiredServices {
		if _, found := observed[required]; !found {
			return fmt.Errorf("required service %s is absent", required)
		}
	}
	if len(observed) != len(requiredServices) {
		return errors.New("unexpected service exists in foundation topology")
	}
	return nil
}

func validatePortfolioHost(environment Name, host string) error {
	if environment == Local {
		if host == "127.0.0.1" || host == "localhost" || serviceNamePattern.MatchString(host) {
			return nil
		}
		return errors.New("local endpoint must remain inside loopback or the compose network")
	}
	wantSuffix := "." + string(environment) + ".atlas.invalid"
	if !strings.HasSuffix(host, wantSuffix) {
		return errors.New("non-local endpoint must use its reserved atlas.invalid namespace")
	}
	return nil
}

func (c Config) validateCredentialRefs() error {
	if len(c.CredentialRefs) != len(requiredCredentials) {
		return errors.New("credential reference set is incomplete")
	}
	for _, purpose := range requiredCredentials {
		value, found := c.CredentialRefs[purpose]
		if !found {
			return fmt.Errorf("credential reference %s is absent", purpose)
		}
		want := "secret://atlas/" + string(c.Environment) + "/" + purpose
		if value != want {
			return fmt.Errorf("credential reference %s is not environment-scoped", purpose)
		}
	}
	return nil
}

func (c Config) validateFeatureFlags(now time.Time) error {
	seen := make(map[string]struct{}, len(c.FeatureFlags))
	for _, feature := range c.FeatureFlags {
		if !flagKeyPattern.MatchString(feature.Key) || strings.TrimSpace(feature.Owner) == "" || strings.TrimSpace(feature.Rollback) == "" {
			return errors.New("feature flag metadata is incomplete")
		}
		if feature.Risk != "low" && feature.Risk != "medium" && feature.Risk != "high" {
			return errors.New("feature flag risk is invalid")
		}
		if feature.Risk == "high" && feature.Default {
			return errors.New("high-risk feature flag cannot default on")
		}
		expires, err := time.Parse("2006-01-02", feature.Expires)
		if err != nil {
			return errors.New("feature flag expiry is invalid")
		}
		if !expires.After(now.UTC().Truncate(24 * time.Hour)) {
			return errors.New("feature flag is expired")
		}
		if _, duplicate := seen[feature.Key]; duplicate {
			return errors.New("duplicate feature flag")
		}
		seen[feature.Key] = struct{}{}
	}
	return nil
}

func ValidateSet(configs []Config) error {
	if len(configs) != len(allNames) {
		return errors.New("local, test, staging, and production-reference configurations are all required")
	}
	seenNames := make(map[Name]struct{}, len(configs))
	seenReferences := make(map[string]Name)
	for _, config := range configs {
		if _, duplicate := seenNames[config.Environment]; duplicate {
			return errors.New("duplicate environment configuration")
		}
		seenNames[config.Environment] = struct{}{}
		keys := make([]string, 0, len(config.CredentialRefs))
		for key := range config.CredentialRefs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			ref := config.CredentialRefs[key]
			if owner, duplicate := seenReferences[ref]; duplicate {
				return fmt.Errorf("credential reference is shared by %s and %s", owner, config.Environment)
			}
			seenReferences[ref] = config.Environment
		}
	}
	for _, name := range allNames {
		if _, found := seenNames[name]; !found {
			return fmt.Errorf("environment %s is absent", name)
		}
	}
	return nil
}

func validName(name Name) bool {
	for _, allowed := range allNames {
		if name == allowed {
			return true
		}
	}
	return false
}

func Names() []Name {
	return append([]Name(nil), allNames...)
}
