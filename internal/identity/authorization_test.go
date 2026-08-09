package identity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestDefaultAuthorizationPolicyMatchesCanonicalContract(t *testing.T) {
	t.Parallel()
	root := identityRepositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(
		root, "docs", "atlas-prd", "03-contracts", "identity-access-policy.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Permissions []string `json:"permissions"`
		Purposes    []string `json:"purposes"`
		Roles       []struct {
			ID             string   `json:"id"`
			Population     string   `json:"population"`
			Permissions    []string `json:"permissions"`
			StandingStatus string   `json:"standing_status"`
		} `json:"roles"`
		FieldRules []struct {
			Resource         string `json:"resource"`
			Field            string `json:"field"`
			Default          string `json:"default"`
			RevealPermission string `json:"reveal_permission"`
			RequiredPurpose  string `json:"required_purpose"`
		} `json:"field_rules"`
	}
	if err := json.Unmarshal(content, &contract); err != nil {
		t.Fatal(err)
	}
	policy := DefaultAuthorizationPolicy()
	if got := sortedSetKeys(policy.permissions); !reflect.DeepEqual(got, sortedCopy(contract.Permissions)) {
		t.Fatalf("runtime permissions diverge from contract: got=%v want=%v", got, contract.Permissions)
	}
	if got := sortedSetKeys(policy.purposes); !reflect.DeepEqual(got, sortedCopy(contract.Purposes)) {
		t.Fatalf("runtime purposes diverge from contract: got=%v want=%v", got, contract.Purposes)
	}
	if len(policy.roles) != len(contract.Roles) {
		t.Fatalf("runtime role count=%d want=%d", len(policy.roles), len(contract.Roles))
	}
	for _, expected := range contract.Roles {
		actual, found := policy.roles[expected.ID]
		if !found || actual.population != Population(expected.Population) ||
			actual.enabled == (expected.StandingStatus == "disabled") ||
			!reflect.DeepEqual(actual.permissions, sortedCopy(expected.Permissions)) {
			t.Fatalf("runtime role %q diverges from contract", expected.ID)
		}
	}
	if len(policy.fields) != len(contract.FieldRules) {
		t.Fatalf("runtime field rule count=%d want=%d", len(policy.fields), len(contract.FieldRules))
	}
	for _, expected := range contract.FieldRules {
		actual, found := policy.fields[expected.Resource+"."+expected.Field]
		if expected.Default != "masked" || !found ||
			actual.revealPermission != expected.RevealPermission ||
			actual.requiredPurpose != expected.RequiredPurpose {
			t.Fatalf("runtime field rule %s.%s diverges from contract", expected.Resource, expected.Field)
		}
	}
}

func TestAuthorizationPolicyDeniesEveryUnknownOrStaleInput(t *testing.T) {
	t.Parallel()
	policy := DefaultAuthorizationPolicy()
	valid := validMemberListAuthorizationFacts(t, policy, "merchant_operator")
	if decision := policy.Evaluate(valid); decision.Effect != AuthorizationAllow ||
		decision.FieldAccess != FieldAccessNone {
		t.Fatalf("valid decision=%+v", decision)
	}

	tests := map[string]func(*AuthorizationFacts){
		"principal":      func(facts *AuthorizationFacts) { facts.PrincipalID = identifierZero() },
		"principal type": func(facts *AuthorizationFacts) { facts.PrincipalType = "workforce" },
		"role":           func(facts *AuthorizationFacts) { facts.Role = "unknown_role" },
		"population":     func(facts *AuthorizationFacts) { facts.Population = PopulationWorkforce },
		"permission removed": func(facts *AuthorizationFacts) {
			facts.CurrentPermissions = facts.CurrentPermissions[1:]
		},
		"permission injected": func(facts *AuthorizationFacts) {
			facts.CurrentPermissions = append(facts.CurrentPermissions, "unknown.permission")
		},
		"action":           func(facts *AuthorizationFacts) { facts.Action = "unknown.action" },
		"resource":         func(facts *AuthorizationFacts) { facts.Resource = "unknown_resource" },
		"purpose":          func(facts *AuthorizationFacts) { facts.Purpose = "unknown_purpose" },
		"assurance":        func(facts *AuthorizationFacts) { facts.Assurance = Assurance("unknown") },
		"session version":  func(facts *AuthorizationFacts) { facts.SessionAuthorizationVersion = 0 },
		"stale authority":  func(facts *AuthorizationFacts) { facts.CurrentAuthorizationVersion++ },
		"resource version": func(facts *AuthorizationFacts) { facts.ResourceVersion = 0 },
		"resource state":   func(facts *AuthorizationFacts) { facts.ResourceStatus = "disabled" },
		"tenant missing":   func(facts *AuthorizationFacts) { facts.ResourceTenantID = identifierZero() },
		"field":            func(facts *AuthorizationFacts) { facts.Field = "unknown_field" },
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			candidate := valid
			candidate.CurrentPermissions = append([]string(nil), valid.CurrentPermissions...)
			mutate(&candidate)
			if decision := policy.Evaluate(candidate); decision.Effect != AuthorizationDeny {
				t.Fatalf("mutated decision=%+v", decision)
			}
		})
	}
}

func TestAuthorizationPolicyConcealsCrossTenantAfterActorAuthorization(t *testing.T) {
	t.Parallel()
	policy := DefaultAuthorizationPolicy()
	facts := validMemberListAuthorizationFacts(t, policy, "merchant_operator")
	facts.ResourceTenantID = mustTestID(t, "ten", 302)
	decision := policy.Evaluate(facts)
	if decision.Effect != AuthorizationConceal || decision.Reason != "tenant_concealed" {
		t.Fatalf("cross-tenant decision=%+v", decision)
	}
}

func TestAuthorizationPolicyFieldMaskRequiresPermissionAndPurpose(t *testing.T) {
	t.Parallel()
	policy := DefaultAuthorizationPolicy()
	for _, test := range []struct {
		name, role, purpose string
		want                FieldAccess
	}{
		{"ordinary role remains masked", "merchant_operator", "organization_administration", FieldAccessMasked},
		{"sensitive permission without purpose remains masked", "merchant_security_admin", "self_service", FieldAccessMasked},
		{"permission and purpose reveal", "merchant_security_admin", "organization_administration", FieldAccessReveal},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			facts := validMemberListAuthorizationFacts(t, policy, test.role)
			facts.Field = "email_hint"
			facts.Purpose = test.purpose
			decision := policy.Evaluate(facts)
			if decision.Effect != AuthorizationAllow || decision.FieldAccess != test.want {
				t.Fatalf("field decision=%+v want=%s", decision, test.want)
			}
		})
	}
}

func validMemberListAuthorizationFacts(
	t *testing.T,
	policy *AuthorizationPolicy,
	roleID string,
) AuthorizationFacts {
	t.Helper()
	role := policy.roles[roleID]
	tenantID := mustTestID(t, "ten", 301)
	return AuthorizationFacts{
		PrincipalID: mustTestID(t, "usr", 301), PrincipalType: "merchant",
		Population: PopulationMerchant, TenantID: tenantID, ResourceTenantID: tenantID,
		Role: roleID, CurrentPermissions: append([]string(nil), role.permissions...),
		Action: "organization.members.list", Resource: "organization_member",
		Purpose: "self_service", Assurance: AssuranceBaseline,
		SessionAuthorizationVersion: 7, CurrentAuthorizationVersion: 7,
		ResourceVersion: 3, ResourceStatus: "active",
	}
}

func sortedSetKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedCopy(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func identifierZero() identifier.ID {
	return identifier.ID{}
}
