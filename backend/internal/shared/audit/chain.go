// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// ChainInput is the deterministic snapshot of an audit-log row that gets
// hashed into the chain. The field order, separator, and null encoding
// MUST stay stable across releases — otherwise old chains will fail
// verification after a code change.
type ChainInput struct {
	ID           string
	OrgID        string
	UserID       string // empty for system / anonymous
	UserEmail    string
	Action       string
	ResourceType string
	ResourceID   string
	ResourceName string
	Details      map[string]string
	IPAddress    string
	CreatedAt    time.Time
}

// chainSeparator delimits fields in the canonical pre-image. The vertical
// bar is chosen because none of the field values are expected to contain
// it; canonicalString defensively escapes any that do.
const chainSeparator = "|"

// canonicalString builds the deterministic pre-image for the chain hash.
// The format is:
//
//	prev_hash_hex | id | org_id | user_id | user_email | action |
//	resource_type | resource_id | resource_name | details_json |
//	ip_address | created_at_unix_micro
//
// details is JSON-encoded with sorted keys (Go's json package preserves
// insertion order, but our map has been canonicalised in canonicalDetails
// below). Empty fields are rendered as the empty string, NOT omitted, so
// adding a future field stays additive.
func canonicalString(prevHash []byte, in ChainInput) string {
	parts := []string{
		hex.EncodeToString(prevHash), // empty hex for nil/first-entry
		in.ID,
		in.OrgID,
		in.UserID,
		in.UserEmail,
		in.Action,
		in.ResourceType,
		in.ResourceID,
		in.ResourceName,
		canonicalDetails(in.Details),
		in.IPAddress,
		strconv.FormatInt(in.CreatedAt.UTC().UnixMicro(), 10),
	}
	for i := range parts {
		parts[i] = escapeSeparator(parts[i])
	}
	return strings.Join(parts, chainSeparator)
}

// EntryHash computes SHA-256 over canonicalString(prevHash, in). prevHash
// may be nil — that is the case for the first chained entry in an org.
func EntryHash(prevHash []byte, in ChainInput) []byte {
	sum := sha256.Sum256([]byte(canonicalString(prevHash, in)))
	return sum[:]
}

// canonicalDetails marshals the details map with sorted keys.  The default
// json.Marshal randomises map iteration in Go, which would make the hash
// non-deterministic across processes.
func canonicalDetails(d map[string]string) string {
	if len(d) == 0 {
		return ""
	}
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	// stable, low-allocation sort
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	type kv struct {
		K string `json:"k"`
		V string `json:"v"`
	}
	out := make([]kv, 0, len(keys))
	for _, k := range keys {
		out = append(out, kv{K: k, V: d[k]})
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// escapeSeparator replaces "|" with a backslash-escape so that a value
// containing the literal separator cannot blur field boundaries in the
// pre-image. Backslashes themselves are escaped first to keep the
// transformation invertible.
func escapeSeparator(s string) string {
	if !strings.ContainsAny(s, `\|`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 4)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			b.WriteString(`\\`)
		case '|':
			b.WriteString(`\|`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
