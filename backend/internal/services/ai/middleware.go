// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package ai

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/matharnica/vakt/internal/license"
)

// RequireAILimit returns an Echo middleware that gates a route on the
// Community-Edition monthly AI quota.  When a CE org has consumed its
// monthly allowance the middleware responds with HTTP 402 (Payment
// Required) and the same JSON shape that the handlers used to compose
// inline.  Pro/Enterprise orgs always pass through.
//
// Audit finding F3: prior to this middleware, the gate was implemented as
// an inline `h.checkCELimit(c)` call at the top of every LLM-producing
// handler.  `GenerateReport` (handler.go) and `AgentRun` (agent_handler.go)
// were never wired up, so two of the most-used AI endpoints had no quota
// enforcement at all — CE customers paid €0 instead of €199 because they
// never hit a paywall.  Centralising the gate in middleware means new AI
// routes inherit the limit by construction.
func RequireAILimit(svc *Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if svc == nil || svc.usage == nil {
				// Tests + early-boot paths can run without a tracker; the
				// gate is then a no-op.  Production wiring ensures svc.usage
				// is non-nil whenever a Pro endpoint is registered.
				return next(c)
			}
			lic, _ := c.Get("license").(*license.License)
			if lic != nil && lic.IsPro() {
				return next(c)
			}
			orgID, _ := c.Get("org_id").(string)
			used := svc.usage.CEMonthlyUsage(c.Request().Context(), orgID)
			if used >= CEMonthlyLimit {
				return c.JSON(http.StatusPaymentRequired, ceLimitResponse{
					Error:     "AI-Limit für Community Edition erreicht",
					Code:      "AI_CE_MONTHLY_LIMIT",
					Used:      used,
					Limit:     CEMonthlyLimit,
					ResetHint: "Limit wird am 1. des nächsten Monats zurückgesetzt. Upgrade auf Vakt Pro für unbegrenzte AI-Anfragen.",
				})
			}
			return next(c)
		}
	}
}
