package persistence

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const (
	putOIDCTransactionSQL = `
INSERT INTO atlas_identity.oidc_transactions (
    transaction_id, transaction_kind, population, state_sha256, nonce_sha256,
    pkce_verifier_ciphertext, encryption_key_version, return_to, principal_id,
    replaced_session_id, requested_action, status, created_at, expires_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'pending', $12, $13)`

	takeOIDCTransactionSQL = `
UPDATE atlas_identity.oidc_transactions
SET status = 'consumed', consumed_at = $2
WHERE state_sha256 = $1
  AND status = 'pending'
  AND expires_at > $2
RETURNING transaction_id, transaction_kind, population, nonce_sha256,
          pkce_verifier_ciphertext, encryption_key_version, return_to,
          principal_id, replaced_session_id, requested_action, created_at, expires_at`

	externalPrincipalSQL = `
SELECT principal.principal_id, principal.principal_type, principal.display_name
FROM atlas_identity.external_subjects AS external_subject
JOIN atlas_identity.principals AS principal
  ON principal.principal_id = external_subject.principal_id
WHERE external_subject.population = $1
  AND external_subject.issuer = $2
  AND external_subject.subject = $3
  AND principal.status = 'active'
FOR SHARE OF principal`

	membershipAuthoritySQL = `
SELECT membership.tenant_id, membership.role_id,
       GREATEST(principal.authorization_version, membership.authorization_version, organization.authorization_version)
FROM atlas_identity.principals AS principal
JOIN atlas_identity.memberships AS membership
  ON membership.principal_id = principal.principal_id
JOIN atlas_identity.organizations AS organization
  ON organization.tenant_id = membership.tenant_id
WHERE principal.principal_id = $1
  AND principal.status = 'active'
  AND membership.status = 'active'
  AND organization.status = 'active'
ORDER BY membership.tenant_id
LIMIT 2
FOR SHARE OF principal, membership, organization`

	workforceAuthoritySQL = `
SELECT principal_role.role_id,
       GREATEST(principal.authorization_version, principal_role.authorization_version)
FROM atlas_identity.principals AS principal
JOIN atlas_identity.principal_roles AS principal_role
  ON principal_role.principal_id = principal.principal_id
WHERE principal.principal_id = $1
  AND principal.status = 'active'
  AND principal_role.status = 'active'
ORDER BY principal_role.role_id
LIMIT 2
FOR SHARE OF principal, principal_role`

	rolePermissionsSQL = `
SELECT permission_id
FROM atlas_identity.role_permissions
WHERE role_id = $1
ORDER BY permission_id`

	insertSessionSQL = `
INSERT INTO atlas_identity.sessions (
    session_id, principal_id, population, tenant_id, global_scope, verifier_sha256,
    assurance, status, authorization_version, rotation_version, version, created_at,
    last_seen_at, idle_expires_at, absolute_expires_at, client_label
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, 'active', $8, $9, 1, $10,
    $10, $11, $12, $13
)`

	activeSessionCountSQL = `
SELECT count(*)
FROM atlas_identity.sessions
WHERE principal_id = $1
  AND status = 'active'
  AND idle_expires_at > $2
  AND absolute_expires_at > $2`

	sessionByVerifierSQL = `
SELECT session.session_id, session.principal_id, principal.principal_type, principal.display_name,
       session.population, session.tenant_id, session.assurance, session.authorization_version,
       session.rotation_version, session.created_at, session.last_seen_at, session.idle_expires_at,
       session.absolute_expires_at, session.revoked_at, session.client_label, session.status
FROM atlas_identity.sessions AS session
JOIN atlas_identity.principals AS principal
  ON principal.principal_id = session.principal_id
WHERE session.verifier_sha256 = $1
FOR UPDATE OF session`

	listSessionsSQL = `
SELECT session_id, population, assurance, client_label, created_at, last_seen_at,
       idle_expires_at, absolute_expires_at, revoked_at
FROM atlas_identity.sessions
WHERE principal_id = $1
ORDER BY created_at DESC, session_id
LIMIT 100`
)

