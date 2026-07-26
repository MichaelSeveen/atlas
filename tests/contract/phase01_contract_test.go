package contract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPhase01OpenAPISurface(t *testing.T) {
	document := readOpenAPIDocument(t)
	paths := objectAt(t, document, "paths")
	expected := map[string]map[string]string{
		"/v1/auth/login":             {"get": "beginBrowserLogin"},
		"/v1/auth/callback":          {"get": "completeBrowserLogin"},
		"/v1/logout":                 {"post": "logoutCurrentSession"},
		"/v1/me":                     {"get": "getCurrentPrincipal"},
		"/v1/sessions":               {"get": "listPrincipalSessions"},
		"/v1/sessions/{session_id}":  {"delete": "revokePrincipalSession"},
		"/v1/sessions/revoke-all":    {"post": "revokeAllPrincipalSessions"},
		"/v1/step-up/challenges":     {"post": "createStepUpChallenge"},
		"/v1/me/active-organization": {"put": "setActiveOrganization"},
		"/v1/organizations":          {"get": "listPrincipalOrganizations"},
		"/v1/organizations/{organization_id}/members":             {"get": "listOrganizationMembers"},
		"/v1/organizations/{organization_id}/invitations":         {"post": "createOrganizationInvitation"},
		"/v1/organization-invitations/{invitation_id}/acceptance": {"post": "acceptOrganizationInvitation"},
		"/v1/organizations/{organization_id}/members/{member_id}": {"patch": "updateOrganizationMember", "delete": "revokeOrganizationMember"},
		"/v1/api-credentials":                                     {"get": "listAPICredentials", "post": "createAPICredential"},
		"/v1/api-credentials/{credential_id}/rotate":              {"post": "rotateAPICredential"},
		"/v1/api-credentials/{credential_id}":                     {"delete": "revokeAPICredential"},
		"/v1/approvals":                                           {"get": "listApprovals", "post": "createApproval"},
		"/v1/approvals/{approval_id}":                             {"get": "getApproval"},
		"/v1/approvals/{approval_id}/decisions":                   {"post": "decideApproval"},
		"/v1/approvals/{approval_id}/executions":                  {"post": "executeApproval"},
		"/v1/approvals/{approval_id}/cancellations":               {"post": "cancelApproval"},
	}

	for path, methods := range expected {
		pathItem, ok := paths[path].(map[string]any)
		if !ok {
			t.Errorf("Phase 01 path %s is absent", path)
			continue
		}
		for method, operationID := range methods {
			operation, ok := pathItem[method].(map[string]any)
			if !ok {
				t.Errorf("Phase 01 operation %s %s is absent", strings.ToUpper(method), path)
				continue
			}
			if got := stringAt(t, operation, "operationId"); got != operationID {
				t.Errorf("%s %s operationId = %q, want %q", strings.ToUpper(method), path, got, operationID)
			}
		}
	}
}

func TestPhase01CanonicalPhaseSurfaceMatchesOpenAPI(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "02-phases", "PHASE-01_IDENTITY_ACCESS_TENANCY.md"))
	if err != nil {
		t.Fatal(err)
	}
	section := strings.SplitN(string(content), "## Frontend requirements", 2)[0]
	matches := regexp.MustCompile("(?m)^- `(GET|POST|PUT|PATCH|DELETE) (/v1/[^`]+)`$").FindAllStringSubmatch(section, -1)
	if len(matches) == 0 {
		t.Fatal("canonical Phase 01 API surface is empty")
	}
	document := readOpenAPIDocument(t)
	paths := objectAt(t, document, "paths")
	for _, match := range matches {
		pathItem, ok := paths[match[2]].(map[string]any)
		if !ok {
			t.Errorf("canonical phase operation %s %s has no OpenAPI path", match[1], match[2])
			continue
		}
		if _, ok := pathItem[strings.ToLower(match[1])].(map[string]any); !ok {
			t.Errorf("canonical phase operation %s %s has no OpenAPI operation", match[1], match[2])
		}
	}
}

