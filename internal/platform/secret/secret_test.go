package secret

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestReferenceIsEnvironmentAndPurposeScoped(t *testing.T) {
	reference, err := ParseReference("secret://atlas/local/signing")
	if err != nil {
		t.Fatal(err)
	}
	if reference.Environment() != "local" || reference.Purpose() != "signing" || reference.String() != "secret://atlas/local/signing" {
		t.Fatal("secret reference lost its boundary metadata")
	}
	for _, unsafe := range []string{"local/signing", "secret://atlas/local/signing/extra", "secret://atlas/../signing", "secret://atlas/production/signing\n"} {
		if _, err := ParseReference(unsafe); err == nil {
			t.Fatalf("unsafe secret reference %q was accepted", unsafe)
		}
	}
}

func TestVersionedRotationOverlapAndDowngradeRejection(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	reference, _ := ParseReference("secret://atlas/local/signing")
	entries := []Entry{
		{Metadata: Metadata{Reference: reference, Owner: "platform-security", Algorithm: "Ed25519", Version: 1, ActivatedAt: now.Add(-48 * time.Hour), ExpiresAt: now.Add(-time.Hour), GraceUntil: now.Add(23 * time.Hour)}, Material: bytes.Repeat([]byte{1}, 64)},
		{Metadata: Metadata{Reference: reference, Owner: "platform-security", Algorithm: "Ed25519", Version: 2, ActivatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(23 * time.Hour), GraceUntil: now.Add(47 * time.Hour)}, Material: bytes.Repeat([]byte{2}, 64)},
	}
	store, err := NewStore(entries)
	if err != nil {
		t.Fatal(err)
	}
	policy := Policy{Algorithm: "Ed25519", MinimumVersion: 2}
	if err := store.WithActive(reference, policy, now, func(candidate Candidate) error {
		if candidate.Metadata.Version != 2 || !bytes.Equal(candidate.Material, bytes.Repeat([]byte{2}, 64)) {
			t.Fatal("active version was not selected")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.WithVerificationCandidates(reference, Policy{Algorithm: "Ed25519", MinimumVersion: 1}, now, func(candidates []Candidate) error {
		if len(candidates) != 2 || candidates[0].Metadata.Version != 2 || candidates[1].Metadata.Version != 1 {
			t.Fatalf("rotation overlap is not deterministic: %+v", candidates)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.WithActive(reference, Policy{Algorithm: "Ed25519", MinimumVersion: 3}, now, func(Candidate) error { return nil }); !errors.Is(err, ErrUnavailable) {
		t.Fatal("version downgrade/rollback floor was not enforced")
	}
	if err := store.WithActive(reference, Policy{Algorithm: "HMAC-SHA-256", MinimumVersion: 1}, now, func(Candidate) error { return nil }); !errors.Is(err, ErrUnavailable) {
		t.Fatal("algorithm purpose was not enforced")
	}
}

func TestSecretProviderUnavailableAndRestored(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	reference, _ := ParseReference("secret://atlas/test/database")
	entry := Entry{Metadata: Metadata{Reference: reference, Owner: "platform", Algorithm: "opaque-credential", Version: 1, ActivatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), GraceUntil: now.Add(2 * time.Hour)}, Material: bytes.Repeat([]byte{3}, 32)}
	store, err := NewStore([]Entry{entry})
	if err != nil {
		t.Fatal(err)
	}
	policy := Policy{Algorithm: "opaque-credential", MinimumVersion: 1}
	store.SetAvailable(false)
	if err := store.WithActive(reference, policy, now, func(Candidate) error { return nil }); !errors.Is(err, ErrUnavailable) {
		t.Fatal("provider outage did not fail closed")
	}
	store.SetAvailable(true)
	if err := store.WithActive(reference, policy, now, func(Candidate) error { return nil }); err != nil {
		t.Fatalf("restored provider did not recover without downgrade: %v", err)
	}
}

func TestSecretMaterialCannotCrossEnvironmentOrPurpose(t *testing.T) {
	now := time.Now().UTC()
	localSigning, _ := ParseReference("secret://atlas/local/signing")
	testSigning, _ := ParseReference("secret://atlas/test/signing")
	store, err := NewStore([]Entry{
		{Metadata: Metadata{Reference: localSigning, Owner: "security", Algorithm: "HMAC-SHA-256", Version: 1, ActivatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), GraceUntil: now.Add(2 * time.Hour)}, Material: bytes.Repeat([]byte{4}, 32)},
		{Metadata: Metadata{Reference: testSigning, Owner: "security", Algorithm: "HMAC-SHA-256", Version: 1, ActivatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour), GraceUntil: now.Add(2 * time.Hour)}, Material: bytes.Repeat([]byte{4}, 32)},
	})
	if err == nil || store != nil {
		t.Fatal("material reused across environments was accepted")
	}
}
