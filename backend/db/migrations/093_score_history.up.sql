-- Migration 092: Compliance score history for trend charts
-- Copyright (c) 2026 NorvikOps. All rights reserved.
-- SPDX-License-Identifier: Elastic-2.0

CREATE TABLE ck_score_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  framework_id UUID,
  score NUMERIC(5,2) NOT NULL,
  controls_total INT NOT NULL DEFAULT 0,
  controls_implemented INT NOT NULL DEFAULT 0,
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON ck_score_history(org_id, framework_id, recorded_at DESC);
