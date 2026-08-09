package operations

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestApprovalStateMachineRejectsTerminalAndInvalidTransitions(t *testing.T) {
	statuses := []ApprovalStatus{
		ApprovalPending, ApprovalApproved, ApprovalRejected, ApprovalCancelled,
		ApprovalExpired, ApprovalExecuted, ApprovalExecutionFailed, ApprovalSuperseded,
	}
	want := map[[2]ApprovalStatus]bool{
		{ApprovalPending, ApprovalApproved}:                true,
		{ApprovalPending, ApprovalRejected}:                true,
		{ApprovalPending, ApprovalCancelled}:               true,
		{ApprovalPending, ApprovalExpired}:                 true,
		{ApprovalPending, ApprovalSuperseded}:              true,
		{ApprovalApproved, ApprovalExecuted}:               true,
		{ApprovalApproved, ApprovalExecutionFailed}:        true,
		{ApprovalApproved, ApprovalExpired}:                true,
		{ApprovalApproved, ApprovalSuperseded}:             true,
		{ApprovalExecutionFailed, ApprovalExecuted}:        true,
		{ApprovalExecutionFailed, ApprovalExecutionFailed}: true,
		{ApprovalExecutionFailed, ApprovalExpired}:         true,
		{ApprovalExecutionFailed, ApprovalSuperseded}:      true,
	}
	for _, from := range statuses {
		for _, to := range statuses {
			if got := from.CanTransitionTo(to); got != want[[2]ApprovalStatus{from, to}] {
				t.Fatalf("transition %s -> %s=%v want=%v", from, to, got, want[[2]ApprovalStatus{from, to}])
			}
		}
		if from.Terminal() && from.CanTransitionTo(ApprovalPending) {
			t.Fatalf("terminal status %s accepted a transition", from)
		}
	}
}

func TestMembershipRoleChangeCanonicalDigestVectorAndTamperDetection(t *testing.T) {
	payload := MembershipRoleChangePayload{
		OrganizationID:            mustApprovalID(t, "ten_01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		MembershipID:              mustApprovalID(t, "mem_01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		ExpectedMembershipVersion: 7,
		RequestedRole:             "merchant_admin",
		Purpose:                   "organization_administration",
	}
	canonical, digest, err := CanonicalMembershipRoleChangePayload(payload)
	if err != nil {
		t.Fatalf("canonicalize payload: %v", err)
	}
	wantCanonical := `{"expected_membership_version":7,"member_id":"mem_01ARZ3NDEKTSV4RRFFQ69G5FAW","organization_id":"ten_01ARZ3NDEKTSV4RRFFQ69G5FAV","purpose":"organization_administration","requested_role":"merchant_admin"}`
	if string(canonical) != wantCanonical {
		t.Fatalf("canonical payload=%s", canonical)
	}
	wantDigest, err := hex.DecodeString("d6ebd3343bc13cb5fd5b4039dcd44d7e12377c0919166f55853416d3eacf2850")
	if err != nil {
		t.Fatal(err)
	}
	if string(digest[:]) != string(wantDigest) {
		t.Fatalf("digest=%x", digest)
	}
	decoded, err := DecodeCanonicalMembershipRoleChangePayload(canonical)
	if err != nil || decoded != payload {
		t.Fatalf("decode canonical payload=%+v err=%v", decoded, err)
	}
	tampered := append([]byte(nil), canonical...)
	tampered[len(tampered)-2] = 'r'
	if sha256.Sum256(tampered) == digest {
		t.Fatal("payload tamper preserved digest")
	}
	if _, err := DecodeCanonicalMembershipRoleChangePayload(tampered); err == nil {
		t.Fatal("payload tamper unexpectedly decoded")
	}
}

func TestApprovalEffectiveExpiryDoesNotReviveTerminalState(t *testing.T) {
	now := time.Date(2026, 8, 9, 7, 0, 0, 0, time.UTC)
	for _, status := range []ApprovalStatus{ApprovalPending, ApprovalApproved, ApprovalExecutionFailed} {
		approval := Approval{Status: status, ExpiresAt: now}
		if got := approval.EffectiveStatus(now); got != ApprovalExpired {
			t.Fatalf("effective status=%s want expired", got)
		}
	}
	for _, status := range []ApprovalStatus{ApprovalRejected, ApprovalCancelled, ApprovalExecuted, ApprovalSuperseded} {
		approval := Approval{Status: status, ExpiresAt: now.Add(-time.Hour)}
		if got := approval.EffectiveStatus(now); got != status {
			t.Fatalf("terminal status changed from %s to %s", status, got)
		}
	}
}

func mustApprovalID(t *testing.T, value string) identifier.ID {
	t.Helper()
	id, err := identifier.Parse(value)
	if err != nil {
		t.Fatalf("parse id: %v", err)
	}
	return id
}
