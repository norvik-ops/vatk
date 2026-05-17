// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package secvitals

import (
	"context"
	"fmt"
	"math"
	"time"
)

// ListMilestones returns all milestones for an org ordered by milestone_date ASC.
// If statusFilter is non-empty only that status is returned.
func (r *Repository) ListMilestones(ctx context.Context, orgID, statusFilter string) ([]AuditMilestone, error) {
	query := `
		SELECT id::text, org_id::text,
		       framework_id::text,
		       title, COALESCE(description,''), milestone_date::text,
		       milestone_type, status,
		       created_by::text,
		       created_at, updated_at
		FROM ck_audit_milestones
		WHERE org_id = $1::uuid`
	args := []any{orgID}

	if statusFilter != "" {
		query += " AND status = $2"
		args = append(args, statusFilter)
	}
	query += " ORDER BY milestone_date ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list milestones: %w", err)
	}
	defer rows.Close()

	today := time.Now().UTC().Truncate(24 * time.Hour)
	var milestones []AuditMilestone
	for rows.Next() {
		var m AuditMilestone
		var fwID, createdBy *string
		if err := rows.Scan(
			&m.ID, &m.OrgID, &fwID,
			&m.Title, &m.Description, &m.MilestoneDate,
			&m.MilestoneType, &m.Status, &createdBy,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan milestone: %w", err)
		}
		m.FrameworkID = fwID
		m.CreatedBy = createdBy
		m.DaysRemaining = computeDaysRemaining(m.MilestoneDate, today)
		milestones = append(milestones, m)
	}
	return milestones, rows.Err()
}

// GetMilestone retrieves a single milestone by ID.
func (r *Repository) GetMilestone(ctx context.Context, orgID, milestoneID string) (*AuditMilestone, error) {
	var m AuditMilestone
	var fwID, createdBy *string
	err := r.db.QueryRow(ctx, `
		SELECT id::text, org_id::text,
		       framework_id::text,
		       title, COALESCE(description,''), milestone_date::text,
		       milestone_type, status,
		       created_by::text,
		       created_at, updated_at
		FROM ck_audit_milestones
		WHERE id = $1::uuid AND org_id = $2::uuid`,
		milestoneID, orgID,
	).Scan(
		&m.ID, &m.OrgID, &fwID,
		&m.Title, &m.Description, &m.MilestoneDate,
		&m.MilestoneType, &m.Status, &createdBy,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get milestone: %w", err)
	}
	m.FrameworkID = fwID
	m.CreatedBy = createdBy
	today := time.Now().UTC().Truncate(24 * time.Hour)
	m.DaysRemaining = computeDaysRemaining(m.MilestoneDate, today)
	return &m, nil
}

// CreateMilestone inserts a new milestone.
func (r *Repository) CreateMilestone(ctx context.Context, orgID, createdBy string, in CreateMilestoneInput) (*AuditMilestone, error) {
	var fwIDArg *string
	if in.FrameworkID != "" {
		fwIDArg = &in.FrameworkID
	}
	var createdByArg *string
	if createdBy != "" {
		createdByArg = &createdBy
	}

	var m AuditMilestone
	var fwIDOut, createdByOut *string
	err := r.db.QueryRow(ctx, `
		INSERT INTO ck_audit_milestones
		  (org_id, framework_id, title, description, milestone_date, milestone_type, created_by)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::date, $6, $7::uuid)
		RETURNING id::text, org_id::text,
		          framework_id::text,
		          title, COALESCE(description,''), milestone_date::text,
		          milestone_type, status,
		          created_by::text,
		          created_at, updated_at`,
		orgID, fwIDArg, in.Title, in.Description, in.MilestoneDate, in.MilestoneType, createdByArg,
	).Scan(
		&m.ID, &m.OrgID, &fwIDOut,
		&m.Title, &m.Description, &m.MilestoneDate,
		&m.MilestoneType, &m.Status, &createdByOut,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create milestone: %w", err)
	}
	m.FrameworkID = fwIDOut
	m.CreatedBy = createdByOut
	today := time.Now().UTC().Truncate(24 * time.Hour)
	m.DaysRemaining = computeDaysRemaining(m.MilestoneDate, today)
	return &m, nil
}

