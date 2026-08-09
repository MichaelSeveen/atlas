package server

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/operations"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
	"go.opentelemetry.io/otel/attribute"
	metricapi "go.opentelemetry.io/otel/metric"
)

type membershipRoleChangeApprovalPayloadRequest struct {
	OrganizationID            string `json:"organization_id"`
	MemberID                  string `json:"member_id"`
	ExpectedMembershipVersion int64  `json:"expected_membership_version"`
	RequestedRole             string `json:"requested_role"`
	Purpose                   string `json:"purpose"`
}

type createApprovalRequest struct {
	ActionType string                                     `json:"action_type"`
	Payload    membershipRoleChangeApprovalPayloadRequest `json:"payload"`
}

type approvalDecisionRequest struct {
	Decision string  `json:"decision"`
	Purpose  string  `json:"purpose"`
	Reason   *string `json:"reason"`
}

type approvalCancellationRequest struct {
	Reason string `json:"reason"`
}

type approvalResponse struct {
	ID                   string `json:"id"`
	OrganizationID       string `json:"organization_id"`
	ActionType           string `json:"action_type"`
	PayloadDigest        string `json:"payload_digest"`
	Status               string `json:"status"`
	RequesterPrincipalID string `json:"requester_principal_id"`
	DeciderPrincipalID   any    `json:"decider_principal_id"`
	Purpose              string `json:"purpose"`
	DecisionReason       any    `json:"decision_reason"`
	Version              int64  `json:"version"`
	ExpiresAt            string `json:"expires_at"`
	CreatedAt            string `json:"created_at"`
	DecidedAt            any    `json:"decided_at"`
	ExecutedAt           any    `json:"executed_at"`
}

type approvalPageResponse struct {
	Data []approvalResponse       `json:"data"`
	Page approvalPageInfoResponse `json:"page"`
}

type approvalPageInfoResponse struct {
	NextCursor any  `json:"next_cursor"`
	HasMore    bool `json:"has_more"`
}

func (a *App) routeApproval(response http.ResponseWriter, request *http.Request) {
	if !a.requireMethod(response, request) {
		return
	}
	if a.approvals == nil {
		a.writeProblem(response, request, http.StatusServiceUnavailable,
			"service-unavailable", "Service unavailable", "SERVICE_UNAVAILABLE", true)
		return
	}
	switch approvalRoute(request.URL.Path) {
	case "/v1/approvals":
		if request.Method == http.MethodPost {
			a.createApproval(response, request)
			return
		}
		a.listApprovals(response, request)
	case "/v1/approvals/{approval_id}":
		a.getApproval(response, request)
	case "/v1/approvals/{approval_id}/decisions":
		a.decideApproval(response, request)
	case "/v1/approvals/{approval_id}/executions":
		a.executeApproval(response, request)
	case "/v1/approvals/{approval_id}/cancellations":
		a.cancelApproval(response, request)
	default:
		a.writeProblem(response, request, http.StatusNotFound,
			"route-not-found", "Not found", "ROUTE_NOT_FOUND", false)
	}
}

func (a *App) createApproval(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" {
		a.malformed(response, request)
		return
	}
	var body createApprovalRequest
	if err := decodeStrictJSON(request, &body); err != nil {
		a.malformed(response, request)
		return
	}
	organizationID, err := parseApprovalPathID(body.Payload.OrganizationID, "ten")
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalValidationFailed)
		return
	}
	membershipID, err := parseApprovalPathID(body.Payload.MemberID, "mem")
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalValidationFailed)
		return
	}
	cookie, csrfToken, idempotencyKey, correlationID, ok := approvalMutationContext(response, request, a)
	if !ok {
		return
	}
	result, err := a.approvals.Create(request.Context(), operations.CreateApprovalRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		CorrelationID: correlationID, ActionType: body.ActionType,
		Payload: operations.MembershipRoleChangePayload{
			OrganizationID: organizationID, MembershipID: membershipID,
			ExpectedMembershipVersion: body.Payload.ExpectedMembershipVersion,
			RequestedRole:             body.Payload.RequestedRole, Purpose: body.Payload.Purpose,
		},
	})
	a.recordApprovalResult(request, "create", result, err)
	writeApprovalDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeApprovalError(response, request, err)
		return
	}
	writeApprovalMutationResponse(response, http.StatusCreated, result)
}