type SessionStore struct {
	pool     *pgxpool.Pool
	recorder audit.Recorder
}

func NewSessionStore(pool *pgxpool.Pool, recorder audit.Recorder) (*SessionStore, error) {
	if pool == nil || recorder == nil {
		return nil, errors.New("session store dependencies are incomplete")
	}
	return &SessionStore{pool: pool, recorder: recorder}, nil
}

func (store *SessionStore) PutOIDCTransaction(ctx context.Context, transaction identity.OIDCTransaction) error {
	var principalID any
	if !transaction.PrincipalID.IsZero() {
		principalID = transaction.PrincipalID.String()
	}
	var replacedSessionID any
	if !transaction.ReplacedSessionID.IsZero() {
		replacedSessionID = transaction.ReplacedSessionID.String()
	}
	var requestedAction any
	if transaction.RequestedAction != "" {
		requestedAction = transaction.RequestedAction
	}
	_, err := store.pool.Exec(
		ctx, putOIDCTransactionSQL,
		transaction.TransactionID.String(), string(transaction.Kind), string(transaction.Population),
		transaction.StateDigest[:], transaction.NonceDigest[:], transaction.EncryptedPKCE,
		transaction.EncryptionKeyVersion, transaction.ReturnTo, principalID, replacedSessionID,
		requestedAction, transaction.CreatedAt, transaction.ExpiresAt,
	)
	if err != nil {
		return identity.ErrIdentityUnavailable
	}
	return nil
}

func (store *SessionStore) TakeOIDCTransaction(
	ctx context.Context,
	stateDigest [32]byte,
	now time.Time,
) (identity.OIDCTransaction, error) {
	var transaction identity.OIDCTransaction
	var transactionID, kind, population string
	var nonceDigest []byte
	var principalID, replacedSessionID, requestedAction *string
	err := store.pool.QueryRow(ctx, takeOIDCTransactionSQL, stateDigest[:], now).Scan(
		&transactionID, &kind, &population, &nonceDigest, &transaction.EncryptedPKCE,
		&transaction.EncryptionKeyVersion, &transaction.ReturnTo, &principalID,
		&replacedSessionID, &requestedAction, &transaction.CreatedAt, &transaction.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.OIDCTransaction{}, identity.ErrOIDCTransactionInvalid
	}
	if err != nil || len(nonceDigest) != 32 {
		return identity.OIDCTransaction{}, identity.ErrIdentityUnavailable
	}
	copy(transaction.StateDigest[:], stateDigest[:])
	copy(transaction.NonceDigest[:], nonceDigest)
	transaction.TransactionID, err = identifier.Parse(transactionID)
	if err != nil {
		return identity.OIDCTransaction{}, identity.ErrIdentityUnavailable
	}
	transaction.Kind = identity.TransactionKind(kind)
	transaction.Population = identity.Population(population)
	if principalID != nil {
		transaction.PrincipalID, err = identifier.Parse(*principalID)
		if err != nil {
			return identity.OIDCTransaction{}, identity.ErrIdentityUnavailable
		}
	}
	if replacedSessionID != nil {
		transaction.ReplacedSessionID, err = identifier.Parse(*replacedSessionID)
		if err != nil {
			return identity.OIDCTransaction{}, identity.ErrIdentityUnavailable
		}
	}
	if requestedAction != nil {
		transaction.RequestedAction = *requestedAction
	}
	return transaction, nil
}

