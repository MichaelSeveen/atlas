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
	stepUpClaimLease  = 30 * time.Second
	stepUpRetention   = 24 * time.Hour
	stepUpFreshness   = 5 * time.Minute
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
	ErrStepUpRequired = domainerror.New(
		domainerror.MustCode("STEP_UP_REQUIRED"),
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
	ErrIdempotencyConflict = domainerror.New(
		domainerror.MustCode("IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_REQUEST"),
		domainerror.KindConflict,
		false,
	)
	ErrIdempotencyInProgress = domainerror.New(
		domainerror.MustCode("IDEMPOTENCY_REQUEST_IN_PROGRESS"),
		domainerror.KindConflict,
		true,
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
	StepUpAction         string
	StepUpVerifiedAt     time.Time
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
	StepUpAction      string
	StepUpVerifiedAt  time.Time
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

type AdminRevocationCommand struct {
	Actor               Session
	TargetSessionID     identifier.ID
	RevocationRequestID identifier.ID
	IdempotencyDigest   [32]byte
	RequestDigest       [32]byte
	Purpose             string
	Reason              string
	Now                 time.Time
	AuditEvent          audit.Event
}

type AdminRevocationResult struct {
	DecisionID     identifier.ID
	CurrentRevoked bool
	Replay         bool
}

type StepUpClaimCommand struct {
	ChallengeRequestID  identifier.ID
	Actor               Session
	IdempotencyScope    [32]byte
	RequestDigest       [32]byte
	Action              string
	CorrelationID       identifier.ID
	Now                 time.Time
	ProcessingExpiresAt time.Time
	RetainedUntil       time.Time
}

type StepUpClaimResult struct {
	ChallengeRequestID identifier.ID
	Owner              bool
	Replay             bool
	TransactionID      identifier.ID
	AuthorizationURL   string
	ExpiresAt          time.Time
}

type CompleteStepUpCommand struct {
	ChallengeRequestID identifier.ID
	IdempotencyScope   [32]byte
	Transaction        OIDCTransaction
	AuthorizationURL   string
	CompletedAt        time.Time
}

type Store interface {
	PutOIDCTransaction(context.Context, OIDCTransaction) error
	TakeOIDCTransaction(context.Context, [32]byte, time.Time) (OIDCTransaction, error)
	ClaimStepUp(context.Context, StepUpClaimCommand) (StepUpClaimResult, error)
	CompleteStepUp(context.Context, CompleteStepUpCommand) error
	FailStepUp(context.Context, identifier.ID, [32]byte, time.Time) error
	CreateSession(context.Context, CreateSessionCommand) (Session, error)
	Authenticate(context.Context, [32]byte, time.Time, time.Duration) (Session, error)
	ListSessions(context.Context, Session, time.Time) ([]SessionSummary, error)
	RevokeCurrent(context.Context, Session, time.Time, audit.Event) error
	RevokeOne(context.Context, RevocationCommand) (RevocationResult, error)
	RevokeAll(context.Context, RevocationCommand) (RevocationResult, error)
	RevokeForSecurity(context.Context, AdminRevocationCommand) (AdminRevocationResult, error)
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
	CookieValue    string
	CSRFToken      string
	Action         string
	IdempotencyKey string
	CorrelationID  identifier.ID
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
	if !validIdempotencyKey(request.IdempotencyKey) ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return BeginLoginResult{}, ErrInputInvalid
	}
	policy := service.sessionPolicies[session.Population]
	now := service.clock.Now().UTC()
	challengeRequestID, err := service.generatedID("idr")
	if err != nil {
		return BeginLoginResult{}, ErrIdentityUnavailable
	}
	scope := "v1\n" + session.PrincipalID.String() + "\n"
	if session.TenantID.IsZero() {
		scope += "global:identity-security\n"
	} else {
		scope += "tenant:" + session.TenantID.String() + "\n"
	}
	scope += "POST\n/v1/step-up/challenges\n" + request.IdempotencyKey
	idempotencyScope := sha256.Sum256([]byte(scope))
	requestDigest := sha256.Sum256([]byte("v1\naction=" + request.Action))
	claim, err := service.store.ClaimStepUp(ctx, StepUpClaimCommand{
		ChallengeRequestID: challengeRequestID, Actor: session,
		IdempotencyScope: idempotencyScope, RequestDigest: requestDigest,
		Action: request.Action, CorrelationID: request.CorrelationID, Now: now,
		ProcessingExpiresAt: now.Add(stepUpClaimLease),
		RetainedUntil:       now.Add(stepUpRetention),
	})
	if err != nil {
		return BeginLoginResult{}, err
	}
	if claim.Replay {
		return BeginLoginResult{
			TransactionID: claim.TransactionID, Population: session.Population,
			AuthorizationURL: claim.AuthorizationURL, ExpiresAt: claim.ExpiresAt,
		}, nil
	}
	if !claim.Owner {
		return BeginLoginResult{}, ErrIdempotencyInProgress
	}
	transaction, authorizationURL, err := service.prepareTransaction(ctx, OIDCTransaction{
		Kind: TransactionStepUp, Population: session.Population,
		ReturnTo: policy.AllowedReturnTo, PrincipalID: session.PrincipalID,
		ReplacedSessionID: session.SessionID, RequestedAction: request.Action,
	})
	if err != nil {
		_ = service.store.FailStepUp(
			ctx,
			claim.ChallengeRequestID,
			idempotencyScope,
			service.clock.Now().UTC(),
		)
		return BeginLoginResult{}, err
	}
	err = service.store.CompleteStepUp(ctx, CompleteStepUpCommand{
		ChallengeRequestID: claim.ChallengeRequestID, IdempotencyScope: idempotencyScope,
		Transaction: transaction, AuthorizationURL: authorizationURL,
		CompletedAt: service.clock.Now().UTC(),
	})
	if err != nil {
		_ = service.store.FailStepUp(
			ctx,
			claim.ChallengeRequestID,
			idempotencyScope,
			service.clock.Now().UTC(),
		)
		return BeginLoginResult{}, err
	}
	return BeginLoginResult{
		TransactionID: transaction.TransactionID, Population: transaction.Population,
		AuthorizationURL: authorizationURL, ExpiresAt: transaction.ExpiresAt,
	}, nil
}

func (service *Service) beginTransaction(ctx context.Context, transaction OIDCTransaction) (BeginLoginResult, error) {
	prepared, authorizationURL, err := service.prepareTransaction(ctx, transaction)
	if err != nil {
		return BeginLoginResult{}, err
	}
	if err := service.store.PutOIDCTransaction(ctx, prepared); err != nil {
		return BeginLoginResult{}, err
	}
	return BeginLoginResult{
		TransactionID: prepared.TransactionID, Population: prepared.Population,
		AuthorizationURL: authorizationURL,
		ExpiresAt:        prepared.ExpiresAt,
	}, nil
}

func (service *Service) prepareTransaction(
	ctx context.Context,
	transaction OIDCTransaction,
) (OIDCTransaction, string, error) {
	state, stateDigest, err := randomToken(service.entropy)
	if err != nil {
		return OIDCTransaction{}, "", ErrIdentityUnavailable
	}
	nonce, nonceDigest, err := randomToken(service.entropy)
	if err != nil {
		return OIDCTransaction{}, "", ErrIdentityUnavailable
	}
	pkce, _, err := randomToken(service.entropy)
	if err != nil {
		return OIDCTransaction{}, "", ErrIdentityUnavailable
	}
	encrypted, version, err := service.cryptor.Encrypt([]byte(pkce))
	if err != nil {
		return OIDCTransaction{}, "", ErrIdentityUnavailable
	}
	authorizationURL, err := service.provider.AuthorizationURL(
		ctx, transaction.Population, state, nonce, pkce, transaction.Kind,
	)
	if err != nil {
		return OIDCTransaction{}, "", ErrIdentityUnavailable
	}
	transactionID, err := service.generatedID("oid")
	if err != nil {
		return OIDCTransaction{}, "", ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	transaction.TransactionID = transactionID
	transaction.StateDigest = stateDigest
	transaction.NonceDigest = nonceDigest
	transaction.EncryptedPKCE = encrypted
	transaction.EncryptionKeyVersion = version
	transaction.CreatedAt = now
	transaction.ExpiresAt = now.Add(defaultOIDCTransactionExpiry)
	return transaction, authorizationURL, nil
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
	if transaction.Kind == TransactionStepUp &&
		!claims.AuthenticatedAt.After(now.Add(-stepUpFreshness)) {
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
	stepUpVerifiedAt := time.Time{}
	if transaction.Kind == TransactionStepUp {
		stepUpVerifiedAt = claims.AuthenticatedAt.UTC()
	}
	session, err := service.store.CreateSession(ctx, CreateSessionCommand{
		Claims: claims, Population: transaction.Population, Kind: transaction.Kind,
		ExpectedPrincipal: transaction.PrincipalID, ReplacedSessionID: transaction.ReplacedSessionID,
		SessionID: sessionID, VerifierDigest: verifierDigest, Assurance: assurance,
		AuthorizationAt: now, StepUpAction: transaction.RequestedAction,
		StepUpVerifiedAt: stepUpVerifiedAt,
		IdleExpiresAt:    idleExpiry, AbsoluteExpiresAt: absoluteExpiry,
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

type AdminRevokeSessionRequest struct {
	CookieValue     string
	CSRFToken       string
	TargetSessionID identifier.ID
	Purpose         string
	Reason          string
	IdempotencyKey  string
	CorrelationID   identifier.ID
}

var adminRevocationReasons = map[string]struct{}{
	"compromised_session":         {},
	"suspected_account_takeover":  {},
	"workforce_security_response": {},
}

func (service *Service) RevokeForSecurity(
	ctx context.Context,
	request AdminRevokeSessionRequest,
) (AdminRevocationResult, error) {
	session, expectedCSRF, err := service.Current(ctx, request.CookieValue)
	if err != nil {
		return AdminRevocationResult{}, err
	}
	if !constantTimeStringEqual(expectedCSRF, request.CSRFToken) {
		return AdminRevocationResult{}, ErrCSRFValidationFailed
	}
	if request.TargetSessionID.IsZero() || request.TargetSessionID.Prefix() != "ses" {
		return AdminRevocationResult{}, ErrSessionNotFound
	}
	if request.Purpose != "security_review" {
		return AdminRevocationResult{}, ErrInputInvalid
	}
	if _, allowed := adminRevocationReasons[request.Reason]; !allowed {
		return AdminRevocationResult{}, ErrInputInvalid
	}
	if !validIdempotencyKey(request.IdempotencyKey) ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return AdminRevocationResult{}, ErrInputInvalid
	}
	requestID, err := service.generatedID("asr")
	if err != nil {
		return AdminRevocationResult{}, ErrIdentityUnavailable
	}
	auditID, err := service.generatedID("aud")
	if err != nil {
		return AdminRevocationResult{}, ErrIdentityUnavailable
	}
	decisionID, err := service.generatedID("dec")
	if err != nil {
		return AdminRevocationResult{}, ErrIdentityUnavailable
	}
	now := service.clock.Now().UTC()
	idempotencyDigest := sha256.Sum256([]byte(request.IdempotencyKey))
	requestDigest := sha256.Sum256([]byte(
		"v1\ntarget=" + request.TargetSessionID.String() +
			"\npurpose=" + request.Purpose +
			"\nreason=" + request.Reason,
	))
	return service.store.RevokeForSecurity(ctx, AdminRevocationCommand{
		Actor: session, TargetSessionID: request.TargetSessionID,
		RevocationRequestID: requestID, IdempotencyDigest: idempotencyDigest,
		RequestDigest: requestDigest, Purpose: request.Purpose, Reason: request.Reason,
		Now: now,
		AuditEvent: audit.Event{
			AuditEventID: auditID, ActorID: session.PrincipalID,
			ActorType: session.PrincipalType, GlobalScope: "identity-security",
			SessionAssurance: string(session.Assurance),
			Action:           "identity.session.admin_revoke", TargetType: "session",
			TargetID: request.TargetSessionID.String(), DecisionID: decisionID,
			Decision: "executed", ReasonCode: request.Reason,
			CorrelationID: request.CorrelationID, OccurredAt: now,
		},
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

func validIdempotencyKey(value string) bool {
	if len(value) < 16 || len(value) > 128 {
		return false
	}
	for index := range value {
		character := value[index]
		if (character >= 'A' && character <= 'Z') ||
			(character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_' || character == ':' || character == '-' {
			continue
		}
		return false
	}
	return true
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
