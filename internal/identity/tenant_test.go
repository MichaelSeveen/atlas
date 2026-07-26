package identity

import "testing"

func TestTenantContextRequiresTenantIdentifier(t *testing.T) {
	context, err := NewTenantContext("ten_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if got := context.TenantID().String(); got != "ten_01JAT1AS00000000000001" {
		t.Fatalf("tenant ID = %q", got)
	}

	for _, value := range []string{
		"",
		"usr_01JAT1AS00000000000001",
		"ten_invalid",
	} {
		if _, err := NewTenantContext(value); err == nil {
			t.Errorf("invalid tenant context %q was accepted", value)
		}
	}
}
