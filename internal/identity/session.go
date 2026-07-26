package identity

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const (
	SessionCookieName = "__Host-atlas_session"
	CSRFHeaderName    = "X-Atlas-CSRF-Token"
)

var (
	ErrAuthenticationRequired = domainerror.New(
		domainerror.MustCode("AUTHENTICATION_REQUIRED"),
		domainerror.KindUnauthenticated,
		false,
	)
	ErrOIDCTransactionInvalid = domainerror.New(
		domainerror.MustCode("OIDC_TRANSACTION_INVALID"),
		domainerror.KindUnauthenticated,
		false,
	)
	ErrSessionExpired = domainerror.New(
		domainerror.MustCode("SESSION_EXPIRED"),
		domainerror.KindUnauthenticated,
		false,
	)
	ErrSessionRevoked = domainerror.New(
		domainerror.MustCode("SESSION_REVOKED"),
		domainerror.KindUnauthenticated,
		false,
	)
	ErrCSRFValidationFailed = domainerror.New(
		domainerror.MustCode("CSRF_VALIDATION_FAILED"),
		domainerror.KindPermissionDenied,
		false,
	)
	ErrActionNotAuthorized = domainerror.New(
		domainerror.MustCode("ACTION_NOT_AUTHORIZED"),
		domainerror.KindPermissionDenied,
		false,
	)
	ErrIdentityUnavailable = domainerror.New(
		domainerror.MustCode("IDENTITY_UNAVAILABLE"),
		domainerror.KindUnavailable,
		true,
	)
	ErrSessionConflict = domainerror.New(
		domainerror.MustCode("SESSION_CONFLICT"),
		domainerror.KindConflict,
		false,
	)
	ErrSessionNotFound = domainerror.New(
		domainerror.MustCode("SESSION_NOT_FOUND"),
		domainerror.KindNotFound,
		false,
	)
	ErrInputInvalid = domainerror.New(
		domainerror.MustCode("IDENTITY_INPUT_INVALID"),
		domainerror.KindInvalidArgument,
		false,
	)
	ErrProviderUnavailable = errors.New("OIDC provider unavailable")
	ErrProviderInvalid     = errors.New("OIDC provider response invalid")
)

type Population string

const (
	PopulationCustomer  Population = "customer"
	PopulationMerchant  Population = "merchant"
	PopulationWorkforce Population = "workforce"
)

func ParsePopulation(value string) (Population, error) {
	population := Population(value)
	switch population {
	case PopulationCustomer, PopulationMerchant, PopulationWorkforce:
		return population, nil
	default:
		return "", ErrInputInvalid
	}
}

type Assurance string

const (
	AssuranceBaseline            Assurance = "baseline"
	AssuranceSteppedUp           Assurance = "stepped-up"
	AssurancePhishingResistant   Assurance = "phishing-resistant"
	defaultOIDCTransactionExpiry           = 5 * time.Minute
)

func (assurance Assurance) ContractValue() string {
	switch assurance {
	case AssuranceSteppedUp:
		return "stepped_up"
	case AssurancePhishingResistant:
		return "phishing_resistant"
	default:
		return "baseline"
	}
}

type SessionPolicy struct {
	Idle            time.Duration
	Absolute        time.Duration
	Minimum         Assurance
	AllowedReturnTo string
}

var DefaultSessionPolicies = map[Population]SessionPolicy{
	PopulationCustomer: {
		Idle: 30 * time.Minute, Absolute: 12 * time.Hour,
		Minimum: AssuranceBaseline, AllowedReturnTo: "/customer",
	},
	PopulationMerchant: {
		Idle: 20 * time.Minute, Absolute: 8 * time.Hour,
		Minimum: AssuranceBaseline, AllowedReturnTo: "/merchant",
	},
	PopulationWorkforce: {
		Idle: 10 * time.Minute, Absolute: time.Hour,
		Minimum: AssurancePhishingResistant, AllowedReturnTo: "/workforce",
	},
}

type TransactionKind string

const (
	TransactionLogin  TransactionKind = "login"
	TransactionStepUp TransactionKind = "step-up"
)

