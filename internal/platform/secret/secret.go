// Package secret defines the provider-neutral, versioned secret boundary used
// by Atlas. It stores no durable key material and performs no cryptography.
package secret

import (
	"crypto/sha256"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrUnavailable = errors.New("secret material is unavailable")
	ErrPolicy      = errors.New("secret policy rejected the request")
)

var segmentPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

type Reference struct {
	environment string
	purpose     string
}

func ParseReference(value string) (Reference, error) {
	const prefix = "secret://atlas/"
	if !strings.HasPrefix(value, prefix) {
		return Reference{}, errors.New("secret reference scheme is invalid")
	}
	parts := strings.Split(strings.TrimPrefix(value, prefix), "/")
	if len(parts) != 2 || !segmentPattern.MatchString(parts[0]) || !segmentPattern.MatchString(parts[1]) {
		return Reference{}, errors.New("secret reference is invalid")
	}
	return Reference{environment: parts[0], purpose: parts[1]}, nil
}

func (reference Reference) String() string {
	if reference.environment == "" || reference.purpose == "" {
		return ""
	}
	return "secret://atlas/" + reference.environment + "/" + reference.purpose
}

func (reference Reference) Environment() string { return reference.environment }
func (reference Reference) Purpose() string     { return reference.purpose }

type Metadata struct {
	Reference   Reference
	Owner       string
	Algorithm   string
	Version     uint64
	ActivatedAt time.Time
	ExpiresAt   time.Time
	GraceUntil  time.Time
	Revoked     bool
}

type Entry struct {
	Metadata Metadata
	Material []byte
}

type Policy struct {
	Algorithm      string
	MinimumVersion uint64
}

type Candidate struct {
	Metadata Metadata
	Material []byte
}

type Store struct {
	mu        sync.RWMutex
	available bool
	entries   []Entry
}

var allowedAlgorithms = map[string]struct{}{
	"AES-256-GCM":       {},
	"Ed25519":           {},
	"HMAC-SHA-256":      {},
	"opaque-credential": {},
}

func NewStore(entries []Entry) (*Store, error) {
	validated, err := copyAndValidateEntries(entries)
	if err != nil {
		return nil, err
	}
	return &Store{available: true, entries: validated}, nil
}

func (store *Store) Replace(entries []Entry) error {
	validated, err := copyAndValidateEntries(entries)
	if err != nil {
		return err
	}
	store.mu.Lock()
	old := store.entries
	store.entries = validated
	store.available = true
	store.mu.Unlock()
	wipeEntries(old)
	return nil
}

// SetAvailable models an external secret-provider outage without changing the
// last known snapshot. Recovery is explicit and does not silently downgrade.
func (store *Store) SetAvailable(available bool) {
	if store == nil {
		return
	}
	store.mu.Lock()
	store.available = available
	store.mu.Unlock()
}

func (store *Store) WithActive(reference Reference, policy Policy, now time.Time, use func(Candidate) error) error {
	if use == nil || !validPolicy(policy) {
		return ErrPolicy
	}
	candidates, err := store.selectCandidates(reference, policy, now, false)
	if err != nil {
		return err
	}
	selected := candidates[0]
	defer wipe(selected.Material)
	return use(selected)
}

func (store *Store) WithVerificationCandidates(reference Reference, policy Policy, now time.Time, use func([]Candidate) error) error {
	if use == nil || !validPolicy(policy) {
		return ErrPolicy
	}
	candidates, err := store.selectCandidates(reference, policy, now, true)
	if err != nil {
		return err
	}
	defer wipeCandidates(candidates)
	return use(candidates)
}