func (a *App) createMembershipRoleApprovalFromPatch(
	response http.ResponseWriter,
	request *http.Request,
	cookie, csrfToken, idempotencyKey string,
	correlationID, organizationID, membershipID identifier.ID,
	ifMatch, role, purpose string,
) {
	expectedVersion, err := parseMembershipVersion(ifMatch)
	if err != nil {
		a.malformed(response, request)
		return
	}
	result, err := a.approvals.Create(request.Context(), operations.CreateApprovalRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		CorrelationID: correlationID, ActionType: operations.ActionMembershipChangeAdmin,
		Payload: operations.MembershipRoleChangePayload{
			OrganizationID: organizationID, MembershipID: membershipID,
			ExpectedMembershipVersion: expectedVersion, RequestedRole: role, Purpose: purpose,
		},
	})
	a.recordApprovalResult(request, "create", result, err)
	writeApprovalDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeApprovalError(response, request, err)
		return
	}
	writeApprovalMutationResponse(response, http.StatusAccepted, result)
}

func parseMembershipVersion(value string) (int64, error) {
	const prefix = `"membership-v`
	if len(value) <= len(prefix)+1 || !strings.HasPrefix(value, prefix) || value[len(value)-1] != '"' ||
		strings.Contains(value, ",") {
		return 0, errors.New("invalid strong ETag")
	}
	version, err := strconv.ParseInt(value[len(prefix):len(value)-1], 10, 64)
	if err != nil || version < 1 || identity.OrganizationMemberETag(version) != value {
		return 0, errors.New("invalid strong ETag")
	}
	return version, nil
}

func (a *App) listApprovals(response http.ResponseWriter, request *http.Request) {
	if requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	values, err := exactQuery(request.URL.RawQuery, "page_size", "cursor")
	if err != nil {
		a.malformed(response, request)
		return
	}
	cookie, err := sessionCookie(request)
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalAuthenticationRequired)
		return
	}
	correlationID, ok := requestCorrelationID(request)
	if !ok {
		a.writeApprovalError(response, request, operations.ErrApprovalUnavailable)
		return
	}
	pageSize, pageProvided := values["page_size"]
	cursor, cursorProvided := values["cursor"]
	requestPage := ""
	if pageProvided {
		requestPage = pageSize[0]
	}
	requestCursor := ""
	if cursorProvided {
		requestCursor = cursor[0]
	}
	page, err := a.approvals.List(request.Context(), operations.ListApprovalsRequest{
		CookieValue: cookie, CorrelationID: correlationID,
		PageSize: requestPage, PageProvided: pageProvided,
		Cursor: requestCursor, CursorProvided: cursorProvided,
	})
	if err != nil {
		a.writeApprovalError(response, request, err)
		return
	}
	for _, approval := range page.Approvals {
		a.recordApprovalObservation(request, "list", approval, nil)
	}
	data := make([]approvalResponse, 0, len(page.Approvals))
	for _, approval := range page.Approvals {
		data = append(data, approvalFromDomain(approval))
	}
	nextCursor := any(nil)
	if page.NextCursor != "" {
		nextCursor = page.NextCursor
	}
	writeJSON(response, http.StatusOK, approvalPageResponse{
		Data: data, Page: approvalPageInfoResponse{NextCursor: nextCursor, HasMore: page.HasMore},
	})
}

func (a *App) getApproval(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	approvalID, err := approvalIDFromPath(request.URL.Path)
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalNotFound)
		return
	}
	cookie, err := sessionCookie(request)
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalAuthenticationRequired)
		return
	}
	correlationID, ok := requestCorrelationID(request)
	if !ok {
		a.writeApprovalError(response, request, operations.ErrApprovalUnavailable)
		return
	}
	result, err := a.approvals.Get(request.Context(), operations.GetApprovalRequest{
		CookieValue: cookie, CorrelationID: correlationID, ApprovalID: approvalID,
	})
	a.recordApprovalResult(request, "get", result, err)
	writeApprovalDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeApprovalError(response, request, err)
		return
	}
	response.Header().Set("ETag", operations.ApprovalETag(result.Approval.Version))
	writeJSON(response, http.StatusOK, approvalFromDomain(result.Approval))
}