type OIDCTransaction struct {
	TransactionID        identifier.ID
	Kind                 TransactionKind
	Population           Population
	StateDigest          [32]byte
	NonceDigest          [32]byte
	EncryptedPKCE        []byte
	EncryptionKeyVersion uint64
	ReturnTo             string
	PrincipalID          identifier.ID
	ReplacedSessionID    identifier.ID
	RequestedAction      string
	CreatedAt            time.Time
	ExpiresAt            time.Time
}

type ProviderClaims struct {
	Issuer          string
	Subject         string
	Nonce           string
	Assurance       Assurance
	AuthenticatedAt time.Time
}

type Provider interface {
	AuthorizationURL(
		context.Context,
		Population,
		string,
		string,
		string,
		TransactionKind,
	) (string, error)
	Exchange(context.Context, Population, string, string, string) (ProviderClaims, error)
}

type Session struct {
	SessionID            identifier.ID
	PrincipalID          identifier.ID
	PrincipalType        string
	DisplayName          string
	Population           Population
	TenantID             identifier.ID
	Assurance            Assurance
	AuthorizationVersion int64
	RotationVersion      int64
	CreatedAt            time.Time
	LastSeenAt           time.Time
	IdleExpiresAt        time.Time
	AbsoluteExpiresAt    time.Time
	RevokedAt            time.Time
	ClientLabel          string
	Permissions          []string
}

type SessionSummary struct {
	SessionID         identifier.ID
	Population        Population
	Assurance         Assurance
	Current           bool
	ClientLabel       string
	CreatedAt         time.Time
	LastSeenAt        time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	RevokedAt         time.Time
}

type CreateSessionCommand struct {
	Claims            ProviderClaims
	Population        Population
	Kind              TransactionKind
	ExpectedPrincipal identifier.ID
	ReplacedSessionID identifier.ID
	SessionID         identifier.ID
	VerifierDigest    [32]byte
	Assurance         Assurance
	AuthorizationAt   time.Time
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
	ClientLabel       string
	AuditEvent        audit.Event
}

type RevocationCommand struct {
	Actor               Session
	TargetSessionID     identifier.ID
	RevocationRequestID identifier.ID
	IncludeCurrent      bool
	IdempotencyDigest   [32]byte
	RequestDigest       [32]byte
	Now                 time.Time
	AuditEvent          audit.Event
}

type RevocationResult struct {
	CurrentRevoked bool
	Replay         bool
}

type Store interface {
	PutOIDCTransaction(context.Context, OIDCTransaction) error
	TakeOIDCTransaction(context.Context, [32]byte, time.Time) (OIDCTransaction, error)
	CreateSession(context.Context, CreateSessionCommand) (Session, error)
	Authenticate(context.Context, [32]byte, time.Time, time.Duration) (Session, error)
	ListSessions(context.Context, Session, time.Time) ([]SessionSummary, error)
	RevokeCurrent(context.Context, Session, time.Time, audit.Event) error
	RevokeOne(context.Context, RevocationCommand) (RevocationResult, error)
	RevokeAll(context.Context, RevocationCommand) (RevocationResult, error)
}

type TransactionCryptor interface {
	Encrypt([]byte) ([]byte, uint64, error)
	Decrypt([]byte, uint64) ([]byte, error)
}

type CSRFProtector interface {
	Token(identifier.ID, int64) (string, error)
}

type IDGenerator func(string) (identifier.ID, error)

type ServiceOptions struct {
	Store           Store
	Provider        Provider
	Cryptor         TransactionCryptor
	CSRF            CSRFProtector
	Clock           clock.Clock
	NewID           IDGenerator
	Entropy         io.Reader
	SessionPolicies map[Population]SessionPolicy
}

type Service struct {
	store           Store
	provider        Provider
	cryptor         TransactionCryptor
	csrf            CSRFProtector
	clock           clock.Clock
	newID           IDGenerator
	entropy         io.Reader
	sessionPolicies map[Population]SessionPolicy
}

func NewService(options ServiceOptions) (*Service, error) {
	if options.Store == nil || options.Provider == nil || options.Cryptor == nil ||
		options.CSRF == nil || options.Entropy == nil {
		return nil, errors.New("identity service dependencies are incomplete")
	}
	if options.Clock == nil {
		options.Clock = clock.System{}
	}
	if options.NewID == nil {
		options.NewID = identifier.New
	}
	policies := options.SessionPolicies
	if policies == nil {
		policies = DefaultSessionPolicies
	}
	if err := validateSessionPolicies(policies); err != nil {
		return nil, err
	}
	copied := make(map[Population]SessionPolicy, len(policies))
	for population, policy := range policies {
		copied[population] = policy
	}
	return &Service{
		store: options.Store, provider: options.Provider, cryptor: options.Cryptor,
		csrf: options.CSRF, clock: options.Clock, newID: options.NewID,
		entropy: options.Entropy, sessionPolicies: copied,
	}, nil
}

