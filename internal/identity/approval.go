package identity

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const (
	ApprovalActionCreate             = "operations.approval.create"
	ApprovalActionRead               = "operations.approval.read"
	ApprovalActionDecide             = "operations.approval.decide"
	ApprovalActionExecute            = "operations.approval.execute"
	ApprovalActionCancel             = "operations.approval.cancel"
	ApprovalCheckerEligibilityAction = "operations.approval.checker_eligible"
	ApprovedRoleChangeAction         = "identity.organization.membership.change_admin"
	ApprovalStepUpCreateAction       = "identity.organization.membership.change_admin"
	ApprovalStepUpDecideAction       = "identity.approval.decide"
	ApprovalStepUpExecuteAction      = "identity.approval.execute"
)

// ApprovalTransaction is the caller-owned PostgreSQL transaction used to keep
// Operations state, the Identity target command, and Audit atomic. Identity
// implementations never begin or commit this transaction.
type ApprovalTransaction interface {
	pgx.Tx
}

type ApprovalSessionAuthorizationRequest struct {
	Actor                Session
	OrganizationID       identifier.ID
	Action               string
	AdditionalAction     string
	Purpose              string
	Resource             string
	ResourceVersion      int64
	ResourceStatus       string
	RequiredStepUpAction string
	Now                  time.Time
}

type ApprovalPrincipalAuthorizationRequest struct {
	PrincipalID     identifier.ID
	OrganizationID  identifier.ID
	Action          string
	Purpose         string
	Resource        string
	ResourceVersion int64
	ResourceStatus  string
}

type ApprovalAuthorizationResult struct {
	Effect AuthorizationEffect
	Reason string
	Role   string
}

type MembershipRoleChangeTargetRequest struct {
	OrganizationID  identifier.ID
	MembershipID    identifier.ID
	ExpectedVersion int64
	RequestedRole   string
}

type MembershipRoleChangeTarget struct {
	OrganizationID identifier.ID
	MembershipID   identifier.ID
	PrincipalID    identifier.ID
	BeforeRole     string
	RequestedRole  string
	Version        int64
	CreatedAt      time.Time
}

type ExecuteApprovedMembershipRoleChangeCommand struct {
	ApprovalID        identifier.ID
	RoleChangeID      identifier.ID
	Executor          Session
	Target            MembershipRoleChangeTargetRequest
	PayloadDigest     [32]byte
	IdempotencyDigest [32]byte
	RequestDigest     [32]byte
	Now               time.Time
	AuditEvent        audit.Event
}

type ExecuteApprovedMembershipRoleChangeResult struct {
	Target        MembershipRoleChangeTarget
	ResultVersion int64
}

// ApprovalBoundary is the Identity-owned API used by Operations inside one
// caller-owned transaction. It exposes authorization and one typed target
// command, never raw Identity-table mutation authority.
type ApprovalBoundary interface {
	AuthorizeApprovalSession(context.Context, ApprovalTransaction, ApprovalSessionAuthorizationRequest) (ApprovalAuthorizationResult, error)
	AuthorizeApprovalPrincipal(context.Context, ApprovalTransaction, ApprovalPrincipalAuthorizationRequest) (ApprovalAuthorizationResult, error)
	ValidateMembershipRoleChangeTarget(context.Context, ApprovalTransaction, MembershipRoleChangeTargetRequest) (MembershipRoleChangeTarget, error)
	ExecuteApprovedMembershipRoleChange(context.Context, ApprovalTransaction, ExecuteApprovedMembershipRoleChangeCommand) (ExecuteApprovedMembershipRoleChangeResult, error)
}