// UpdateMilestone applies a partial update to an existing milestone.
func (r *Repository) UpdateMilestone(ctx context.Context, orgID, milestoneID string, in UpdateMilestoneInput) (*AuditMilestone, error) {
	// Fetch current to merge
	cur, err := r.GetMilestone(ctx, orgID, milestoneID)
	if err != nil {
		return nil, err
	}

	title := cur.Title
	description := cur.Description
	milestoneDate := cur.MilestoneDate
	milestoneType := cur.MilestoneType
	status := cur.Status

	if in.Title != nil {
		title = *in.Title
	}
	if in.Description != nil {
		description = *in.Description
	}
	if in.MilestoneDate != nil {
		milestoneDate = *in.MilestoneDate
	}
	if in.MilestoneType != nil {
		milestoneType = *in.MilestoneType
	}
	if in.Status != nil {
		status = *in.Status
	}

	var m AuditMilestone
	var fwIDOut, createdByOut *string
	err = r.db.QueryRow(ctx, `
		UPDATE ck_audit_milestones
		SET title = $1, description = $2, milestone_date = $3::date,
		    milestone_type = $4, status = $5, updated_at = NOW()
		WHERE id = $6::uuid AND org_id = $7::uuid
		RETURNING id::text, org_id::text,
		          framework_id::text,
		          title, COALESCE(description,''), milestone_date::text,
		          milestone_type, status,
		          created_by::text,
		          created_at, updated_at`,
		title, description, milestoneDate, milestoneType, status,
		milestoneID, orgID,
	).Scan(
		&m.ID, &m.OrgID, &fwIDOut,
		&m.Title, &m.Description, &m.MilestoneDate,
		&m.MilestoneType, &m.Status, &createdByOut,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update milestone: %w", err)
	}
	m.FrameworkID = fwIDOut
	m.CreatedBy = createdByOut
	today := time.Now().UTC().Truncate(24 * time.Hour)
	m.DaysRemaining = computeDaysRemaining(m.MilestoneDate, today)
	return &m, nil
}

// DeleteMilestone removes a milestone.
func (r *Repository) DeleteMilestone(ctx context.Context, orgID, milestoneID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM ck_audit_milestones WHERE id = $1::uuid AND org_id = $2::uuid`,
		milestoneID, orgID,
	)
	if err != nil {
		return fmt.Errorf("delete milestone: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("milestone not found")
	}
	return nil
}

// NextMilestone returns the nearest upcoming milestone or nil if none exist.
func (r *Repository) NextMilestone(ctx context.Context, orgID string) (*AuditMilestone, error) {
	var m AuditMilestone
	var fwIDOut, createdByOut *string
	err := r.db.QueryRow(ctx, `
		SELECT id::text, org_id::text,
		       framework_id::text,
		       title, COALESCE(description,''), milestone_date::text,
		       milestone_type, status,
		       created_by::text,
		       created_at, updated_at
		FROM ck_audit_milestones
		WHERE org_id = $1::uuid
		  AND status = 'upcoming'
		  AND milestone_date >= CURRENT_DATE
		ORDER BY milestone_date ASC
		LIMIT 1`,
		orgID,
	).Scan(
		&m.ID, &m.OrgID, &fwIDOut,
		&m.Title, &m.Description, &m.MilestoneDate,
		&m.MilestoneType, &m.Status, &createdByOut,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, err // caller checks pgx.ErrNoRows
	}
	m.FrameworkID = fwIDOut
	m.CreatedBy = createdByOut
	today := time.Now().UTC().Truncate(24 * time.Hour)
	m.DaysRemaining = computeDaysRemaining(m.MilestoneDate, today)
	return &m, nil
}

// computeDaysRemaining returns a pointer to the number of days between today and the milestone date.
// Negative values mean the milestone is overdue.
func computeDaysRemaining(dateStr string, today time.Time) *int {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil
	}
	days := int(math.Round(t.Sub(today).Hours() / 24))
	return &days
}
