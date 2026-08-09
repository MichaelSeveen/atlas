// Package operations owns Atlas approval coordination and other privileged
// operational workflows. It does not own or directly mutate target-domain state.
package operations

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const (
	ActionMembershipChangeAdmin = "identity.organization.membership.change_admin"
	approvalPurpose             = "organization_administration"
	approvalReviewPurpose       = "approval_review"
	defaultApprovalLifetime     = 24 * time.Hour
)

var (
	ErrApprovalInputInvalid = domainerror.New(
		domainerror.MustCode("APPROVAL_INPUT_INVALID"), domainerror.KindInvalidArgument, false,
	)
	ErrApprovalValidationFailed = domainerror.New(
		domainerror.MustCode("APPROVAL_VALIDATION_FAILED"), domainerror.KindInvalidArgument, false,
	)
	ErrApprovalAuthenticationRequired = domainerror.New(
		domainerror.MustCode("APPROVAL_AUTHENTICATION_REQUIRED"), domainerror.KindUnauthenticated, false,
	)
	ErrApprovalCSRFValidationFailed = domainerror.New(
		domainerror.MustCode("APPROVAL_CSRF_VALIDATION_FAILED"), domainerror.KindPermissionDenied, false,
	)
	ErrApprovalNotAuthorized = domainerror.New(
		domainerror.MustCode("APPROVAL_NOT_AUTHORIZED"), domainerror.KindPermissionDenied, false,
	)
	ErrApprovalStepUpRequired = domainerror.New(
		domainerror.MustCode("APPROVAL_STEP_UP_REQUIRED"), domainerror.KindPermissionDenied, false,
	)
	ErrApprovalNotFound = domainerror.New(
		domainerror.MustCode("APPROVAL_NOT_FOUND_OR_CONCEALED"), domainerror.KindNotFound, false,
	)
	ErrApprovalConflict = domainerror.New(
		domainerror.MustCode("APPROVAL_STATE_CONFLICT"), domainerror.KindConflict, false,
	)
	ErrApprovalIdempotencyConflict = domainerror.New(
		domainerror.MustCode("APPROVAL_IDEMPOTENCY_CONFLICT"), domainerror.KindConflict, false,
	)
	ErrApprovalPreconditionFailed = domainerror.New(
		domainerror.MustCode("APPROVAL_PRECONDITION_FAILED"), domainerror.KindFailedPrecondition, false,
	)
	ErrApprovalUnavailable = domainerror.New(
		domainerror.MustCode("APPROVAL_SERVICE_UNAVAILABLE"), domainerror.KindUnavailable, true,
	)
)

// ApprovalStatus is the closed Phase 01 approval state vocabulary.
type ApprovalStatus string

const (
	ApprovalPending         ApprovalStatus = "pending"
	ApprovalApproved        ApprovalStatus = "approved"
	ApprovalRejected        ApprovalStatus = "rejected"
	ApprovalCancelled       ApprovalStatus = "cancelled"
	ApprovalExpired         ApprovalStatus = "expired"
	ApprovalExecuted        ApprovalStatus = "executed"
	ApprovalExecutionFailed ApprovalStatus = "execution_failed"
	ApprovalSuperseded      ApprovalStatus = "superseded"
)

// Terminal reports whether no later transition is allowed.
func (status ApprovalStatus) Terminal() bool {
	switch status {
	case ApprovalRejected, ApprovalCancelled, ApprovalExpired, ApprovalExecuted, ApprovalSuperseded:
		return true
	default:
		return false
	}
}

// CanTransitionTo encodes the state machine independently of persistence.
func (status ApprovalStatus) CanTransitionTo(next ApprovalStatus) bool {
	switch status {
	case ApprovalPending:
		switch next {
		case ApprovalApproved, ApprovalRejected, ApprovalCancelled, ApprovalExpired, ApprovalSuperseded:
			return true
		}
	case ApprovalApproved:
		switch next {
		case ApprovalExecuted, ApprovalExecutionFailed, ApprovalExpired, ApprovalSuperseded:
			return true
		}
	case ApprovalExecutionFailed:
		switch next {
		case ApprovalExecuted, ApprovalExecutionFailed, ApprovalExpired, ApprovalSuperseded:
			return true
		}
	}
	return false
}

