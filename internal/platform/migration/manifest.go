// Package migration validates Atlas's forward-only, checksum-bound migration set.
package migration

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	CurrentVersion  = 2
	CurrentChecksum = "94fdc5112a045e595ee0a6300b8e7cc50b64e09a60a562336011f176283c1dc6"
)

var (
	filenamePattern = regexp.MustCompile(`^(\d{6})_([a-z][a-z0-9_]{2,63})\.sql$`)
	namePattern     = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)
	hashPattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	transactionSQL  = regexp.MustCompile(`(?im)^\s*(begin|commit|rollback|savepoint|release\s+savepoint)\b`)
	dangerousSQL    = regexp.MustCompile(`(?i)\b(alter\s+system|copy\s+[^;]+\s+program|security\s+definer)\b`)
)

type RiskMetadata struct {
	Version            int    `json:"version"`
	Name               string `json:"name"`
	Released           bool   `json:"released"`
	LockTimeoutMS      int    `json:"lock_timeout_ms"`
	StatementTimeoutMS int    `json:"statement_timeout_ms"`
	LockRisk           string `json:"lock_risk"`
	RepresentativeData string `json:"representative_data"`
	QueryPlanReview    string `json:"query_plan_review"`
	SpaceRisk          string `json:"space_risk"`
	ForwardFix         string `json:"forward_fix"`
	Rollback           string `json:"rollback"`
}

type Migration struct {
	Version  int
	Name     string
	SQLPath  string
	Checksum string
	Risk     RiskMetadata
}

type manifestEntry struct {
	checksum string
	filename string
}

func Load(directory string) ([]Migration, error) {
	if strings.TrimSpace(directory) == "" {
		return nil, errors.New("migration directory is required")
	}
	entries, err := readManifest(filepath.Join(directory, "MANIFEST.sha256"))
	if err != nil {
		return nil, err
	}
	if err := verifyClosedInventory(directory, entries); err != nil {
		return nil, err
	}
	if len(entries)%2 != 0 {
		return nil, errors.New("migration manifest must contain SQL/metadata pairs")
	}

	migrations := make([]Migration, 0, len(entries)/2)
	for index := 0; index < len(entries); index += 2 {
		sqlEntry := entries[index]
		metadataEntry := entries[index+1]
		match := filenamePattern.FindStringSubmatch(sqlEntry.filename)
		if match == nil {
			return nil, fmt.Errorf("manifest SQL entry %q is not an ordered migration", sqlEntry.filename)
		}
		wantMetadata := strings.TrimSuffix(sqlEntry.filename, ".sql") + ".metadata.json"
		if metadataEntry.filename != wantMetadata {
			return nil, fmt.Errorf("migration %s metadata is absent or out of order", sqlEntry.filename)
		}
		version, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, errors.New("migration version is invalid")
		}
		if version != len(migrations)+1 {
			return nil, fmt.Errorf("migration versions must be contiguous from 1; got %d", version)
		}
		sqlPath := filepath.Join(directory, sqlEntry.filename)
		// #nosec G304 -- the manifest filename passed filenamePattern and is checksum-bound below.
		sqlContent, err := os.ReadFile(sqlPath)
		if err != nil {
			return nil, fmt.Errorf("read migration SQL: %w", err)
		}
		if err := validateSQL(sqlContent); err != nil {
			return nil, fmt.Errorf("migration %06d: %w", version, err)
		}
		risk, err := loadRiskMetadata(filepath.Join(directory, metadataEntry.filename))
		if err != nil {
			return nil, fmt.Errorf("migration %06d metadata: %w", version, err)
		}
		if risk.Version != version || risk.Name != match[2] {
			return nil, fmt.Errorf("migration %06d metadata identity does not match its filename", version)
		}
		if err := risk.validate(); err != nil {
			return nil, fmt.Errorf("migration %06d metadata: %w", version, err)
		}
		migrations = append(migrations, Migration{
			Version: version, Name: match[2], SQLPath: sqlPath, Checksum: sqlEntry.checksum, Risk: risk,
		})
	}
	if len(migrations) != CurrentVersion || migrations[len(migrations)-1].Checksum != CurrentChecksum {
		return nil, errors.New("current migration constants do not match the released manifest")
	}
	return migrations, nil
}

