package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/MichaelSeveen/atlas/internal/audit"
	auditapplication "github.com/MichaelSeveen/atlas/internal/audit/application"
	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestOrganizationInvitationRealPostgresDelegationReplayAndAuditRollback(t *testing.T) {
	apiURL := os.Getenv("ATLAS_P01_DATABASE_URL")
	migrationURL := os.Getenv("ATLAS_P01_MIGRATION_DATABASE_URL")
	if apiURL == "" || migrationURL == "" {
		t.Skip("real Phase 01 database URLs are not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	apiPool, err := pgxpool.New(ctx, apiURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(apiPool.Close)
	migrationPool, err := pgxpool.New(ctx, migrationURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(migrationPool.Close)
	recorder := auditapplication.NewRecorder()
	sessionStore, err := NewSessionStore(apiPool, recorder)
	if err != nil {
		t.Fatal(err)
	}
	invitationStore, err := NewOrganizationStore(apiPool, recorder)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	principalID := newIntegrationID(t, "usr")
	externalSubjectID := newIntegrationID(t, "ext")
	membershipID := newIntegrationID(t, "mem")
	steppedSessionID := newIntegrationID(t, "ses")
	baselineSessionID := newIntegrationID(t, "ses")
	recipientPrincipalID := newIntegrationID(t, "usr")
	recipientExternalSubjectID := newIntegrationID(t, "ext")
	bootstrapSessionID := newIntegrationID(t, "ses")
	acceptedSessionID := newIntegrationID(t, "ses")
	acceptedMembershipID := newIntegrationID(t, "mem")
	failurePrincipalID := newIntegrationID(t, "usr")
	failureExternalSubjectID := newIntegrationID(t, "ext")
	failureBootstrapSessionID := newIntegrationID(t, "ses")
	failureAcceptedSessionID := newIntegrationID(t, "ses")
	failureMembershipID := newIntegrationID(t, "mem")
	tenantID, err := identifier.Parse("ten_01JAT1AS00000000000002")
	if err != nil {
		t.Fatal(err)
	}
	personAnchor := "syn_person_invitation_" + strings.ToLower(strings.TrimPrefix(principalID.String(), "usr_"))
	issuer := "https://identity.test.invalid/realms/invitation-integration"
	subject := "invitation-" + strings.ToLower(strings.TrimPrefix(principalID.String(), "usr_"))
	setup, err := migrationPool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = setup.Rollback(ctx) }()
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', 'Synthetic Invitation Security Administrator', $2, 'active', 1, 1, $3, $3)`,
		principalID.String(), personAnchor, now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.external_subjects (
    external_subject_id, principal_id, population, issuer, subject, created_at
) VALUES ($1, $2, 'merchant', $3, $4, $5)`,
		externalSubjectID.String(), principalID.String(), issuer, subject, now,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `
INSERT INTO atlas_identity.memberships (
    membership_id, tenant_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at
) VALUES ($1, $2, $3, 'merchant_security_admin', 'merchant', 'active', 1, 1, $4, $4)`,
		membershipID.String(), tenantID.String(), principalID.String(), now,
	); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.organization_invitations WHERE invited_by_principal_id = $1`, principalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_audit.audit_events WHERE actor_id = $1`, principalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_audit.audit_events WHERE actor_id = $1`, recipientPrincipalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.sessions WHERE principal_id = $1`, recipientPrincipalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.memberships WHERE membership_id = $1`, acceptedMembershipID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.external_subjects WHERE external_subject_id = $1`, recipientExternalSubjectID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.principals WHERE principal_id = $1`, recipientPrincipalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_audit.audit_events WHERE actor_id = $1`, failurePrincipalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.sessions WHERE principal_id = $1`, failurePrincipalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.memberships WHERE membership_id = $1`, failureMembershipID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.external_subjects WHERE external_subject_id = $1`, failureExternalSubjectID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.principals WHERE principal_id = $1`, failurePrincipalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.sessions WHERE principal_id = $1`, principalID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.memberships WHERE membership_id = $1`, membershipID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.external_subjects WHERE external_subject_id = $1`, externalSubjectID.String())
		_, _ = migrationPool.Exec(cleanupCtx, `DELETE FROM atlas_identity.principals WHERE principal_id = $1`, principalID.String())
	})

	_, steppedDigest := integrationToken(t)
	stepped := createInvitationIntegrationSession(
		t, ctx, sessionStore, principalID, issuer, subject, steppedSessionID, steppedDigest,
		now, identity.AssurancePhishingResistant, "identity.organization.invitation.create_admin",
	)
	_, baselineDigest := integrationToken(t)
	baseline := createInvitationIntegrationSession(
		t, ctx, sessionStore, principalID, issuer, subject, baselineSessionID, baselineDigest,
		now.Add(time.Minute), identity.AssuranceBaseline, "",
	)

	first, _ := invitationIntegrationCommand(
		t, stepped, tenantID, now.Add(2*time.Minute), "integration-invitation-replay-0001",
		"admin@example.test", "merchant_admin",
	)
	second, _ := invitationIntegrationCommand(
		t, stepped, tenantID, now.Add(2*time.Minute), "integration-invitation-replay-0001",
		"admin@example.test", "merchant_admin",
	)
	type outcome struct {
		result identity.CreateInvitationStoreResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	for _, command := range []identity.CreateInvitationCommand{first, second} {
		go func(value identity.CreateInvitationCommand) {
			defer wait.Done()
			result, callErr := invitationStore.CreateInvitation(ctx, value)
			outcomes <- outcome{result: result, err: callErr}
		}(command)
	}
	wait.Wait()
	close(outcomes)
	var created, replayed identity.CreateInvitationStoreResult
	for current := range outcomes {
		if current.err != nil {
			t.Fatal(current.err)
		}
		if current.result.Replay {
			replayed = current.result
		} else {
			created = current.result
		}
	}
	if created.Invitation.InvitationID.IsZero() || !replayed.Replay ||
		created.Invitation != replayed.Invitation || created.DecisionID != replayed.DecisionID ||
		created.Invitation.ExpiresAt.Sub(created.Invitation.CreatedAt) != 72*time.Hour {
		t.Fatalf("created=%+v replayed=%+v", created, replayed)
	}
	expiredIssue, _ := invitationIntegrationCommand(
		t, stepped, tenantID, now.Add(2*time.Minute), "integration-invitation-expired-0001",
		"expired@example.test", "merchant_viewer",
	)
	expiredInvitation, err := invitationStore.CreateInvitation(ctx, expiredIssue)
	if err != nil {
		t.Fatal(err)
	}
	_, expiredVerifier := integrationToken(t)
	if _, err := sessionStore.CreateSession(ctx, identity.CreateSessionCommand{
		Claims: identity.ProviderClaims{
			Issuer: issuer, Subject: "expired-recipient", Assurance: identity.AssuranceBaseline,
			AuthenticatedAt: now.Add(75 * time.Hour), Email: "expired@example.test", EmailVerified: true,
		},
		Population: identity.PopulationMerchant, Kind: identity.TransactionInvitationAcceptance,
		SessionID: newIntegrationID(t, "ses"), VerifierDigest: expiredVerifier,
		Assurance: identity.AssuranceBaseline, AuthorizationAt: now.Add(75 * time.Hour),
		IdleExpiresAt: now.Add(75*time.Hour + 15*time.Minute), AbsoluteExpiresAt: now.Add(75*time.Hour + 15*time.Minute),
		InvitationID:          expiredInvitation.Invitation.InvitationID,
		InvitationTokenDigest: expiredIssue.TokenDigest, VerifiedEmailDigest: expiredIssue.EmailDigest,
		ProvisionalPrincipalID:       newIntegrationID(t, "usr"),
		ProvisionalExternalSubjectID: newIntegrationID(t, "ext"),
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), Action: "identity.organization.invitation.authenticate",
			TargetType: "session", TargetID: "expired-bootstrap", DecisionID: newIntegrationID(t, "dec"),
			Decision: "executed", ReasonCode: "oidc_invitation_authentication",
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now.Add(75 * time.Hour),
		},
	}); !errors.Is(err, identity.ErrOIDCTransactionInvalid) {
		t.Fatalf("expired invitation callback error=%v", err)
	}
	acceptanceIssue, _ := invitationIntegrationCommand(
		t, stepped, tenantID, now.Add(2*time.Minute), "integration-invitation-accept-issue-0001",
		"recipient@example.test", "merchant_viewer",
	)
	acceptanceIssued, err := invitationStore.CreateInvitation(ctx, acceptanceIssue)
	if err != nil {
		t.Fatal(err)
	}
	_, bootstrapDigest := integrationToken(t)
	bootstrapCommand := identity.CreateSessionCommand{
		Claims: identity.ProviderClaims{
			Issuer: issuer, Subject: "recipient-" + strings.ToLower(strings.TrimPrefix(recipientPrincipalID.String(), "usr_")),
			Assurance: identity.AssuranceBaseline, AuthenticatedAt: now.Add(150 * time.Second),
			Email: "recipient@example.test", EmailVerified: true,
		},
		Population: identity.PopulationMerchant, Kind: identity.TransactionInvitationAcceptance,
		SessionID: bootstrapSessionID, VerifierDigest: bootstrapDigest,
		Assurance: identity.AssuranceBaseline, AuthorizationAt: now.Add(150 * time.Second),
		IdleExpiresAt:         now.Add(17*time.Minute + 30*time.Second),
		AbsoluteExpiresAt:     now.Add(17*time.Minute + 30*time.Second),
		ClientLabel:           "integration-invitation-recipient",
		InvitationID:          acceptanceIssued.Invitation.InvitationID,
		InvitationTokenDigest: acceptanceIssue.TokenDigest,
		VerifiedEmailDigest:   acceptanceIssue.EmailDigest, ProvisionalPrincipalID: recipientPrincipalID,
		ProvisionalExternalSubjectID: recipientExternalSubjectID,
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), GlobalScope: "identity-security",
			SessionAssurance: string(identity.AssuranceBaseline),
			Action:           "identity.organization.invitation.authenticate", TargetType: "session",
			TargetID: bootstrapSessionID.String(), DecisionID: newIntegrationID(t, "dec"),
			Decision: "executed", ReasonCode: "oidc_invitation_authentication",
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now.Add(150 * time.Second),
		},
	}
	wrongRecipient := bootstrapCommand
	wrongRecipient.SessionID = newIntegrationID(t, "ses")
	wrongRecipient.ProvisionalPrincipalID = newIntegrationID(t, "usr")
	wrongRecipient.ProvisionalExternalSubjectID = newIntegrationID(t, "ext")
	wrongRecipient.Claims.Email = "attacker@example.test"
	wrongRecipient.VerifiedEmailDigest = sha256.Sum256([]byte("attacker@example.test"))
	wrongRecipient.AuditEvent.AuditEventID = newIntegrationID(t, "aud")
	wrongRecipient.AuditEvent.TargetID = wrongRecipient.SessionID.String()
	if _, err := sessionStore.CreateSession(ctx, wrongRecipient); !errors.Is(err, identity.ErrOIDCTransactionInvalid) {
		t.Fatalf("wrong recipient callback error=%v", err)
	}
	wrongToken := bootstrapCommand
	wrongToken.SessionID = newIntegrationID(t, "ses")
	wrongToken.ProvisionalPrincipalID = newIntegrationID(t, "usr")
	wrongToken.ProvisionalExternalSubjectID = newIntegrationID(t, "ext")
	wrongToken.InvitationTokenDigest = sha256.Sum256([]byte("wrong-invitation-token"))
	wrongToken.AuditEvent.AuditEventID = newIntegrationID(t, "aud")
	wrongToken.AuditEvent.TargetID = wrongToken.SessionID.String()
	if _, err := sessionStore.CreateSession(ctx, wrongToken); !errors.Is(err, identity.ErrOIDCTransactionInvalid) {
		t.Fatalf("wrong token callback error=%v", err)
	}
	bootstrap, err := sessionStore.CreateSession(ctx, bootstrapCommand)
	if err != nil || !bootstrap.InvitationAcceptanceOnly || !bootstrap.TenantID.IsZero() ||
		bootstrap.InvitationID != acceptanceIssued.Invitation.InvitationID || len(bootstrap.Permissions) != 0 {
		t.Fatalf("bootstrap=%+v err=%v", bootstrap, err)
	}
	_, acceptedDigest := integrationToken(t)
	acceptNow := now.Add(3 * time.Minute)
	acceptRequestDigest := sha256.Sum256([]byte(
		"v1\ninvitation=" + acceptanceIssued.Invitation.InvitationID.String() +
			"\ntoken_sha256=" + hex.EncodeToString(acceptanceIssue.TokenDigest[:]),
	))
	acceptCommand := identity.AcceptInvitationCommand{
		Actor: bootstrap, InvitationID: acceptanceIssued.Invitation.InvitationID,
		TokenDigest: acceptanceIssue.TokenDigest, MembershipID: acceptedMembershipID,
		NewSessionID: acceptedSessionID, NewSessionVerifierDigest: acceptedDigest,
		IdempotencyDigest: sha256.Sum256([]byte("integration-invitation-accept-0001")),
		RequestDigest:     acceptRequestDigest, Now: acceptNow,
		IdleExpiresAt: acceptNow.Add(20 * time.Minute), AbsoluteExpiresAt: acceptNow.Add(8 * time.Hour),
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), ActorID: bootstrap.PrincipalID,
			ActorType: bootstrap.PrincipalType, SessionAssurance: string(bootstrap.Assurance),
			Action: "identity.organization.invitation.accept", TargetType: "invitation",
			TargetID: acceptanceIssued.Invitation.InvitationID.String(), DecisionID: newIntegrationID(t, "dec"),
			Decision: "executed", ReasonCode: "organization_invitation_accepted",
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: acceptNow,
			SafeBeforeReference: "invitation:pending", SafeAfterReference: "invitation:accepted",
		},
	}
	type acceptanceOutcome struct {
		result identity.AcceptInvitationStoreResult
		err    error
	}
	acceptanceOutcomes := make(chan acceptanceOutcome, 2)
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			result, callErr := invitationStore.AcceptInvitation(ctx, acceptCommand)
			acceptanceOutcomes <- acceptanceOutcome{result: result, err: callErr}
		}()
	}
	wait.Wait()
	close(acceptanceOutcomes)
	var accepted, concurrentReplay identity.AcceptInvitationStoreResult
	for current := range acceptanceOutcomes {
		if current.err != nil {
			t.Fatal(current.err)
		}
		if current.result.Replay {
			concurrentReplay = current.result
		} else {
			accepted = current.result
		}
	}
	if accepted.Member.MembershipID != acceptedMembershipID ||
		accepted.Member.Role != "merchant_viewer" || accepted.Session.TenantID != tenantID ||
		accepted.Session.InvitationAcceptanceOnly || !concurrentReplay.Replay ||
		concurrentReplay.Member.MembershipID != acceptedMembershipID ||
		concurrentReplay.DecisionID != accepted.DecisionID {
		t.Fatalf("accepted=%+v concurrent replay=%+v", accepted, concurrentReplay)
	}
	if _, err := sessionStore.Authenticate(ctx, bootstrapDigest, acceptNow, 0); !errors.Is(err, identity.ErrSessionRevoked) {
		t.Fatalf("bootstrap session survived acceptance: %v", err)
	}
	authorized, err := sessionStore.Authenticate(ctx, acceptedDigest, acceptNow, 0)
	if err != nil || authorized.TenantID != tenantID || !containsPermission(authorized.Permissions, "organization.members.read") {
		t.Fatalf("promoted session=%+v err=%v", authorized, err)
	}
	replayedAcceptance, err := invitationStore.AcceptInvitation(ctx, acceptCommand)
	if err != nil || !replayedAcceptance.Replay ||
		replayedAcceptance.Member.MembershipID != acceptedMembershipID ||
		replayedAcceptance.DecisionID != accepted.DecisionID {
		t.Fatalf("acceptance replay=%+v err=%v", replayedAcceptance, err)
	}

	failureIssue, _ := invitationIntegrationCommand(
		t, stepped, tenantID, now.Add(3*time.Minute), "integration-invitation-accept-audit-0001",
		"rollback@example.test", "merchant_viewer",
	)
	failureInvitation, err := invitationStore.CreateInvitation(ctx, failureIssue)
	if err != nil {
		t.Fatal(err)
	}
	_, failureBootstrapDigest := integrationToken(t)
	failureBootstrap, err := sessionStore.CreateSession(ctx, identity.CreateSessionCommand{
		Claims: identity.ProviderClaims{
			Issuer: issuer, Subject: "rollback-" + strings.ToLower(strings.TrimPrefix(failurePrincipalID.String(), "usr_")),
			Assurance: identity.AssuranceBaseline, AuthenticatedAt: now.Add(210 * time.Second),
			Email: "rollback@example.test", EmailVerified: true,
		},
		Population: identity.PopulationMerchant, Kind: identity.TransactionInvitationAcceptance,
		SessionID: failureBootstrapSessionID, VerifierDigest: failureBootstrapDigest,
		Assurance: identity.AssuranceBaseline, AuthorizationAt: now.Add(210 * time.Second),
		IdleExpiresAt:         now.Add(18*time.Minute + 30*time.Second),
		AbsoluteExpiresAt:     now.Add(18*time.Minute + 30*time.Second),
		ClientLabel:           "integration-invitation-rollback",
		InvitationID:          failureInvitation.Invitation.InvitationID,
		InvitationTokenDigest: failureIssue.TokenDigest,
		VerifiedEmailDigest:   failureIssue.EmailDigest, ProvisionalPrincipalID: failurePrincipalID,
		ProvisionalExternalSubjectID: failureExternalSubjectID,
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), GlobalScope: "identity-security",
			SessionAssurance: string(identity.AssuranceBaseline),
			Action:           "identity.organization.invitation.authenticate", TargetType: "session",
			TargetID: failureBootstrapSessionID.String(), DecisionID: newIntegrationID(t, "dec"),
			Decision: "executed", ReasonCode: "oidc_invitation_authentication",
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now.Add(210 * time.Second),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	failingAcceptanceStore, err := NewOrganizationStore(apiPool, failingAuditRecorder{})
	if err != nil {
		t.Fatal(err)
	}
	failureAcceptNow := now.Add(4 * time.Minute)
	_, failureAcceptedDigest := integrationToken(t)
	failureAccept := identity.AcceptInvitationCommand{
		Actor: failureBootstrap, InvitationID: failureInvitation.Invitation.InvitationID,
		TokenDigest: failureIssue.TokenDigest, MembershipID: failureMembershipID,
		NewSessionID: failureAcceptedSessionID, NewSessionVerifierDigest: failureAcceptedDigest,
		IdempotencyDigest: sha256.Sum256([]byte("integration-invitation-accept-audit-0001")),
		RequestDigest: sha256.Sum256([]byte(
			"v1\ninvitation=" + failureInvitation.Invitation.InvitationID.String() +
				"\ntoken_sha256=" + hex.EncodeToString(failureIssue.TokenDigest[:]),
		)),
		Now: failureAcceptNow, IdleExpiresAt: failureAcceptNow.Add(20 * time.Minute),
		AbsoluteExpiresAt: failureAcceptNow.Add(8 * time.Hour),
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), ActorID: failureBootstrap.PrincipalID,
			ActorType: failureBootstrap.PrincipalType, SessionAssurance: string(failureBootstrap.Assurance),
			Action: "identity.organization.invitation.accept", TargetType: "invitation",
			TargetID: failureInvitation.Invitation.InvitationID.String(), DecisionID: newIntegrationID(t, "dec"),
			Decision: "executed", ReasonCode: "organization_invitation_accepted",
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: failureAcceptNow,
			SafeBeforeReference: "invitation:pending", SafeAfterReference: "invitation:accepted",
		},
	}
	if _, err := failingAcceptanceStore.AcceptInvitation(ctx, failureAccept); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("acceptance Audit outage error=%v", err)
	}
	var failureInvitationStatus, failureBootstrapStatus string
	var failureMembershipCount, failureSessionCount int
	if err := migrationPool.QueryRow(ctx, `
SELECT status FROM atlas_identity.organization_invitations WHERE invitation_id = $1`,
		failureInvitation.Invitation.InvitationID.String(),
	).Scan(&failureInvitationStatus); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT status FROM atlas_identity.sessions WHERE session_id = $1`,
		failureBootstrapSessionID.String(),
	).Scan(&failureBootstrapStatus); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.memberships WHERE membership_id = $1`,
		failureMembershipID.String(),
	).Scan(&failureMembershipCount); err != nil {
		t.Fatal(err)
	}
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*) FROM atlas_identity.sessions WHERE session_id = $1`,
		failureAcceptedSessionID.String(),
	).Scan(&failureSessionCount); err != nil {
		t.Fatal(err)
	}
	if failureInvitationStatus != "pending" || failureBootstrapStatus != "active" ||
		failureMembershipCount != 0 || failureSessionCount != 0 {
		t.Fatalf("acceptance Audit rollback invitation=%s bootstrap=%s membership=%d session=%d",
			failureInvitationStatus, failureBootstrapStatus, failureMembershipCount, failureSessionCount)
	}

	staleStepUpReplay := first
	staleStepUpReplay.Now = now.Add(8 * time.Minute)
	staleStepUpReplay.AuditEvent.OccurredAt = staleStepUpReplay.Now
	redacted, err := invitationStore.CreateInvitation(ctx, staleStepUpReplay)
	if err != nil || !redacted.Replay || redacted.Invitation != created.Invitation ||
		redacted.DecisionID != created.DecisionID {
		t.Fatalf("redacted response-loss replay result=%+v err=%v", redacted, err)
	}

	mismatch := first
	mismatch.InvitationID = newIntegrationID(t, "inv")
	mismatch.RequestDigest = sha256.Sum256([]byte("different-request"))
	conflict, err := invitationStore.CreateInvitation(ctx, mismatch)
	if !errors.Is(err, identity.ErrIdempotencyConflict) || !conflict.Replay ||
		conflict.Invitation.InvitationID != created.Invitation.InvitationID {
		t.Fatalf("mismatch result=%+v err=%v", conflict, err)
	}

	tooHigh, _ := invitationIntegrationCommand(
		t, stepped, tenantID, now.Add(3*time.Minute), "integration-invitation-delegation-0001",
		"security@example.test", "merchant_security_admin",
	)
	denied, err := invitationStore.CreateInvitation(ctx, tooHigh)
	if !errors.Is(err, identity.ErrActionNotAuthorized) || denied.DecisionID.IsZero() {
		t.Fatalf("ADV-IAM-007 result=%+v err=%v", denied, err)
	}

	missingStepUp, _ := invitationIntegrationCommand(
		t, baseline, tenantID, now.Add(3*time.Minute), "integration-invitation-stepup-0001",
		"admin2@example.test", "merchant_admin",
	)
	stepUpDenied, err := invitationStore.CreateInvitation(ctx, missingStepUp)
	if !errors.Is(err, identity.ErrStepUpRequired) || stepUpDenied.DecisionID.IsZero() {
		t.Fatalf("step-up result=%+v err=%v", stepUpDenied, err)
	}

	failingStore, err := NewOrganizationStore(apiPool, failingAuditRecorder{})
	if err != nil {
		t.Fatal(err)
	}
	auditFailure, _ := invitationIntegrationCommand(
		t, stepped, tenantID, now.Add(4*time.Minute), "integration-invitation-audit-0001",
		"viewer@example.test", "merchant_viewer",
	)
	if _, err := failingStore.CreateInvitation(ctx, auditFailure); !errors.Is(err, identity.ErrIdentityUnavailable) {
		t.Fatalf("Audit outage error=%v", err)
	}
	var failedCount int
	if err := migrationPool.QueryRow(ctx, `