func TestPhase01CookieMutationsRequireCSRF(t *testing.T) {
	document := readOpenAPIDocument(t)
	paths := objectAt(t, document, "paths")
	mutations := []struct {
		method string
		path   string
	}{
		{"post", "/v1/logout"},
		{"delete", "/v1/sessions/{session_id}"},
		{"post", "/v1/sessions/revoke-all"},
		{"post", "/v1/step-up/challenges"},
		{"put", "/v1/me/active-organization"},
		{"post", "/v1/organizations/{organization_id}/invitations"},
		{"post", "/v1/organization-invitations/{invitation_id}/acceptance"},
		{"patch", "/v1/organizations/{organization_id}/members/{member_id}"},
		{"delete", "/v1/organizations/{organization_id}/members/{member_id}"},
		{"post", "/v1/api-credentials"},
		{"post", "/v1/api-credentials/{credential_id}/rotate"},
		{"delete", "/v1/api-credentials/{credential_id}"},
		{"post", "/v1/approvals"},
		{"post", "/v1/approvals/{approval_id}/decisions"},
		{"post", "/v1/approvals/{approval_id}/executions"},
		{"post", "/v1/approvals/{approval_id}/cancellations"},
	}
	for _, mutation := range mutations {
		pathItem := objectAt(t, paths, mutation.path)
		operation := objectAt(t, pathItem, mutation.method)
		parameters, ok := operation["parameters"].([]any)
		if !ok {
			t.Errorf("%s %s has no operation parameters", strings.ToUpper(mutation.method), mutation.path)
			continue
		}
		found := false
		for _, raw := range parameters {
			parameter, ok := raw.(map[string]any)
			if ok && parameter["$ref"] == "#/components/parameters/XAtlasCSRFToken" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s %s does not require X-Atlas-CSRF-Token", strings.ToUpper(mutation.method), mutation.path)
		}
	}
}

func TestPhase01MachineCredentialIsLeastPrivilege(t *testing.T) {
	document := readOpenAPIDocument(t)
	schemes := objectAt(t, objectAt(t, document, "components"), "securitySchemes")
	machine := objectAt(t, schemes, "merchantApiKey")
	if got := stringAt(t, machine, "name"); got != "Authorization" {
		t.Errorf("merchantApiKey header = %q", got)
	}
	me := objectAt(t, objectAt(t, objectAt(t, document, "paths"), "/v1/me"), "get")
	security, ok := me["security"].([]any)
	if !ok || len(security) != 2 {
		t.Fatalf("GET /v1/me security alternatives = %#v, want BFF session or merchant API key", me["security"])
	}
	source := readOpenAPI(t)
	if !strings.Contains(source, "The `identity:read` scope") {
		t.Error("merchant API key scope is not bounded to Phase 01 identity read")
	}
}

func TestPhase01SchemasAreClosedAndOneTimeSecretsAreNotReplayable(t *testing.T) {
	source := readOpenAPI(t)
	for _, schema := range []string{
		"Session",
		"RevokeAllSessionsRequest",
		"StepUpChallengeRequest",
		"StepUpChallenge",
		"SetActiveOrganizationRequest",
		"Organization",
		"OrganizationMember",
		"CreateOrganizationInvitationRequest",
		"OrganizationInvitation",
		"OrganizationInvitationCreated",
		"AcceptOrganizationInvitationRequest",
		"UpdateOrganizationMemberRequest",
		"APICredential",
		"CreateAPICredentialRequest",
		"APICredentialCreated",
		"MembershipRoleChangeApprovalRequest",
		"CreateApprovalRequest",
		"Approval",
		"ApprovalDecisionRequest",
		"ApprovalCancellationRequest",
	} {
		if !strings.Contains(componentBlock(t, source, schema), "additionalProperties: false") {
			t.Errorf("%s schema permits undeclared fields", schema)
		}
	}
	for _, schema := range []string{"OrganizationInvitationCreated", "APICredentialCreated"} {
		block := componentBlock(t, source, schema)
		for _, fragment := range []string{"type: [string, 'null']", "secret_disclosed:", "null", "replay"} {
			if !strings.Contains(strings.ToLower(block), strings.ToLower(fragment)) {
				t.Errorf("%s does not close one-time secret replay rule %q", schema, fragment)
			}
		}
	}
}