func (a *App) decideApproval(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" {
		a.malformed(response, request)
		return
	}
	approvalID, err := approvalIDFromPath(request.URL.Path)
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalNotFound)
		return
	}
	var body approvalDecisionRequest
	if err := decodeStrictJSON(request, &body); err != nil {
		a.malformed(response, request)
		return
	}
	cookie, csrfToken, idempotencyKey, correlationID, ok := approvalMutationContext(response, request, a)
	if !ok {
		return
	}
	ifMatch, ok := singleHeader(request.Header, "If-Match")
	if !ok {
		a.malformed(response, request)
		return
	}
	reason := ""
	if body.Reason != nil {
		reason = *body.Reason
	}
	result, err := a.approvals.Decide(request.Context(), operations.DecideApprovalRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		CorrelationID: correlationID, ApprovalID: approvalID, IfMatch: ifMatch,
		Decision: body.Decision, Purpose: body.Purpose, Reason: reason,
	})
	a.recordApprovalResult(request, "decide", result, err)
	writeApprovalDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeApprovalError(response, request, err)
		return
	}
	writeApprovalMutationResponse(response, http.StatusOK, result)
}

func (a *App) executeApproval(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" || requestHasBody(request) {
		a.malformed(response, request)
		return
	}
	approvalID, err := approvalIDFromPath(request.URL.Path)
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalNotFound)
		return
	}
	cookie, csrfToken, idempotencyKey, correlationID, ok := approvalMutationContext(response, request, a)
	if !ok {
		return
	}
	ifMatch, ok := singleHeader(request.Header, "If-Match")
	if !ok {
		a.malformed(response, request)
		return
	}
	result, err := a.approvals.Execute(request.Context(), operations.ExecuteApprovalRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		CorrelationID: correlationID, ApprovalID: approvalID, IfMatch: ifMatch,
	})
	a.recordApprovalResult(request, "execute", result, err)
	writeApprovalDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeApprovalError(response, request, err)
		return
	}
	writeApprovalMutationResponse(response, http.StatusOK, result)
}

func (a *App) cancelApproval(response http.ResponseWriter, request *http.Request) {
	if request.URL.RawQuery != "" {
		a.malformed(response, request)
		return
	}
	approvalID, err := approvalIDFromPath(request.URL.Path)
	if err != nil {
		a.writeApprovalError(response, request, operations.ErrApprovalNotFound)
		return
	}
	var body approvalCancellationRequest
	if err := decodeStrictJSON(request, &body); err != nil {
		a.malformed(response, request)
		return
	}
	cookie, csrfToken, idempotencyKey, correlationID, ok := approvalMutationContext(response, request, a)
	if !ok {
		return
	}
	ifMatch, ok := singleHeader(request.Header, "If-Match")
	if !ok {
		a.malformed(response, request)
		return
	}
	result, err := a.approvals.Cancel(request.Context(), operations.CancelApprovalRequest{
		CookieValue: cookie, CSRFToken: csrfToken, IdempotencyKey: idempotencyKey,
		CorrelationID: correlationID, ApprovalID: approvalID, IfMatch: ifMatch, Reason: body.Reason,
	})
	a.recordApprovalResult(request, "cancel", result, err)
	writeApprovalDecisionHeader(response, result.DecisionID)
	if err != nil {
		a.writeApprovalError(response, request, err)
		return
	}
	writeApprovalMutationResponse(response, http.StatusOK, result)
}