func readManifest(path string) ([]manifestEntry, error) {
	// #nosec G304 -- path is the fixed MANIFEST.sha256 below the operator-selected migration directory.
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open migration manifest: %w", err)
	}
	defer file.Close()

	entries := make([]manifestEntry, 0)
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(io.LimitReader(file, 1<<20))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "  ./", 2)
		if len(parts) != 2 || !hashPattern.MatchString(parts[0]) || filepath.Base(parts[1]) != parts[1] {
			return nil, errors.New("migration manifest contains an unsafe or malformed entry")
		}
		if _, duplicate := seen[parts[1]]; duplicate {
			return nil, errors.New("migration manifest contains a duplicate path")
		}
		seen[parts[1]] = struct{}{}
		entries = append(entries, manifestEntry{checksum: parts[0], filename: parts[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read migration manifest: %w", err)
	}
	if len(entries) == 0 {
		return nil, errors.New("migration manifest is empty")
	}
	return entries, nil
}

func verifyClosedInventory(directory string, entries []manifestEntry) error {
	want := make(map[string]string, len(entries))
	for _, entry := range entries {
		want[entry.filename] = entry.checksum
		// #nosec G304 -- readManifest rejects separators and requires a base filename.
		content, err := os.ReadFile(filepath.Join(directory, entry.filename))
		if err != nil {
			return fmt.Errorf("read manifested migration file: %w", err)
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != entry.checksum {
			return fmt.Errorf("released migration file %s failed checksum verification", entry.filename)
		}
	}

	actual := make([]string, 0)
	directoryEntries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("list migration directory: %w", err)
	}
	for _, entry := range directoryEntries {
		if entry.IsDir() || entry.Name() == "MANIFEST.sha256" {
			continue
		}
		actual = append(actual, entry.Name())
	}
	sort.Strings(actual)
	if len(actual) != len(want) {
		return errors.New("migration directory contains an unmanifested or missing file")
	}
	for _, filename := range actual {
		if _, found := want[filename]; !found {
			return fmt.Errorf("migration file %s is not in the released manifest", filename)
		}
	}
	return nil
}

func loadRiskMetadata(path string) (RiskMetadata, error) {
	// #nosec G304 -- path uses the already-validated migration base filename and directory.
	file, err := os.Open(path)
	if err != nil {
		return RiskMetadata{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 64<<10))
	decoder.DisallowUnknownFields()
	var metadata RiskMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return RiskMetadata{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return RiskMetadata{}, errors.New("metadata contains trailing JSON")
	}
	return metadata, nil
}

func (m RiskMetadata) validate() error {
	if m.Version < 1 || !namePattern.MatchString(m.Name) || !m.Released {
		return errors.New("released migration identity is incomplete")
	}
	if m.LockTimeoutMS < 100 || m.LockTimeoutMS > 2000 {
		return errors.New("lock timeout must be between 100ms and 2s")
	}
	if m.StatementTimeoutMS < m.LockTimeoutMS || m.StatementTimeoutMS > 30000 {
		return errors.New("statement timeout is missing or unsafe")
	}
	for field, value := range map[string]string{
		"lock risk": m.LockRisk, "representative data": m.RepresentativeData,
		"query-plan review": m.QueryPlanReview, "space risk": m.SpaceRisk,
		"forward fix": m.ForwardFix, "rollback": m.Rollback,
	} {
		if len(strings.TrimSpace(value)) < 16 {
			return fmt.Errorf("%s analysis is incomplete", field)
		}
	}
	return nil
}

func validateSQL(content []byte) error {
	if len(content) == 0 || len(content) > 1<<20 {
		return errors.New("SQL has unsafe size")
	}
	source := string(content)
	if transactionSQL.MatchString(source) {
		return errors.New("migration files cannot control transactions")
	}
	if dangerousSQL.MatchString(source) {
		return errors.New("migration contains a forbidden privileged statement")
	}
	lower := strings.ToLower(source)
	for _, forbidden := range []string{"wallet", "ledger", "journal", "posting", "balance", "payment", "transfer", "customer", "identity"} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("feature-free migration contains product term %q", forbidden)
		}
	}
	return nil
}
