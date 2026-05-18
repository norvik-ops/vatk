// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package secvitals

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// ChangeLogEntry represents one field change on a control.
type ChangeLogEntry struct {
	ID        string    `json:"id"`
	ControlID string    `json:"control_id"`
	UserEmail string    `json:"user_email"`
	Field     string    `json:"field"`
	OldValue  string    `json:"old_value"`
	NewValue  string    `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
}

// AppendControlChange inserts a change log entry into ck_control_changelog.
// Errors are logged but not returned — a changelog write failure must never
// abort the primary update operation.
func (r *Repository) AppendControlChange(ctx context.Context, orgID, controlID, userID, userEmail, field, oldVal, newVal string) {
	var userIDParam interface{}
	if userID != "" {
		userIDParam = userID
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO ck_control_changelog
			(control_id, org_id, user_id, user_email, field, old_value, new_value, changed_at)
		VALUES
			($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, NOW())
	`, controlID, orgID, userIDParam, userEmail, field, oldVal, newVal)
	if err != nil {
		log.Error().
			Err(err).
			Str("control_id", controlID).
			Str("field", field).
			Msg("changelog: failed to append control change")
	}
}

// ListControlChanges returns the last 50 field-level changes for a control,
// ordered newest first.
func (r *Repository) ListControlChanges(ctx context.Context, orgID, controlID string) ([]ChangeLogEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, control_id::text, COALESCE(user_email,''),
		       field, COALESCE(old_value,''), COALESCE(new_value,''), changed_at
		FROM ck_control_changelog
		WHERE org_id = $1::uuid
		  AND control_id = $2::uuid
		ORDER BY changed_at DESC
		LIMIT 50
	`, orgID, controlID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ChangeLogEntry
	for rows.Next() {
		var e ChangeLogEntry
		if err := rows.Scan(&e.ID, &e.ControlID, &e.UserEmail, &e.Field, &e.OldValue, &e.NewValue, &e.ChangedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
