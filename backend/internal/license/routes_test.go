// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package license

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// TestRequireAdminRole_RoleNameMatchesDBSeed verifies the fix for audit
// finding F10. The middleware MUST accept the PascalCase role name "Admin"
// that the DB seed in migrations/001_core_schema.up.sql installs and that
// internal/auth/middleware.go emits.
//
// The previous implementation checked for lowercase "admin", which is never
// produced anywhere in the codebase — making the Pro license activation
// endpoint un-callable for every legitimate admin.
func TestRequireAdminRole_RoleNameMatchesDBSeed(t *testing.T) {
	cases := []struct {
		name       string
		roles      []string
		wantStatus int
	}{
		{
			name:       "Admin (PascalCase, as seeded in DB) — allowed",
			roles:      []string{"Admin"},
			wantStatus: http.StatusNoContent, // next() handler answers 204
		},
		{
			name:       "admin (lowercase — historical bug shape) — rejected",
			roles:      []string{"admin"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "SecurityAnalyst — rejected",
			roles:      []string{"SecurityAnalyst"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "Viewer — rejected",
			roles:      []string{"Viewer"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "AuditorReadOnly — rejected",
			roles:      []string{"AuditorReadOnly"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "no roles — rejected",
			roles:      []string{},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "Admin alongside others — allowed",
			roles:      []string{"SecurityAnalyst", "Admin"},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/license/activate", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("roles", tc.roles)

			handler := requireAdminRole()(func(c echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			})

			err := handler(c)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

// TestRequireAdminRole_NilRolesContext exercises the edge case where the
// auth middleware did not set a "roles" key at all. The middleware MUST
// treat that as "no roles" rather than panicking.
func TestRequireAdminRole_NilRolesContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/license/activate", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// intentionally no c.Set("roles", ...)

	handler := requireAdminRole()(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	err := handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
