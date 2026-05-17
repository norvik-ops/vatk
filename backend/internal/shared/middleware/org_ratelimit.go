// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package middleware

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

const (
	// orgRateLimitPerMinute is the default per-org request cap (300 req/min).
	orgRateLimitPerMinute = 300
	// orgRateLimitExpiresIn controls how long an idle org entry is kept in the store.
	orgRateLimitExpiresIn = 5 * time.Minute
)

// orgVisitor holds the per-org token-bucket limiter.
type orgVisitor struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// orgRateLimitStore is an in-memory store keyed by org_id.
type orgRateLimitStore struct {
	mu        sync.Mutex
	visitors  map[string]*orgVisitor
	rateLimit rate.Limit
	burst     int
	expiresIn time.Duration
	lastClean time.Time
}

func newOrgRateLimitStore(reqPerMinute int, expiresIn time.Duration) *orgRateLimitStore {
	r := rate.Limit(float64(reqPerMinute) / 60.0)
	burst := reqPerMinute
	if burst <= 0 {
		burst = int(math.Max(1, math.Ceil(float64(r))))
	}
	return &orgRateLimitStore{
		visitors:  make(map[string]*orgVisitor),
		rateLimit: r,
		burst:     burst,
		expiresIn: expiresIn,
		lastClean: time.Now(),
	}
}

// allow checks whether the given org_id is within its rate limit.
// Returns (allowed, remaining, resetUnix).
func (s *orgRateLimitStore) allow(orgID string) (bool, int, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	v, ok := s.visitors[orgID]
	if !ok {
		v = &orgVisitor{lim: rate.NewLimiter(s.rateLimit, s.burst)}
		s.visitors[orgID] = v
	}
	v.lastSeen = now

	// Periodic cleanup of stale entries.
	if now.Sub(s.lastClean) > s.expiresIn {
		for id, vis := range s.visitors {
			if now.Sub(vis.lastSeen) > s.expiresIn {
				delete(s.visitors, id)
			}
		}
		s.lastClean = now
	}

	allowed := v.lim.Allow()

	// Calculate remaining tokens after the Allow() call.
	tokens := v.lim.Tokens()
	remaining := int(math.Floor(tokens))
	if remaining < 0 {
		remaining = 0
	}

	// X-RateLimit-Reset: when the bucket will next have ≥1 token.
	var waitSeconds float64
	if tokens < 1 {
		waitSeconds = (1 - tokens) / float64(s.rateLimit)
	}
	reset := now.Add(time.Duration(waitSeconds * float64(time.Second))).Unix()

	return allowed, remaining, reset
}

// OrgRateLimit returns an Echo middleware that enforces a per-org_id rate limit
// of 300 requests per minute on all protected routes.
//
// The org_id is read from the echo.Context key "org_id" which is populated by
// AuthMiddleware before any protected handler runs.  If org_id is not set the
// middleware falls through without limiting (e.g. anonymous health checks).
//
// On every allowed request the middleware sets X-RateLimit-* response headers.
// On rejection it returns 429 with a JSON body and the same headers.
func OrgRateLimit() echo.MiddlewareFunc {
	store := newOrgRateLimitStore(orgRateLimitPerMinute, orgRateLimitExpiresIn)
	limitStr := strconv.Itoa(orgRateLimitPerMinute)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			orgID, _ := c.Get("org_id").(string)
			if orgID == "" {
				// No org_id in context — not an authenticated request; skip limiting.
				return next(c)
			}

			allowed, remaining, reset := store.allow(orgID)

			c.Response().Header().Set("X-RateLimit-Limit", limitStr)
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))

			if !allowed {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "rate limit exceeded",
					"code":  "RATE_LIMIT_EXCEEDED",
				})
			}
			return next(c)
		}
	}
}
