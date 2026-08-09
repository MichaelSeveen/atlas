package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const (
	APICredentialScheme                   = "AtlasKey"
	APICredentialAudience                 = "atlas-api"
	APICredentialScopeIdentityRead        = "identity:read"
	CredentialPurposeManagement           = "credential_management"
	CredentialActionRead                  = "identity.api_credential.read"
	CredentialActionCreate                = "identity.api_credential.create"
	CredentialActionRotate                = "identity.api_credential.rotate"
	CredentialActionRevoke                = "identity.api_credential.revoke"
	CredentialDefaultExpiry               = 90 * 24 * time.Hour
	CredentialMaximumExpiry               = 365 * 24 * time.Hour
	CredentialRotationOverlap             = 10 * time.Minute
	CredentialStepUpFreshness             = 5 * time.Minute
	CredentialRateWindow                  = time.Minute
	CredentialRatePerKey                  = 120
	CredentialRatePerTenant               = 600
	CredentialRatePerNetwork              = 60
	CredentialFallbackRatePerKey          = 30
	CredentialFallbackRatePerTenant       = 120
	CredentialFallbackRatePerNetwork      = 15
	CredentialProcessLocalCounterCapacity = 4096
)

var (
	ErrCredentialNotFound = domainerror.New(
		domainerror.MustCode("API_CREDENTIAL_NOT_FOUND"), domainerror.KindNotFound, false,
	)
	ErrCredentialConflict = domainerror.New(
		domainerror.MustCode("API_CREDENTIAL_CONFLICT"), domainerror.KindConflict, false,
	)
	ErrCredentialRateLimited = domainerror.New(
		domainerror.MustCode("API_CREDENTIAL_RATE_LIMITED"), domainerror.KindRateLimited, true,
	)
)

type APICredentialStatus string

const (
	APICredentialActive   APICredentialStatus = "active"
	APICredentialRotating APICredentialStatus = "rotating"
	APICredentialRevoked  APICredentialStatus = "revoked"
	APICredentialExpired  APICredentialStatus = "expired"
)

// APICredential contains only non-secret metadata. Verifier and network-signal
// digests never cross the persistence boundary.
type APICredential struct {
	CredentialID            identifier.ID
	OrganizationID          identifier.ID
	Name                    string
	SecretHint              string
	Scopes                  []string
	Status                  APICredentialStatus
	Environment             string
	Audience                string
	Version                 int64
	ExpiresAt               time.Time
	OverlapEndsAt           time.Time
	LastUsedAt              time.Time
	CreatedAt               time.Time
	RevokedAt               time.Time
	PreviousCredentialID    identifier.ID
	ReplacementCredentialID identifier.ID
}

type CredentialRateSubject struct {
	CredentialID  identifier.ID
	TenantID      identifier.ID
	NetworkSignal [32]byte
}

type CredentialRateDecision struct {
	Allowed  bool
	Fallback bool
}

// CredentialRateLimiter owns only reconstructible counters. It never receives
// or returns authority, scopes, verifier material, or a raw network address.
type CredentialRateLimiter interface {
	Allow(context.Context, CredentialRateSubject, time.Time) (CredentialRateDecision, error)
}

type CredentialStore interface {
	ListCredentials(context.Context, ListCredentialsCommand) ([]APICredential, error)
	CreateCredential(context.Context, CreateCredentialCommand) (CredentialMutationResult, error)
	RotateCredential(context.Context, RotateCredentialCommand) (CredentialMutationResult, error)
	RevokeCredential(context.Context, RevokeCredentialCommand) (CredentialMutationResult, error)
	AuthenticateCredential(context.Context, AuthenticateCredentialCommand) (CredentialAuthentication, error)
}

type ListCredentialsCommand struct {
	Actor       Session
	Environment string
	Now         time.Time
}

type CreateAPICredentialRequest struct {
	CookieValue    string
	CSRFToken      string
	IdempotencyKey string
	Name           string
	Scopes         []string
	ExpiresInDays  int
	Purpose        string
	CorrelationID  identifier.ID
}

type CreateCredentialCommand struct {
	Actor             Session
	Credential        APICredential
	Verifier          [32]byte
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	AuditEvent        audit.Event
}

type RotateAPICredentialRequest struct {
	CookieValue    string
	CSRFToken      string
	IdempotencyKey string
	CredentialID   identifier.ID
	CorrelationID  identifier.ID
}

type RotateCredentialCommand struct {
	Actor                 Session
	SourceCredentialID    identifier.ID
	ReplacementID         identifier.ID
	ReplacementVerifier   [32]byte
	ReplacementSecretHint string
	Environment           string
	IdempotencyDigest     [32]byte
	RequestDigest         [32]byte
	Now                   time.Time
	AuditEvent            audit.Event
}