func TestPhase01IdentityAccessPolicyIsClosedAndConsistent(t *testing.T) {
	policy := readJSONDocument(t, identityAccessPolicyPath(t))
	if got := intAt(t, policy, "schema_version"); got != 1 {
		t.Errorf("schema_version = %d", got)
	}
	for key, want := range map[string]string{
		"decision":              "ADR-0014",
		"phase":                 "PHASE-01_IDENTITY_ACCESS_TENANCY",
		"default_authorization": "deny",
		"authoritative_store":   "postgresql",
	} {
		if got := stringAt(t, policy, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	sessions := objectAt(t, policy, "sessions")
	cookie := objectAt(t, sessions, "cookie")
	csrf := objectAt(t, sessions, "csrf")
	assertString(t, cookie, "name", "__Host-atlas_session")
	assertString(t, cookie, "domain_attribute", "forbidden")
	assertString(t, csrf, "header", "X-Atlas-CSRF-Token")
	assertString(t, csrf, "browser_storage", "memory-only")
	assertString(t, sessions, "upstream_token_retention", "none-after-validated-callback")
	if got := intAt(t, sessions, "old_session_grace_seconds"); got != 0 {
		t.Errorf("old_session_grace_seconds = %d, want zero", got)
	}
	if got := intAt(t, sessions, "maximum_concurrent_sessions_per_principal"); got != 10 {
		t.Errorf("maximum concurrent sessions = %d, want 10", got)
	}
	if got := intAt(t, sessions, "oidc_clock_skew_seconds"); got != 60 {
		t.Errorf("OIDC clock skew = %d seconds, want 60", got)
	}

	permissions := stringSetAt(t, policy, "permissions")
	if len(permissions) < 20 {
		t.Fatalf("permission catalogue unexpectedly small: %d", len(permissions))
	}
	roles := arrayAt(t, policy, "roles")
	roleIDs := map[string]bool{}
	for _, raw := range roles {
		role, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("role entry has type %T", raw)
		}
		id := stringAt(t, role, "id")
		if roleIDs[id] {
			t.Errorf("duplicate role %s", id)
		}
		roleIDs[id] = true
		for permission := range stringSetAt(t, role, "permissions") {
			if !permissions[permission] {
				t.Errorf("role %s references undeclared permission %s", id, permission)
			}
		}
	}
	for _, raw := range roles {
		role := raw.(map[string]any)
		for delegated := range stringSetAt(t, role, "delegable_roles") {
			if !roleIDs[delegated] {
				t.Errorf("role %s delegates undeclared role %s", stringAt(t, role, "id"), delegated)
			}
		}
	}

	authorization := objectAt(t, policy, "authorization")
	assertString(t, authorization, "decision_cache", "none-in-phase-01")
	assertString(t, authorization, "redis_authorization_truth", "forbidden")
	assertString(t, authorization, "list_filter_order", "authorize-before-count-cursor-suggestion")

	stepUp := objectAt(t, policy, "step_up")
	if got := intAt(t, stepUp, "freshness_minutes"); got != 5 {
		t.Errorf("step-up freshness = %d minutes, want 5", got)
	}
	if len(arrayAt(t, stepUp, "reserved_deny_until_owner_phase")) != 6 {
		t.Error("future high-risk actions are not explicitly reserved-deny")
	}
	openAPI := readOpenAPIDocument(t)
	schemas := objectAt(t, objectAt(t, openAPI, "components"), "schemas")
	stepUpSchema := objectAt(t, objectAt(t, objectAt(t, schemas, "StepUpChallengeRequest"), "properties"), "action")
	if got, want := sortedKeys(stringSetAt(t, stepUpSchema, "enum")), sortedKeys(stringSetAt(t, stepUp, "current_actions")); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("OpenAPI step-up actions = %v, policy actions = %v", got, want)
	}
	purposeSchema := objectAt(t, schemas, "Purpose")
	if got, want := sortedKeys(stringSetAt(t, purposeSchema, "enum")), sortedKeys(stringSetAt(t, policy, "purposes")); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("OpenAPI purposes = %v, policy purposes = %v", got, want)
	}

	approvals := objectAt(t, policy, "approvals")
	assertString(t, approvals, "owner", "operations")
	assertString(t, approvals, "payload_canonicalization", "rfc8785")
	assertString(t, approvals, "event_publication", "none-in-phase-01")
	assertString(t, approvals, "eligible_checker_policy", "dynamic-at-decision-and-execution")
	if same, ok := approvals["same_principal_prohibited"].(bool); !ok || !same {
		t.Error("maker/checker identity separation is not mandatory")
	}

	credentials := objectAt(t, policy, "api_credentials")
	assertString(t, credentials, "owner", "identity")
	assertString(t, credentials, "stored_verifier", "sha256")
	assertString(t, credentials, "idempotent_replay_secret", "redacted")
	assertString(t, credentials, "entropy_failure", "deny-creation-or-rotation")
	assertString(t, credentials, "audience", "atlas-api")
	assertString(t, credentials, "environment_binding", "exact-configured-environment")
	if got := intAt(t, credentials, "rotation_overlap_minutes"); got != 10 {
		t.Errorf("credential overlap = %d minutes, want 10", got)
	}

	audit := objectAt(t, policy, "audit")
	assertString(t, audit, "classification", "confidential-security")
	assertString(t, audit, "retention", "retain-through-phase-01;phase-11-may-pseudonymize-identifiers-but-not-delete-security-facts")
	assertString(t, audit, "read_permission", "audit.events.read")
	assertString(t, audit, "write_path", "synchronous-application-service")
	assertString(t, audit, "privileged_mutation_atomicity", "same-postgresql-transaction")
	assertString(t, audit, "asyncapi_publication", "not-implemented-in-phase-01")

	client := objectAt(t, policy, "generated_client")
	assertString(t, client, "runtime", "bun")
	assertString(t, client, "source", "docs/atlas-prd/03-contracts/openapi.yaml")

	adr, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "06-governance", "adrs", "0014-phase-01-identity-access-contract-boundary.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{"exact configured origins", "`Vary: Origin`", "Wildcard credentialed CORS is forbidden"} {
		if !strings.Contains(string(adr), rule) {
			t.Errorf("ADR 0014 is missing browser CORS rule %q", rule)
		}
	}
}