func (store *SessionStore) ClaimStepUp(
	ctx context.Context,
	command identity.StepUpClaimCommand,
) (identity.StepUpClaimResult, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	inserted, err := insertStepUpClaim(ctx, transaction, command)
	if err != nil {
		return identity.StepUpClaimResult{}, err
	}
	if inserted {
		if err := transaction.Commit(ctx); err != nil {
			return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
		}
		return identity.StepUpClaimResult{
			ChallengeRequestID: command.ChallengeRequestID, Owner: true,
		}, nil
	}

	var requestID, principalID, lifecycle string
	var requestDigest []byte
	var transactionID, authorizationURL *string
	var expiresAt, processingExpiresAt *time.Time
	var retainedUntil time.Time
	err = transaction.QueryRow(ctx, `
SELECT challenge_request_id, principal_id, request_sha256, lifecycle, transaction_id,
       authorization_url, challenge_expires_at, processing_expires_at, retained_until
FROM atlas_identity.step_up_challenge_requests
WHERE idempotency_scope_sha256 = $1
FOR UPDATE`,
		command.IdempotencyScope[:],
	).Scan(
		&requestID, &principalID, &requestDigest, &lifecycle, &transactionID,
		&authorizationURL, &expiresAt, &processingExpiresAt, &retainedUntil,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
	}
	if err != nil {
		return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
	}
	if !command.Now.Before(retainedUntil) {
		tag, updateErr := transaction.Exec(ctx, `
UPDATE atlas_identity.step_up_challenge_requests
SET challenge_request_id = $2,
    principal_id = $3,
    population = $4,
    tenant_id = $5,
    global_scope = $6,
    request_sha256 = $7,
    requested_action = $8,
    lifecycle = 'processing',
    transaction_id = NULL,
    authorization_url = NULL,
    challenge_expires_at = NULL,
    processing_expires_at = $9,
    correlation_id = $10,
    created_at = $11,
    updated_at = $11,
    retained_until = $12
WHERE idempotency_scope_sha256 = $1`,
			command.IdempotencyScope[:], command.ChallengeRequestID.String(),
			command.Actor.PrincipalID.String(), string(command.Actor.Population),
			stepUpTenant(command.Actor), stepUpGlobalScope(command.Actor),
			command.RequestDigest[:], command.Action, command.ProcessingExpiresAt,
			command.CorrelationID.String(), command.Now, command.RetainedUntil,
		)
		if updateErr != nil || tag.RowsAffected() != 1 {
			return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
		}
		return identity.StepUpClaimResult{
			ChallengeRequestID: command.ChallengeRequestID, Owner: true,
		}, nil
	}
	if principalID != command.Actor.PrincipalID.String() ||
		len(requestDigest) != 32 || !equalDigest(requestDigest, command.RequestDigest) {
		return identity.StepUpClaimResult{}, identity.ErrIdempotencyConflict
	}
	parsedRequestID, err := identifier.Parse(requestID)
	if err != nil {
		return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
	}
	if lifecycle == "completed" {
		if transactionID == nil || authorizationURL == nil || expiresAt == nil {
			return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
		}
		parsedTransactionID, parseErr := identifier.Parse(*transactionID)
		if parseErr != nil {
			return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
		}
		return identity.StepUpClaimResult{
			ChallengeRequestID: parsedRequestID, Replay: true,
			TransactionID: parsedTransactionID, AuthorizationURL: *authorizationURL,
			ExpiresAt: expiresAt.UTC(),
		}, nil
	}
	if lifecycle == "processing" && processingExpiresAt != nil &&
		command.Now.Before(*processingExpiresAt) {
		return identity.StepUpClaimResult{}, identity.ErrIdempotencyInProgress
	}
	tag, err := transaction.Exec(ctx, `
UPDATE atlas_identity.step_up_challenge_requests
SET challenge_request_id = $2,
    lifecycle = 'processing',
    processing_expires_at = $4,
    correlation_id = $5,
    updated_at = $3,
    retained_until = GREATEST(retained_until, $6)
WHERE idempotency_scope_sha256 = $1`,
		command.IdempotencyScope[:], command.ChallengeRequestID.String(),
		command.Now, command.ProcessingExpiresAt,
		command.CorrelationID.String(), command.RetainedUntil,
	)
	if err != nil || tag.RowsAffected() != 1 {
		return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.StepUpClaimResult{}, identity.ErrIdentityUnavailable
	}
	return identity.StepUpClaimResult{
		ChallengeRequestID: command.ChallengeRequestID, Owner: true,
	}, nil
}

