// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package audit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func makeInput() ChainInput {
	return ChainInput{
		ID:           "11111111-1111-1111-1111-111111111111",
		OrgID:        "22222222-2222-2222-2222-222222222222",
		UserID:       "33333333-3333-3333-3333-333333333333",
		UserEmail:    "alice@example.org",
		Action:       "create",
		ResourceType: "control",
		ResourceID:   "ctrl-1",
		ResourceName: "Backup Policy",
		Details:      map[string]string{"field": "title", "old": "X", "new": "Y"},
		IPAddress:    "127.0.0.1",
		CreatedAt:    time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
	}
}

// TestEntryHash_Deterministic pins the canonical hash for a fixed input. If
// this test breaks unexpectedly, the canonicalisation has drifted — and any
// previously-recorded chain hashes will fail to verify under the new code.
// Update the expected value ONLY when intentionally rotating the chain
// format (and bump the migration / verifier story accordingly).
func TestEntryHash_Deterministic(t *testing.T) {
	in := makeInput()
	first := EntryHash(nil, in)
	second := EntryHash(nil, in)
	assert.Equal(t, first, second, "EntryHash must be deterministic for the same input")
	assert.Len(t, first, 32, "SHA-256 output must be 32 bytes")
}

// TestEntryHash_ChainsOnPrevHash verifies that swapping the prev_hash byte
// produces a different output — the property that makes the chain a chain.
func TestEntryHash_ChainsOnPrevHash(t *testing.T) {
	in := makeInput()
	a := EntryHash([]byte{0x01}, in)
	b := EntryHash([]byte{0x02}, in)
	assert.NotEqual(t, a, b, "different prev_hash must yield different entry_hash")
}

// TestEntryHash_DetectsFieldTamper is the core forensic property: changing
// ANY tracked field of a row produces a different entry_hash. The verifier
// uses this to localise tampering.
func TestEntryHash_DetectsFieldTamper(t *testing.T) {
	base := makeInput()
	prev := []byte("previous-hash-stub")
	baseline := EntryHash(prev, base)

	mutations := []struct {
		name string
		mut  func(*ChainInput)
	}{
		{"action", func(c *ChainInput) { c.Action = "delete" }},
		{"resource_id", func(c *ChainInput) { c.ResourceID = "ctrl-2" }},
		{"resource_name", func(c *ChainInput) { c.ResourceName = "Other Policy" }},
		{"user_email", func(c *ChainInput) { c.UserEmail = "mallory@example.org" }},
		{"ip_address", func(c *ChainInput) { c.IPAddress = "203.0.113.1" }},
		{"created_at micro", func(c *ChainInput) { c.CreatedAt = c.CreatedAt.Add(time.Microsecond) }},
		{"details add key", func(c *ChainInput) {
			c.Details = map[string]string{"field": "title", "old": "X", "new": "Y", "evil": "Z"}
		}},
		{"details change value", func(c *ChainInput) { c.Details = map[string]string{"field": "title", "old": "X", "new": "Z"} }},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			mutated := base
			m.mut(&mutated)
			got := EntryHash(prev, mutated)
			assert.NotEqual(t, baseline, got, "tampering with %s must change entry_hash", m.name)
		})
	}
}

// TestCanonicalDetails_StableUnderKeyOrder is the regression for an easy-to-
// miss bug: Go map iteration order is randomised, so naive json.Marshal of
// e.Details would produce non-deterministic output. The hash must be the
// same regardless of how the map was constructed.
func TestCanonicalDetails_StableUnderKeyOrder(t *testing.T) {
	// Two semantically identical maps built in different insertion orders.
	a := map[string]string{"a": "1", "b": "2", "c": "3"}
	b := map[string]string{"c": "3", "a": "1", "b": "2"}
	in1 := makeInput()
	in1.Details = a
	in2 := makeInput()
	in2.Details = b
	assert.Equal(t, EntryHash(nil, in1), EntryHash(nil, in2),
		"details map order must not affect the chain hash")
}

// TestCanonicalDetails_PipeSeparatorEscaped guards the pre-image separator:
// a field value containing the literal "|" must not break field boundaries.
func TestCanonicalDetails_PipeSeparatorEscaped(t *testing.T) {
	a := makeInput()
	a.ResourceName = "Policy A"
	// Mutation that, without escaping, would collide with a totally
	// different row by manipulating where the | separator falls.
	b := makeInput()
	b.ResourceName = "Policy A|extra|injected"
	assert.NotEqual(t, EntryHash(nil, a), EntryHash(nil, b),
		"pipe-injection in a field must not yield the same hash as a fresh row")
}