func approvalMutationContext(
	response http.ResponseWriter,
	request *http.Request,
	app *App,
) (string, string, string, identifier.ID, bool) {
	cookie, err := sessionCookie(request)
	if err != nil {
		app.writeApprovalError(response, request, operations.ErrApprovalAuthenticationRequired)
		return "", "", "", identifier.ID{}, false
	}
	csrfToken, ok := singleHeader(request.Header, identity.CSRFHeaderName)
	if !ok {
		app.writeApprovalError(response, request, operations.ErrApprovalCSRFValidationFailed)
		return "", "", "", identifier.ID{}, false
	}
	idempotencyKey, ok := singleHeader(request.Header, "Idempotency-Key")
	if !ok {
		app.malformed(response, request)
		return "", "", "", identifier.ID{}, false
	}
	correlationID, ok := requestCorrelationID(request)
	if !ok {
		app.writeApprovalError(response, request, operations.ErrApprovalUnavailable)
		return "", "", "", identifier.ID{}, false
	}
	return cookie, csrfToken, idempotencyKey, correlationID, true
}

func approvalIDFromPath(path string) (identifier.ID, error) {
	remainder := strings.TrimPrefix(path, "/v1/approvals/")
	identifierText := strings.Split(remainder, "/")[0]
	return parseApprovalPathID(identifierText, "apr")
}

func parseApprovalPathID(value, prefix string) (identifier.ID, error) {
	id, err := identifier.Parse(value)
	if err != nil || id.Prefix() != prefix {
		return identifier.ID{}, errors.New("invalid approval identifier")
	}
	return id, nil
}

func approvalFromDomain(approval operations.Approval) approvalResponse {
	decider := any(nil)
	if !approval.DeciderPrincipalID.IsZero() {
		decider = approval.DeciderPrincipalID.String()
	}
	reason := any(nil)
	if approval.DecisionReason != "" {
		reason = approval.DecisionReason
	}
	decidedAt := any(nil)
	if !approval.DecidedAt.IsZero() {
		decidedAt = approval.DecidedAt.UTC().Format(time.RFC3339Nano)
	}
	executedAt := any(nil)
	if !approval.ExecutedAt.IsZero() {
		executedAt = approval.ExecutedAt.UTC().Format(time.RFC3339Nano)
	}
	return approvalResponse{
		ID: approval.ApprovalID.String(), OrganizationID: approval.OrganizationID.String(),
		ActionType: approval.ActionType, PayloadDigest: hex.EncodeToString(approval.PayloadDigest[:]),
		Status: string(approval.Status), RequesterPrincipalID: approval.RequesterPrincipalID.String(),
		DeciderPrincipalID: decider, Purpose: approval.Purpose, DecisionReason: reason,
		Version: approval.Version, ExpiresAt: approval.ExpiresAt.UTC().Format(time.RFC3339Nano),
		CreatedAt: approval.CreatedAt.UTC().Format(time.RFC3339Nano), DecidedAt: decidedAt,
		ExecutedAt: executedAt,
	}
}

func writeApprovalMutationResponse(response http.ResponseWriter, status int, result operations.ApprovalResult) {
	response.Header().Set("ETag", operations.ApprovalETag(result.Approval.Version))
	response.Header().Set("Idempotency-Replayed", strconv.FormatBool(result.Replay))
	response.Header().Set("Location", "/v1/approvals/"+result.Approval.ApprovalID.String())
	writeJSON(response, status, approvalFromDomain(result.Approval))
}

func writeApprovalDecisionHeader(response http.ResponseWriter, decisionID identifier.ID) {
	if !decisionID.IsZero() {
		response.Header().Set("X-Authorization-Decision-Id", decisionID.String())
	}
}

func (a *App) recordApprovalResult(
	request *http.Request,
	operation string,
	result operations.ApprovalResult,
	err error,
) {
	a.recordApprovalObservation(request, operation, result.Approval, err)
}