func insertStepUpClaim(
	ctx context.Context,
	transaction pgx.Tx,
	command identity.StepUpClaimCommand,
) (bool, error) {
	tag, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.step_up_challenge_requests (
    challenge_request_id, principal_id, population, tenant_id, global_scope,
    idempotency_scope_sha256, request_sha256, requested_action, lifecycle,
    processing_expires_at, correlation_id, created_at, updated_at, retained_until
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, 'processing',
    $9, $10, $11, $11, $12
)
ON CONFLICT (idempotency_scope_sha256) DO NOTHING`,
		command.ChallengeRequestID.String(), command.Actor.PrincipalID.String(),
		string(command.Actor.Population), stepUpTenant(command.Actor),
		stepUpGlobalScope(command.Actor), command.IdempotencyScope[:],
		command.RequestDigest[:], command.Action, command.ProcessingExpiresAt,
		command.CorrelationID.String(), command.Now, command.RetainedUntil,
	)
	if err != nil {
		return false, identity.ErrIdentityUnavailable
	}
	return tag.RowsAffected() == 1, nil
}

func (store *SessionStore) CompleteStepUp(
	ctx context.Context,
	command identity.CompleteStepUpCommand,
) error {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	oidc := command.Transaction
	_, err = transaction.Exec(
		ctx, putOIDCTransactionSQL,
		oidc.TransactionID.String(), string(oidc.Kind), string(oidc.Population),
		oidc.StateDigest[:], oidc.NonceDigest[:], oidc.EncryptedPKCE,
		oidc.EncryptionKeyVersion, oidc.ReturnTo, oidc.PrincipalID.String(),
		oidc.ReplacedSessionID.String(), oidc.RequestedAction, oidc.CreatedAt, oidc.ExpiresAt,
	)
	if err != nil {
		return identity.ErrIdentityUnavailable
	}
	tag, err := transaction.Exec(ctx, `
UPDATE atlas_identity.step_up_challenge_requests
SET lifecycle = 'completed',
    transaction_id = $3,
    authorization_url = $4,
    challenge_expires_at = $5,
    processing_expires_at = NULL,
    updated_at = $6
WHERE challenge_request_id = $1
  AND idempotency_scope_sha256 = $2
  AND lifecycle = 'processing'`,
		command.ChallengeRequestID.String(), command.IdempotencyScope[:],
		oidc.TransactionID.String(), command.AuthorizationURL, oidc.ExpiresAt,
		command.CompletedAt,
	)
	if err != nil || tag.RowsAffected() != 1 {
		return identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.ErrIdentityUnavailable
	}
	return nil
}

func (store *SessionStore) FailStepUp(
	ctx context.Context,
	challengeRequestID identifier.ID,
	idempotencyScope [32]byte,
	now time.Time,
) error {
	_, err := store.pool.Exec(ctx, `
UPDATE atlas_identity.step_up_challenge_requests
SET lifecycle = 'failed-retryable',
    processing_expires_at = NULL,
    updated_at = $3
WHERE challenge_request_id = $1
  AND idempotency_scope_sha256 = $2
  AND lifecycle = 'processing'`,
		challengeRequestID.String(), idempotencyScope[:], now,
	)
	if err != nil {
		return identity.ErrIdentityUnavailable
	}
	return nil
}

func stepUpTenant(actor identity.Session) any {
	if actor.TenantID.IsZero() {
		return nil
	}
	return actor.TenantID.String()
}

func stepUpGlobalScope(actor identity.Session) any {
	if actor.TenantID.IsZero() {
		return "identity-security"
	}
	return nil
}

func (store *SessionStore) CreateSession(
	ctx context.Context,
	command identity.CreateSessionCommand,
) (identity.Session, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var principalIDText, principalType, displayName string
	err = transaction.QueryRow(
		ctx, externalPrincipalSQL,
		string(command.Population), command.Claims.Issuer, command.Claims.Subject,
	).Scan(&principalIDText, &principalType, &displayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.Session{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	principalID, err := identifier.Parse(principalIDText)
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if !command.ExpectedPrincipal.IsZero() && command.ExpectedPrincipal != principalID {
		return identity.Session{}, identity.ErrOIDCTransactionInvalid
	}

	tenantID, roleID, authorizationVersion, err := authorityForPrincipal(
		ctx, transaction, principalID, command.Population,
	)
	if err != nil {
		return identity.Session{}, err
	}
	permissions, err := permissionsForRole(ctx, transaction, roleID)
	if err != nil {
		return identity.Session{}, err
	}

	rotationVersion := int64(1)
	if !command.ReplacedSessionID.IsZero() {
		var replacedRotationVersion int64
		err = transaction.QueryRow(ctx, `
SELECT rotation_version
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR UPDATE`,
			command.ReplacedSessionID.String(), principalID.String(),
		).Scan(&replacedRotationVersion)
		if err == nil {
			rotationVersion = replacedRotationVersion + 1
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return identity.Session{}, identity.ErrIdentityUnavailable
		}
		_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $3, version = version + 1
WHERE session_id = $1
  AND principal_id = $2
  AND status = 'active'`,
			command.ReplacedSessionID.String(), principalID.String(), command.AuthorizationAt,
		)
		if err != nil {
			return identity.Session{}, identity.ErrIdentityUnavailable
		}
	}
	var activeCount int
	if err := transaction.QueryRow(
		ctx, activeSessionCountSQL, principalID.String(), command.AuthorizationAt,
	).Scan(&activeCount); err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if activeCount >= 10 {
		return identity.Session{}, identity.ErrSessionConflict
	}

	var tenantValue any
	var globalScope any
	if tenantID.IsZero() {
		globalScope = "workforce"
	} else {
		tenantValue = tenantID.String()
	}
	var clientLabel any
	if command.ClientLabel != "" {
		clientLabel = command.ClientLabel
	}
	_, err = transaction.Exec(
		ctx, insertSessionSQL,
		command.SessionID.String(), principalID.String(), string(command.Population),
		tenantValue, globalScope, command.VerifierDigest[:], string(command.Assurance),
		authorizationVersion, rotationVersion, command.AuthorizationAt, command.IdleExpiresAt,
		command.AbsoluteExpiresAt, clientLabel,
	)
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	event := command.AuditEvent
	event.ActorID = principalID
	event.ActorType = principalType
	if tenantID.IsZero() {
		event.GlobalScope = "identity-security"
		event.TenantID = identifier.ID{}
	} else {
		event.GlobalScope = ""
		event.TenantID = tenantID
	}
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	return identity.Session{
		SessionID: command.SessionID, PrincipalID: principalID, PrincipalType: principalType,
		DisplayName: displayName, Population: command.Population, TenantID: tenantID,
		Assurance: command.Assurance, AuthorizationVersion: authorizationVersion,
		RotationVersion: rotationVersion, CreatedAt: command.AuthorizationAt,
		LastSeenAt: command.AuthorizationAt, IdleExpiresAt: command.IdleExpiresAt,
		AbsoluteExpiresAt: command.AbsoluteExpiresAt, ClientLabel: command.ClientLabel,
		Permissions: permissions,
	}, nil
}