// MembershipRoleChangePayload is the sole typed executable approval payload in Phase 01.
type MembershipRoleChangePayload struct {
	OrganizationID            identifier.ID
	MembershipID              identifier.ID
	ExpectedMembershipVersion int64
	RequestedRole             string
	Purpose                   string
}

type canonicalMembershipRoleChangePayload struct {
	ExpectedMembershipVersion int64  `json:"expected_membership_version"`
	MemberID                  string `json:"member_id"`
	OrganizationID            string `json:"organization_id"`
	Purpose                   string `json:"purpose"`
	RequestedRole             string `json:"requested_role"`
}

// CanonicalMembershipRoleChangePayload returns RFC 8785-equivalent canonical
// JSON for the deliberately restricted typed schema plus its SHA-256 digest.
// Every string field is either an Atlas opaque ID or a closed ASCII enum, and
// the integer is positive; encoding/json therefore has no non-JCS number or
// Unicode-normalization ambiguity for this schema.
func CanonicalMembershipRoleChangePayload(payload MembershipRoleChangePayload) ([]byte, [32]byte, error) {
	if err := validateMembershipRoleChangePayload(payload); err != nil {
		return nil, [32]byte{}, err
	}
	canonical, err := json.Marshal(canonicalMembershipRoleChangePayload{
		ExpectedMembershipVersion: payload.ExpectedMembershipVersion,
		MemberID:                  payload.MembershipID.String(),
		OrganizationID:            payload.OrganizationID.String(),
		Purpose:                   payload.Purpose,
		RequestedRole:             payload.RequestedRole,
	})
	if err != nil {
		return nil, [32]byte{}, ErrApprovalUnavailable
	}
	return canonical, sha256.Sum256(canonical), nil
}

