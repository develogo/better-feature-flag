package main

import (
	"better-feature-flag/src/internal/config"
	"better-feature-flag/src/internal/handlers"
	"better-feature-flag/src/internal/middleware"
	"better-feature-flag/src/internal/services"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
)

func main() {
	// Setup logger estruturado
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Better Feature Flag")

	// Carrega configuração
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("configuration loaded",
		slog.String("environment", cfg.Environment),
		slog.String("goff_endpoint", cfg.GoffEndpoint),
		slog.String("port", cfg.ServerPort),
	)

	// Inicializa Feature Flag Service
	ffService, err := services.NewFeatureFlagService(cfg.GoffEndpoint, logger)
	if err != nil {
		logger.Error("failed to initialize feature flag service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Inicializa Keycloak Service
	keycloakService := services.NewKeycloakService(
		cfg.KeycloakURL,
		cfg.KeycloakRealm,
		cfg.KeycloakClientID,
		cfg.KeycloakClientSecret,
		logger,
	)

	logger.Info("keycloak service initialized",
		slog.String("url", cfg.KeycloakURL),
		slog.String("realm", cfg.KeycloakRealm),
		slog.String("client_id", cfg.KeycloakClientID),
	)

	// Inicializa handlers
	flagsHandler := handlers.NewFlagsHandler(ffService, logger)
	healthHandler := handlers.NewHealthHandler(ffService, logger)

	// Inicializa middlewares
	authMiddleware := middleware.NewAuthMiddleware(keycloakService)

	// Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middlewares globais
	e.Use(middleware.CORS())
	e.Use(middleware.Logger(logger))

	// Health checks (sem autenticação)
	e.GET("/health", healthHandler.Health)
	e.GET("/ready", healthHandler.Ready)

	// API routes
	api := e.Group("/api/v1")
	api.Use(authMiddleware.OptionalJWT())
	api.GET("/flags", flagsHandler.GetFlags)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Start server
	go func() {
		logger.Info("server started", slog.String("port", cfg.ServerPort))
		if err := e.Start(":" + cfg.ServerPort); err != nil {
			logger.Error("server error", slog.String("error", err.Error()))
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	logger.Info("shutting down server")

	// Graceful shutdown com timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}