func TestPhase01ErrorCodesAreCatalogued(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", "ERROR_CATALOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"OIDC_TRANSACTION_INVALID",
		"CSRF_VALIDATION_FAILED",
		"TENANT_CONTEXT_INVALID",
		"INVITATION_INVALID_OR_EXPIRED",
		"INVITATION_ALREADY_USED",
		"MAKER_CANNOT_APPROVE",
		"APPROVAL_STATE_CONFLICT",
		"APPROVAL_EXECUTION_FAILED",
		"API_CREDENTIAL_REVOKED",
		"API_CREDENTIAL_EXPIRED",
		"API_CREDENTIAL_SECRET_NOT_RECOVERABLE",
	} {
		if !strings.Contains(string(content), "`"+code+"`") {
			t.Errorf("Phase 01 error code %s is not catalogued", code)
		}
	}
}

func TestPhase01DoesNotPublishUnjustifiedEvents(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", "asyncapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{
		"identity.session.revoked.v1",
		"identity.organization.membership.changed.v1",
		"identity.api_credential.rotated.v1",
		"operations.approval.decided.v1",
	} {
		if strings.Contains(string(content), event) {
			t.Errorf("Phase 01 event %s was published without a consumer/delivery decision", event)
		}
	}
}

func readOpenAPIDocument(t *testing.T) map[string]any {
	t.Helper()
	content, err := os.ReadFile(openAPIPath(t))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatalf("parse OpenAPI: %v", err)
	}
	return document
}

func identityAccessPolicyPath(t *testing.T) string {
	t.Helper()
	if override := os.Getenv("ATLAS_IDENTITY_ACCESS_POLICY_PATH"); override != "" {
		return override
	}
	return filepath.Join(repositoryRoot(t), "docs", "atlas-prd", "03-contracts", "identity-access-policy.json")
}

func readJSONDocument(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return document
}

func objectAt(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key].(map[string]any)
	if !ok {
		t.Fatalf("%s has type %T, want object", key, object[key])
	}
	return value
}

func arrayAt(t *testing.T, object map[string]any, key string) []any {
	t.Helper()
	value, ok := object[key].([]any)
	if !ok {
		t.Fatalf("%s has type %T, want array", key, object[key])
	}
	return value
}

func stringAt(t *testing.T, object map[string]any, key string) string {
	t.Helper()
	value, ok := object[key].(string)
	if !ok {
		t.Fatalf("%s has type %T, want string", key, object[key])
	}
	return value
}

func intAt(t *testing.T, object map[string]any, key string) int {
	t.Helper()
	switch value := object[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		t.Fatalf("%s has type %T, want number", key, object[key])
		return 0
	}
}

func stringSetAt(t *testing.T, object map[string]any, key string) map[string]bool {
	t.Helper()
	result := map[string]bool{}
	for _, raw := range arrayAt(t, object, key) {
		value, ok := raw.(string)
		if !ok {
			t.Fatalf("%s contains %T, want string", key, raw)
		}
		if result[value] {
			t.Errorf("%s contains duplicate %s", key, value)
		}
		result[value] = true
	}
	return result
}

func assertString(t *testing.T, object map[string]any, key, want string) {
	t.Helper()
	if got := stringAt(t, object, key); got != want {
		t.Errorf("%s = %q, want %q", key, got, want)
	}
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
