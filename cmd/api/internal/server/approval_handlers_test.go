package server

import (
	"context"
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/operations"
	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	metricdata "go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type approvalHTTPIdentity struct {
	session identity.Session
	csrf    string
	err     error
}

func (boundary approvalHTTPIdentity) Current(context.Context, string) (identity.Session, string, error) {
	return boundary.session, boundary.csrf, boundary.err
}

type approvalHTTPStore struct {
	approval       operations.Approval
	decisionID     identifier.ID
	createCommand  operations.CreateApprovalCommand
	listCommand    operations.ListApprovalsCommand
	getCommand     operations.GetApprovalCommand
	decideCommand  operations.DecideApprovalCommand
	executeCommand operations.ExecuteApprovalCommand
	cancelCommand  operations.CancelApprovalCommand
}

func (store *approvalHTTPStore) Create(
	_ context.Context,
	command operations.CreateApprovalCommand,
) (operations.ApprovalResult, error) {
	store.createCommand = command
	if !store.approval.ApprovalID.IsZero() {
		return operations.ApprovalResult{
			Approval: store.approval, DecisionID: store.decisionID, Replay: true,
		}, nil
	}
	store.approval = operations.Approval{
		ApprovalID: command.ApprovalID, OrganizationID: command.Payload.OrganizationID,
		ActionType: command.ActionType, PayloadDigest: command.PayloadDigest,
		Status: operations.ApprovalPending, RequesterPrincipalID: command.Actor.PrincipalID,
		Purpose: command.Payload.Purpose, Version: 1, ExpiresAt: command.ExpiresAt,
		CreatedAt: command.Now,
	}
	store.decisionID = command.AuditEvent.DecisionID
	return operations.ApprovalResult{Approval: store.approval, DecisionID: store.decisionID}, nil
}

func (store *approvalHTTPStore) List(
	_ context.Context,
	command operations.ListApprovalsCommand,
) (operations.ApprovalPage, error) {
	store.listCommand = command
	return operations.ApprovalPage{Approvals: []operations.Approval{store.approval}}, nil
}

func (store *approvalHTTPStore) Get(
	_ context.Context,
	command operations.GetApprovalCommand,
) (operations.ApprovalResult, error) {
	store.getCommand = command
	return operations.ApprovalResult{Approval: store.approval, DecisionID: command.AuditEvent.DecisionID}, nil
}

func (store *approvalHTTPStore) Decide(
	_ context.Context,
	command operations.DecideApprovalCommand,
) (operations.ApprovalResult, error) {
	store.decideCommand = command
	store.approval.Status = operations.ApprovalApproved
	if command.Decision == "reject" {
		store.approval.Status = operations.ApprovalRejected
	}
	store.approval.DeciderPrincipalID = command.Actor.PrincipalID
	store.approval.DecisionReason = command.Reason
	store.approval.DecidedAt = command.Now
	store.approval.Version++
	return operations.ApprovalResult{Approval: store.approval, DecisionID: command.AuditEvent.DecisionID}, nil
}

func (store *approvalHTTPStore) Execute(
	_ context.Context,
	command operations.ExecuteApprovalCommand,
) (operations.ApprovalResult, error) {
	store.executeCommand = command
	store.approval.Status = operations.ApprovalExecuted
	store.approval.ExecutedAt = command.Now
	store.approval.Version++
	return operations.ApprovalResult{Approval: store.approval, DecisionID: command.AuditEvent.DecisionID}, nil
}

func (store *approvalHTTPStore) Cancel(
	_ context.Context,
	command operations.CancelApprovalCommand,
) (operations.ApprovalResult, error) {
	store.cancelCommand = command
	store.approval.Status = operations.ApprovalCancelled
	store.approval.DecisionReason = command.Reason
	store.approval.Version++
	return operations.ApprovalResult{Approval: store.approval, DecisionID: command.AuditEvent.DecisionID}, nil
}