// DecodeCanonicalMembershipRoleChangePayload validates that stored bytes are
// the one canonical representation accepted for the typed action.
func DecodeCanonicalMembershipRoleChangePayload(canonical []byte) (MembershipRoleChangePayload, error) {
	var encoded canonicalMembershipRoleChangePayload
	decoder := json.NewDecoder(strings.NewReader(string(canonical)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&encoded); err != nil {
		return MembershipRoleChangePayload{}, ErrApprovalValidationFailed
	}
	organizationID, err := identifier.Parse(encoded.OrganizationID)
	if err != nil {
		return MembershipRoleChangePayload{}, ErrApprovalValidationFailed
	}
	membershipID, err := identifier.Parse(encoded.MemberID)
	if err != nil {
		return MembershipRoleChangePayload{}, ErrApprovalValidationFailed
	}
	payload := MembershipRoleChangePayload{
		OrganizationID: organizationID, MembershipID: membershipID,
		ExpectedMembershipVersion: encoded.ExpectedMembershipVersion,
		RequestedRole:             encoded.RequestedRole, Purpose: encoded.Purpose,
	}
	reencoded, _, err := CanonicalMembershipRoleChangePayload(payload)
	if err != nil || subtle.ConstantTimeCompare(reencoded, canonical) != 1 {
		return MembershipRoleChangePayload{}, ErrApprovalValidationFailed
	}
	return payload, nil
}

func validateMembershipRoleChangePayload(payload MembershipRoleChangePayload) error {
	if payload.OrganizationID.IsZero() || payload.OrganizationID.Prefix() != "ten" ||
		payload.MembershipID.IsZero() || payload.MembershipID.Prefix() != "mem" ||
		payload.ExpectedMembershipVersion < 1 || payload.Purpose != approvalPurpose {
		return ErrApprovalValidationFailed
	}
	switch payload.RequestedRole {
	case "merchant_viewer", "merchant_operator", "merchant_admin":
		return nil
	default:
		return ErrApprovalValidationFailed
	}
}

// Approval is the immutable payload binding plus its versioned workflow state.
type Approval struct {
	ApprovalID           identifier.ID
	OrganizationID       identifier.ID
	ActionType           string
	PayloadDigest        [32]byte
	Status               ApprovalStatus
	RequesterPrincipalID identifier.ID
	DeciderPrincipalID   identifier.ID
	Purpose              string
	DecisionReason       string
	Version              int64
	ExpiresAt            time.Time
	CreatedAt            time.Time
	DecidedAt            time.Time
	ExecutedAt           time.Time
}

// EffectiveStatus projects expiry without turning a read into a state mutation.
func (approval Approval) EffectiveStatus(now time.Time) ApprovalStatus {
	if (approval.Status == ApprovalPending || approval.Status == ApprovalApproved ||
		approval.Status == ApprovalExecutionFailed) && !now.Before(approval.ExpiresAt) {
		return ApprovalExpired
	}
	return approval.Status
}

type ApprovalPage struct {
	Approvals  []Approval
	NextCursor string
	HasMore    bool
}

type CreateApprovalRequest struct {
	CookieValue    string
	CSRFToken      string
	IdempotencyKey string
	CorrelationID  identifier.ID
	ActionType     string
	Payload        MembershipRoleChangePayload
}

type ListApprovalsRequest struct {
	CookieValue    string
	CorrelationID  identifier.ID
	PageSize       string
	PageProvided   bool
	Cursor         string
	CursorProvided bool
}

type GetApprovalRequest struct {
	CookieValue   string
	CorrelationID identifier.ID
	ApprovalID    identifier.ID
}

type DecideApprovalRequest struct {
	CookieValue    string
	CSRFToken      string
	IdempotencyKey string
	CorrelationID  identifier.ID
	ApprovalID     identifier.ID
	IfMatch        string
	Decision       string
	Purpose        string
	Reason         string
}

type ExecuteApprovalRequest struct {
	CookieValue    string
	CSRFToken      string
	IdempotencyKey string
	CorrelationID  identifier.ID
	ApprovalID     identifier.ID
	IfMatch        string
}

type CancelApprovalRequest struct {
	CookieValue    string
	CSRFToken      string
	IdempotencyKey string
	CorrelationID  identifier.ID
	ApprovalID     identifier.ID
	IfMatch        string
	Reason         string
}

type ApprovalResult struct {
	Approval   Approval
	DecisionID identifier.ID
	Replay     bool
}

type CreateApprovalCommand struct {
	Actor             identity.Session
	ApprovalID        identifier.ID
	ActionType        string
	Payload           MembershipRoleChangePayload
	CanonicalPayload  []byte
	PayloadDigest     [32]byte
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	ExpiresAt         time.Time
	AuditEvent        audit.Event
}

type ListApprovalsCommand struct {
	Actor          identity.Session
	PageSize       string
	PageProvided   bool
	Cursor         string
	CursorProvided bool
	Now            time.Time
	AuditEvent     audit.Event
}

type GetApprovalCommand struct {
	Actor      identity.Session
	ApprovalID identifier.ID
	Now        time.Time
	AuditEvent audit.Event
}

type DecideApprovalCommand struct {
	Actor             identity.Session
	DecisionRecordID  identifier.ID
	ApprovalID        identifier.ID
	ExpectedVersion   int64
	Decision          string
	Purpose           string
	Reason            string
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	AuditEvent        audit.Event
}

type ExecuteApprovalCommand struct {
	Actor             identity.Session
	ExecutionRecordID identifier.ID
	RoleChangeID      identifier.ID
	ApprovalID        identifier.ID
	ExpectedVersion   int64
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	AuditEvent        audit.Event
	TargetAuditEvent  audit.Event
}

type CancelApprovalCommand struct {
	Actor                identity.Session
	CancellationRecordID identifier.ID
	ApprovalID           identifier.ID
	ExpectedVersion      int64
	Reason               string
	IdempotencyDigest    [32]byte
	RequestDigest        [32]byte
	Now                  time.Time
	AuditEvent           audit.Event
}

// Store is the Operations-owned durable approval boundary.
type Store interface {
	Create(context.Context, CreateApprovalCommand) (ApprovalResult, error)
	List(context.Context, ListApprovalsCommand) (ApprovalPage, error)
	Get(context.Context, GetApprovalCommand) (ApprovalResult, error)
	Decide(context.Context, DecideApprovalCommand) (ApprovalResult, error)
	Execute(context.Context, ExecuteApprovalCommand) (ApprovalResult, error)
	Cancel(context.Context, CancelApprovalCommand) (ApprovalResult, error)
}

type identitySessionBoundary interface {
	Current(context.Context, string) (identity.Session, string, error)
}

type ServiceOptions struct {
	Store    Store
	Identity identitySessionBoundary
	Clock    clock.Clock
	NewID    func(string) (identifier.ID, error)
}

type Service struct {
	store    Store
	identity identitySessionBoundary
	clock    clock.Clock
	newID    func(string) (identifier.ID, error)
}

func NewService(options ServiceOptions) (*Service, error) {
	if options.Store == nil || options.Identity == nil {
		return nil, errors.New("approval service dependencies are incomplete")
	}
	if options.Clock == nil {
		options.Clock = clock.System{}
	}
	if options.NewID == nil {
		options.NewID = identifier.New
	}
	return &Service{store: options.Store, identity: options.Identity, clock: options.Clock, newID: options.NewID}, nil
}

func (service *Service) Create(ctx context.Context, request CreateApprovalRequest) (ApprovalResult, error) {
	actor, err := service.authenticate(ctx, request.CookieValue, request.CSRFToken, true)
	if err != nil {
		return ApprovalResult{}, err
	}
	if request.ActionType != ActionMembershipChangeAdmin || !validIdempotencyKey(request.IdempotencyKey) ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return ApprovalResult{}, ErrApprovalInputInvalid
	}
	canonical, payloadDigest, err := CanonicalMembershipRoleChangePayload(request.Payload)
	if err != nil || request.Payload.OrganizationID != actor.TenantID {
		return ApprovalResult{}, ErrApprovalValidationFailed
	}
	approvalID, auditID, decisionID, err := service.generatedAuditIDs("apr")
	if err != nil {
		return ApprovalResult{}, err
	}
	now := service.clock.Now().UTC()
	idempotencyDigest := sha256.Sum256([]byte(request.IdempotencyKey))
	requestDigest := sha256.Sum256(append([]byte(request.ActionType+"\n"), canonical...))
	return service.store.Create(ctx, CreateApprovalCommand{
		Actor: actor, ApprovalID: approvalID, ActionType: request.ActionType,
		Payload: request.Payload, CanonicalPayload: canonical, PayloadDigest: payloadDigest,
		IdempotencyDigest: idempotencyDigest, RequestDigest: requestDigest,
		Now: now, ExpiresAt: now.Add(defaultApprovalLifetime),
		AuditEvent: approvalAuditEvent(auditID, decisionID, actor, request.CorrelationID,
			approvalID, "operations.approval.create", "approval_created", now),
	})
}