SELECT count(*)
FROM atlas_identity.organization_invitations
WHERE tenant_id = $1 AND invited_by_principal_id = $2 AND idempotency_key_sha256 = $3`,
		tenantID.String(), principalID.String(), auditFailure.IdempotencyDigest[:],
	).Scan(&failedCount); err != nil || failedCount != 0 {
		t.Fatalf("Audit outage invitation count=%d err=%v", failedCount, err)
	}

	rows, err := migrationPool.Query(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = 'atlas_identity' AND table_name = 'organization_invitations'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatal(err)
		}
		if column == "email" || column == "token" || column == "acceptance_token" {
			t.Fatalf("recoverable invitation material column exists: %s", column)
		}
	}
}

func createInvitationIntegrationSession(
	t *testing.T,
	ctx context.Context,
	store *SessionStore,
	principalID identifier.ID,
	issuer string,
	subject string,
	sessionID identifier.ID,
	verifierDigest [32]byte,
	now time.Time,
	assurance identity.Assurance,
	stepUpAction string,
) identity.Session {
	t.Helper()
	stepUpAt := time.Time{}
	kind := identity.TransactionLogin
	if stepUpAction != "" {
		kind = identity.TransactionStepUp
		stepUpAt = now
	}
	result, err := store.CreateSession(ctx, identity.CreateSessionCommand{
		Claims: identity.ProviderClaims{
			Issuer: issuer, Subject: subject, Assurance: assurance, AuthenticatedAt: now,
		},
		Population: identity.PopulationMerchant, Kind: kind,
		ExpectedPrincipal: principalID, SessionID: sessionID, VerifierDigest: verifierDigest,
		Assurance: assurance, AuthorizationAt: now,
		StepUpAction: stepUpAction, StepUpVerifiedAt: stepUpAt,
		IdleExpiresAt: now.Add(20 * time.Minute), AbsoluteExpiresAt: now.Add(8 * time.Hour),
		ClientLabel: "integration-invitation-browser",
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), GlobalScope: "identity-security",
			SessionAssurance: string(assurance), Action: "identity.session.login",
			TargetType: "session", TargetID: sessionID.String(),
			DecisionID: newIntegrationID(t, "dec"), Decision: "executed",
			ReasonCode: "oidc_login", CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func invitationIntegrationCommand(
	t *testing.T,
	actor identity.Session,
	tenantID identifier.ID,
	now time.Time,
	idempotencyKey string,
	email string,
	role string,
) (identity.CreateInvitationCommand, string) {
	t.Helper()
	normalizedEmail := email
	if local, domain, found := strings.Cut(email, "@"); found {
		normalizedEmail = local + "@" + strings.ToLower(domain)
	}
	emailDigest := sha256.Sum256([]byte(normalizedEmail))
	requestDigest := sha256.Sum256([]byte(
		"v1\nemail_sha256=" + hex.EncodeToString(emailDigest[:]) + "\nrole=" + role,
	))
	token, tokenDigest := integrationToken(t)
	invitationID := newIntegrationID(t, "inv")
	return identity.CreateInvitationCommand{
		Actor: actor, InvitationID: invitationID, OrganizationID: tenantID,
		EmailDigest: emailDigest, EmailHint: "a***@example.test", Role: role,
		TokenDigest: tokenDigest, IdempotencyDigest: sha256.Sum256([]byte(idempotencyKey)),
		RequestDigest: requestDigest, Now: now, ExpiresAt: now.Add(72 * time.Hour),
		AuditEvent: audit.Event{
			AuditEventID: newIntegrationID(t, "aud"), ActorID: actor.PrincipalID,
			ActorType: actor.PrincipalType, TenantID: tenantID,
			SessionAssurance: string(actor.Assurance),
			Action:           "identity.organization.invitation.create", TargetType: "invitation",
			TargetID: invitationID.String(), DecisionID: newIntegrationID(t, "dec"),
			Decision: "executed", ReasonCode: "organization_invitation_created",
			CorrelationID: newIntegrationID(t, "cor"), OccurredAt: now,
			SafeAfterReference: "invitation:pending",
		},
	}, token
}
