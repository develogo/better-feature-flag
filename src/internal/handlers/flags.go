package handlers

import (
	"better-feature-flag/src/internal/middleware"
	"better-feature-flag/src/internal/models"
	"better-feature-flag/src/internal/services"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type FlagsHandler struct {
	service *services.FeatureFlagService
	logger  *slog.Logger
}

func NewFlagsHandler(service *services.FeatureFlagService, logger *slog.Logger) *FlagsHandler {
	return &FlagsHandler{
		service: service,
		logger:  logger,
	}
}

func (h *FlagsHandler) GetFlags(c echo.Context) error {
	ctx := c.Request().Context()

	// Obtém contexto do cliente (populado pelo middleware)
	clientCtx := middleware.GetClientContext(c)

	h.logger.Info("evaluating all flags",
		slog.String("targeting_key", clientCtx.GetTargetingKey()),
		slog.String("app_version", clientCtx.AppVersion),
		slog.String("platform", clientCtx.Platform),
		slog.Bool("authenticated", clientCtx.IsAuthenticated()),
	)

	// Avalia todas as flags (bulk para frontend)
	flags, err := h.service.EvaluateAllFlags(ctx, clientCtx)
	if err != nil {
		h.logger.Error("failed to evaluate flags", slog.String("error", err.Error()))
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Failed to evaluate feature flags",
		})
	}

	return c.JSON(http.StatusOK, models.FlagsResponse{
		Flags: flags,
	})
}
