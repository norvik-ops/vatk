// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0

package notifications

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// PreferencesHandler handles HTTP requests for notification preferences.
type PreferencesHandler struct {
	svc      *PreferencesService
	validate *validator.Validate
}

// NewPreferencesHandler creates a new PreferencesHandler.
func NewPreferencesHandler(svc *PreferencesService) *PreferencesHandler {
	return &PreferencesHandler{svc: svc, validate: validator.New()}
}

// RegisterPreferences mounts notification preference routes under the provided group.
// Expected base: /api/v1/notifications
func RegisterPreferences(g *echo.Group, h *PreferencesHandler) {
	g.GET("/preferences", h.Get)
	g.PUT("/preferences", h.Update)
}

// Get handles GET /api/v1/notifications/preferences.
// Returns the notification preferences for the authenticated user.
func (h *PreferencesHandler) Get(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "not authenticated",
			"code":  "UNAUTHORIZED",
		})
	}

	prefs, err := h.svc.GetPreferences(c.Request().Context(), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("get notification preferences failed")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get notification preferences",
			"code":  "NOTIF_PREFS_GET_ERROR",
		})
	}
	return c.JSON(http.StatusOK, prefs)
}

// Update handles PUT /api/v1/notifications/preferences.
// Updates the notification preferences for the authenticated user.
func (h *PreferencesHandler) Update(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "not authenticated",
			"code":  "UNAUTHORIZED",
		})
	}

	var input UpdatePreferencesInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
			"code":  "NOTIF_PREFS_BAD_REQUEST",
		})
	}
	if err := h.validate.Struct(input); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{
			"error": err.Error(),
			"code":  "NOTIF_PREFS_VALIDATION_ERROR",
		})
	}

	prefs, err := h.svc.UpdatePreferences(c.Request().Context(), userID, input)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("update notification preferences failed")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to update notification preferences",
			"code":  "NOTIF_PREFS_UPDATE_ERROR",
		})
	}
	return c.JSON(http.StatusOK, prefs)
}
