// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	sharedcrypto "github.com/matharnica/vakt/internal/shared/crypto"
)

const (
	// samlRequestIDCookieName is the cookie that carries the signed AuthnRequest
	// ID between /api/v1/auth/saml/initiate and /api/v1/auth/saml/acs. It is
	// HttpOnly + SameSite=None (the IdP POSTs back from a foreign origin) and
	// scoped to /api/v1/auth/saml.
	samlRequestIDCookieName = "saml_req_id"

	// samlRequestIDTTLSeconds is how long the cookie is valid. SAML round-
	// trips through an IdP login screen typically complete in under a minute,
	// but 10 minutes accommodates 2FA prompts and reauth flows. Longer means
	// a larger replay window after the cookie was set.
	samlRequestIDTTLSeconds = 600

	// samlRequestIDHKDFPurpose is the HKDF info string used to derive a
	// dedicated signing key from the master key.  Reusing the master key
	// directly would let a key compromise extend to PASETO/vault/etc. (see
	// cmd/api/main.go HKDF wiring).
	samlRequestIDHKDFPurpose = "vakt-saml-reqid-v1"
)

// errSAMLRequestIDInvalid is returned when the cookie cannot be decoded or
// the HMAC does not match. We intentionally return the same error for both
// cases so a downstream caller cannot distinguish them — small but real
// timing-channel hygiene.
var errSAMLRequestIDInvalid = errors.New("saml: invalid request_id cookie")

// signSAMLRequestID HMAC-signs the AuthnRequest ID with a per-purpose HKDF-
// derived key from the master key. Output format is "<id>.<base64-hmac>".
//
// Storing the ID alongside its HMAC means we do not need server-side state
// (Redis or DB) for replay protection — the cookie itself is the receipt.
// The ID is short and not secret (crewjam/saml generates "id-<40-hex>" via
// crypto/rand), so plain-text encoding is fine.
func signSAMLRequestID(masterKey []byte, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("saml: empty request id")
	}
	key, err := sharedcrypto.DeriveServiceKey(masterKey, samlRequestIDHKDFPurpose)
	if err != nil {
		return "", fmt.Errorf("saml: derive signing key: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return id + "." + sig, nil
}

// verifySAMLRequestID validates the cookie value and returns the embedded ID.
// Returns errSAMLRequestIDInvalid when the format is wrong or the HMAC does
// not verify — never returns a partial result.
func verifySAMLRequestID(masterKey []byte, signed string) (string, error) {
	idx := strings.LastIndexByte(signed, '.')
	if idx <= 0 || idx == len(signed)-1 {
		return "", errSAMLRequestIDInvalid
	}
	id := signed[:idx]
	sigB64 := signed[idx+1:]

	gotSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return "", errSAMLRequestIDInvalid
	}

	key, err := sharedcrypto.DeriveServiceKey(masterKey, samlRequestIDHKDFPurpose)
	if err != nil {
		return "", errSAMLRequestIDInvalid
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id))
	wantSig := mac.Sum(nil)

	if !hmac.Equal(gotSig, wantSig) {
		return "", errSAMLRequestIDInvalid
	}
	return id, nil
}
