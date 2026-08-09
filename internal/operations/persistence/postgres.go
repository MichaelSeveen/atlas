// Package persistence implements the Operations-owned PostgreSQL approval store.
package persistence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/operations"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var errRetryApprovalTransaction = errors.New("retry approval transaction")

type Store struct {
	pool     *pgxpool.Pool
	recorder audit.Recorder
	identity identity.ApprovalBoundary
}

func NewStore(
	pool *pgxpool.Pool,
	recorder audit.Recorder,
	identityBoundary identity.ApprovalBoundary,
) (*Store, error) {
	if pool == nil || recorder == nil || identityBoundary == nil {
		return nil, errors.New("approval persistence dependencies are incomplete")
	}
	return &Store{pool: pool, recorder: recorder, identity: identityBoundary}, nil
}

type storedApproval struct {
	approval             operations.Approval
	targetID             identifier.ID
	targetVersion        int64
	payloadCanonical     []byte
	payloadSchemaVersion int64
	canonicalization     string
	hashAlgorithm        string
	requesterSessionID   identifier.ID
	deciderSessionID     identifier.ID
	checkerPolicy        string
}

func scanStoredApproval(row pgx.Row) (storedApproval, error) {
	var (
		approvalIDText, tenantIDText, actionType, targetIDText       string
		payloadDigest, payloadCanonical                              []byte
		status, requesterPrincipalIDText, requesterSessionIDText     string
		purpose, checkerPolicy, canonicalization, hashAlgorithm      string
		deciderPrincipalIDText, deciderSessionIDText, decisionReason *string
		expiresAt, createdAt, updatedAt                              time.Time
		decidedAt, executedAt                                        *time.Time
		targetVersion, payloadSchemaVersion, version                 int64
	)
	err := row.Scan(
		&approvalIDText, &tenantIDText, &actionType, &targetIDText, &targetVersion,
		&payloadCanonical, &payloadDigest, &payloadSchemaVersion, &canonicalization,
		&hashAlgorithm, &requesterPrincipalIDText, &requesterSessionIDText,
		&checkerPolicy, &purpose, &status, &decisionReason, &deciderPrincipalIDText,
		&deciderSessionIDText, &version, &expiresAt, &createdAt, &decidedAt,
		&executedAt, &updatedAt,
	)
	if err != nil {
		return storedApproval{}, err
	}
	approvalID, err := parseApprovalIdentifier(approvalIDText, "apr")
	if err != nil {
		return storedApproval{}, operations.ErrApprovalUnavailable
	}
	tenantID, err := parseApprovalIdentifier(tenantIDText, "ten")
	if err != nil {
		return storedApproval{}, operations.ErrApprovalUnavailable
	}
	targetID, err := parseApprovalIdentifier(targetIDText, "mem")
	if err != nil {
		return storedApproval{}, operations.ErrApprovalUnavailable
	}
	requesterPrincipalID, err := parseApprovalIdentifier(requesterPrincipalIDText, "usr")
	if err != nil {
		return storedApproval{}, operations.ErrApprovalUnavailable
	}
	requesterSessionID, err := parseApprovalIdentifier(requesterSessionIDText, "ses")
	if err != nil {
		return storedApproval{}, operations.ErrApprovalUnavailable
	}
	var deciderPrincipalID, deciderSessionID identifier.ID
	if deciderPrincipalIDText != nil {
		deciderPrincipalID, err = parseApprovalIdentifier(*deciderPrincipalIDText, "usr")
		if err != nil {
			return storedApproval{}, operations.ErrApprovalUnavailable
		}
	}
	if deciderSessionIDText != nil {
		deciderSessionID, err = parseApprovalIdentifier(*deciderSessionIDText, "ses")
		if err != nil {
			return storedApproval{}, operations.ErrApprovalUnavailable
		}
	}
	if len(payloadDigest) != sha256Size || payloadSchemaVersion != 1 ||
		canonicalization != "rfc8785" || hashAlgorithm != "sha256" ||
		checkerPolicy != "dynamic-at-decision-and-execution" {
		return storedApproval{}, operations.ErrApprovalUnavailable
	}
	var digest [sha256Size]byte
	copy(digest[:], payloadDigest)
	reason := ""
	if decisionReason != nil {
		reason = *decisionReason
	}
	approval := operations.Approval{
		ApprovalID: approvalID, OrganizationID: tenantID, ActionType: actionType,
		PayloadDigest: digest, Status: operations.ApprovalStatus(status),
		RequesterPrincipalID: requesterPrincipalID, DeciderPrincipalID: deciderPrincipalID,
		Purpose: purpose, DecisionReason: reason, Version: version,
		ExpiresAt: expiresAt.UTC(), CreatedAt: createdAt.UTC(),
	}
	if decidedAt != nil {
		approval.DecidedAt = decidedAt.UTC()
	}
	if executedAt != nil {
		approval.ExecutedAt = executedAt.UTC()
	}
	if !validStoredApproval(approval, updatedAt.UTC()) {
		return storedApproval{}, operations.ErrApprovalUnavailable
	}
	return storedApproval{
		approval: approval, targetID: targetID, targetVersion: targetVersion,
		payloadCanonical:     append([]byte(nil), payloadCanonical...),
		payloadSchemaVersion: payloadSchemaVersion, canonicalization: canonicalization,
		hashAlgorithm: hashAlgorithm, requesterSessionID: requesterSessionID,
		deciderSessionID: deciderSessionID, checkerPolicy: checkerPolicy,
	}, nil
}