type RevokeAPICredentialRequest struct {
	CookieValue    string
	CSRFToken      string
	IdempotencyKey string
	CredentialID   identifier.ID
	CorrelationID  identifier.ID
}

type RevokeCredentialCommand struct {
	Actor             Session
	CredentialID      identifier.ID
	Environment       string
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	AuditEvent        audit.Event
}

type CredentialMutationResult struct {
	Credential APICredential
	DecisionID identifier.ID
	Replay     bool
}

type APICredentialCreated struct {
	Credential      APICredential
	Secret          string
	SecretDisclosed bool
	DecisionID      identifier.ID
	Replay          bool
}

type AuthenticateCredentialCommand struct {
	CredentialID  identifier.ID
	Verifier      [32]byte
	Environment   string
	Audience      string
	Scope         string
	NetworkSignal [32]byte
	Now           time.Time
}

type CredentialAuthentication struct {
	Credential       APICredential
	Permission       string
	AnomalousNetwork bool
}

type MachinePrincipal struct {
	PrincipalID          identifier.ID
	TenantID             identifier.ID
	DisplayName          string
	Permissions          []string
	AuthorizationVersion int64
	ExpiresAt            time.Time
	AnomalousNetwork     bool
	RateFallback         bool
}

func (service *Service) ListAPICredentials(
	ctx context.Context,
	cookieValue string,
) ([]APICredential, error) {
	actor, _, err := service.Current(ctx, cookieValue)
	if err != nil {
		return nil, err
	}
	if actor.InvitationAcceptanceOnly || actor.Population != PopulationMerchant {
		return nil, ErrActionNotAuthorized
	}
	if service.credentials == nil {
		return nil, ErrIdentityUnavailable
	}
	return service.credentials.ListCredentials(ctx, ListCredentialsCommand{
		Actor: actor, Environment: service.credentialEnvironment, Now: service.clock.Now().UTC(),
	})
}

func (service *Service) CreateAPICredential(
	ctx context.Context,
	request CreateAPICredentialRequest,
) (APICredentialCreated, error) {
	actor, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return APICredentialCreated{}, err
	}
	if actor.InvitationAcceptanceOnly || actor.Population != PopulationMerchant {
		return APICredentialCreated{}, ErrActionNotAuthorized
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return APICredentialCreated{}, ErrCSRFValidationFailed
	}
	if service.credentials == nil || !validCredentialName(request.Name) ||
		!validCredentialScopes(request.Scopes) || request.ExpiresInDays < 1 ||
		request.ExpiresInDays > int(CredentialMaximumExpiry/(24*time.Hour)) ||
		request.Purpose != CredentialPurposeManagement {
		return APICredentialCreated{}, ErrValidationFailed
	}
	if !validIdempotencyKey(request.IdempotencyKey) || request.CorrelationID.IsZero() ||
		request.CorrelationID.Prefix() != "cor" {
		return APICredentialCreated{}, ErrInputInvalid
	}
	now := service.clock.Now().UTC()
	credentialID, err := service.generatedID("key")
	if err != nil {
		return APICredentialCreated{}, ErrIdentityUnavailable
	}
	secret, verifier, err := randomToken(service.entropy)
	if err != nil {
		return APICredentialCreated{}, ErrIdentityUnavailable
	}
	auditEvent, err := service.credentialAudit(actor, credentialID, request.CorrelationID,
		CredentialActionCreate, "api_credential_created", now)
	if err != nil {
		return APICredentialCreated{}, ErrIdentityUnavailable
	}
	expiresAt := now.Add(time.Duration(request.ExpiresInDays) * 24 * time.Hour)
	requestDigest := sha256.Sum256([]byte(
		"v1\noperation=create\ntenant=" + actor.TenantID.String() +
			"\nname=" + request.Name + "\nscopes=" + strings.Join(request.Scopes, ",") +
			"\nexpires_in_days=" + strconv.Itoa(request.ExpiresInDays) +
			"\npurpose=" + request.Purpose + "\nenvironment=" + service.credentialEnvironment,
	))
	stored, err := service.credentials.CreateCredential(ctx, CreateCredentialCommand{
		Actor: actor,
		Credential: APICredential{
			CredentialID: credentialID, OrganizationID: actor.TenantID, Name: request.Name,
			SecretHint: secret[len(secret)-8:], Scopes: append([]string(nil), request.Scopes...),
			Status: APICredentialActive, Environment: service.credentialEnvironment,
			Audience: APICredentialAudience, Version: 1, ExpiresAt: expiresAt, CreatedAt: now,
		},
		Verifier: verifier, IdempotencyDigest: sha256.Sum256([]byte(request.IdempotencyKey)),
		RequestDigest: requestDigest, Now: now, AuditEvent: auditEvent,
	})
	if err != nil {
		return APICredentialCreated{DecisionID: stored.DecisionID, Replay: stored.Replay}, err
	}
	result := APICredentialCreated{
		Credential: stored.Credential, DecisionID: stored.DecisionID, Replay: stored.Replay,
	}
	if !stored.Replay {
		result.Secret = APICredentialScheme + " " + stored.Credential.CredentialID.String() + "." + secret
		result.SecretDisclosed = true
	}
	return result, nil
}