func (service *Service) List(ctx context.Context, request ListApprovalsRequest) (ApprovalPage, error) {
	actor, err := service.authenticate(ctx, request.CookieValue, "", false)
	if err != nil {
		return ApprovalPage{}, err
	}
	if request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return ApprovalPage{}, ErrApprovalInputInvalid
	}
	auditID, err := service.newID("aud")
	if err != nil {
		return ApprovalPage{}, ErrApprovalUnavailable
	}
	decisionID, err := service.newID("dec")
	if err != nil {
		return ApprovalPage{}, ErrApprovalUnavailable
	}
	now := service.clock.Now().UTC()
	return service.store.List(ctx, ListApprovalsCommand{
		Actor: actor, PageSize: request.PageSize, PageProvided: request.PageProvided,
		Cursor: request.Cursor, CursorProvided: request.CursorProvided, Now: now,
		AuditEvent: approvalAuditEvent(auditID, decisionID, actor, request.CorrelationID,
			identifier.ID{}, "operations.approval.list", "approvals_listed", now),
	})
}

func (service *Service) Get(ctx context.Context, request GetApprovalRequest) (ApprovalResult, error) {
	actor, err := service.authenticate(ctx, request.CookieValue, "", false)
	if err != nil {
		return ApprovalResult{}, err
	}
	if request.ApprovalID.IsZero() || request.ApprovalID.Prefix() != "apr" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" {
		return ApprovalResult{}, ErrApprovalInputInvalid
	}
	auditID, decisionID, err := service.generatedDecisionIDs()
	if err != nil {
		return ApprovalResult{}, err
	}
	now := service.clock.Now().UTC()
	return service.store.Get(ctx, GetApprovalCommand{
		Actor: actor, ApprovalID: request.ApprovalID, Now: now,
		AuditEvent: approvalAuditEvent(auditID, decisionID, actor, request.CorrelationID,
			request.ApprovalID, "operations.approval.get", "approval_read", now),
	})
}

