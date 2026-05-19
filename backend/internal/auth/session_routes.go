// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package auth

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

// RegisterSessions mounts session management endpoints onto g.
// g must already be behind auth middleware.
func RegisterSessions(g *echo.Group, db *pgxpool.Pool, rdb *redis.Client) {
	h := NewSessionHandler(db, rdb)
	g.GET("", h.ListSessions)
	g.DELETE("/:id", h.RevokeSession)
	g.DELETE("", h.RevokeAllOtherSessions)
}
