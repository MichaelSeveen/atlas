package persistence

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var credentialDummyVerifier = sha256.Sum256([]byte("atlas-invalid-credential-verifier-v1"))

// CredentialStore owns authoritative merchant API-credential lifecycle state.
// Redis is deliberately absent from this adapter: validity and revocation are
// checked under a PostgreSQL row lock on every machine request.
type CredentialStore struct {
	pool          *pgxpool.Pool
	recorder      audit.Recorder
	authorization *identity.AuthorizationPolicy
}

func NewCredentialStore(pool *pgxpool.Pool, recorder audit.Recorder) (*CredentialStore, error) {
	if pool == nil || recorder == nil {
		return nil, errors.New("credential store dependencies are incomplete")
	}
	return &CredentialStore{
		pool: pool, recorder: recorder, authorization: identity.DefaultAuthorizationPolicy(),
	}, nil
}

func (store *CredentialStore) ListCredentials(
	ctx context.Context,
	command identity.ListCredentialsCommand,
) ([]identity.APICredential, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	decision, reason, err := store.authorizeActor(
		ctx, transaction, command.Actor, identity.CredentialActionRead, false, command.Now,
	)
	if err != nil {
		return nil, err
	}
	if decision != identity.AuthorizationAllow {
		return nil, authorizationFailure(reason)
	}
	rows, err := transaction.Query(ctx, `
SELECT credential_id, tenant_id, name, secret_hint, scopes,
       CASE
         WHEN expires_at <= $2 THEN 'expired'
         WHEN status = 'rotating' AND overlap_ends_at <= $2 THEN 'revoked'
         ELSE status
       END AS effective_status,
       environment, audience, version, expires_at, overlap_ends_at, last_used_at,
       created_at, revoked_at, previous_credential_id, replacement_credential_id
FROM atlas_identity.api_credentials
WHERE tenant_id = $1 AND environment = $3
ORDER BY created_at DESC, credential_id DESC
LIMIT 100`, command.Actor.TenantID.String(), command.Now, command.Environment)
	if err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	defer rows.Close()
	credentials := make([]identity.APICredential, 0)
	for rows.Next() {
		credential, scanErr := scanCredential(rows)
		if scanErr != nil {
			return nil, identity.ErrIdentityUnavailable
		}
		credentials = append(credentials, credential)
	}
	if rows.Err() != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return nil, identity.ErrIdentityUnavailable
	}
	return credentials, nil
}