func (store *Store) selectCandidates(reference Reference, policy Policy, now time.Time, verification bool) ([]Candidate, error) {
	if store == nil || reference.String() == "" {
		return nil, ErrUnavailable
	}
	now = now.UTC()
	store.mu.RLock()
	if !store.available {
		store.mu.RUnlock()
		return nil, ErrUnavailable
	}
	selected := make([]Candidate, 0, 2)
	for _, entry := range store.entries {
		metadata := entry.Metadata
		if metadata.Reference != reference || metadata.Algorithm != policy.Algorithm || metadata.Version < policy.MinimumVersion || metadata.Revoked {
			continue
		}
		eligible := !now.Before(metadata.ActivatedAt) && now.Before(metadata.ExpiresAt)
		if verification {
			eligible = !now.Before(metadata.ActivatedAt) && !now.After(metadata.GraceUntil)
		}
		if eligible {
			selected = append(selected, Candidate{Metadata: metadata, Material: append([]byte(nil), entry.Material...)})
		}
	}
	store.mu.RUnlock()
	if len(selected) == 0 {
		return nil, ErrUnavailable
	}
	sort.Slice(selected, func(left, right int) bool { return selected[left].Metadata.Version > selected[right].Metadata.Version })
	if !verification && len(selected) > 1 {
		wipeCandidates(selected[1:])
		selected = selected[:1]
	}
	return selected, nil
}

func validPolicy(policy Policy) bool {
	_, allowed := allowedAlgorithms[policy.Algorithm]
	return allowed && policy.MinimumVersion > 0
}

func copyAndValidateEntries(entries []Entry) ([]Entry, error) {
	if len(entries) == 0 {
		return nil, errors.New("secret snapshot is empty")
	}
	result := make([]Entry, 0, len(entries))
	identities := make(map[string]struct{}, len(entries))
	materials := make(map[[32]byte]string, len(entries))
	for _, entry := range entries {
		metadata := entry.Metadata
		if metadata.Reference.String() == "" || strings.TrimSpace(metadata.Owner) == "" || len(metadata.Owner) > 64 {
			wipeEntries(result)
			return nil, errors.New("secret ownership metadata is invalid")
		}
		if _, allowed := allowedAlgorithms[metadata.Algorithm]; !allowed || metadata.Version == 0 {
			wipeEntries(result)
			return nil, errors.New("secret algorithm or version is invalid")
		}
		metadata.ActivatedAt = metadata.ActivatedAt.UTC()
		metadata.ExpiresAt = metadata.ExpiresAt.UTC()
		metadata.GraceUntil = metadata.GraceUntil.UTC()
		if metadata.ActivatedAt.IsZero() || !metadata.ExpiresAt.After(metadata.ActivatedAt) || metadata.GraceUntil.Before(metadata.ExpiresAt) || metadata.GraceUntil.Sub(metadata.ExpiresAt) > 7*24*time.Hour {
			wipeEntries(result)
			return nil, errors.New("secret lifecycle metadata is invalid")
		}
		if len(entry.Material) < 32 || len(entry.Material) > 4096 {
			wipeEntries(result)
			return nil, errors.New("secret material length is invalid")
		}
		identity := metadata.Reference.String() + "/" + metadata.Algorithm + "/" + strconv.FormatUint(metadata.Version, 10)
		if _, duplicate := identities[identity]; duplicate {
			wipeEntries(result)
			return nil, errors.New("duplicate secret version")
		}
		identities[identity] = struct{}{}
		fingerprint := sha256.Sum256(entry.Material)
		if owner, reused := materials[fingerprint]; reused && owner != metadata.Reference.String() {
			wipeEntries(result)
			return nil, errors.New("secret material is reused across boundaries")
		}
		materials[fingerprint] = metadata.Reference.String()
		result = append(result, Entry{Metadata: metadata, Material: append([]byte(nil), entry.Material...)})
	}
	return result, nil
}

func wipeCandidates(candidates []Candidate) {
	for index := range candidates {
		wipe(candidates[index].Material)
	}
}

func wipeEntries(entries []Entry) {
	for index := range entries {
		wipe(entries[index].Material)
	}
}

func wipe(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