func newHTTPApprovalService(
	t *testing.T,
	boundary interface {
		Current(context.Context, string) (identity.Session, string, error)
	},
) (*operations.Service, *approvalHTTPStore) {
	t.Helper()
	store := &approvalHTTPStore{}
	service, err := operations.NewService(operations.ServiceOptions{
		Store: store, Identity: boundary, Clock: clock.NewFixed(testBuildTime), NewID: staticID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, store
}

func TestApprovalHTTPLifecycleAndStrictBoundary(t *testing.T) {
	principalID, _ := identifier.Parse("usr_00000000000000000201")
	sessionID, _ := identifier.Parse("ses_00000000000000000201")
	tenantID, _ := identifier.Parse("ten_00000000000000000201")
	membershipID, _ := identifier.Parse("mem_00000000000000000202")
	cookieValue := strings.Repeat("Q", 43)
	const csrfToken = "approval-http-csrf"
	boundary := approvalHTTPIdentity{session: identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		Population: identity.PopulationMerchant, TenantID: tenantID,
		Assurance: identity.AssurancePhishingResistant,
	}, csrf: csrfToken}
	service, store := newHTTPApprovalService(t, boundary)
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Approvals = service
		options.WebOrigin = "https://web.test.invalid"
		options.Meter = provider.Meter("atlas-approval-http-test")
	})

	createBody := `{"action_type":"identity.organization.membership.change_admin","payload":{` +
		`"organization_id":"` + tenantID.String() + `","member_id":"` + membershipID.String() + `",` +
		`"expected_membership_version":7,"requested_role":"merchant_admin",` +
		`"purpose":"organization_administration"}}`
	create := approvalMutationRequest(http.MethodPost, "/v1/approvals", createBody,
		cookieValue, csrfToken, "approval-create-0001", "")
	created := httptest.NewRecorder()
	app.Handler().ServeHTTP(created, create)
	if created.Code != http.StatusCreated || created.Header().Get("ETag") != `"1"` ||
		created.Header().Get("Idempotency-Replayed") != "false" ||
		created.Header().Get("Location") != "/v1/approvals/"+store.approval.ApprovalID.String() ||
		!strings.HasPrefix(created.Header().Get("X-Authorization-Decision-Id"), "dec_") ||
		!strings.Contains(created.Body.String(), `"status":"pending"`) {
		t.Fatalf("create status=%d headers=%v body=%s", created.Code, created.Header(), created.Body)
	}
	if store.createCommand.Payload.OrganizationID != tenantID ||
		store.createCommand.Payload.MembershipID != membershipID ||
		store.createCommand.Payload.ExpectedMembershipVersion != 7 ||
		store.createCommand.PayloadDigest == ([32]byte{}) ||
		sha256.Sum256(store.createCommand.CanonicalPayload) != store.createCommand.PayloadDigest {
		t.Fatalf("create command was not canonically bound: %#v", store.createCommand)
	}
	replayCreate := approvalMutationRequest(http.MethodPost, "/v1/approvals", createBody,
		cookieValue, csrfToken, "approval-create-0001", "")
	replayedCreate := httptest.NewRecorder()
	app.Handler().ServeHTTP(replayedCreate, replayCreate)
	if replayedCreate.Code != http.StatusCreated ||
		replayedCreate.Header().Get("Idempotency-Replayed") != "true" ||
		replayedCreate.Header().Get("X-Authorization-Decision-Id") != created.Header().Get("X-Authorization-Decision-Id") {
		t.Fatalf("create replay status=%d headers=%v body=%s",
			replayedCreate.Code, replayedCreate.Header(), replayedCreate.Body)
	}

	list := httptest.NewRequest(http.MethodGet, "/v1/approvals?page_size=25", nil)
	list.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	listed := httptest.NewRecorder()
	app.Handler().ServeHTTP(listed, list)
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), store.approval.ApprovalID.String()) ||
		store.listCommand.PageSize != "25" || !store.listCommand.PageProvided {
		t.Fatalf("list status=%d body=%s command=%#v", listed.Code, listed.Body, store.listCommand)
	}

	approvalPath := "/v1/approvals/" + store.approval.ApprovalID.String()
	get := httptest.NewRequest(http.MethodGet, approvalPath, nil)
	get.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	fetched := httptest.NewRecorder()
	app.Handler().ServeHTTP(fetched, get)
	if fetched.Code != http.StatusOK || fetched.Header().Get("ETag") != `"1"` ||
		store.getCommand.ApprovalID != store.approval.ApprovalID {
		t.Fatalf("get status=%d headers=%v body=%s", fetched.Code, fetched.Header(), fetched.Body)
	}

	decide := approvalMutationRequest(http.MethodPost, approvalPath+"/decisions",
		`{"decision":"approve","purpose":"approval_review","reason":"independent review"}`,
		cookieValue, csrfToken, "approval-decide-0001", `"1"`)
	decided := httptest.NewRecorder()
	app.Handler().ServeHTTP(decided, decide)
	if decided.Code != http.StatusOK || decided.Header().Get("ETag") != `"2"` ||
		store.decideCommand.ExpectedVersion != 1 || store.decideCommand.Decision != "approve" ||
		store.decideCommand.Purpose != "approval_review" {
		t.Fatalf("decide status=%d headers=%v body=%s", decided.Code, decided.Header(), decided.Body)
	}

	execute := approvalMutationRequest(http.MethodPost, approvalPath+"/executions", "",
		cookieValue, csrfToken, "approval-execute-0001", `"2"`)
	executed := httptest.NewRecorder()
	app.Handler().ServeHTTP(executed, execute)
	if executed.Code != http.StatusOK || executed.Header().Get("ETag") != `"3"` ||
		store.executeCommand.ExpectedVersion != 2 ||
		!strings.Contains(executed.Body.String(), `"status":"executed"`) {
		t.Fatalf("execute status=%d headers=%v body=%s", executed.Code, executed.Header(), executed.Body)
	}

	store.approval.Status = operations.ApprovalPending
	store.approval.Version = 4
	cancel := approvalMutationRequest(http.MethodPost, approvalPath+"/cancellations",
		`{"reason":"request withdrawn"}`, cookieValue, csrfToken, "approval-cancel-0001", `"4"`)
	cancelled := httptest.NewRecorder()
	app.Handler().ServeHTTP(cancelled, cancel)
	if cancelled.Code != http.StatusOK || cancelled.Header().Get("ETag") != `"5"` ||
		cancelled.Header().Get("Location") != approvalPath ||
		store.cancelCommand.ExpectedVersion != 4 || store.cancelCommand.Reason != "request withdrawn" ||
		!strings.Contains(cancelled.Body.String(), `"status":"cancelled"`) {
		t.Fatalf("cancel status=%d headers=%v body=%s", cancelled.Code, cancelled.Header(), cancelled.Body)
	}

	unknown := approvalMutationRequest(http.MethodPost, "/v1/approvals", createBody[:len(createBody)-1]+`,"admin":true}`,
		cookieValue, csrfToken, "approval-create-0002", "")
	unknownResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(unknownResponse, unknown)
	if unknownResponse.Code != http.StatusBadRequest || store.createCommand.IdempotencyDigest == sha256.Sum256([]byte("approval-create-0002")) {
		t.Fatalf("unknown field status=%d body=%s", unknownResponse.Code, unknownResponse.Body)
	}

	duplicate := approvalMutationRequest(http.MethodPost, "/v1/approvals", createBody,
		cookieValue, csrfToken, "approval-create-0003", "")
	duplicate.Header.Add("Idempotency-Key", "approval-create-duplicate")
	duplicateResponse := httptest.NewRecorder()
	app.Handler().ServeHTTP(duplicateResponse, duplicate)
	if duplicateResponse.Code != http.StatusBadRequest {
		t.Fatalf("duplicate header status=%d body=%s", duplicateResponse.Code, duplicateResponse.Body)
	}
	tampered := store.approval
	tampered.Status = operations.ApprovalSuperseded
	tampered.DecisionReason = "payload_digest_mismatch"
	app.recordApprovalObservation(execute, "execute", tampered, operations.ErrApprovalConflict)
	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &metrics); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, scope := range metrics.ScopeMetrics {
		for _, metric := range scope.Metrics {
			seen[metric.Name] = true
		}
	}
	for _, name := range []string{
		"atlas.operations.approval.operation.count",
		"atlas.operations.approval.operation.duration",
		"atlas.operations.approval.status.count",
		"atlas.operations.approval.age",
		"atlas.operations.approval.conflict.count",
		"atlas.operations.approval.integrity_failure.count",
	} {
		if !seen[name] {
			t.Errorf("approval metric %s was not emitted", name)
		}
	}
}

