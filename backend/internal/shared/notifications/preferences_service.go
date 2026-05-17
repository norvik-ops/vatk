// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotificationPreferences holds a user's notification opt-in/out settings.
type NotificationPreferences struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	EmailWeeklyDigest     bool      `json:"email_weekly_digest"`
	EmailFindingsSeverity string    `json:"email_findings_severity"`
	EmailNewIncidents     bool      `json:"email_new_incidents"`
	EmailOverdueControls  bool      `json:"email_overdue_controls"`
	EmailEvidenceExpiry   bool      `json:"email_evidence_expiry"`
	InappComments         bool      `json:"inapp_comments"`
	InappApprovals        bool      `json:"inapp_approvals"`
	InappSystemUpdates    bool      `json:"inapp_system_updates"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// UpdatePreferencesInput is the validated input for updating notification preferences.
type UpdatePreferencesInput struct {
	EmailWeeklyDigest     *bool   `json:"email_weekly_digest"`
	EmailFindingsSeverity *string `json:"email_findings_severity" validate:"omitempty,oneof=critical high all none"`
	EmailNewIncidents     *bool   `json:"email_new_incidents"`
	EmailOverdueControls  *bool   `json:"email_overdue_controls"`
	EmailEvidenceExpiry   *bool   `json:"email_evidence_expiry"`
	InappComments         *bool   `json:"inapp_comments"`
	InappApprovals        *bool   `json:"inapp_approvals"`
	InappSystemUpdates    *bool   `json:"inapp_system_updates"`
}

// PreferencesService manages per-user notification preferences.
type PreferencesService struct {
	db *pgxpool.Pool
}

// NewPreferencesService creates a new PreferencesService.
func NewPreferencesService(db *pgxpool.Pool) *PreferencesService {
	return &PreferencesService{db: db}
}

// GetPreferences returns the notification preferences for the given user.
// If no row exists yet, a default row is created via upsert and returned.
func (s *PreferencesService) GetPreferences(ctx context.Context, userID string) (*NotificationPreferences, error) {
	// Upsert: insert defaults if not present, then return the row.
	var p NotificationPreferences
	err := s.db.QueryRow(ctx, `
		INSERT INTO notification_preferences (user_id)
		VALUES ($1::uuid)
		ON CONFLICT (user_id) DO UPDATE
		  SET updated_at = notification_preferences.updated_at
		RETURNING id::text, user_id::text,
		          email_weekly_digest, email_findings_severity,
		          email_new_incidents, email_overdue_controls, email_evidence_expiry,
		          inapp_comments, inapp_approvals, inapp_system_updates,
		          updated_at`,
		userID,
	).Scan(
		&p.ID, &p.UserID,
		&p.EmailWeeklyDigest, &p.EmailFindingsSeverity,
		&p.EmailNewIncidents, &p.EmailOverdueControls, &p.EmailEvidenceExpiry,
		&p.InappComments, &p.InappApprovals, &p.InappSystemUpdates,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get notification preferences: %w", err)
	}
	return &p, nil
}

// UpdatePreferences applies a partial update to the user's notification preferences.
// If no row exists yet, defaults are inserted first (upsert pattern).
func (s *PreferencesService) UpdatePreferences(ctx context.Context, userID string, input UpdatePreferencesInput) (*NotificationPreferences, error) {
	// Ensure row exists with defaults before patching.
	if _, err := s.db.Exec(ctx, `
		INSERT INTO notification_preferences (user_id)
		VALUES ($1::uuid)
		ON CONFLICT (user_id) DO NOTHING`,
		userID,
	); err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("upsert notification preferences: %w", err)
	}

	var p NotificationPreferences
	err := s.db.QueryRow(ctx, `
		UPDATE notification_preferences
		SET
			email_weekly_digest     = COALESCE($2, email_weekly_digest),
			email_findings_severity = COALESCE($3, email_findings_severity),
			email_new_incidents     = COALESCE($4, email_new_incidents),
			email_overdue_controls  = COALESCE($5, email_overdue_controls),
			email_evidence_expiry   = COALESCE($6, email_evidence_expiry),
			inapp_comments          = COALESCE($7, inapp_comments),
			inapp_approvals         = COALESCE($8, inapp_approvals),
			inapp_system_updates    = COALESCE($9, inapp_system_updates),
			updated_at              = NOW()
		WHERE user_id = $1::uuid
		RETURNING id::text, user_id::text,
		          email_weekly_digest, email_findings_severity,
		          email_new_incidents, email_overdue_controls, email_evidence_expiry,
		          inapp_comments, inapp_approvals, inapp_system_updates,
		          updated_at`,
		userID,
		input.EmailWeeklyDigest, input.EmailFindingsSeverity,
		input.EmailNewIncidents, input.EmailOverdueControls, input.EmailEvidenceExpiry,
		input.InappComments, input.InappApprovals, input.InappSystemUpdates,
	).Scan(
		&p.ID, &p.UserID,
		&p.EmailWeeklyDigest, &p.EmailFindingsSeverity,
		&p.EmailNewIncidents, &p.EmailOverdueControls, &p.EmailEvidenceExpiry,
		&p.InappComments, &p.InappApprovals, &p.InappSystemUpdates,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update notification preferences: %w", err)
	}
	return &p, nil
}

// IsWeeklyDigestEnabled returns true if email_weekly_digest is enabled for the given user.
// Returns true (safe default) if no row exists or a DB error occurs.
func IsWeeklyDigestEnabled(ctx context.Context, db *pgxpool.Pool, userID string) bool {
	var enabled bool
	err := db.QueryRow(ctx, `
		SELECT email_weekly_digest
		FROM notification_preferences
		WHERE user_id = $1::uuid`,
		userID,
	).Scan(&enabled)
	if err != nil {
		// No row yet → default is true.
		return true
	}
	return enabled
}