type BeginLoginRequest struct {
	Population  Population
	ReturnTo    string
	CookieValue string
}

type BeginLoginResult struct {
	TransactionID    identifier.ID
	Population       Population
	AuthorizationURL string
	ExpiresAt        time.Time
}

func (service *Service) BeginLogin(ctx context.Context, request BeginLoginRequest) (BeginLoginResult, error) {
	policy, found := service.sessionPolicies[request.Population]
	if !found {
		return BeginLoginResult{}, ErrInputInvalid
	}
	returnTo := request.ReturnTo
	if returnTo == "" {
		returnTo = policy.AllowedReturnTo
	}
	if returnTo != policy.AllowedReturnTo {
		return BeginLoginResult{}, ErrInputInvalid
	}
	transaction := OIDCTransaction{
		Kind: TransactionLogin, Population: request.Population, ReturnTo: returnTo,
	}
	if request.CookieValue != "" {
		session, _, err := service.Current(ctx, request.CookieValue)
		switch {
		case err == nil:
			transaction.ReplacedSessionID = session.SessionID
		case errors.Is(err, ErrAuthenticationRequired),
			errors.Is(err, ErrSessionExpired),
			errors.Is(err, ErrSessionRevoked):
		default:
			return BeginLoginResult{}, err
		}
	}
	return service.beginTransaction(ctx, transaction)
}

type BeginStepUpRequest struct {
	CookieValue string
	CSRFToken   string
	Action      string
}

var stepUpActions = map[string]struct{}{
	"identity.session.admin_revoke":                 {},
	"identity.organization.invitation.create_admin": {},
	"identity.organization.membership.change_admin": {},
	"identity.organization.membership.remove_admin": {},
	"identity.approval.decide":                      {},
	"identity.approval.execute":                     {},
	"identity.api_credential.create":                {},
	"identity.api_credential.rotate":                {},
	"identity.api_credential.revoke":                {},
}

func (service *Service) BeginStepUp(ctx context.Context, request BeginStepUpRequest) (BeginLoginResult, error) {
	session, csrf, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return BeginLoginResult{}, err
	}
	if !constantTimeStringEqual(csrf, request.CSRFToken) {
		return BeginLoginResult{}, ErrCSRFValidationFailed
	}
	if _, allowed := stepUpActions[request.Action]; !allowed {
		return BeginLoginResult{}, ErrInputInvalid
	}
	policy := service.sessionPolicies[session.Population]
	return service.beginTransaction(ctx, OIDCTransaction{
		Kind: TransactionStepUp, Population: session.Population,
		ReturnTo: policy.AllowedReturnTo, PrincipalID: session.PrincipalID,
		ReplacedSessionID: session.SessionID, RequestedAction: request.Action,
	})
}

