// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package webhooks

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// randomKey returns a fresh 32-byte master key for the test.
func randomKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, 32)
	_, err := rand.Read(k)
	require.NoError(t, err)
	return k
}

// TestEncryptDecrypt_RoundTrip is the basic happy path: encrypt under a
// master key, decrypt back, get the same plaintext.
func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	svc := &WebhookService{masterKey: randomKey(t)}
	plain := "very-secret-webhook-signing-key"

	stored, err := svc.encryptSecret(plain)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(stored, encSecretPrefix), "stored value must carry the enc:v1: prefix")

	got, err := svc.decryptSecret(stored)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

// TestDecrypt_RecognisesLegacyPlaintext: a value without the enc:v1: prefix
// is treated as legacy plaintext and returned as-is. This is what makes
// MigrateLegacyPlaintextSecrets safe to run on a freshly-upgraded DB.
func TestDecrypt_RecognisesLegacyPlaintext(t *testing.T) {
	svc := &WebhookService{masterKey: randomKey(t)}

	got, err := svc.decryptSecret("plain-legacy-secret")
	require.NoError(t, err)
	assert.Equal(t, "plain-legacy-secret", got)
}

// TestEncrypt_DevModeNoKey: with no master key (dev), encryptSecret
// returns the plaintext value unchanged.  This is a deliberate dev-mode
// behaviour — the API boot logs a warning when running without a key.
func TestEncrypt_DevModeNoKey(t *testing.T) {
	svc := &WebhookService{}

	stored, err := svc.encryptSecret("foo")
	require.NoError(t, err)
	assert.Equal(t, "foo", stored, "no master key → plaintext passthrough")
	assert.False(t, strings.HasPrefix(stored, encSecretPrefix))
}

// TestDecrypt_WrongKeyFails: a ciphertext encrypted under one key must not
// decrypt under another. Property keeps tamper detection useful — a
// rotated DB with the wrong env should fail loudly, not silently return
// garbage.
func TestDecrypt_WrongKeyFails(t *testing.T) {
	svcA := &WebhookService{masterKey: randomKey(t)}
	svcB := &WebhookService{masterKey: randomKey(t)}

	stored, err := svcA.encryptSecret("foo")
	require.NoError(t, err)

	_, err = svcB.decryptSecret(stored)
	assert.Error(t, err)
}

// TestEncrypt_PlaintextValueIsHidden ensures the literal plaintext does
// not appear inside the stored ciphertext value — guards against a future
// regression that accidentally double-encodes or wraps but doesn't encrypt.
func TestEncrypt_PlaintextValueIsHidden(t *testing.T) {
	svc := &WebhookService{masterKey: randomKey(t)}
	plain := "super-secret-string-1234567890"

	stored, err := svc.encryptSecret(plain)
	require.NoError(t, err)
	assert.NotContains(t, stored, plain, "encrypted output must not contain the plaintext")
}
