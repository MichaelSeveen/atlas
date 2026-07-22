package environment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type SeedEntity struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id,omitempty"`
	Label     string `json:"label"`
	Synthetic bool   `json:"synthetic"`
}

type SeedScenario struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Seed     int64  `json:"seed"`
}

type SeedManifest struct {
	Version     int            `json:"version"`
	SeedID      string         `json:"seed_id"`
	VirtualTime string         `json:"virtual_time"`
	Tenants     []SeedEntity   `json:"tenants"`
	Users       []SeedEntity   `json:"users"`
	Accounts    []SeedEntity   `json:"accounts"`
	Scenarios   []SeedScenario `json:"provider_scenarios"`
}

func LoadSeedManifest(path string) (SeedManifest, string, error) {
	// #nosec G304 -- envctl supplies the repository-owned synthetic seed manifest path.
	content, err := os.ReadFile(path)
	if err != nil {
		return SeedManifest{}, "", fmt.Errorf("read seed manifest: %w", err)
	}
	if len(content) > 1<<20 {
		return SeedManifest{}, "", errors.New("seed manifest is too large")
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	var manifest SeedManifest
	if err := decoder.Decode(&manifest); err != nil {
		return SeedManifest{}, "", fmt.Errorf("decode seed manifest: %w", err)
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		return SeedManifest{}, "", err
	}
	if err := manifest.Validate(); err != nil {
		return SeedManifest{}, "", err
	}
	digest := sha256.Sum256(content)
	return manifest, hex.EncodeToString(digest[:]), nil
}

func (m SeedManifest) Validate() error {
	if m.Version != 1 || m.SeedID != "atlas-phase00-foundation-v1" {
		return errors.New("unsupported seed manifest identity")
	}
	if _, err := time.Parse(time.RFC3339, m.VirtualTime); err != nil {
		return errors.New("seed virtual time is invalid")
	}
	if len(m.Tenants) == 0 || len(m.Users) == 0 || len(m.Accounts) == 0 || len(m.Scenarios) == 0 {
		return errors.New("seed manifest is incomplete")
	}
	seen := make(map[string]struct{})
	tenantIDs := make(map[string]struct{}, len(m.Tenants))
	validateEntities := func(kind string, entities []SeedEntity) error {
		for _, entity := range entities {
			parsed, err := identifier.Parse(entity.ID)
			if err != nil || strings.TrimSpace(entity.Label) == "" || !entity.Synthetic {
				return fmt.Errorf("invalid synthetic %s fixture", kind)
			}
			if _, duplicate := seen[parsed.String()]; duplicate {
				return errors.New("duplicate seed identifier")
			}
			seen[parsed.String()] = struct{}{}
			if entity.TenantID != "" {
				if _, err := identifier.Parse(entity.TenantID); err != nil {
					return fmt.Errorf("invalid %s tenant reference", kind)
				}
			}
		}
		return nil
	}
	if err := validateEntities("tenant", m.Tenants); err != nil {
		return err
	}
	for _, tenant := range m.Tenants {
		tenantIDs[tenant.ID] = struct{}{}
	}
	if err := validateEntities("user", m.Users); err != nil {
		return err
	}
	if err := validateEntities("account", m.Accounts); err != nil {
		return err
	}
	for _, user := range m.Users {
		if user.TenantID == "" {
			continue // Workforce fixtures are deliberately outside customer/merchant tenancy.
		}
		if _, found := tenantIDs[user.TenantID]; !found {
			return errors.New("seed fixture references an unknown tenant")
		}
	}
	for _, account := range m.Accounts {
		if _, found := tenantIDs[account.TenantID]; !found {
			return errors.New("seed fixture references an unknown tenant")
		}
	}
	seenScenarios := make(map[string]struct{})
	for _, scenario := range m.Scenarios {
		if !strings.HasPrefix(scenario.ID, "provider.") || scenario.Category == "" || scenario.Seed <= 0 {
			return errors.New("invalid provider scenario fixture")
		}
		if _, duplicate := seenScenarios[scenario.ID]; duplicate {
			return errors.New("duplicate provider scenario")
		}
		seenScenarios[scenario.ID] = struct{}{}
	}
	return nil
}