func (service *Service) RotateAPICredential(
	ctx context.Context,
	request RotateAPICredentialRequest,
) (APICredentialCreated, error) {
	actor, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return APICredentialCreated{}, err
	}
	if actor.InvitationAcceptanceOnly || actor.Population != PopulationMerchant {
		return APICredentialCreated{}, ErrActionNotAuthorized
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return APICredentialCreated{}, ErrCSRFValidationFailed
	}
	if service.credentials == nil || request.CredentialID.IsZero() ||
		request.CredentialID.Prefix() != "key" || !validIdempotencyKey(request.IdempotencyKey) ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return APICredentialCreated{}, ErrInputInvalid
	}
	now := service.clock.Now().UTC()
	replacementID, err := service.generatedID("key")
	if err != nil {
		return APICredentialCreated{}, ErrIdentityUnavailable
	}
	secret, verifier, err := randomToken(service.entropy)
	if err != nil {
		return APICredentialCreated{}, ErrIdentityUnavailable
	}
	auditEvent, err := service.credentialAudit(actor, request.CredentialID, request.CorrelationID,
		CredentialActionRotate, "api_credential_rotated", now)
	if err != nil {
		return APICredentialCreated{}, ErrIdentityUnavailable
	}
	requestDigest := sha256.Sum256([]byte(
		"v1\noperation=rotate\ntenant=" + actor.TenantID.String() +
			"\ncredential=" + request.CredentialID.String() +
			"\nenvironment=" + service.credentialEnvironment,
	))
	stored, err := service.credentials.RotateCredential(ctx, RotateCredentialCommand{
		Actor: actor, SourceCredentialID: request.CredentialID, ReplacementID: replacementID,
		ReplacementVerifier: verifier, ReplacementSecretHint: secret[len(secret)-8:],
		Environment:       service.credentialEnvironment,
		IdempotencyDigest: sha256.Sum256([]byte(request.IdempotencyKey)),
		RequestDigest:     requestDigest, Now: now, AuditEvent: auditEvent,
	})
	if err != nil {
		return APICredentialCreated{DecisionID: stored.DecisionID, Replay: stored.Replay}, err
	}
	result := APICredentialCreated{
		Credential: stored.Credential, DecisionID: stored.DecisionID, Replay: stored.Replay,
	}
	if !stored.Replay {
		result.Secret = APICredentialScheme + " " + stored.Credential.CredentialID.String() + "." + secret
		result.SecretDisclosed = true
	}
	return result, nil
}

func (service *Service) RevokeAPICredential(
	ctx context.Context,
	request RevokeAPICredentialRequest,
) (CredentialMutationResult, error) {
	actor, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return CredentialMutationResult{}, err
	}
	if actor.InvitationAcceptanceOnly || actor.Population != PopulationMerchant {
		return CredentialMutationResult{}, ErrActionNotAuthorized
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return CredentialMutationResult{}, ErrCSRFValidationFailed
	}
	if service.credentials == nil || request.CredentialID.IsZero() ||
		request.CredentialID.Prefix() != "key" || !validIdempotencyKey(request.IdempotencyKey) ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return CredentialMutationResult{}, ErrInputInvalid
	}
	now := service.clock.Now().UTC()
	auditEvent, err := service.credentialAudit(actor, request.CredentialID, request.CorrelationID,
		CredentialActionRevoke, "api_credential_revoked", now)
	if err != nil {
		return CredentialMutationResult{}, ErrIdentityUnavailable
	}
	requestDigest := sha256.Sum256([]byte(
		"v1\noperation=revoke\ntenant=" + actor.TenantID.String() +
			"\ncredential=" + request.CredentialID.String() +
			"\nenvironment=" + service.credentialEnvironment,
	))
	return service.credentials.RevokeCredential(ctx, RevokeCredentialCommand{
		Actor: actor, CredentialID: request.CredentialID,
		Environment:       service.credentialEnvironment,
		IdempotencyDigest: sha256.Sum256([]byte(request.IdempotencyKey)),
		RequestDigest:     requestDigest, Now: now, AuditEvent: auditEvent,
	})
}