func (service *Service) Decide(ctx context.Context, request DecideApprovalRequest) (ApprovalResult, error) {
	actor, err := service.authenticate(ctx, request.CookieValue, request.CSRFToken, true)
	if err != nil {
		return ApprovalResult{}, err
	}
	if request.ApprovalID.IsZero() || request.ApprovalID.Prefix() != "apr" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" ||
		!validIdempotencyKey(request.IdempotencyKey) || request.Purpose != approvalReviewPurpose ||
		(request.Decision != "approve" && request.Decision != "reject") || len(request.Reason) > 500 {
		return ApprovalResult{}, ErrApprovalValidationFailed
	}
	expectedVersion, err := parseApprovalETag(request.IfMatch)
	if err != nil {
		return ApprovalResult{}, err
	}
	recordID, auditID, decisionID, err := service.generatedAuditIDs("apd")
	if err != nil {
		return ApprovalResult{}, err
	}
	now := service.clock.Now().UTC()
	requestBytes, _ := json.Marshal(struct {
		Decision string `json:"decision"`
		Purpose  string `json:"purpose"`
		Reason   string `json:"reason"`
	}{request.Decision, request.Purpose, request.Reason})
	return service.store.Decide(ctx, DecideApprovalCommand{
		Actor: actor, DecisionRecordID: recordID, ApprovalID: request.ApprovalID,
		ExpectedVersion: expectedVersion, Decision: request.Decision, Purpose: request.Purpose,
		Reason: request.Reason, IdempotencyDigest: sha256.Sum256([]byte(request.IdempotencyKey)),
		RequestDigest: sha256.Sum256(requestBytes), Now: now,
		AuditEvent: approvalAuditEvent(auditID, decisionID, actor, request.CorrelationID,
			request.ApprovalID, "operations.approval.decide", "approval_decided", now),
	})
}

func (service *Service) Execute(ctx context.Context, request ExecuteApprovalRequest) (ApprovalResult, error) {
	actor, err := service.authenticate(ctx, request.CookieValue, request.CSRFToken, true)
	if err != nil {
		return ApprovalResult{}, err
	}
	if request.ApprovalID.IsZero() || request.ApprovalID.Prefix() != "apr" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" ||
		!validIdempotencyKey(request.IdempotencyKey) {
		return ApprovalResult{}, ErrApprovalInputInvalid
	}
	expectedVersion, err := parseApprovalETag(request.IfMatch)
	if err != nil {
		return ApprovalResult{}, err
	}
	recordID, auditID, decisionID, err := service.generatedAuditIDs("aex")
	if err != nil {
		return ApprovalResult{}, err
	}
	roleChangeID, err := service.newID("mrc")
	if err != nil {
		return ApprovalResult{}, ErrApprovalUnavailable
	}
	targetAuditID, err := service.newID("aud")
	if err != nil {
		return ApprovalResult{}, ErrApprovalUnavailable
	}
	targetDecisionID, err := service.newID("dec")
	if err != nil {
		return ApprovalResult{}, ErrApprovalUnavailable
	}
	now := service.clock.Now().UTC()
	requestDigest := sha256.Sum256([]byte("v1\napproval=" + request.ApprovalID.String() +
		"\nversion=" + strconv.FormatInt(expectedVersion, 10)))
	targetEvent := approvalAuditEvent(targetAuditID, targetDecisionID, actor, request.CorrelationID,
		request.ApprovalID, "identity.organization.membership.role.change", "approved_membership_role_changed", now)
	targetEvent.TargetType = "membership"
	return service.store.Execute(ctx, ExecuteApprovalCommand{
		Actor: actor, ExecutionRecordID: recordID, RoleChangeID: roleChangeID,
		ApprovalID:      request.ApprovalID,
		ExpectedVersion: expectedVersion, IdempotencyDigest: sha256.Sum256([]byte(request.IdempotencyKey)),
		RequestDigest: requestDigest, Now: now,
		AuditEvent: approvalAuditEvent(auditID, decisionID, actor, request.CorrelationID,
			request.ApprovalID, "operations.approval.execute", "approval_executed", now),
		TargetAuditEvent: targetEvent,
	})
}