func (service *Service) beginTransaction(ctx context.Context, transaction OIDCTransaction) (BeginLoginResult, error) {
	state, stateDigest, err := randomToken(service.entropy)
	if err != nil {
		return BeginLoginResult{}, ErrIdentityUnavailable
	}
	nonce, nonceDigest, err := randomToken(service.entropy)
	if err != nil {
		return BeginLoginResult{}, ErrIdentityUnavailable
	}
	pkce, _, err := randomToken(service.entropy)
	if err != nil {
		return BeginLoginResult{}, ErrIdentityUnavailable
	}
	encrypted, version, err := service.cryptor.Encrypt([]byte(pkce))
	if err != nil {
		return BeginLoginResult{}, ErrIdentityUnavailable
	}
	authorizationURL, err := service.provider.AuthorizationURL(
		ctx, transaction.Population, state, nonce, pkce, transaction.Kind,
	)
	if err != nil {
		return BeginLoginResult{}, ErrIdentityUnavailable
	}
	transactionID, err := service.generatedID("oid")
	if err != nil {
		return BeginLoginResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	transaction.TransactionID = transactionID
	transaction.StateDigest = stateDigest
	transaction.NonceDigest = nonceDigest
	transaction.EncryptedPKCE = encrypted
	transaction.EncryptionKeyVersion = version
	transaction.CreatedAt = now
	transaction.ExpiresAt = now.Add(defaultOIDCTransactionExpiry)
	if err := service.store.PutOIDCTransaction(ctx, transaction); err != nil {
		return BeginLoginResult{}, err
	}
	return BeginLoginResult{
		TransactionID: transactionID, Population: transaction.Population,
		AuthorizationURL: authorizationURL,
		ExpiresAt:        transaction.ExpiresAt,
	}, nil
}

type CompleteLoginRequest struct {
	State               string
	Code                string
	AuthorizationIssuer string
	CorrelationID       identifier.ID
	ClientLabel         string
}

type CompleteLoginResult struct {
	CookieValue string
	ReturnTo    string
	Session     Session
}

func (service *Service) CompleteLogin(ctx context.Context, request CompleteLoginRequest) (CompleteLoginResult, error) {
	if !validProtocolToken(request.State) || len(request.Code) < 16 || len(request.Code) > 2048 ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return CompleteLoginResult{}, ErrOIDCTransactionInvalid
	}
	stateDigest := sha256.Sum256([]byte(request.State))
	now := service.clock.Now().UTC()
	transaction, err := service.store.TakeOIDCTransaction(ctx, stateDigest, now)
	if err != nil {
		return CompleteLoginResult{}, ErrOIDCTransactionInvalid
	}
	pkce, err := service.cryptor.Decrypt(transaction.EncryptedPKCE, transaction.EncryptionKeyVersion)
	if err != nil {
		return CompleteLoginResult{}, ErrOIDCTransactionInvalid
	}
	defer wipe(pkce)
	claims, err := service.provider.Exchange(
		ctx,
		transaction.Population,
		request.Code,
		string(pkce),
		request.AuthorizationIssuer,
	)
	if err != nil {
		if errors.Is(err, ErrProviderUnavailable) {
			return CompleteLoginResult{}, ErrIdentityUnavailable
		}
		return CompleteLoginResult{}, ErrOIDCTransactionInvalid
	}
	nonceDigest := sha256.Sum256([]byte(claims.Nonce))
	if subtle.ConstantTimeCompare(nonceDigest[:], transaction.NonceDigest[:]) != 1 ||
		claims.AuthenticatedAt.IsZero() || claims.AuthenticatedAt.After(now.Add(time.Minute)) {
		return CompleteLoginResult{}, ErrOIDCTransactionInvalid
	}
	policy := service.sessionPolicies[transaction.Population]
	assurance := claims.Assurance
	if transaction.Kind == TransactionStepUp && assurance == AssuranceBaseline {
		return CompleteLoginResult{}, ErrOIDCTransactionInvalid
	}
	if !assuranceSatisfies(assurance, policy.Minimum) {
		return CompleteLoginResult{}, ErrOIDCTransactionInvalid
	}
	cookieValue, verifierDigest, err := randomToken(service.entropy)
	if err != nil {
		return CompleteLoginResult{}, ErrIdentityUnavailable
	}
	sessionID, err := service.generatedID("ses")
	if err != nil {
		return CompleteLoginResult{}, ErrIdentityUnavailable
	}
	auditEvent, err := service.authenticationAudit(
		transaction, sessionID, claims, request.CorrelationID, now,
	)
	if err != nil {
		return CompleteLoginResult{}, ErrIdentityUnavailable
	}
	idleExpiry := now.Add(policy.Idle)
	absoluteExpiry := now.Add(policy.Absolute)
	session, err := service.store.CreateSession(ctx, CreateSessionCommand{
		Claims: claims, Population: transaction.Population, Kind: transaction.Kind,
		ExpectedPrincipal: transaction.PrincipalID, ReplacedSessionID: transaction.ReplacedSessionID,
		SessionID: sessionID, VerifierDigest: verifierDigest, Assurance: assurance,
		AuthorizationAt: now, IdleExpiresAt: idleExpiry, AbsoluteExpiresAt: absoluteExpiry,
		ClientLabel: sanitizeClientLabel(request.ClientLabel), AuditEvent: auditEvent,
	})
	if err != nil {
		return CompleteLoginResult{}, err
	}
	return CompleteLoginResult{
		CookieValue: cookieValue, ReturnTo: transaction.ReturnTo, Session: session,
	}, nil
}

