package identity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type SeedRole struct {
	ID             string   `json:"id"`
	Population     string   `json:"population"`
	Permissions    []string `json:"permissions"`
	DelegableRoles []string `json:"delegable_roles"`
	StandingStatus string   `json:"standing_status,omitempty"`
}

type SeedOrganization struct {
	TenantID           string `json:"tenant_id"`
	OrganizationType   string `json:"organization_type"`
	DisplayName        string `json:"display_name"`
	NormalizedName     string `json:"normalized_name"`
	ConfusableSkeleton string `json:"confusable_skeleton"`
}

type SeedPrincipal struct {
	PrincipalID   string `json:"principal_id"`
	PrincipalType string `json:"principal_type"`
	DisplayName   string `json:"display_name"`
	PersonAnchor  string `json:"person_anchor,omitempty"`
}

type SeedExternalSubject struct {
	ExternalSubjectID string `json:"external_subject_id"`
	PrincipalID       string `json:"principal_id"`
	Population        string `json:"population"`
	Issuer            string `json:"issuer"`
	Subject           string `json:"subject"`
}

type SeedMembership struct {
	MembershipID string `json:"membership_id"`
	TenantID     string `json:"tenant_id"`
	PrincipalID  string `json:"principal_id"`
	RoleID       string `json:"role_id"`
}

type SeedPrincipalRole struct {
	PrincipalRoleID string `json:"principal_role_id"`
	PrincipalID     string `json:"principal_id"`
	RoleID          string `json:"role_id"`
}

type SeedSession struct {
	SessionID         string `json:"session_id"`
	PrincipalID       string `json:"principal_id"`
	TenantID          string `json:"tenant_id,omitempty"`
	GlobalScope       string `json:"global_scope,omitempty"`
	VerifierSHA256    string `json:"verifier_sha256"`
	Assurance         string `json:"assurance"`
	Status            string `json:"status"`
	IdleExpiresAt     string `json:"idle_expires_at"`
	AbsoluteExpiresAt string `json:"absolute_expires_at"`
	RevokedAt         string `json:"revoked_at,omitempty"`
}

type SeedAuditEvent struct {
	AuditEventID        string `json:"audit_event_id"`
	ActorID             string `json:"actor_id"`
	ActorType           string `json:"actor_type"`
	TenantID            string `json:"tenant_id,omitempty"`
	GlobalScope         string `json:"global_scope,omitempty"`
	SessionAssurance    string `json:"session_assurance"`
	Action              string `json:"action"`
	TargetType          string `json:"target_type"`
	TargetID            string `json:"target_id"`
	DecisionID          string `json:"decision_id"`
	Decision            string `json:"decision"`
	ReasonCode          string `json:"reason_code"`
	CorrelationID       string `json:"correlation_id"`
	ApprovalID          string `json:"approval_id,omitempty"`
	SafeBeforeReference string `json:"safe_before_reference,omitempty"`
	SafeAfterReference  string `json:"safe_after_reference,omitempty"`
}

type SeedManifest struct {
	SchemaVersion    int                   `json:"schema_version"`
	SeedID           string                `json:"seed_id"`
	VirtualTime      string                `json:"virtual_time"`
	PolicySHA256     string                `json:"policy_sha256"`
	Permissions      []string              `json:"permissions"`
	Roles            []SeedRole            `json:"roles"`
	Organizations    []SeedOrganization    `json:"organizations"`
	Principals       []SeedPrincipal       `json:"principals"`
	ExternalSubjects []SeedExternalSubject `json:"external_subjects"`
	Memberships      []SeedMembership      `json:"memberships"`
	PrincipalRoles   []SeedPrincipalRole   `json:"principal_roles"`
	Sessions         []SeedSession         `json:"sessions"`
	AuditEvents      []SeedAuditEvent      `json:"audit_events"`
}

var (
	hexDigestPattern   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	localIssuerPattern = regexp.MustCompile(`^http://keycloak:8080/realms/atlas-(customer|merchant|workforce)-local$`)
)