const sha256Size = 32

func validStoredApproval(approval operations.Approval, updatedAt time.Time) bool {
	if approval.ApprovalID.IsZero() || approval.OrganizationID.IsZero() ||
		approval.ActionType != operations.ActionMembershipChangeAdmin ||
		approval.RequesterPrincipalID.IsZero() || approval.Purpose != "organization_administration" ||
		approval.Version < 1 || approval.ExpiresAt.IsZero() || approval.CreatedAt.IsZero() ||
		approval.ExpiresAt.Location() != time.UTC || approval.CreatedAt.Location() != time.UTC ||
		updatedAt.Location() != time.UTC || updatedAt.Before(approval.CreatedAt) {
		return false
	}
	switch approval.Status {
	case operations.ApprovalPending, operations.ApprovalApproved, operations.ApprovalRejected,
		operations.ApprovalCancelled, operations.ApprovalExpired, operations.ApprovalExecuted,
		operations.ApprovalExecutionFailed, operations.ApprovalSuperseded:
	default:
		return false
	}
	return true
}

const approvalColumns = `
approval_id, tenant_id, action_type, target_id, target_version,
payload_canonical, payload_sha256, payload_schema_version, payload_canonicalization,
payload_hash_algorithm, requester_principal_id, requester_session_id,
eligible_checker_policy, purpose, status, decision_reason, decider_principal_id,
decider_session_id, version, expires_at, created_at, decided_at, executed_at, updated_at`

func parseApprovalIdentifier(value, prefix string) (identifier.ID, error) {
	id, err := identifier.Parse(value)
	if err != nil || id.Prefix() != prefix {
		return identifier.ID{}, operations.ErrApprovalUnavailable
	}
	return id, nil
}

func effectiveApproval(stored storedApproval, now time.Time) operations.Approval {
	result := stored.approval
	result.Status = result.EffectiveStatus(now)
	return result
}

func beginApprovalTransaction(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	transaction, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, operations.ErrApprovalUnavailable
	}
	return transaction, nil
}

func approvalDatabaseError(err error) error {
	var databaseError *pgconn.PgError
	if errors.As(err, &databaseError) &&
		(databaseError.Code == "40001" || databaseError.Code == "40P01" || databaseError.Code == "55P03") {
		return errRetryApprovalTransaction
	}
	return operations.ErrApprovalUnavailable
}

func mapIdentityAuthorization(result identity.ApprovalAuthorizationResult) error {
	if result.Effect == identity.AuthorizationAllow {
		return nil
	}
	if result.Effect == identity.AuthorizationConceal || result.Reason == "tenant_concealed" {
		return operations.ErrApprovalNotFound
	}
	switch result.Reason {
	case "step_up_required", "assurance_denied":
		return operations.ErrApprovalStepUpRequired
	case "authority_stale_or_revoked", "stale_authority":
		return operations.ErrApprovalAuthenticationRequired
	default:
		return operations.ErrApprovalNotAuthorized
	}
}

func (store *Store) commitDenial(
	ctx context.Context,
	transaction pgx.Tx,
	event audit.Event,
	reason string,
	denial error,
) error {
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeBeforeReference = "approval:unchanged"
	event.SafeAfterReference = "approval:unchanged"
	if errors.Is(denial, operations.ErrApprovalNotFound) {
		event.TargetID = "approval:concealed"
	}
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return operations.ErrApprovalUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return approvalDatabaseError(err)
	}
	return denial
}

func commitApprovalTransaction(ctx context.Context, transaction pgx.Tx) error {
	if err := transaction.Commit(ctx); err != nil {
		return approvalDatabaseError(err)
	}
	return nil
}

type approvalCursor struct {
	CreatedAt  string `json:"created_at"`
	ApprovalID string `json:"approval_id"`
}

func parseApprovalPage(pageSize string, provided bool) (int, error) {
	if !provided {
		return 50, nil
	}
	value, err := strconv.Atoi(pageSize)
	if err != nil || value < 1 || value > 100 || strconv.Itoa(value) != pageSize {
		return 0, operations.ErrApprovalInputInvalid
	}
	return value, nil
}

func encodeApprovalCursor(createdAt time.Time, approvalID identifier.ID) (string, error) {
	payload, err := json.Marshal(approvalCursor{
		CreatedAt: createdAt.UTC().Format(time.RFC3339Nano), ApprovalID: approvalID.String(),
	})
	if err != nil {
		return "", operations.ErrApprovalUnavailable
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeApprovalCursor(value string, provided bool) (time.Time, identifier.ID, error) {
	if !provided {
		return time.Time{}, identifier.ID{}, nil
	}
	if len(value) < 16 || len(value) > 512 || strings.TrimSpace(value) != value {
		return time.Time{}, identifier.ID{}, operations.ErrApprovalInputInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, identifier.ID{}, operations.ErrApprovalInputInvalid
	}
	var cursor approvalCursor
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return time.Time{}, identifier.ID{}, operations.ErrApprovalInputInvalid
	}
	createdAt, err := time.Parse(time.RFC3339Nano, cursor.CreatedAt)
	if err != nil || createdAt.Location() != time.UTC {
		return time.Time{}, identifier.ID{}, operations.ErrApprovalInputInvalid
	}
	approvalID, err := parseApprovalIdentifier(cursor.ApprovalID, "apr")
	if err != nil {
		return time.Time{}, identifier.ID{}, operations.ErrApprovalInputInvalid
	}
	return createdAt, approvalID, nil
}
