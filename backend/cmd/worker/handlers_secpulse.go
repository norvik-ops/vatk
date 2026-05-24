// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0
// See LICENSE file in the project root for terms.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/matharnica/vakt/internal/config"
	"github.com/matharnica/vakt/internal/modules/secpulse"
	"github.com/matharnica/vakt/internal/services/alerting"
)

func handleScanJob(cfg *config.Config, pool *pgxpool.Pool) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload secpulse.ScanPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("parse scan payload: %w", err)
		}

		// Build alertSvc once — nil when alerting is not configured.
		var alertSvc *alerting.Service
		if cfg != nil && cfg.SecretKey != "" {
			if masterKey, keyErr := hexDecodeKey(cfg.SecretKey); keyErr == nil {
				alertSvc = alerting.NewService(pool, masterKey, alerting.SMTPConfig{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Pass: cfg.SMTPPass, From: cfg.SMTPFrom})
			}
		}

		// Sprint 17 S17-2: Live-Progress via Redis Pub/Sub. rdb darf nil sein
		// (lokales Dev ohne Redis) — PublishProgress ist dann no-op.
		var rdb *redis.Client
		if cfg != nil && cfg.RedisUrl != "" {
			if opt, err := redis.ParseURL(cfg.RedisUrl); err == nil {
				rdb = redis.NewClient(opt)
				defer rdb.Close()
			}
		}
		secpulse.PublishProgress(ctx, rdb, secpulse.ProgressEvent{
			ScanID: payload.ScanID, Phase: "started", Message: payload.Scanner + " scan started",
		})

		var scanErr error
		switch t.Type() {
		case secpulse.TaskScanTrivy:
			scanErr = secpulse.RunTrivyScan(ctx, pool, payload)
			if scanErr != nil {
				log.Error().Err(scanErr).Str("scan_id", payload.ScanID).Msg("trivy scan failed")
			}
		case secpulse.TaskScanNuclei:
			scanErr = secpulse.RunNucleiScan(ctx, pool, payload)
			if scanErr != nil {
				log.Error().Err(scanErr).Str("scan_id", payload.ScanID).Msg("nuclei scan failed")
			}
		case secpulse.TaskScanOpenVAS:
			scanErr = secpulse.RunOpenVASScan(ctx, pool, payload)
			if scanErr != nil {
				log.Error().Err(scanErr).Str("scan_id", payload.ScanID).Msg("openvas scan failed")
			}
		default:
			return fmt.Errorf("unknown scan task type: %s", t.Type())
		}

		// S17-2: Terminal-Event publizieren — beendet aktive SSE-Streams.
		finalPhase := "finished"
		if scanErr != nil {
			finalPhase = "failed"
		}
		secpulse.PublishProgress(ctx, rdb, secpulse.ProgressEvent{
			ScanID: payload.ScanID, Phase: finalPhase, Percent: 100,
			Message: payload.Scanner + " scan " + finalPhase,
		})

		if scanErr != nil {
			// Fire scan.failed alert — non-fatal, best-effort.
			if alertSvc != nil {
				alertSvc.Fire(ctx, payload.OrgID, alerting.EventScanFailed, map[string]any{
					"scan_id": payload.ScanID,
					"scanner": payload.Scanner,
					"asset":   payload.AssetName,
					"error":   scanErr.Error(),
				})
			}
			return scanErr
		}

		// After a successful scan, fire finding.new_critical for any critical
		// findings that belong to this scan and were seen for the first time today.
		if alertSvc != nil {
			var criticalCount int
			row := pool.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM vb_findings
				WHERE org_id = $1::uuid
				  AND scan_id = $2::uuid
				  AND severity = 'critical'
				  AND status = 'open'
				  AND created_at >= NOW() - INTERVAL '1 hour'
			`, payload.OrgID, payload.ScanID)
			if scanErr2 := row.Scan(&criticalCount); scanErr2 == nil && criticalCount > 0 {
				alertSvc.Fire(ctx, payload.OrgID, alerting.EventFindingNewCritical, map[string]any{
					"scan_id":        payload.ScanID,
					"scanner":        payload.Scanner,
					"asset":          payload.AssetName,
					"critical_count": criticalCount,
				})
			}
		}

		return nil
	}
}

// EnqueueScanTask enqueues a scan job with retry and timeout options appropriate
// for long-running scanner processes (Trivy, Nuclei, OpenVAS).
// OpenVAS scans can take up to 10 minutes and are placed on the "low" queue
// to avoid blocking faster Trivy/Nuclei jobs on the "default" queue.

func handleGenerateReport(cfg *config.Config, pool *pgxpool.Pool) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload secpulse.GenerateReportPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("parse report payload: %w", err)
		}

		repo := secpulse.NewRepository(pool)

		if err := repo.UpdateReport(ctx, payload.ReportID, "", "processing", nil); err != nil {
			return fmt.Errorf("mark report processing: %w", err)
		}

		title := payload.Scope.Title

		log.Info().Str("report_id", payload.ReportID).Str("org_id", payload.OrgID).Msg("generating PDF report")

		pdfBytes, err := secpulse.GenerateReportPDF(ctx, pool, payload.OrgID, title)
		if err != nil {
			_ = repo.UpdateReport(ctx, payload.ReportID, "", "failed", nil)
			return fmt.Errorf("generate PDF: %w", err)
		}

		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		if err := repo.StoreReportContent(ctx, payload.ReportID, pdfBytes, expiresAt); err != nil {
			return fmt.Errorf("store report content: %w", err)
		}

		log.Info().Str("report_id", payload.ReportID).Int("bytes", len(pdfBytes)).Msg("PDF report generated")
		return nil
	}
}

// handleEPSSEnrich enriches all open findings across all organisations with
// EPSS scores fetched from the FIRST.org API. Errors for individual orgs are
// logged but do not abort processing of remaining orgs.
// Enrichment is opt-in via VAKT_EPSS_ENABLED=true because it sends CVE IDs to
// the external api.first.org service, which contradicts the self-hosted
// data-privacy promise.
func handleEPSSEnrich(cfg *config.Config, pool *pgxpool.Pool) asynq.HandlerFunc {
	return func(ctx context.Context, _ *asynq.Task) error {
		if cfg == nil || !cfg.EPSSEnabled {
			log.Info().Msg("epss_enrich: skipped — set VAKT_EPSS_ENABLED=true to enable (sends CVE IDs to api.first.org)")
			return nil
		}

		rows, err := pool.Query(ctx, `SELECT id::text FROM organizations WHERE is_deleted = false`)
		if err != nil {
			return fmt.Errorf("epss_enrich: list orgs: %w", err)
		}
		defer rows.Close()

		var orgIDs []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				continue
			}
			orgIDs = append(orgIDs, id)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		for _, orgID := range orgIDs {
			if err := secpulse.UpdateEPSSScores(ctx, pool, orgID); err != nil {
				log.Error().Err(err).Str("org_id", orgID).Msg("epss_enrich: org failed")
			}
		}
		log.Info().Int("orgs", len(orgIDs)).Msg("epss_enrich: completed")
		return nil
	}
}

// handleSBOMGenerate handles secpulse:sbom:generate jobs.
// It calls RunSyftScan to generate a CycloneDX SBOM and persist it in the DB.
func handleSBOMGenerate(cfg *config.Config, pool *pgxpool.Pool) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload secpulse.SBOMGeneratePayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("parse sbom generate payload: %w", err)
		}

		if err := secpulse.RunSyftScan(ctx, pool, payload.OrgID, payload.AssetID, payload.Target); err != nil {
			log.Error().Err(err).
				Str("org_id", payload.OrgID).
				Str("asset_id", payload.AssetID).
				Msg("syft SBOM scan failed")
			return err
		}
		return nil
	}
}

// handleEOLCheck handles secpulse:eol:check jobs.
// It calls EOLChecker.CheckComponents to resolve EOL status for all SBOM components.
func handleEOLCheck(cfg *config.Config, pool *pgxpool.Pool) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload secpulse.EOLCheckPayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("parse eol check payload: %w", err)
		}

		checker := secpulse.NewEOLChecker(pool)
		if err := checker.CheckComponents(ctx, payload.OrgID, payload.SBOMID); err != nil {
			log.Error().Err(err).
				Str("org_id", payload.OrgID).
				Str("sbom_id", payload.SBOMID).
				Msg("EOL check failed")
			return err
		}
		return nil
	}
}