func (service *Service) Current(ctx context.Context, cookieValue string) (Session, string, error) {
	if !validProtocolToken(cookieValue) {
		return Session{}, "", ErrAuthenticationRequired
	}
	digest := sha256.Sum256([]byte(cookieValue))
	now := service.clock.Now().UTC()
	session, err := service.store.Authenticate(
		ctx, digest, now, service.sessionPolicies[PopulationCustomer].Idle,
	)
	if err != nil {
		return Session{}, "", err
	}
	policy, found := service.sessionPolicies[session.Population]
	if !found {
		return Session{}, "", ErrAuthenticationRequired
	}
	if session.IdleExpiresAt.Before(now) || session.IdleExpiresAt.Equal(now) {
		return Session{}, "", ErrSessionExpired
	}
	if session.AbsoluteExpiresAt.Before(now) || session.AbsoluteExpiresAt.Equal(now) {
		return Session{}, "", ErrSessionExpired
	}
	if !assuranceSatisfies(session.Assurance, policy.Minimum) {
		return Session{}, "", ErrAuthenticationRequired
	}
	csrfToken, err := service.csrf.Token(session.SessionID, session.RotationVersion)
	if err != nil {
		return Session{}, "", ErrIdentityUnavailable
	}
	sort.Strings(session.Permissions)
	return session, csrfToken, nil
}

func (service *Service) Sessions(ctx context.Context, cookieValue string) ([]SessionSummary, error) {
	session, _, err := service.Current(ctx, cookieValue)
	if err != nil {
		return nil, err
	}
	return service.store.ListSessions(ctx, session, service.clock.Now().UTC())
}

func (service *Service) Logout(
	ctx context.Context,
	cookieValue string,
	csrfToken string,
	correlationID identifier.ID,
) error {
	session, expectedCSRF, err := service.Current(ctx, cookieValue)
	if errors.Is(err, ErrAuthenticationRequired) || errors.Is(err, ErrSessionExpired) || errors.Is(err, ErrSessionRevoked) {
		return nil
	}
	if err != nil {
		return err
	}
	if !constantTimeStringEqual(expectedCSRF, csrfToken) {
		return ErrCSRFValidationFailed
	}
	event, err := service.revocationAudit(session, session.SessionID, correlationID, "logout", service.clock.Now().UTC())
	if err != nil {
		return ErrIdentityUnavailable
	}
	return service.store.RevokeCurrent(ctx, session, service.clock.Now().UTC(), event)
}

func (service *Service) RevokeOne(
	ctx context.Context,
	cookieValue string,
	csrfToken string,
	target identifier.ID,
	correlationID identifier.ID,
) (RevocationResult, error) {
	session, expectedCSRF, err := service.Current(ctx, cookieValue)
	if err != nil {
		return RevocationResult{}, err
	}
	if !constantTimeStringEqual(expectedCSRF, csrfToken) {
		return RevocationResult{}, ErrCSRFValidationFailed
	}
	if target.IsZero() || target.Prefix() != "ses" {
		return RevocationResult{}, ErrSessionNotFound
	}
	now := service.clock.Now().UTC()
	event, err := service.revocationAudit(session, target, correlationID, "revoke_one", now)
	if err != nil {
		return RevocationResult{}, ErrIdentityUnavailable
	}
	return service.store.RevokeOne(ctx, RevocationCommand{
		Actor: session, TargetSessionID: target, Now: now, AuditEvent: event,
	})
}

