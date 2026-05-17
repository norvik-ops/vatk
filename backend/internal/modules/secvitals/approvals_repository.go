// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package secvitals

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Approval represents a pending or resolved control status change approval request.
type Approval struct {
	ID              string     `json:"id"`
	OrgID           string     `json:"org_id"`
	ControlID       string     `json:"control_id"`
	RequestedBy     string     `json:"requested_by"`
	RequestedStatus string     `json:"requested_status"`
	CurrentStatus   string     `json:"current_status"`
	Comment         string     `json:"comment,omitempty"`
	Status          string     `json:"status"` // pending | approved | rejected
	ReviewedBy      string     `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	ReviewComment   string     `json:"review_comment,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ApprovalWithDetails enriches an Approval with control title and requester name.
type ApprovalWithDetails struct {
	Approval
	ControlTitle      string `json:"control_title"`
	ControlRef        string `json:"control_ref"`
	RequesterName     string `json:"requester_name"`
	RequesterEmail    string `json:"requester_email"`
}

// CreateApprovalRequest inserts a new pending approval request.
func (r *Repository) CreateApprovalRequest(
	ctx context.Context,
	orgID, controlID, requestedBy, requestedStatus, currentStatus, comment string,
) (*Approval, error) {
	var a Approval
	err := r.db.QueryRow(ctx, `
		INSERT INTO ck_control_approvals
		  (org_id, control_id, requested_by, requested_status, current_status, comment)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, NULLIF($6,''))
		RETURNING
		  id::text, org_id::text, control_id::text, requested_by::text,
		  requested_status, current_status, COALESCE(comment,''),
		  status, COALESCE(reviewed_by::text,''), reviewed_at, COALESCE(review_comment,''), created_at`,
		orgID, controlID, requestedBy, requestedStatus, currentStatus, comment,
	).Scan(
		&a.ID, &a.OrgID, &a.ControlID, &a.RequestedBy,
		&a.RequestedStatus, &a.CurrentStatus, &a.Comment,
		&a.Status, &a.ReviewedBy, &a.ReviewedAt, &a.ReviewComment, &a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create approval request: %w", err)
	}
	return &a, nil
}

// ListPendingApprovals returns all pending approvals for an org, joined with control and user info.
func (r *Repository) ListPendingApprovals(ctx context.Context, orgID string) ([]ApprovalWithDetails, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
		  a.id::text,
		  a.org_id::text,
		  a.control_id::text,
		  a.requested_by::text,
		  a.requested_status,
		  a.current_status,
		  COALESCE(a.comment,''),
		  a.status,
		  COALESCE(a.reviewed_by::text,''),
		  a.reviewed_at,
		  COALESCE(a.review_comment,''),
		  a.created_at,
		  COALESCE(c.title,''),
		  COALESCE(c.control_id,''),
		  COALESCE(u.display_name, u.email,''),
		  COALESCE(u.email,'')
		FROM ck_control_approvals a
		LEFT JOIN ck_controls c ON c.id = a.control_id
		LEFT JOIN users u ON u.id = a.requested_by
		WHERE a.org_id = $1::uuid AND a.status = 'pending'
		ORDER BY a.created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending approvals: %w", err)
	}
	defer rows.Close()

	var results []ApprovalWithDetails
	for rows.Next() {
		var d ApprovalWithDetails
		if err := rows.Scan(
			&d.ID, &d.OrgID, &d.ControlID, &d.RequestedBy,
			&d.RequestedStatus, &d.CurrentStatus, &d.Comment,
			&d.Status, &d.ReviewedBy, &d.ReviewedAt, &d.ReviewComment, &d.CreatedAt,
			&d.ControlTitle, &d.ControlRef,
			&d.RequesterName, &d.RequesterEmail,
		); err != nil {
			return nil, fmt.Errorf("scan approval: %w", err)
		}
		results = append(results, d)
	}
	return results, rows.Err()
}

// GetApproval returns a single approval request by ID within an org.
func (r *Repository) GetApproval(ctx context.Context, orgID, approvalID string) (*Approval, error) {
	var a Approval
	err := r.db.QueryRow(ctx, `
		SELECT
		  id::text, org_id::text, control_id::text, requested_by::text,
		  requested_status, current_status, COALESCE(comment,''),
		  status, COALESCE(reviewed_by::text,''), reviewed_at, COALESCE(review_comment,''), created_at
		FROM ck_control_approvals
		WHERE id = $1::uuid AND org_id = $2::uuid`,
		approvalID, orgID,
	).Scan(
		&a.ID, &a.OrgID, &a.ControlID, &a.RequestedBy,
		&a.RequestedStatus, &a.CurrentStatus, &a.Comment,
		&a.Status, &a.ReviewedBy, &a.ReviewedAt, &a.ReviewComment, &a.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("approval not found")
		}
		return nil, fmt.Errorf("get approval: %w", err)
	}
	return &a, nil
}

// ReviewApproval marks an approval as approved or rejected and optionally updates the control status.
func (r *Repository) ReviewApproval(
	ctx context.Context,
	orgID, approvalID, reviewerID string,
	approve bool,
	comment string,
) error {
	newStatus := "rejected"
	if approve {
		newStatus = "approved"
	}

	now := time.Now().UTC()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Update approval record.
	var controlID, requestedStatus string
	err = tx.QueryRow(ctx, `
		UPDATE ck_control_approvals
		SET status = $3, reviewed_by = $4::uuid, reviewed_at = $5, review_comment = NULLIF($6,'')
		WHERE id = $1::uuid AND org_id = $2::uuid AND status = 'pending'
		RETURNING control_id::text, requested_status`,
		approvalID, orgID, newStatus, reviewerID, now, comment,
	).Scan(&controlID, &requestedStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("approval not found or already reviewed")
		}
		return fmt.Errorf("update approval: %w", err)
	}

	// If approved, apply the status change to the control.
	if approve {
		var notApplicable bool
		var manualStatus string
		switch requestedStatus {
		case "not_applicable":
			notApplicable = true
			manualStatus = ""
		case "missing":
			notApplicable = false
			manualStatus = ""
		default:
			notApplicable = false
			manualStatus = requestedStatus
		}

		_, err = tx.Exec(ctx, `
			UPDATE ck_controls
			SET not_applicable = $3, manual_status = $4, updated_at = NOW()
			WHERE id = $1::uuid AND org_id = $2::uuid`,
			controlID, orgID, notApplicable, manualStatus,
		)
		if err != nil {
			return fmt.Errorf("update control status: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// OrgApprovalRequired returns whether the organisation requires approval for control status changes.
func (r *Repository) OrgApprovalRequired(ctx context.Context, orgID string) (bool, error) {
	var required bool
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(approval_required, false) FROM organizations WHERE id = $1::uuid`, orgID,
	).Scan(&required)
	if err != nil {
		return false, fmt.Errorf("get org approval_required: %w", err)
	}
	return required, nil
}

// SetOrgApprovalRequired updates the approval_required flag for an organisation.
func (r *Repository) SetOrgApprovalRequired(ctx context.Context, orgID string, required bool) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE organizations SET approval_required = $2, updated_at = NOW() WHERE id = $1::uuid`,
		orgID, required,
	)
	if err != nil {
		return fmt.Errorf("set org approval_required: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("org not found: %s", orgID)
	}
	return nil
}

// CountPendingApprovals returns the number of pending approvals for an org.
func (r *Repository) CountPendingApprovals(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM ck_control_approvals WHERE org_id = $1::uuid AND status = 'pending'`,
		orgID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending approvals: %w", err)
	}
	return count, nil
}