func (service *Service) AuthenticateAPICredential(
	ctx context.Context,
	authorization string,
	sourceAddress string,
) (MachinePrincipal, error) {
	if service.credentials == nil || service.credentialLimiter == nil {
		return MachinePrincipal{}, ErrAuthenticationRequired
	}
	credentialID, verifier, err := parseAPICredential(authorization)
	if err != nil {
		return MachinePrincipal{}, ErrAuthenticationRequired
	}
	networkSignal, err := service.networkSignal(sourceAddress)
	if err != nil {
		return MachinePrincipal{}, ErrAuthenticationRequired
	}
	now := service.clock.Now().UTC()
	authentication, err := service.credentials.AuthenticateCredential(ctx, AuthenticateCredentialCommand{
		CredentialID: credentialID, Verifier: verifier,
		Environment: service.credentialEnvironment, Audience: APICredentialAudience,
		Scope: APICredentialScopeIdentityRead, NetworkSignal: networkSignal, Now: now,
	})
	if err != nil {
		return MachinePrincipal{}, err
	}
	rate, err := service.credentialLimiter.Allow(ctx, CredentialRateSubject{
		CredentialID: credentialID, TenantID: authentication.Credential.OrganizationID,
		NetworkSignal: networkSignal,
	}, now)
	if err != nil {
		return MachinePrincipal{}, ErrIdentityUnavailable
	}
	if !rate.Allowed {
		return MachinePrincipal{RateFallback: rate.Fallback}, ErrCredentialRateLimited
	}
	return MachinePrincipal{
		PrincipalID: credentialID, TenantID: authentication.Credential.OrganizationID,
		DisplayName:          authentication.Credential.Name,
		Permissions:          []string{authentication.Permission},
		AuthorizationVersion: authentication.Credential.Version,
		ExpiresAt:            authentication.Credential.ExpiresAt,
		AnomalousNetwork:     authentication.AnomalousNetwork, RateFallback: rate.Fallback,
	}, nil
}

func (service *Service) credentialAudit(
	actor Session,
	target identifier.ID,
	correlationID identifier.ID,
	action string,
	reason string,
	now time.Time,
) (audit.Event, error) {
	auditID, err := service.generatedID("aud")
	if err != nil {
		return audit.Event{}, err
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return audit.Event{}, err
	}
	return audit.Event{
		AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
		TenantID: actor.TenantID, SessionAssurance: string(actor.Assurance), Action: action,
		TargetType: "api_credential", TargetID: target.String(), DecisionID: decisionID,
		Decision: "executed", ReasonCode: reason, CorrelationID: correlationID,
		OccurredAt: now, SafeAfterReference: "credential-status:active",
	}, nil
}

func parseAPICredential(value string) (identifier.ID, [32]byte, error) {
	if strings.TrimSpace(value) != value || strings.Count(value, " ") != 1 ||
		!strings.HasPrefix(value, APICredentialScheme+" ") {
		return identifier.ID{}, [32]byte{}, errors.New("invalid credential scheme")
	}
	payload := strings.TrimPrefix(value, APICredentialScheme+" ")
	if strings.Count(payload, ".") != 1 {
		return identifier.ID{}, [32]byte{}, errors.New("invalid credential payload")
	}
	parts := strings.SplitN(payload, ".", 2)
	credentialID, err := identifier.Parse(parts[0])
	if err != nil || credentialID.Prefix() != "key" {
		return identifier.ID{}, [32]byte{}, errors.New("invalid credential identifier")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != parts[1] {
		return identifier.ID{}, [32]byte{}, errors.New("invalid credential secret")
	}
	return credentialID, sha256.Sum256([]byte(parts[1])), nil
}

func (service *Service) networkSignal(sourceAddress string) ([32]byte, error) {
	if len(service.credentialNetworkSignalKey) != 32 {
		return [32]byte{}, errors.New("network signal key unavailable")
	}
	host := sourceAddress
	if parsedHost, _, err := net.SplitHostPort(sourceAddress); err == nil {
		host = parsedHost
	}
	ip, err := netipFromString(host)
	if err != nil {
		return [32]byte{}, err
	}
	mac := hmac.New(sha256.New, service.credentialNetworkSignalKey)
	_, _ = mac.Write([]byte("atlas-credential-network-v1\n" + ip))
	var signal [32]byte
	copy(signal[:], mac.Sum(nil))
	return signal, nil
}

func netipFromString(value string) (string, error) {
	ip := net.ParseIP(strings.Trim(value, "[]"))
	if ip == nil {
		return "", errors.New("source network is invalid")
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.String(), nil
	}
	return ip.String(), nil
}

func validCredentialName(value string) bool {
	if !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	count := utf8.RuneCountInString(value)
	if count < 1 || count > 120 {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validCredentialScopes(scopes []string) bool {
	return len(scopes) == 1 && scopes[0] == APICredentialScopeIdentityRead
}

func validCredentialEnvironment(value string) bool {
	switch value {
	case "local", "test", "staging", "production-reference":
		return true
	default:
		return false
	}
}