func TestApprovalRouteInventoryIsClosed(t *testing.T) {
	want := []string{
		"/v1/approvals",
		"/v1/approvals/{approval_id}",
		"/v1/approvals/{approval_id}/decisions",
		"/v1/approvals/{approval_id}/executions",
		"/v1/approvals/{approval_id}/cancellations",
	}
	if strings.Join(approvalRoutes, ",") != strings.Join(want, ",") {
		t.Fatalf("approval route inventory=%v", approvalRoutes)
	}
}

func TestAdministratorPromotionPATCHCreatesApproval(t *testing.T) {
	identityService, identityStore, _ := newHTTPIdentityService(t)
	cookieValue := strings.Repeat("R", 43)
	principalID, _ := identifier.Parse("usr_00000000000000000211")
	sessionID, _ := identifier.Parse("ses_00000000000000000211")
	tenantID, _ := identifier.Parse("ten_00000000000000000211")
	membershipID, _ := identifier.Parse("mem_00000000000000000212")
	identityStore.sessions[sha256.Sum256([]byte(cookieValue))] = identity.Session{
		SessionID: sessionID, PrincipalID: principalID, PrincipalType: "merchant",
		Population: identity.PopulationMerchant, TenantID: tenantID,
		Assurance:            identity.AssurancePhishingResistant,
		AuthorizationVersion: 1, RotationVersion: 1, CreatedAt: testBuildTime,
		LastSeenAt: testBuildTime, IdleExpiresAt: testBuildTime.AddDate(0, 0, 1),
		AbsoluteExpiresAt: testBuildTime.AddDate(0, 0, 1),
	}
	_, csrfToken, err := identityService.Current(context.Background(), cookieValue)
	if err != nil {
		t.Fatal(err)
	}
	approvalService, store := newHTTPApprovalService(t, identityService)
	app := newTestApp(t, ReadinessState{DependenciesReady: true, MigrationsCurrent: true}, func(options *Options) {
		options.Identity = identityService
		options.Approvals = approvalService
		options.WebOrigin = "https://web.test.invalid"
	})

	path := "/v1/organizations/" + tenantID.String() + "/members/" + membershipID.String()
	request := approvalMutationRequest(http.MethodPatch, path,
		`{"role":"merchant_admin","purpose":"organization_administration"}`,
		cookieValue, csrfToken, "approval-patch-0001", `"membership-v4"`)
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || response.Header().Get("ETag") != `"1"` ||
		store.createCommand.Payload.ExpectedMembershipVersion != 4 ||
		store.createCommand.Payload.RequestedRole != "merchant_admin" ||
		!strings.Contains(response.Body.String(), `"status":"pending"`) {
		t.Fatalf("promotion status=%d headers=%v body=%s", response.Code, response.Header(), response.Body)
	}
}

func approvalMutationRequest(
	method, path, body, cookieValue, csrfToken, idempotencyKey, ifMatch string,
) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set(identity.CSRFHeaderName, csrfToken)
	request.Header.Set("Idempotency-Key", idempotencyKey)
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	request.AddCookie(&http.Cookie{Name: identity.SessionCookieName, Value: cookieValue})
	return request
}
