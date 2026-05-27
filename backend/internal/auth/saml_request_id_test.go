// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package auth

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// randomMasterKey returns 32 bytes of CSPRNG-derived key material for tests.
func randomMasterKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

// TestSAMLRequestID_RoundTrip checks the basic sign → verify happy path.
// A signed request ID must decode back to exactly the original ID.
func TestSAMLRequestID_RoundTrip(t *testing.T) {
	key := randomMasterKey(t)
	id := "id-ab12cd34ef56"

	signed, err := signSAMLRequestID(key, id)
	require.NoError(t, err)
	require.Contains(t, signed, ".", "format must be <id>.<sig>")

	recovered, err := verifySAMLRequestID(key, signed)
	require.NoError(t, err)
	assert.Equal(t, id, recovered)
}

// TestSAMLRequestID_RejectsWrongKey is the core replay-protection assertion:
// a cookie signed under one master key must not verify under another. This
// is what prevents an attacker from forging arbitrary InResponseTo values.
func TestSAMLRequestID_RejectsWrongKey(t *testing.T) {
	keyA := randomMasterKey(t)
	keyB := randomMasterKey(t)
	signed, err := signSAMLRequestID(keyA, "id-deadbeef")
	require.NoError(t, err)

	_, err = verifySAMLRequestID(keyB, signed)
	assert.ErrorIs(t, err, errSAMLRequestIDInvalid)
}

// TestSAMLRequestID_RejectsTamperedSignature checks that flipping a single
// character inside the HMAC fails verification.
func TestSAMLRequestID_RejectsTamperedSignature(t *testing.T) {
	key := randomMasterKey(t)
	signed, err := signSAMLRequestID(key, "id-foo")
	require.NoError(t, err)

	// Mutate a character mid-signature.  Last-char tampering can hit the
	// base64 padding-bit invariant in RawURLEncoding without changing the
	// decoded bytes, so we pick a position deep in the HMAC body instead.
	dot := strings.IndexByte(signed, '.')
	require.GreaterOrEqual(t, dot, 0)
	tamperAt := dot + 5 // well inside the sig
	tampered := signed[:tamperAt] + flipChar(signed[tamperAt]) + signed[tamperAt+1:]
	require.NotEqual(t, signed, tampered)

	_, err = verifySAMLRequestID(key, tampered)
	assert.ErrorIs(t, err, errSAMLRequestIDInvalid)
}

// TestSAMLRequestID_RejectsTamperedID checks that changing the embedded ID
// also fails verification — the HMAC binds both parts together.
func TestSAMLRequestID_RejectsTamperedID(t *testing.T) {
	key := randomMasterKey(t)
	signed, err := signSAMLRequestID(key, "id-real")
	require.NoError(t, err)

	// replace the ID portion with a different value but keep the (now-stale) sig
	dotIdx := strings.LastIndexByte(signed, '.')
	tampered := "id-evil" + signed[dotIdx:]
	_, err = verifySAMLRequestID(key, tampered)
	assert.ErrorIs(t, err, errSAMLRequestIDInvalid)
}

// TestSAMLRequestID_RejectsMalformed exercises every shape that should be
// rejected without panicking: empty, no dot, trailing dot, leading dot.
func TestSAMLRequestID_RejectsMalformed(t *testing.T) {
	key := randomMasterKey(t)
	cases := []string{
		"",
		"no-dot-anywhere",
		"trailing-dot.",
		".leading-dot",
		"only-dot.",
		".",
	}
	for _, c := range cases {
		_, err := verifySAMLRequestID(key, c)
		assert.ErrorIs(t, err, errSAMLRequestIDInvalid, "input %q must be rejected", c)
	}
}

// TestSAMLRequestID_EmptyIDFails prevents the caller from accidentally
// signing an empty string — would otherwise produce a parseable but useless
// cookie.
func TestSAMLRequestID_EmptyIDFails(t *testing.T) {
	_, err := signSAMLRequestID(randomMasterKey(t), "")
	assert.Error(t, err)
}

func flipChar(b byte) string {
	if b == 'A' {
		return "B"
	}
	return "A"
}
