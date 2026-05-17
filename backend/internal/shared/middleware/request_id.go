// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	// HeaderXRequestID is the canonical header name for request tracing.
	HeaderXRequestID = "X-Request-ID"
	// ContextKeyRequestID is the echo.Context key used to store the request ID.
	ContextKeyRequestID = "request_id"
)

// RequestID is an Echo middleware that reads the X-Request-ID header from the
// incoming request.  If the header is absent or empty a new UUID v4 is
// generated.  The resolved ID is:
//   - written back to the request header (so downstream handlers can read it)
//   - echoed in the response header
//   - stored in the echo.Context under the key "request_id"
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(HeaderXRequestID)
			if id == "" {
				id = uuid.New().String()
			}

			// Propagate to downstream handlers via the request header.
			c.Request().Header.Set(HeaderXRequestID, id)
			// Echo on the response so clients can correlate logs.
			c.Response().Header().Set(HeaderXRequestID, id)
			// Store in context for use in structured log entries.
			c.Set(ContextKeyRequestID, id)

			return next(c)
		}
	}
}