func (store *SessionStore) Authenticate(
	ctx context.Context,
	verifierDigest [32]byte,
	now time.Time,
	_ time.Duration,
) (identity.Session, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	session, status, err := scanSession(transaction.QueryRow(ctx, sessionByVerifierSQL, verifierDigest[:]))
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.Session{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if status == "revoked" {
		return identity.Session{}, identity.ErrSessionRevoked
	}
	if status == "expired" || !now.Before(session.IdleExpiresAt) || !now.Before(session.AbsoluteExpiresAt) {
		_, _ = transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'expired', version = version + 1
WHERE session_id = $1 AND status = 'active'`, session.SessionID.String())
		_ = transaction.Commit(ctx)
		return identity.Session{}, identity.ErrSessionExpired
	}
	tenantID, roleID, authorizationVersion, err := authorityForPrincipal(
		ctx, transaction, session.PrincipalID, session.Population,
	)
	if err != nil || tenantID != session.TenantID || authorizationVersion != session.AuthorizationVersion {
		return identity.Session{}, identity.ErrAuthenticationRequired
	}
	permissions, err := permissionsForRole(ctx, transaction, roleID)
	if err != nil {
		return identity.Session{}, err
	}
	idleDuration := sessionIdleDuration(session.Population)
	nextIdle := now.Add(idleDuration)
	if nextIdle.After(session.AbsoluteExpiresAt) {
		nextIdle = session.AbsoluteExpiresAt
	}
	_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET last_seen_at = GREATEST(last_seen_at, $2),
    idle_expires_at = GREATEST(idle_expires_at, $3),
    version = version + 1
WHERE session_id = $1`, session.SessionID.String(), now, nextIdle)
	if err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.Session{}, identity.ErrIdentityUnavailable
	}
	session.LastSeenAt = now
	session.IdleExpiresAt = nextIdle
	session.Permissions = permissions
	return session, nil
}

