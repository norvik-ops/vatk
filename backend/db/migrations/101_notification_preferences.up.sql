-- Copyright (c) 2026 NorvikOps. All rights reserved.
-- SPDX-License-Identifier: Elastic-2.0

CREATE TABLE notification_preferences (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- E-Mail
  email_weekly_digest BOOLEAN NOT NULL DEFAULT true,
  email_findings_severity TEXT NOT NULL DEFAULT 'critical' CHECK (email_findings_severity IN ('critical','high','all','none')),
  email_new_incidents BOOLEAN NOT NULL DEFAULT true,
  email_overdue_controls BOOLEAN NOT NULL DEFAULT true,
  email_evidence_expiry BOOLEAN NOT NULL DEFAULT true,
  -- In-App
  inapp_comments BOOLEAN NOT NULL DEFAULT true,
  inapp_approvals BOOLEAN NOT NULL DEFAULT true,
  inapp_system_updates BOOLEAN NOT NULL DEFAULT true,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX ON notification_preferences(user_id);