func (service *Service) RevokeAll(
	ctx context.Context,
	cookieValue string,
	csrfToken string,
	includeCurrent bool,
	idempotencyKey string,
	correlationID identifier.ID,
) (RevocationResult, error) {
	session, expectedCSRF, err := service.Current(ctx, cookieValue)
	if err != nil {
		return RevocationResult{}, err
	}
	if !constantTimeStringEqual(expectedCSRF, csrfToken) {
		return RevocationResult{}, ErrCSRFValidationFailed
	}
	if len(idempotencyKey) < 16 || len(idempotencyKey) > 128 {
		return RevocationResult{}, ErrInputInvalid
	}
	idempotencyDigest := sha256.Sum256([]byte(idempotencyKey))
	requestDigest := sha256.Sum256([]byte("include_current=" + boolString(includeCurrent)))
	now := service.clock.Now().UTC()
	revocationRequestID, err := service.generatedID("rev")
	if err != nil {
		return RevocationResult{}, ErrIdentityUnavailable
	}
	event, err := service.revocationAudit(session, session.PrincipalID, correlationID, "revoke_all", now)
	if err != nil {
		return RevocationResult{}, ErrIdentityUnavailable
	}
	return service.store.RevokeAll(ctx, RevocationCommand{
		Actor: session, IncludeCurrent: includeCurrent, RevocationRequestID: revocationRequestID,
		IdempotencyDigest: idempotencyDigest, RequestDigest: requestDigest,
		Now: now, AuditEvent: event,
	})
}

func (service *Service) generatedID(prefix string) (identifier.ID, error) {
	value, err := service.newID(prefix)
	if err != nil || value.IsZero() || value.Prefix() != prefix {
		return identifier.ID{}, errors.New("invalid generated identifier")
	}
	return value, nil
}

func (service *Service) authenticationAudit(
	transaction OIDCTransaction,
	sessionID identifier.ID,
	claims ProviderClaims,
	correlationID identifier.ID,
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
	action := "identity.session.login"
	reason := "oidc_login"
	if transaction.Kind == TransactionStepUp {
		action = "identity.session.step_up"
		reason = "oidc_step_up"
	}
	return audit.Event{
		AuditEventID: auditID, ActorID: transaction.PrincipalID,
		ActorType: string(transaction.Population), GlobalScope: "identity-security",
		SessionAssurance: string(claims.Assurance), Action: action,
		TargetType: "session", TargetID: sessionID.String(), DecisionID: decisionID,
		Decision: "executed", ReasonCode: reason, CorrelationID: correlationID, OccurredAt: now,
	}, nil
}

func (service *Service) revocationAudit(
	actor Session,
	target identifier.ID,
	correlationID identifier.ID,
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
	event := audit.Event{
		AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
		SessionAssurance: string(actor.Assurance), Action: "identity.session.revoke",
		TargetType: "session", TargetID: target.String(), DecisionID: decisionID,
		Decision: "executed", ReasonCode: reason, CorrelationID: correlationID, OccurredAt: now,
	}
	if actor.TenantID.IsZero() {
		event.GlobalScope = "identity-security"
	} else {
		event.TenantID = actor.TenantID
	}
	return event, nil
}

func randomToken(reader io.Reader) (string, [32]byte, error) {
	var raw [32]byte
	if _, err := io.ReadFull(reader, raw[:]); err != nil {
		return "", [32]byte{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw[:])
	return token, sha256.Sum256([]byte(token)), nil
}

func validProtocolToken(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) >= 32
}

func constantTimeStringEqual(left, right string) bool {
	if len(left) == 0 || len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func assuranceSatisfies(actual, minimum Assurance) bool {
	rank := map[Assurance]int{
		AssuranceBaseline: 1, AssuranceSteppedUp: 2, AssurancePhishingResistant: 3,
	}
	return rank[actual] >= rank[minimum] && rank[minimum] > 0
}

func validateSessionPolicies(policies map[Population]SessionPolicy) error {
	if len(policies) != 3 {
		return errors.New("session policy population set is incomplete")
	}
	for _, population := range []Population{PopulationCustomer, PopulationMerchant, PopulationWorkforce} {
		policy, found := policies[population]
		if !found || policy.Idle < time.Minute || policy.Absolute < policy.Idle ||
			policy.Absolute > 24*time.Hour || !assuranceSatisfies(policy.Minimum, AssuranceBaseline) {
			return errors.New("session policy is invalid")
		}
		parsed, err := url.Parse(policy.AllowedReturnTo)
		if err != nil || parsed.IsAbs() || parsed.RawQuery != "" || parsed.Fragment != "" ||
			!strings.HasPrefix(policy.AllowedReturnTo, "/") {
			return errors.New("session return path is invalid")
		}
	}
	return nil
}

func sanitizeClientLabel(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 120 {
		value = value[:120]
	}
	var safe strings.Builder
	for _, character := range value {
		if character >= 0x20 && character <= 0x7e {
			safe.WriteRune(character)
		}
	}
	return safe.String()
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func wipe(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