func (store *SessionStore) ListSessions(
	ctx context.Context,
	actor identity.Session,
	_ time.Time,
) ([]identity.SessionSummary, error) {
	rows, err := store.pool.Query(ctx, listSessionsSQL, actor.PrincipalID.String())
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	defer rows.Close()
	result := make([]identity.SessionSummary, 0)
	for rows.Next() {
		var summary identity.SessionSummary
		var sessionID, population, assurance string
		var clientLabel *string
		var revokedAt *time.Time
		if err := rows.Scan(
			&sessionID, &population, &assurance, &clientLabel, &summary.CreatedAt,
			&summary.LastSeenAt, &summary.IdleExpiresAt, &summary.AbsoluteExpiresAt, &revokedAt,
		); err != nil {
			return nil, identity.ErrIdentityUnavailable
		}
		summary.SessionID, err = identifier.Parse(sessionID)
		if err != nil {
			return nil, identity.ErrIdentityUnavailable
		}
		summary.Population = identity.Population(population)
		summary.Assurance = identity.Assurance(assurance)
		summary.Current = summary.SessionID == actor.SessionID
		if clientLabel != nil {
			summary.ClientLabel = *clientLabel
		}
		if revokedAt != nil {
			summary.RevokedAt = revokedAt.UTC()
		}
		result = append(result, summary)
	}
	if rows.Err() != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	return result, nil
}

func (store *SessionStore) RevokeCurrent(
	ctx context.Context,
	actor identity.Session,
	now time.Time,
	event audit.Event,
) error {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	tag, err := transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $3, version = version + 1
WHERE session_id = $1 AND principal_id = $2 AND status = 'active'`,
		actor.SessionID.String(), actor.PrincipalID.String(), now,
	)
	if err != nil {
		return identity.ErrIdentityUnavailable
	}
	if tag.RowsAffected() > 0 {
		if err := store.recorder.Record(ctx, transaction, event); err != nil {
			return identity.ErrIdentityUnavailable
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.ErrIdentityUnavailable
	}
	return nil
}

func (store *SessionStore) RevokeOne(
	ctx context.Context,
	command identity.RevocationCommand,
) (identity.RevocationResult, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var status string
	err = transaction.QueryRow(ctx, `
SELECT status
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR UPDATE`,
		command.TargetSessionID.String(), command.Actor.PrincipalID.String(),
	).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.RevocationResult{}, identity.ErrSessionNotFound
	}
	if err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}
	if status == "active" {
		if _, err := transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $2, version = version + 1
WHERE session_id = $1`, command.TargetSessionID.String(), command.Now); err != nil {
			return identity.RevocationResult{}, identity.ErrIdentityUnavailable
		}
		if err := store.recorder.Record(ctx, transaction, command.AuditEvent); err != nil {
			return identity.RevocationResult{}, identity.ErrIdentityUnavailable
		}
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.RevocationResult{
		CurrentRevoked: command.TargetSessionID == command.Actor.SessionID,
	}, nil
}

