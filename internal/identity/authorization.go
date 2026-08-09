package identity

import (
	"sort"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

// AuthorizationEffect is the bounded server-side authorization outcome.
type AuthorizationEffect string

const (
	AuthorizationDeny    AuthorizationEffect = "deny"
	AuthorizationAllow   AuthorizationEffect = "allow"
	AuthorizationConceal AuthorizationEffect = "conceal"
)

// FieldAccess is the bounded field-level disclosure outcome.
type FieldAccess string

const (
	FieldAccessNone   FieldAccess = "none"
	FieldAccessMasked FieldAccess = "masked"
	FieldAccessReveal FieldAccess = "reveal"
)

// AuthorizationFacts contains only authoritative facts needed for one decision.
// Callers must obtain CurrentPermissions and CurrentAuthorizationVersion from
// PostgreSQL in the same transaction or locked protocol as the protected work.
type AuthorizationFacts struct {
	PrincipalID                 identifier.ID
	PrincipalType               string
	Population                  Population
	TenantID                    identifier.ID
	ResourceTenantID            identifier.ID
	Role                        string
	CurrentPermissions          []string
	Action                      string
	Resource                    string
	Field                       string
	Purpose                     string
	Assurance                   Assurance
	SessionAuthorizationVersion int64
	CurrentAuthorizationVersion int64
	ResourceVersion             int64
	ResourceStatus              string
}

// AuthorizationDecision exposes a bounded reason suitable for metrics and
// Audit. It never contains policy internals, identifiers, or sensitive values.
type AuthorizationDecision struct {
	Effect      AuthorizationEffect
	FieldAccess FieldAccess
	Reason      string
}

type authorizationRole struct {
	population  Population
	permissions []string
	enabled     bool
}

type authorizationAction struct {
	permission       string
	resource         string
	populations      map[Population]struct{}
	purposes         map[string]struct{}
	minimumAssurance Assurance
	tenantScoped     bool
	resourceStatuses map[string]struct{}
}

type authorizationFieldRule struct {
	revealPermission string
	requiredPurpose  string
}

// AuthorizationPolicy is the immutable Phase 01 deny-default evaluator input.
// The canonical JSON policy and database seed are checked against this runtime
// catalogue by tests; any drift denies at runtime and fails verification.
type AuthorizationPolicy struct {
	permissions map[string]struct{}
	roles       map[string]authorizationRole
	purposes    map[string]struct{}
	actions     map[string]authorizationAction
	fields      map[string]authorizationFieldRule
}

// DefaultAuthorizationPolicy returns an isolated copy of the source-controlled
// Phase 01 runtime catalogue.
func DefaultAuthorizationPolicy() *AuthorizationPolicy {
	policy := &AuthorizationPolicy{
		permissions: stringSet(
			"identity.me.read",
			"identity.sessions.read",
			"identity.sessions.revoke_self",
			"identity.sessions.revoke_all_self",
			"identity.sessions.revoke_admin",
			"identity.step_up.create",
			"organization.list",
			"organization.members.read",
			"organization.members.sensitive.read",
			"organization.invitations.create",
			"organization.members.roles.update",
			"organization.members.remove",
			"organization.active.switch",
			"approvals.read",
			"approvals.create",
			"approvals.decide",
			"approvals.execute",
			"approvals.cancel",
			"api_credentials.read",
			"api_credentials.create",
			"api_credentials.rotate",
			"api_credentials.revoke",
			"audit.events.read",
		),
		roles: map[string]authorizationRole{
			"customer_self": rolePolicy(PopulationCustomer, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create"),
			"merchant_viewer": rolePolicy(PopulationMerchant, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "organization.list",
				"organization.members.read", "organization.active.switch", "approvals.read",
				"api_credentials.read"),
			"merchant_operator": rolePolicy(PopulationMerchant, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "organization.list",
				"organization.members.read", "organization.active.switch", "approvals.read",
				"approvals.create", "approvals.execute", "api_credentials.read"),
			"merchant_admin": rolePolicy(PopulationMerchant, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "organization.list",
				"organization.members.read", "organization.invitations.create",
				"organization.members.roles.update", "organization.members.remove",
				"organization.active.switch", "approvals.read", "approvals.create",
				"approvals.execute", "approvals.cancel", "api_credentials.read",
				"api_credentials.create", "api_credentials.rotate", "api_credentials.revoke"),
			"merchant_security_admin": rolePolicy(PopulationMerchant, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "organization.list",
				"organization.members.read", "organization.members.sensitive.read",
				"organization.invitations.create", "organization.members.roles.update",
				"organization.members.remove", "organization.active.switch", "approvals.read",
				"approvals.create", "approvals.decide", "approvals.execute", "approvals.cancel",
				"api_credentials.read", "api_credentials.create", "api_credentials.rotate",
				"api_credentials.revoke", "audit.events.read"),
			"support": rolePolicy(PopulationWorkforce, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "organization.list",
				"organization.members.read"),
			"operations": rolePolicy(PopulationWorkforce, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "organization.list",
				"organization.members.read", "approvals.read", "approvals.create", "approvals.execute"),
			"risk_analyst": rolePolicy(PopulationWorkforce, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "approvals.read",
				"approvals.decide"),
			"finance_operator": rolePolicy(PopulationWorkforce, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "approvals.read",
				"approvals.decide"),
			"security_auditor": rolePolicy(PopulationWorkforce, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.step_up.create", "approvals.read",
				"audit.events.read"),
			"platform_administrator": rolePolicy(PopulationWorkforce, true,
				"identity.me.read", "identity.sessions.read", "identity.sessions.revoke_self",
				"identity.sessions.revoke_all_self", "identity.sessions.revoke_admin",
				"identity.step_up.create", "organization.list", "organization.members.read",
				"organization.members.sensitive.read", "approvals.read", "approvals.decide",
				"audit.events.read"),
			"break_glass":      rolePolicy(PopulationWorkforce, false),
			"merchant_machine": rolePolicy(Population("machine"), true, "identity.me.read"),
		},
		purposes: stringSet(
			"self_service",
			"organization_administration",
			"approval_review",
			"credential_management",
			"security_review",
		),
		actions: map[string]authorizationAction{
			"organization.members.list": {
				permission: "organization.members.read", resource: "organization_member",
				populations:      populationSet(PopulationMerchant),
				purposes:         stringSet("self_service", "organization_administration"),
				minimumAssurance: AssuranceBaseline, tenantScoped: true,
				resourceStatuses: stringSet("active"),
			},
		},
		fields: map[string]authorizationFieldRule{
			"organization_member.email_hint": {
				revealPermission: "organization.members.sensitive.read",
				requiredPurpose:  "organization_administration",
			},
			"session.network_hint": {
				revealPermission: "identity.sessions.revoke_admin",
				requiredPurpose:  "security_review",
			},
			"api_credential.last_used_network_hint": {
				revealPermission: "api_credentials.read",
				requiredPurpose:  "credential_management",
			},
		},
	}
	return policy
}

// Evaluate denies every unknown, inconsistent, stale, or policy-divergent fact.
func (policy *AuthorizationPolicy) Evaluate(facts AuthorizationFacts) AuthorizationDecision {
	deny := func(reason string) AuthorizationDecision {
		return AuthorizationDecision{Effect: AuthorizationDeny, FieldAccess: FieldAccessNone, Reason: reason}
	}
	if policy == nil || facts.PrincipalID.IsZero() || facts.PrincipalType != string(facts.Population) {
		return deny("invalid_principal")
	}
	role, knownRole := policy.roles[facts.Role]
	if !knownRole || !role.enabled {
		return deny("unknown_or_disabled_role")
	}
	if role.population != facts.Population {
		return deny("role_population_mismatch")
	}
	if !samePermissionSet(facts.CurrentPermissions, role.permissions, policy.permissions) {
		return deny("policy_binding_mismatch")
	}
	action, knownAction := policy.actions[facts.Action]
	if !knownAction {
		return deny("unknown_action")
	}
	if _, allowedPopulation := action.populations[facts.Population]; !allowedPopulation ||
		facts.Resource != action.resource {
		return deny("action_resource_mismatch")
	}
	if !containsSorted(role.permissions, action.permission) {
		return deny("permission_denied")
	}
	if _, knownPurpose := policy.purposes[facts.Purpose]; !knownPurpose {
		return deny("unknown_purpose")
	}
	if _, allowedPurpose := action.purposes[facts.Purpose]; !allowedPurpose {
		return deny("purpose_denied")
	}
	if !assuranceSatisfies(facts.Assurance, action.minimumAssurance) {
		return deny("assurance_denied")
	}
	if facts.SessionAuthorizationVersion < 1 ||
		facts.CurrentAuthorizationVersion != facts.SessionAuthorizationVersion {
		return deny("stale_authority")
	}
	if action.tenantScoped {
		if facts.TenantID.IsZero() || facts.ResourceTenantID.IsZero() {
			return deny("invalid_tenant")
		}
		if facts.TenantID != facts.ResourceTenantID {
			return AuthorizationDecision{
				Effect: AuthorizationConceal, FieldAccess: FieldAccessNone, Reason: "tenant_concealed",
			}
		}
	}
	if facts.ResourceVersion < 1 {
		return deny("invalid_resource_version")
	}
	if _, allowedStatus := action.resourceStatuses[facts.ResourceStatus]; !allowedStatus {
		return deny("resource_state_denied")
	}
	if facts.Field == "" {
		return AuthorizationDecision{Effect: AuthorizationAllow, FieldAccess: FieldAccessNone, Reason: "allowed"}
	}
	fieldRule, knownField := policy.fields[facts.Resource+"."+facts.Field]
	if !knownField {
		return deny("unknown_field")
	}
	fieldAccess := FieldAccessMasked
	if facts.Purpose == fieldRule.requiredPurpose &&
		containsSorted(role.permissions, fieldRule.revealPermission) {
		fieldAccess = FieldAccessReveal
	}
	return AuthorizationDecision{Effect: AuthorizationAllow, FieldAccess: fieldAccess, Reason: "allowed"}
}

func rolePolicy(population Population, enabled bool, permissions ...string) authorizationRole {
	copied := append([]string(nil), permissions...)
	sort.Strings(copied)
	return authorizationRole{population: population, permissions: copied, enabled: enabled}
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func populationSet(values ...Population) map[Population]struct{} {
	result := make(map[Population]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func samePermissionSet(actual, expected []string, known map[string]struct{}) bool {
	if len(actual) != len(expected) {
		return false
	}
	seen := make(map[string]struct{}, len(actual))
	for _, permission := range actual {
		if _, valid := known[permission]; !valid {
			return false
		}
		if _, duplicate := seen[permission]; duplicate {
			return false
		}
		seen[permission] = struct{}{}
	}
	for _, permission := range expected {
		if _, found := seen[permission]; !found {
			return false
		}
	}
	return true
}

func containsSorted(values []string, wanted string) bool {
	index := sort.SearchStrings(values, wanted)
	return index < len(values) && values[index] == wanted
}
