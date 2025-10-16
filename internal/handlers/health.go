package handlers

import (
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	service *services.FeatureFlagService
	logger  *slog.Logger
}

func NewHealthHandler(service *services.FeatureFlagService, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{
		service: service,
		logger:  logger,
	}
}

// Health é o liveness probe - verifica se a aplicação está rodando
func (h *HealthHandler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, models.HealthResponse{
		Status: "ok",
	})
}

// Ready é o readiness probe - verifica se a aplicação está pronta para receber tráfego
func (h *HealthHandler) Ready(c echo.Context) error {
	ctx := c.Request().Context()

	// Verifica conectividade com GO Feature Flag
	if err := h.service.HealthCheck(ctx); err != nil {
		h.logger.Error("readiness check failed", slog.String("error", err.Error()))
		return c.JSON(http.StatusServiceUnavailable, models.HealthResponse{
			Status:  "unavailable",
			Message: "GO Feature Flag service is not available",
		})
	}

	return c.JSON(http.StatusOK, models.HealthResponse{
		Status: "ready",
	})
}