// LoadSeedManifest validates the deterministic product seed against the canonical access policy.
func LoadSeedManifest(seedPath, policyPath string) (SeedManifest, string, error) {
	// #nosec G304 -- repository tests and operators supply repository-owned policy and seed paths.
	content, err := os.ReadFile(seedPath)
	if err != nil {
		return SeedManifest{}, "", fmt.Errorf("read identity seed: %w", err)
	}
	if len(content) == 0 || len(content) > 1<<20 {
		return SeedManifest{}, "", errors.New("identity seed has unsafe size")
	}
	var manifest SeedManifest
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return SeedManifest{}, "", fmt.Errorf("decode identity seed: %w", err)
	}
	if err := rejectTrailingSeedJSON(decoder); err != nil {
		return SeedManifest{}, "", err
	}

	// #nosec G304 -- see seed path rationale above.
	policyContent, err := os.ReadFile(policyPath)
	if err != nil {
		return SeedManifest{}, "", fmt.Errorf("read identity policy: %w", err)
	}
	policyDigest := sha256.Sum256(policyContent)
	if manifest.PolicySHA256 != hex.EncodeToString(policyDigest[:]) {
		return SeedManifest{}, "", errors.New("identity seed is not bound to the canonical access policy")
	}
	var policy struct {
		Permissions []string   `json:"permissions"`
		Roles       []SeedRole `json:"roles"`
	}
	if err := json.Unmarshal(policyContent, &policy); err != nil {
		return SeedManifest{}, "", fmt.Errorf("decode identity policy: %w", err)
	}
	if !reflect.DeepEqual(manifest.Permissions, policy.Permissions) ||
		!reflect.DeepEqual(manifest.Roles, policy.Roles) {
		return SeedManifest{}, "", errors.New("identity seed role catalogue diverges from the canonical access policy")
	}
	if err := manifest.Validate(); err != nil {
		return SeedManifest{}, "", err
	}
	digest := sha256.Sum256(content)
	return manifest, hex.EncodeToString(digest[:]), nil
}

func rejectTrailingSeedJSON(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("identity seed contains trailing JSON")
	}
	return nil
}