func (store *CredentialStore) CreateCredential(
	ctx context.Context,
	command identity.CreateCredentialCommand,
) (identity.CredentialMutationResult, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	decision, reason, err := store.authorizeActor(
		ctx, transaction, command.Actor, identity.CredentialActionCreate, true, command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, err
	}
	if decision != identity.AuthorizationAllow {
		return store.commitDenied(ctx, transaction, command.AuditEvent, reason)
	}
	claimed, err := claimCredentialMutation(
		ctx, transaction, command.Actor, "create", command.IdempotencyDigest,
		command.RequestDigest, nil, command.Credential.CredentialID,
		command.AuditEvent, command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if !claimed {
		return replayCredentialMutation(
			ctx, transaction, command.Actor, "create", command.IdempotencyDigest,
			command.RequestDigest, command.Now,
		)
	}
	_, err = transaction.Exec(ctx, `
INSERT INTO atlas_identity.api_credentials (
    credential_id, tenant_id, name, secret_verifier_sha256, secret_hint,
    verifier_algorithm, verifier_version, scopes, environment, audience,
    status, version, expires_at, created_by_principal_id, created_at
) VALUES (
    $1, $2, $3, $4, $5,
    'sha256', 1, $6, $7, 'atlas-api',
    'active', 1, $8, $9, $10
)`,
		command.Credential.CredentialID.String(), command.Actor.TenantID.String(),
		command.Credential.Name, command.Verifier[:], command.Credential.SecretHint,
		command.Credential.Scopes, command.Credential.Environment, command.Credential.ExpiresAt,
		command.Actor.PrincipalID.String(), command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if err := store.recorder.Record(ctx, transaction, command.AuditEvent); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.CredentialMutationResult{
		Credential: command.Credential, DecisionID: command.AuditEvent.DecisionID,
	}, nil
}

func (store *CredentialStore) RotateCredential(
	ctx context.Context,
	command identity.RotateCredentialCommand,
) (identity.CredentialMutationResult, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	decision, reason, err := store.authorizeActor(
		ctx, transaction, command.Actor, identity.CredentialActionRotate, true, command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, err
	}
	if decision != identity.AuthorizationAllow {
		return store.commitDenied(ctx, transaction, command.AuditEvent, reason)
	}
	claimTransaction, err := transaction.Begin(ctx)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = claimTransaction.Rollback(ctx) }()
	requested := command.SourceCredentialID
	claimed, err := claimCredentialMutation(
		ctx, claimTransaction, command.Actor, "rotate", command.IdempotencyDigest,
		command.RequestDigest, &requested, command.ReplacementID, command.AuditEvent, command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if !claimed {
		if err := claimTransaction.Rollback(ctx); err != nil {
			return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
		}
		return replayCredentialMutation(
			ctx, transaction, command.Actor, "rotate", command.IdempotencyDigest,
			command.RequestDigest, command.Now,
		)
	}
	source, err := credentialByIDForUpdate(
		ctx, claimTransaction, command.Actor.TenantID, command.SourceCredentialID, command.Environment, command.Now,
	)
	if errors.Is(err, identity.ErrCredentialNotFound) {
		if rollbackErr := claimTransaction.Rollback(ctx); rollbackErr != nil {
			return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
		}
		return store.commitCredentialRejected(
			ctx, transaction, command.AuditEvent, "credential_not_found", identity.ErrCredentialNotFound,
		)
	}
	if err != nil {
		return identity.CredentialMutationResult{}, err
	}
	if source.Status != identity.APICredentialActive ||
		source.Audience != identity.APICredentialAudience || len(source.Scopes) != 1 ||
		source.Scopes[0] != identity.APICredentialScopeIdentityRead {
		if rollbackErr := claimTransaction.Rollback(ctx); rollbackErr != nil {
			return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
		}
		return store.commitCredentialRejected(
			ctx, transaction, command.AuditEvent, "credential_state_conflict", identity.ErrCredentialConflict,
		)
	}
	if err := claimTransaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	replacement := identity.APICredential{
		CredentialID: command.ReplacementID, OrganizationID: source.OrganizationID,
		Name: source.Name, SecretHint: command.ReplacementSecretHint,
		Scopes: append([]string(nil), source.Scopes...), Status: identity.APICredentialActive,
		Environment: source.Environment, Audience: source.Audience, Version: 1,
		ExpiresAt: command.Now.Add(identity.CredentialDefaultExpiry), CreatedAt: command.Now,
		PreviousCredentialID: source.CredentialID,
	}
	_, err = transaction.Exec(ctx, `
INSERT INTO atlas_identity.api_credentials (
    credential_id, tenant_id, name, secret_verifier_sha256, secret_hint,
    verifier_algorithm, verifier_version, scopes, environment, audience,
    status, version, expires_at, created_by_principal_id, previous_credential_id, created_at
) VALUES (
    $1, $2, $3, $4, $5,
    'sha256', 1, $6, $7, 'atlas-api',
    'active', 1, $8, $9, $10, $11
)`,
		replacement.CredentialID.String(), replacement.OrganizationID.String(), replacement.Name,
		command.ReplacementVerifier[:], replacement.SecretHint, replacement.Scopes,
		replacement.Environment, replacement.ExpiresAt, command.Actor.PrincipalID.String(),
		source.CredentialID.String(), command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	overlapEndsAt := command.Now.Add(identity.CredentialRotationOverlap)
	updated, err := transaction.Exec(ctx, `
UPDATE atlas_identity.api_credentials
SET status = 'rotating', overlap_ends_at = $3, replacement_credential_id = $4,
    version = version + 1
WHERE tenant_id = $1 AND credential_id = $2 AND status = 'active'`,
		command.Actor.TenantID.String(), source.CredentialID.String(), overlapEndsAt,
		replacement.CredentialID.String(),
	)
	if err != nil || updated.RowsAffected() != 1 {
		return identity.CredentialMutationResult{}, identity.ErrCredentialConflict
	}
	event := command.AuditEvent
	event.SafeBeforeReference = "credential-status:active"
	event.SafeAfterReference = "credential-status:rotating"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.CredentialMutationResult{
		Credential: replacement, DecisionID: command.AuditEvent.DecisionID,
	}, nil
}

func (store *CredentialStore) RevokeCredential(
	ctx context.Context,
	command identity.RevokeCredentialCommand,
) (identity.CredentialMutationResult, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	decision, reason, err := store.authorizeActor(
		ctx, transaction, command.Actor, identity.CredentialActionRevoke, true, command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, err
	}
	if decision != identity.AuthorizationAllow {
		return store.commitDenied(ctx, transaction, command.AuditEvent, reason)
	}
	claimTransaction, err := transaction.Begin(ctx)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = claimTransaction.Rollback(ctx) }()
	requested := command.CredentialID
	claimed, err := claimCredentialMutation(
		ctx, claimTransaction, command.Actor, "revoke", command.IdempotencyDigest,
		command.RequestDigest, &requested, command.CredentialID, command.AuditEvent, command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if !claimed {
		if err := claimTransaction.Rollback(ctx); err != nil {
			return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
		}
		return replayCredentialMutation(
			ctx, transaction, command.Actor, "revoke", command.IdempotencyDigest,
			command.RequestDigest, command.Now,
		)
	}
	credential, err := credentialByIDForUpdate(
		ctx, claimTransaction, command.Actor.TenantID, command.CredentialID, command.Environment, command.Now,
	)
	if errors.Is(err, identity.ErrCredentialNotFound) {
		if rollbackErr := claimTransaction.Rollback(ctx); rollbackErr != nil {
			return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
		}
		return store.commitCredentialRejected(
			ctx, transaction, command.AuditEvent, "credential_not_found", identity.ErrCredentialNotFound,
		)
	}
	if err != nil {
		return identity.CredentialMutationResult{DecisionID: command.AuditEvent.DecisionID}, err
	}
	if credential.Status == identity.APICredentialRevoked || credential.Status == identity.APICredentialExpired {
		if rollbackErr := claimTransaction.Rollback(ctx); rollbackErr != nil {
			return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
		}
		return store.commitCredentialRejected(
			ctx, transaction, command.AuditEvent, "credential_state_conflict", identity.ErrCredentialConflict,
		)
	}
	if err := claimTransaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	beforeStatus := credential.Status
	_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.api_credentials
SET status = 'revoked', revoked_at = $3, version = version + 1
WHERE tenant_id = $1 AND credential_id = $2`,
		command.Actor.TenantID.String(), command.CredentialID.String(), command.Now,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	credential.Status = identity.APICredentialRevoked
	credential.RevokedAt = command.Now
	credential.Version++
	event := command.AuditEvent
	event.SafeBeforeReference = "credential-status:" + string(beforeStatus)
	event.SafeAfterReference = "credential-status:revoked"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.CredentialMutationResult{
		Credential: credential, DecisionID: command.AuditEvent.DecisionID,
	}, nil
}

func (store *CredentialStore) commitCredentialRejected(
	ctx context.Context,
	transaction pgx.Tx,
	event audit.Event,
	reason string,
	resultErr error,
) (identity.CredentialMutationResult, error) {
	event.Decision = "denied"
	event.ReasonCode = reason
	event.SafeAfterReference = "credential-status:unchanged"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.CredentialMutationResult{DecisionID: event.DecisionID}, resultErr
}

func (store *CredentialStore) AuthenticateCredential(
	ctx context.Context,
	command identity.AuthenticateCredentialCommand,
) (identity.CredentialAuthentication, error) {
	transaction, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return identity.CredentialAuthentication{}, identity.ErrIdentityUnavailable
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var (
		credentialIDText, tenantIDText, name, secretHint, verifierAlgorithm string
		environment, audience, status                                       string
		scopes                                                              []string
		storedVerifier, previousNetwork                                     []byte
		version, verifierVersion                                            int64
		expiresAt, createdAt                                                time.Time
		overlapEndsAt, lastUsedAt, revokedAt                                *time.Time
		previousIDText, replacementIDText                                   *string
	)
	err = transaction.QueryRow(ctx, `
SELECT credential_id, tenant_id, name, secret_hint, secret_verifier_sha256,
       verifier_algorithm, verifier_version, scopes, environment, audience, status, version,
       expires_at, overlap_ends_at, last_used_at, last_used_network_signal_sha256,
       created_at, revoked_at, previous_credential_id, replacement_credential_id
FROM atlas_identity.api_credentials
WHERE credential_id = $1
FOR UPDATE`, command.CredentialID.String()).Scan(
		&credentialIDText, &tenantIDText, &name, &secretHint, &storedVerifier,
		&verifierAlgorithm, &verifierVersion, &scopes, &environment, &audience, &status, &version,
		&expiresAt, &overlapEndsAt, &lastUsedAt, &previousNetwork,
		&createdAt, &revokedAt, &previousIDText, &replacementIDText,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = subtle.ConstantTimeCompare(credentialDummyVerifier[:], command.Verifier[:])
		return identity.CredentialAuthentication{}, identity.ErrAuthenticationRequired
	}
	if err != nil {
		return identity.CredentialAuthentication{}, identity.ErrIdentityUnavailable
	}
	verifierMatches := len(storedVerifier) == 32 &&
		subtle.ConstantTimeCompare(storedVerifier, command.Verifier[:]) == 1
	statusAllows := status == string(identity.APICredentialActive) ||
		(status == string(identity.APICredentialRotating) && overlapEndsAt != nil && command.Now.Before(*overlapEndsAt))
	if !verifierMatches || verifierAlgorithm != "sha256" || verifierVersion != 1 ||
		environment != command.Environment || audience != command.Audience ||
		len(scopes) != 1 || scopes[0] != command.Scope || !statusAllows ||
		!command.Now.Before(expiresAt) || revokedAt != nil {
		return identity.CredentialAuthentication{}, identity.ErrAuthenticationRequired
	}
	credentialID, err := identifier.Parse(credentialIDText)
	if err != nil {
		return identity.CredentialAuthentication{}, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.CredentialAuthentication{}, identity.ErrIdentityUnavailable
	}
	anomalous := len(previousNetwork) == 32 &&
		subtle.ConstantTimeCompare(previousNetwork, command.NetworkSignal[:]) != 1
	_, err = transaction.Exec(ctx, `
UPDATE atlas_identity.api_credentials
SET last_used_at = $2,
    last_used_network_signal_sha256 = $3,
    last_use_anomalous = $4,
    authentication_count = authentication_count + 1
WHERE credential_id = $1`,
		credentialID.String(), command.Now, command.NetworkSignal[:], anomalous,
	)
	if err != nil {
		return identity.CredentialAuthentication{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CredentialAuthentication{}, identity.ErrIdentityUnavailable
	}
	credential := identity.APICredential{
		CredentialID: credentialID, OrganizationID: tenantID, Name: name, SecretHint: secretHint,
		Scopes: append([]string(nil), scopes...), Status: identity.APICredentialStatus(status),
		Environment: environment, Audience: audience, Version: version,
		ExpiresAt: expiresAt.UTC(), CreatedAt: createdAt.UTC(), LastUsedAt: command.Now,
	}
	credential.OverlapEndsAt = utcTime(overlapEndsAt)
	credential.RevokedAt = utcTime(revokedAt)
	credential.PreviousCredentialID = parsedOptionalID(previousIDText)
	credential.ReplacementCredentialID = parsedOptionalID(replacementIDText)
	return identity.CredentialAuthentication{
		Credential: credential, Permission: "identity.me.read", AnomalousNetwork: anomalous,
	}, nil
}

func (store *CredentialStore) authorizeActor(
	ctx context.Context,
	transaction pgx.Tx,
	actor identity.Session,
	action string,
	requireStepUp bool,
	now time.Time,
) (identity.AuthorizationEffect, string, error) {
	if actor.SessionID.IsZero() || actor.PrincipalID.IsZero() ||
		actor.Population != identity.PopulationMerchant || actor.TenantID.IsZero() {
		return identity.AuthorizationDeny, "invalid_authorization_facts", nil
	}
	var (
		status, population, tenantIDText, assurance string
		authorizationVersion, rotationVersion       int64
		idleExpiresAt, absoluteExpiresAt            time.Time
		stepUpAction                                *string
		stepUpVerifiedAt                            *time.Time
	)
	err := transaction.QueryRow(ctx, `
SELECT status, population, tenant_id, assurance, authorization_version, rotation_version,
       idle_expires_at, absolute_expires_at, step_up_action, step_up_verified_at
FROM atlas_identity.sessions
WHERE session_id = $1 AND principal_id = $2
FOR SHARE`, actor.SessionID.String(), actor.PrincipalID.String()).Scan(
		&status, &population, &tenantIDText, &assurance, &authorizationVersion,
		&rotationVersion, &idleExpiresAt, &absoluteExpiresAt, &stepUpAction, &stepUpVerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.AuthorizationDeny, "authority_stale_or_revoked", nil
	}
	if err != nil {
		return identity.AuthorizationDeny, "", identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.AuthorizationDeny, "", identity.ErrIdentityUnavailable
	}
	if status != "active" || population != string(identity.PopulationMerchant) ||
		tenantID != actor.TenantID || assurance != string(actor.Assurance) ||
		authorizationVersion != actor.AuthorizationVersion || rotationVersion != actor.RotationVersion ||
		!now.Before(idleExpiresAt) || !now.Before(absoluteExpiresAt) {
		return identity.AuthorizationDeny, "authority_stale_or_revoked", nil
	}
	_, role, currentAuthorizationVersion, err := authorityForPrincipal(
		ctx, transaction, actor.PrincipalID, identity.PopulationMerchant, tenantID,
	)
	if errors.Is(err, identity.ErrAuthenticationRequired) {
		return identity.AuthorizationDeny, "authority_stale_or_revoked", nil
	}
	if err != nil {
		return identity.AuthorizationDeny, "", err
	}
	permissions, err := permissionsForRole(ctx, transaction, role)
	if err != nil {
		return identity.AuthorizationDeny, "", err
	}
	decision := store.authorization.Evaluate(identity.AuthorizationFacts{
		PrincipalID: actor.PrincipalID, PrincipalType: actor.PrincipalType,
		Population: identity.PopulationMerchant, TenantID: tenantID, ResourceTenantID: tenantID,
		Role: role, CurrentPermissions: permissions, Action: action, Resource: "api_credential",
		Purpose: identity.CredentialPurposeManagement, Assurance: actor.Assurance,
		SessionAuthorizationVersion: authorizationVersion,
		CurrentAuthorizationVersion: currentAuthorizationVersion,
		ResourceVersion:             1, ResourceStatus: "active",
	})
	if decision.Effect != identity.AuthorizationAllow {
		return decision.Effect, decision.Reason, nil
	}
	if requireStepUp {
		stepUpSatisfied := actor.Assurance == identity.AssurancePhishingResistant &&
			stepUpAction != nil && *stepUpAction == action && stepUpVerifiedAt != nil &&
			!stepUpVerifiedAt.Before(now.Add(-identity.CredentialStepUpFreshness)) &&
			!stepUpVerifiedAt.After(now.Add(time.Minute))
		if !stepUpSatisfied {
			return identity.AuthorizationDeny, "step_up_required", nil
		}
	}
	return identity.AuthorizationAllow, "allowed", nil
}

func (store *CredentialStore) commitDenied(
	ctx context.Context,
	transaction pgx.Tx,
	event audit.Event,
	reason string,
) (identity.CredentialMutationResult, error) {
	event.Decision = "denied"
	event.ReasonCode = "credential_" + reason
	event.SafeAfterReference = "credential-status:unchanged"
	if err := store.recorder.Record(ctx, transaction, event); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.CredentialMutationResult{DecisionID: event.DecisionID}, authorizationFailure(reason)
}

func authorizationFailure(reason string) error {
	if reason == "step_up_required" {
		return identity.ErrStepUpRequired
	}
	return identity.ErrActionNotAuthorized
}

func claimCredentialMutation(
	ctx context.Context,
	transaction pgx.Tx,
	actor identity.Session,
	operation string,
	idempotencyDigest [32]byte,
	requestDigest [32]byte,
	requested *identifier.ID,
	result identifier.ID,
	event audit.Event,
	now time.Time,
) (bool, error) {
	var requestedValue any
	if requested != nil {
		requestedValue = requested.String()
	}
	inserted, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.api_credential_mutation_requests (
    tenant_id, actor_principal_id, actor_session_id, operation,
    idempotency_key_sha256, request_sha256, requested_credential_id,
    result_credential_id, authorization_decision_id, audit_event_id, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (tenant_id, actor_principal_id, operation, idempotency_key_sha256)
DO NOTHING`,
		actor.TenantID.String(), actor.PrincipalID.String(), actor.SessionID.String(), operation,
		idempotencyDigest[:], requestDigest[:], requestedValue, result.String(),
		event.DecisionID.String(), event.AuditEventID.String(), now,
	)
	return err == nil && inserted.RowsAffected() == 1, err
}

func replayCredentialMutation(
	ctx context.Context,
	transaction pgx.Tx,
	actor identity.Session,
	operation string,
	idempotencyDigest [32]byte,
	requestDigest [32]byte,
	now time.Time,
) (identity.CredentialMutationResult, error) {
	var storedDigest []byte
	var resultIDText, decisionIDText string
	err := transaction.QueryRow(ctx, `
SELECT request_sha256, result_credential_id, authorization_decision_id
FROM atlas_identity.api_credential_mutation_requests
WHERE tenant_id = $1 AND actor_principal_id = $2 AND operation = $3
  AND idempotency_key_sha256 = $4`, actor.TenantID.String(), actor.PrincipalID.String(), operation, idempotencyDigest[:]).Scan(
		&storedDigest, &resultIDText, &decisionIDText,
	)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	decisionID, err := identifier.Parse(decisionIDText)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	if len(storedDigest) != 32 || subtle.ConstantTimeCompare(storedDigest, requestDigest[:]) != 1 {
		return identity.CredentialMutationResult{DecisionID: decisionID, Replay: true}, identity.ErrIdempotencyConflict
	}
	resultID, err := identifier.Parse(resultIDText)
	if err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	credential, err := credentialByID(ctx, transaction, actor.TenantID, resultID, now)
	if err != nil {
		return identity.CredentialMutationResult{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return identity.CredentialMutationResult{}, identity.ErrIdentityUnavailable
	}
	return identity.CredentialMutationResult{
		Credential: credential, DecisionID: decisionID, Replay: true,
	}, nil
}

func credentialByID(
	ctx context.Context,
	transaction pgx.Tx,
	tenantID identifier.ID,
	credentialID identifier.ID,
	now time.Time,
) (identity.APICredential, error) {
	return scanCredential(transaction.QueryRow(ctx, `
SELECT credential_id, tenant_id, name, secret_hint, scopes,
       CASE
         WHEN expires_at <= $3 THEN 'expired'
         WHEN status = 'rotating' AND overlap_ends_at <= $3 THEN 'revoked'
         ELSE status
       END AS effective_status,
       environment, audience, version, expires_at, overlap_ends_at, last_used_at,
       created_at, revoked_at, previous_credential_id, replacement_credential_id
FROM atlas_identity.api_credentials
WHERE tenant_id = $1 AND credential_id = $2`, tenantID.String(), credentialID.String(), now))
}

func credentialByIDForUpdate(
	ctx context.Context,
	transaction pgx.Tx,
	tenantID identifier.ID,
	credentialID identifier.ID,
	environment string,
	now time.Time,
) (identity.APICredential, error) {
	return scanCredential(transaction.QueryRow(ctx, `
SELECT credential_id, tenant_id, name, secret_hint, scopes,
       CASE
         WHEN expires_at <= $3 THEN 'expired'
         WHEN status = 'rotating' AND overlap_ends_at <= $3 THEN 'revoked'
         ELSE status
       END AS effective_status,
       environment, audience, version, expires_at, overlap_ends_at, last_used_at,
       created_at, revoked_at, previous_credential_id, replacement_credential_id
FROM atlas_identity.api_credentials
WHERE tenant_id = $1 AND credential_id = $2 AND environment = $4
FOR UPDATE`, tenantID.String(), credentialID.String(), now, environment))
}

type credentialScanner interface {
	Scan(...any) error
}

func scanCredential(row credentialScanner) (identity.APICredential, error) {
	var (
		credentialIDText, tenantIDText, name, secretHint, status, environment, audience string
		scopes                                                                          []string
		version                                                                         int64
		expiresAt, createdAt                                                            time.Time
		overlapEndsAt, lastUsedAt, revokedAt                                            *time.Time
		previousIDText, replacementIDText                                               *string
	)
	err := row.Scan(
		&credentialIDText, &tenantIDText, &name, &secretHint, &scopes, &status,
		&environment, &audience, &version, &expiresAt, &overlapEndsAt, &lastUsedAt,
		&createdAt, &revokedAt, &previousIDText, &replacementIDText,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.APICredential{}, identity.ErrCredentialNotFound
	}
	if err != nil {
		return identity.APICredential{}, identity.ErrIdentityUnavailable
	}
	credentialID, err := identifier.Parse(credentialIDText)
	if err != nil {
		return identity.APICredential{}, identity.ErrIdentityUnavailable
	}
	tenantID, err := identifier.Parse(tenantIDText)
	if err != nil {
		return identity.APICredential{}, identity.ErrIdentityUnavailable
	}
	return identity.APICredential{
		CredentialID: credentialID, OrganizationID: tenantID, Name: name, SecretHint: secretHint,
		Scopes: append([]string(nil), scopes...), Status: identity.APICredentialStatus(status),
		Environment: environment, Audience: audience, Version: version,
		ExpiresAt: expiresAt.UTC(), OverlapEndsAt: utcTime(overlapEndsAt),
		LastUsedAt: utcTime(lastUsedAt), CreatedAt: createdAt.UTC(), RevokedAt: utcTime(revokedAt),
		PreviousCredentialID:    parsedOptionalID(previousIDText),
		ReplacementCredentialID: parsedOptionalID(replacementIDText),
	}, nil
}

func utcTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

func parsedOptionalID(value *string) identifier.ID {
	if value == nil {
		return identifier.ID{}
	}
	parsed, err := identifier.Parse(*value)
	if err != nil {
		return identifier.ID{}
	}
	return parsed
}

var _ identity.CredentialStore = (*CredentialStore)(nil)
