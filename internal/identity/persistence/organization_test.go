package persistence

import (
	"errors"
	"testing"

	"github.com/MichaelSeveen/atlas/internal/identity"
)

func TestOrganizationMemberCursorIsCanonicalAndTenantBound(t *testing.T) {
	tenant := newIntegrationID(t, "ten")
	otherTenant := newIntegrationID(t, "ten")
	principal := newIntegrationID(t, "usr")
	cursor := encodeOrganizationMemberCursor(tenant, principal)
	decoded, err := decodeOrganizationMemberCursor(cursor, true, tenant)
	if err != nil || decoded != principal.String() {
		t.Fatalf("decoded=%q err=%v", decoded, err)
	}
	if _, err := decodeOrganizationMemberCursor(cursor, true, otherTenant); !errors.Is(err, identity.ErrInputInvalid) {
		t.Fatalf("cross-tenant cursor error=%v", err)
	}
	for _, invalid := range []string{"", "short", cursor + "=", cursor[:len(cursor)-1] + "!"} {
		if _, err := decodeOrganizationMemberCursor(invalid, true, tenant); !errors.Is(err, identity.ErrInputInvalid) {
			t.Fatalf("invalid cursor %q error=%v", invalid, err)
		}
	}
}

func TestOrganizationMemberPageSizeIsClosedAndBounded(t *testing.T) {
	got, err := organizationMemberPageSize("", false)
	if err != nil || got != 25 {
		t.Fatalf("default page_size got=%d err=%v", got, err)
	}
	for value, want := range map[string]int{"1": 1, "25": 25, "100": 100} {
		got, err := organizationMemberPageSize(value, true)
		if err != nil || got != want {
			t.Fatalf("page_size=%q got=%d err=%v", value, got, err)
		}
	}
	for _, invalid := range []string{"", "0", "101", "-1", "1.5", " 25", "25 "} {
		if _, err := organizationMemberPageSize(invalid, true); !errors.Is(err, identity.ErrInputInvalid) {
			t.Fatalf("invalid page_size=%q error=%v", invalid, err)
		}
	}
}