func (service *Service) Cancel(ctx context.Context, request CancelApprovalRequest) (ApprovalResult, error) {
	actor, err := service.authenticate(ctx, request.CookieValue, request.CSRFToken, true)
	if err != nil {
		return ApprovalResult{}, err
	}
	if request.ApprovalID.IsZero() || request.ApprovalID.Prefix() != "apr" ||
		request.CorrelationID.IsZero() || request.CorrelationID.Prefix() != "cor" ||
		!validIdempotencyKey(request.IdempotencyKey) || len(request.Reason) < 1 || len(request.Reason) > 500 {
		return ApprovalResult{}, ErrApprovalValidationFailed
	}
	expectedVersion, err := parseApprovalETag(request.IfMatch)
	if err != nil {
		return ApprovalResult{}, err
	}
	recordID, auditID, decisionID, err := service.generatedAuditIDs("apc")
	if err != nil {
		return ApprovalResult{}, err
	}
	now := service.clock.Now().UTC()
	requestBytes, _ := json.Marshal(struct {
		Reason string `json:"reason"`
	}{request.Reason})
	return service.store.Cancel(ctx, CancelApprovalCommand{
		Actor: actor, CancellationRecordID: recordID, ApprovalID: request.ApprovalID,
		ExpectedVersion: expectedVersion, Reason: request.Reason,
		IdempotencyDigest: sha256.Sum256([]byte(request.IdempotencyKey)),
		RequestDigest:     sha256.Sum256(requestBytes), Now: now,
		AuditEvent: approvalAuditEvent(auditID, decisionID, actor, request.CorrelationID,
			request.ApprovalID, "operations.approval.cancel", "approval_cancelled", now),
	})
}

func (service *Service) authenticate(ctx context.Context, cookieValue, csrfToken string, mutation bool) (identity.Session, error) {
	actor, expectedCSRF, err := service.identity.Current(ctx, cookieValue)
	if err != nil {
		if errors.Is(err, identity.ErrIdentityUnavailable) {
			return identity.Session{}, ErrApprovalUnavailable
		}
		return identity.Session{}, ErrApprovalAuthenticationRequired
	}
	if actor.InvitationAcceptanceOnly || actor.Population != identity.PopulationMerchant || actor.TenantID.IsZero() {
		return identity.Session{}, ErrApprovalNotAuthorized
	}
	if mutation && subtle.ConstantTimeCompare([]byte(expectedCSRF), []byte(csrfToken)) != 1 {
		return identity.Session{}, ErrApprovalCSRFValidationFailed
	}
	return actor, nil
}

func (service *Service) generatedDecisionIDs() (identifier.ID, identifier.ID, error) {
	auditID, err := service.newID("aud")
	if err != nil {
		return identifier.ID{}, identifier.ID{}, ErrApprovalUnavailable
	}
	decisionID, err := service.newID("dec")
	if err != nil {
		return identifier.ID{}, identifier.ID{}, ErrApprovalUnavailable
	}
	return auditID, decisionID, nil
}

func (service *Service) generatedAuditIDs(prefix string) (identifier.ID, identifier.ID, identifier.ID, error) {
	recordID, err := service.newID(prefix)
	if err != nil {
		return identifier.ID{}, identifier.ID{}, identifier.ID{}, ErrApprovalUnavailable
	}
	auditID, decisionID, err := service.generatedDecisionIDs()
	return recordID, auditID, decisionID, err
}

func approvalAuditEvent(
	auditID, decisionID identifier.ID,
	actor identity.Session,
	correlationID, approvalID identifier.ID,
	action, reason string,
	now time.Time,
) audit.Event {
	targetID := approvalID.String()
	if targetID == "" {
		targetID = "approvals:tenant-page"
	}
	return audit.Event{
		AuditEventID: auditID, ActorID: actor.PrincipalID, ActorType: actor.PrincipalType,
		TenantID: actor.TenantID, SessionAssurance: string(actor.Assurance), Action: action,
		TargetType: "approval", TargetID: targetID, DecisionID: decisionID,
		Decision: "executed", ReasonCode: reason, CorrelationID: correlationID,
		ApprovalID: approvalID, OccurredAt: now,
	}
}

func ApprovalETag(version int64) string {
	return `"` + strconv.FormatInt(version, 10) + `"`
}

func parseApprovalETag(value string) (int64, error) {
	if len(value) < 3 || value[0] != '"' || value[len(value)-1] != '"' || strings.Contains(value, ",") {
		return 0, ErrApprovalInputInvalid
	}
	version, err := strconv.ParseInt(value[1:len(value)-1], 10, 64)
	if err != nil || version < 1 {
		return 0, ErrApprovalInputInvalid
	}
	return version, nil
}

func validIdempotencyKey(value string) bool {
	if len(value) < 8 || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}