func (a *App) recordApprovalObservation(
	request *http.Request,
	operation string,
	approval operations.Approval,
	err error,
) {
	if approval.ApprovalID.IsZero() {
		return
	}
	status := string(approval.Status)
	attributes := []attribute.KeyValue{
		attribute.String("atlas.operations.approval.operation", operation),
		attribute.String("atlas.operations.approval.status", status),
	}
	safeAdd(a.approvalStatus, request.Context(), 1, metricapi.WithAttributes(attributes...))
	if !approval.CreatedAt.IsZero() {
		age := a.clock.Now().UTC().Sub(approval.CreatedAt.UTC()).Seconds()
		if age >= 0 {
			safeRecord(a.approvalAge, request.Context(), age, metricapi.WithAttributes(attributes...))
		}
	}
	if category := approvalConflictCategory(err, approval); category != "" {
		safeAdd(a.approvalConflict, request.Context(), 1, metricapi.WithAttributes(
			attribute.String("atlas.operations.approval.operation", operation),
			attribute.String("atlas.operations.approval.conflict", category),
		))
	}
	if reason, ok := approvalIntegrityReason(approval.DecisionReason); ok {
		safeAdd(a.approvalIntegrity, request.Context(), 1, metricapi.WithAttributes(
			attribute.String("atlas.operations.approval.integrity_reason", reason),
		))
	}
}

func approvalConflictCategory(err error, approval operations.Approval) string {
	switch {
	case errors.Is(err, operations.ErrApprovalIdempotencyConflict):
		return "idempotency"
	case errors.Is(err, operations.ErrApprovalPreconditionFailed):
		return "precondition"
	case errors.Is(err, operations.ErrApprovalConflict) && approval.Status == operations.ApprovalSuperseded:
		return "superseded"
	case errors.Is(err, operations.ErrApprovalConflict):
		return "state"
	default:
		return ""
	}
}

func approvalIntegrityReason(reason string) (string, bool) {
	switch reason {
	case "payload_metadata_invalid", "payload_digest_mismatch", "payload_canonicalization_invalid",
		"payload_revalidation_failed", "payload_binding_mismatch", "maker_checker_integrity_failure":
		return reason, true
	default:
		return "", false
	}
}

func (a *App) writeApprovalError(response http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, operations.ErrApprovalAuthenticationRequired):
		a.writeProblem(response, request, http.StatusUnauthorized,
			"authentication-required", "Authentication required", "AUTHENTICATION_REQUIRED", false)
	case errors.Is(err, operations.ErrApprovalCSRFValidationFailed):
		a.writeProblem(response, request, http.StatusForbidden,
			"csrf-validation-failed", "Action not authorized", "CSRF_VALIDATION_FAILED", false)
	case errors.Is(err, operations.ErrApprovalStepUpRequired):
		a.writeProblem(response, request, http.StatusForbidden,
			"step-up-required", "Additional verification required", "STEP_UP_REQUIRED", false)
	case errors.Is(err, operations.ErrApprovalNotAuthorized):
		a.writeProblem(response, request, http.StatusForbidden,
			"action-not-authorized", "Action not authorized", "ACTION_NOT_AUTHORIZED", false)
	case errors.Is(err, operations.ErrApprovalNotFound):
		a.writeProblem(response, request, http.StatusNotFound,
			"not-found-or-concealed", "Not found", "NOT_FOUND_OR_CONCEALED", false)
	case errors.Is(err, operations.ErrApprovalPreconditionFailed):
		a.writeProblem(response, request, http.StatusPreconditionFailed,
			"precondition-failed", "Precondition failed", "APPROVAL_PRECONDITION_FAILED", false)
	case errors.Is(err, operations.ErrApprovalIdempotencyConflict):
		a.writeProblem(response, request, http.StatusConflict,
			"idempotency-conflict", "Conflict", "IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_REQUEST", false)
	case errors.Is(err, operations.ErrApprovalConflict):
		a.writeProblem(response, request, http.StatusConflict,
			"approval-conflict", "Conflict", "APPROVAL_STATE_CONFLICT", false)
	case errors.Is(err, operations.ErrApprovalValidationFailed):
		a.writeProblem(response, request, http.StatusUnprocessableEntity,
			"validation-failed", "Validation failed", "VALIDATION_FAILED", false)
	case errors.Is(err, operations.ErrApprovalInputInvalid):
		a.malformed(response, request)
	default:
		a.writeProblem(response, request, http.StatusServiceUnavailable,
			"service-unavailable", "Service unavailable", "SERVICE_UNAVAILABLE", true)
	}
}
