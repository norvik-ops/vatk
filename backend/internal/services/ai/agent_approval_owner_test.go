// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package ai

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentRunManager_DecideEnforcesOrgOwnership verifies the audit-finding
// fix for the cross-org Approve hijack. A run registered for org A must NOT
// be decidable by a caller from org B — that would let any authenticated
// user approve a write-tool call in another customer's agent run by
// guessing the run_id (UUID, but leakable via logs).
func TestAgentRunManager_DecideEnforcesOrgOwnership(t *testing.T) {
	mgr := &AgentRunManager{}

	// Run owner: org-A, user-1
	ch := mgr.Register("run-xyz", "org-A", "user-1")
	defer mgr.Unregister("run-xyz")

	cases := []struct {
		name        string
		callerOrg   string
		callerUser  string
		wantErrType error
	}{
		{"owner approves", "org-A", "user-1", nil},
		{"other-org caller", "org-B", "user-1", ErrApprovalForbidden},
		{"empty org caller", "", "user-1", ErrApprovalForbidden},
		{"same org different user", "org-A", "user-2", ErrApprovalForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Re-register before each sub-test because the channel buffer
			// holds 1 — the previous "owner approves" iteration consumed it.
			if tc.name != "owner approves" {
				// Drain leftover channel and re-register so the slot is fresh.
				mgr.Unregister("run-xyz")
				ch = mgr.Register("run-xyz", "org-A", "user-1")
				_ = ch // referenced to keep buffer alive
			}
			err := mgr.Decide("run-xyz", tc.callerOrg, tc.callerUser, ApprovalDecision{Approved: true, UserID: tc.callerUser})
			if tc.wantErrType == nil {
				assert.NoError(t, err)
			} else {
				assert.True(t, errors.Is(err, tc.wantErrType), "expected %v, got %v", tc.wantErrType, err)
			}
		})
	}
}

// TestAgentRunManager_DecideUnknownRun ensures we return NotFound (not
// Forbidden) when no slot exists at all — keeps the response-shape symmetric
// regardless of "wrong org" vs "expired run", so the handler can map both
// to a 404 without leaking which case applies.
func TestAgentRunManager_DecideUnknownRun(t *testing.T) {
	mgr := &AgentRunManager{}
	err := mgr.Decide("does-not-exist", "org-A", "user-1", ApprovalDecision{Approved: true})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrApprovalNotFound))
}

// TestAgentRunManager_DecideUnblocksRunner is the basic happy-path: the
// returned channel must receive the decision when the owner approves.
func TestAgentRunManager_DecideUnblocksRunner(t *testing.T) {
	mgr := &AgentRunManager{}
	ch := mgr.Register("run-abc", "org-A", "user-1")
	defer mgr.Unregister("run-abc")

	err := mgr.Decide("run-abc", "org-A", "user-1", ApprovalDecision{Approved: true, UserID: "user-1"})
	require.NoError(t, err)

	select {
	case d := <-ch:
		assert.True(t, d.Approved)
		assert.Equal(t, "user-1", d.UserID)
	default:
		t.Fatal("expected the runner channel to receive the decision")
	}
}