func (store *SessionStore) RevokeAll(
	ctx context.Context,
	command identity.RevocationCommand,
) (identity.RevocationResult, error) {
	transaction, err := store.pool.Begin(ctx)
	if err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, `
SELECT pg_advisory_xact_lock(
    hashtextextended($1 || ':' || encode($2::bytea, 'hex'), 0)
)`,
		command.Actor.PrincipalID.String(), command.IdempotencyDigest[:],
	); err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}

	var storedRequest []byte
	var currentRevoked bool
	err = transaction.QueryRow(ctx, `
SELECT request_sha256, current_revoked
FROM atlas_identity.session_revocation_requests
WHERE principal_id = $1 AND idempotency_key_sha256 = $2`,
		command.Actor.PrincipalID.String(), command.IdempotencyDigest[:],
	).Scan(&storedRequest, &currentRevoked)
	if err == nil {
		if len(storedRequest) != 32 || !equalDigest(storedRequest, command.RequestDigest) {
			return identity.RevocationResult{}, identity.ErrSessionConflict
		}
		if err := transaction.Commit(ctx); err != nil {
			return identity.RevocationResult{}, identity.ErrIdentityUnavailable
		}
		return identity.RevocationResult{CurrentRevoked: currentRevoked, Replay: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}

	exclusion := ""
	arguments := []any{command.Actor.PrincipalID.String(), command.Now}
	if !command.IncludeCurrent {
		exclusion = " AND session_id <> $3"
		arguments = append(arguments, command.Actor.SessionID.String())
	}
	tag, err := transaction.Exec(ctx, `
UPDATE atlas_identity.sessions
SET status = 'revoked', revoked_at = $2, version = version + 1
WHERE principal_id = $1 AND status = 'active'`+exclusion, arguments...)
	if err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}
	currentRevoked = command.IncludeCurrent && tag.RowsAffected() > 0
	if tag.RowsAffected() > 0 {
		if err := store.recorder.Record(ctx, transaction, command.AuditEvent); err != nil {
			return identity.RevocationResult{}, identity.ErrIdentityUnavailable
		}
	}
	_, err = transaction.Exec(ctx, `
INSERT INTO atlas_identity.session_revocation_requests (
    revocation_request_id, principal_id, idempotency_key_sha256, request_sha256,
    include_current, current_revoked, committed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		command.RevocationRequestID.String(), command.Actor.PrincipalID.String(),
		command.IdempotencyDigest[:], command.RequestDigest[:], command.IncludeCurrent,
		currentRevoked, command.Now,
	)
	if err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.RevocationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.RevocationResult{CurrentRevoked: currentRevoked}, nil
}

func authorityForPrincipal(
	ctx context.Context,
	transaction pgx.Tx,
	principalID identifier.ID,
	population identity.Population,
) (identifier.ID, string, int64, error) {
	if population == identity.PopulationCustomer || population == identity.PopulationMerchant {
		rows, err := transaction.Query(ctx, membershipAuthoritySQL, principalID.String())
		if err != nil {
			return identifier.ID{}, "", 0, identity.ErrIdentityUnavailable
		}
		defer rows.Close()
		type authority struct {
			tenantID string
			roleID   string
			version  int64
		}
		var found []authority
		for rows.Next() {
			var value authority
			if err := rows.Scan(&value.tenantID, &value.roleID, &value.version); err != nil {
				return identifier.ID{}, "", 0, identity.ErrIdentityUnavailable
			}
			found = append(found, value)
		}
		if rows.Err() != nil {
			return identifier.ID{}, "", 0, identity.ErrIdentityUnavailable
		}
		if len(found) != 1 {
			return identifier.ID{}, "", 0, identity.ErrAuthenticationRequired
		}
		tenantID, err := identifier.Parse(found[0].tenantID)
		if err != nil {
			return identifier.ID{}, "", 0, identity.ErrIdentityUnavailable
		}
		return tenantID, found[0].roleID, found[0].version, nil
	}
	if population == identity.PopulationWorkforce {
		rows, err := transaction.Query(ctx, workforceAuthoritySQL, principalID.String())
		if err != nil {
			return identifier.ID{}, "", 0, identity.ErrIdentityUnavailable
		}
		defer rows.Close()
		type authority struct {
			roleID  string
			version int64
		}
		var found []authority
		for rows.Next() {
			var value authority
			if err := rows.Scan(&value.roleID, &value.version); err != nil {
				return identifier.ID{}, "", 0, identity.ErrIdentityUnavailable
			}
			found = append(found, value)
		}
		if rows.Err() != nil {
			return identifier.ID{}, "", 0, identity.ErrIdentityUnavailable
		}
		if len(found) != 1 {
			return identifier.ID{}, "", 0, identity.ErrAuthenticationRequired
		}
		return identifier.ID{}, found[0].roleID, found[0].version, nil
	}
	return identifier.ID{}, "", 0, identity.ErrAuthenticationRequired
}

func permissionsForRole(ctx context.Context, transaction pgx.Tx, roleID string) ([]string, error) {
	rows, err := transaction.Query(ctx, rolePermissionsSQL, roleID)
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	defer rows.Close()
	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, identity.ErrIdentityUnavailable
		}
		permissions = append(permissions, permission)
	}
	if rows.Err() != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	sort.Strings(permissions)
	return permissions, nil
}

func scanSession(row pgx.Row) (identity.Session, string, error) {
	var session identity.Session
	var sessionID, principalID, population, assurance, status string
	var tenantID, clientLabel *string
	var revokedAt *time.Time
	err := row.Scan(
		&sessionID, &principalID, &session.PrincipalType, &session.DisplayName,
		&population, &tenantID, &assurance, &session.AuthorizationVersion,
		&session.RotationVersion, &session.CreatedAt, &session.LastSeenAt,
		&session.IdleExpiresAt, &session.AbsoluteExpiresAt, &revokedAt, &clientLabel, &status,
	)
	if err != nil {
		return identity.Session{}, "", err
	}
	session.SessionID, err = identifier.Parse(sessionID)
	if err != nil {
		return identity.Session{}, "", err
	}
	session.PrincipalID, err = identifier.Parse(principalID)
	if err != nil {
		return identity.Session{}, "", err
	}
	if tenantID != nil {
		session.TenantID, err = identifier.Parse(*tenantID)
		if err != nil {
			return identity.Session{}, "", err
		}
	}
	session.Population = identity.Population(population)
	session.Assurance = identity.Assurance(assurance)
	if clientLabel != nil {
		session.ClientLabel = *clientLabel
	}
	if revokedAt != nil {
		session.RevokedAt = revokedAt.UTC()
	}
	return session, status, nil
}

func sessionIdleDuration(population identity.Population) time.Duration {
	switch population {
	case identity.PopulationCustomer:
		return 30 * time.Minute
	case identity.PopulationMerchant:
		return 20 * time.Minute
	case identity.PopulationWorkforce:
		return 10 * time.Minute
	default:
		return 0
	}
}

func equalDigest(left []byte, right [32]byte) bool {
	if len(left) != len(right) {
		return false
	}
	var difference byte
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	return difference == 0
}

var _ identity.Store = (*SessionStore)(nil)
