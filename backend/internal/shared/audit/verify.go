// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VerifyOrgChain replays the per-org audit-log hash chain and returns the
// UUID of the first row whose recomputed entry_hash does not match the
// stored value — or the empty string if the chain verifies cleanly.
//
// The walk is in created_at ASC + id ASC order (matching the writer-side
// chain extension). Rows whose entry_hash is NULL are treated as a pre-
// chain prefix and skipped: they were inserted before migration 149.
//
// A returned non-empty bad-row id always reflects a real chain break — the
// caller should treat it as forensic evidence (preserve DB snapshot,
// rotate access keys, invoke the response under ADR-0040).
func VerifyOrgChain(ctx context.Context, pool *pgxpool.Pool, orgID string) (string, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			id::text,
			COALESCE(user_id::text, ''),
			COALESCE(user_email, ''),
			action,
			resource_type,
			COALESCE(resource_id, ''),
			COALESCE(resource_name, ''),
			details,
			COALESCE(ip_address, ''),
			created_at,
			prev_hash,
			entry_hash
		FROM audit_log
		WHERE org_id = $1::uuid
		ORDER BY created_at ASC, id ASC`, orgID)
	if err != nil {
		return "", fmt.Errorf("query audit_log for org %s: %w", orgID, err)
	}
	defer rows.Close()

	var expectedPrev []byte
	chainStarted := false

	for rows.Next() {
		var (
			id, userID, userEmail, action, resourceType, resourceID, resourceName, ipAddress string
			detailsJSON                                                                      []byte
			createdAt                                                                        time.Time
			storedPrev, storedEntry                                                          []byte
		)
		if err := rows.Scan(&id, &userID, &userEmail, &action, &resourceType, &resourceID, &resourceName, &detailsJSON, &ipAddress, &createdAt, &storedPrev, &storedEntry); err != nil {
			return "", fmt.Errorf("scan audit row: %w", err)
		}

		// Skip pre-chain rows (migration 149 hasn't kicked in yet for them).
		if storedEntry == nil {
			continue
		}

		// First chained row of this org: storedPrev must be NULL.
		if !chainStarted {
			if storedPrev != nil {
				return id, nil // chain starts mid-stream — link missing
			}
			chainStarted = true
		} else if !bytes.Equal(storedPrev, expectedPrev) {
			return id, nil // link does not match the previous entry_hash
		}

		// Rebuild the input and recompute the hash.
		details := map[string]string{}
		if len(detailsJSON) > 0 {
			_ = json.Unmarshal(detailsJSON, &details)
		}

		in := ChainInput{
			ID:           id,
			OrgID:        orgID,
			UserID:       userID,
			UserEmail:    userEmail,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			ResourceName: resourceName,
			Details:      details,
			IPAddress:    ipAddress,
			CreatedAt:    createdAt,
		}
		recomputed := EntryHash(storedPrev, in)
		if !bytes.Equal(recomputed, storedEntry) {
			return id, nil
		}
		expectedPrev = storedEntry
	}
	return "", rows.Err()
}