// Validate checks fixed synthetic identities, referential integrity, and recovery canaries.
func (manifest SeedManifest) Validate() error {
	virtualTime, err := time.Parse(time.RFC3339, manifest.VirtualTime)
	if err != nil || manifest.SchemaVersion != 1 ||
		manifest.SeedID != "atlas-phase01-identity-v1" ||
		manifest.VirtualTime != "2026-07-26T00:00:00Z" ||
		!hexDigestPattern.MatchString(manifest.PolicySHA256) {
		return errors.New("unsupported identity seed identity")
	}
	if len(manifest.Permissions) != 23 || len(manifest.Roles) != 13 ||
		len(manifest.Organizations) != 2 || len(manifest.Principals) != 3 ||
		len(manifest.ExternalSubjects) != 3 || len(manifest.Memberships) != 2 ||
		len(manifest.PrincipalRoles) != 1 || len(manifest.Sessions) != 1 ||
		len(manifest.AuditEvents) != 1 {
		return errors.New("identity seed closed inventory is incomplete")
	}

	seenIDs := make(map[string]struct{})
	requireID := func(value, prefix string) error {
		id, parseErr := identifier.Parse(value)
		if parseErr != nil || id.Prefix() != prefix {
			return errors.New("identity seed contains an invalid opaque identifier")
		}
		if _, duplicate := seenIDs[value]; duplicate {
			return errors.New("identity seed contains a duplicate opaque identifier")
		}
		seenIDs[value] = struct{}{}
		return nil
	}

	tenants := make(map[string]string, len(manifest.Organizations))
	for _, organization := range manifest.Organizations {
		if err := requireID(organization.TenantID, "ten"); err != nil {
			return err
		}
		if organization.DisplayName == "" || organization.NormalizedName == "" ||
			organization.ConfusableSkeleton == "" ||
			(organization.OrganizationType != "customer" && organization.OrganizationType != "merchant") {
			return errors.New("identity seed organization naming fixture is incomplete")
		}
		tenants[organization.TenantID] = organization.OrganizationType
	}

	principals := make(map[string]SeedPrincipal, len(manifest.Principals))
	for _, principal := range manifest.Principals {
		if err := requireID(principal.PrincipalID, "usr"); err != nil {
			return err
		}
		if principal.DisplayName == "" {
			return errors.New("identity seed principal display name is absent")
		}
		switch principal.PrincipalType {
		case "merchant", "workforce":
			if !strings.HasPrefix(principal.PersonAnchor, "syn_person_") {
				return errors.New("merchant or workforce synthetic person anchor is absent")
			}
		case "customer", "machine":
			if principal.PersonAnchor != "" {
				return errors.New("customer or machine seed has an unexpected person anchor")
			}
		default:
			return errors.New("identity seed principal population is invalid")
		}
		principals[principal.PrincipalID] = principal
	}

	for _, subject := range manifest.ExternalSubjects {
		if err := requireID(subject.ExternalSubjectID, "ext"); err != nil {
			return err
		}
		principal, found := principals[subject.PrincipalID]
		if !found || principal.PrincipalType != subject.Population ||
			!localIssuerPattern.MatchString(subject.Issuer) ||
			!strings.HasPrefix(subject.Subject, "00000000-0000-4000-8000-") {
			return errors.New("identity seed external subject is invalid")
		}
	}

	roles := make(map[string]SeedRole, len(manifest.Roles))
	permissions := make(map[string]struct{}, len(manifest.Permissions))
	for _, permission := range manifest.Permissions {
		if _, duplicate := permissions[permission]; duplicate || strings.TrimSpace(permission) == "" {
			return errors.New("identity seed permission catalogue is invalid")
		}
		permissions[permission] = struct{}{}
	}
	for _, role := range manifest.Roles {
		if _, duplicate := roles[role.ID]; duplicate || role.ID == "" {
			return errors.New("identity seed role catalogue is invalid")
		}
		for _, permission := range role.Permissions {
			if _, found := permissions[permission]; !found {
				return errors.New("identity seed role references an unknown permission")
			}
		}
		roles[role.ID] = role
	}
	for _, role := range manifest.Roles {
		for _, delegated := range role.DelegableRoles {
			target, found := roles[delegated]
			if !found || target.Population != role.Population || delegated == role.ID {
				return errors.New("identity seed delegation is invalid")
			}
		}
	}

	for _, membership := range manifest.Memberships {
		if err := requireID(membership.MembershipID, "mem"); err != nil {
			return err
		}
		principal, principalFound := principals[membership.PrincipalID]
		role, roleFound := roles[membership.RoleID]
		tenantType, tenantFound := tenants[membership.TenantID]
		if !principalFound || !roleFound || !tenantFound ||
			principal.PrincipalType != role.Population ||
			tenantType != role.Population ||
			(role.Population != "customer" && role.Population != "merchant") {
			return errors.New("identity seed tenant membership is invalid")
		}
	}
	for _, assignment := range manifest.PrincipalRoles {
		if err := requireID(assignment.PrincipalRoleID, "prr"); err != nil {
			return err
		}
		principal, principalFound := principals[assignment.PrincipalID]
		role, roleFound := roles[assignment.RoleID]
		if !principalFound || !roleFound || principal.PrincipalType != role.Population ||
			(role.Population != "workforce" && role.Population != "machine") {
			return errors.New("identity seed global principal role is invalid")
		}
	}

	for _, session := range manifest.Sessions {
		if err := requireID(session.SessionID, "ses"); err != nil {
			return err
		}
		if _, found := principals[session.PrincipalID]; !found ||
			session.Status != "revoked" || session.RevokedAt != manifest.VirtualTime ||
			!hexDigestPattern.MatchString(session.VerifierSHA256) ||
			(session.TenantID == "" && session.GlobalScope == "") ||
			(session.TenantID != "" && session.GlobalScope != "") {
			return errors.New("identity seed revoked-session recovery canary is invalid")
		}
		principal := principals[session.PrincipalID]
		if session.TenantID != "" && tenants[session.TenantID] != principal.PrincipalType {
			return errors.New("identity seed session tenant population is invalid")
		}
		idle, idleErr := time.Parse(time.RFC3339, session.IdleExpiresAt)
		absolute, absoluteErr := time.Parse(time.RFC3339, session.AbsoluteExpiresAt)
		if idleErr != nil || absoluteErr != nil || idle.Before(virtualTime) || absolute.Before(idle) {
			return errors.New("identity seed session expiry is invalid")
		}
	}

	for _, event := range manifest.AuditEvents {
		if err := requireID(event.AuditEventID, "aud"); err != nil {
			return err
		}
		if err := requireID(event.DecisionID, "dec"); err != nil {
			return err
		}
		if err := requireID(event.CorrelationID, "cor"); err != nil {
			return err
		}
		if _, found := principals[event.ActorID]; !found ||
			event.Action != "identity.seed.applied" ||
			event.Decision != "executed" ||
			(event.TenantID == "" && event.GlobalScope == "") ||
			(event.TenantID != "" && event.GlobalScope != "") {
			return errors.New("identity seed audit fact is invalid")
		}
	}
	return nil
}
