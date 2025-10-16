package fx

import (
	"better-feature-flag/src/internal/config"
	"better-feature-flag/src/internal/handlers"
	"better-feature-flag/src/internal/middleware"
	"context"
	"log/slog"
	"os"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

var Module = fx.Module("server",
	fx.Provide(
		ProvideLogger,
		ProvideEcho,
	),
	fx.Invoke(RegisterRoutes),
)

func ProvideLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	return logger
}

func ProvideEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	return e
}

type RouteParams struct {
	fx.In

	Lifecycle        fx.Lifecycle
	Logger           *slog.Logger
	Config           *config.Config
	Echo             *echo.Echo
	FlagsHandler     *handlers.FlagsHandler
	HealthHandler    *handlers.HealthHandler
	AuthMiddleware   *middleware.AuthMiddleware
	CORSMiddleware   echo.MiddlewareFunc `name:"cors"`
	LoggerMiddleware echo.MiddlewareFunc `name:"logger"`
}

func RegisterRoutes(p RouteParams) {
	// Log startup
	p.Logger.Info("Starting Better Feature Flag")
	p.Logger.Info("configuration loaded",
		slog.String("environment", p.Config.Environment),
		slog.String("goff_endpoint", p.Config.GoffEndpoint),
		slog.String("port", p.Config.ServerPort),
	)

	// Middlewares globais
	p.Echo.Use(p.CORSMiddleware)
	p.Echo.Use(p.LoggerMiddleware)

	// Health checks (sem autenticação)
	p.Echo.GET("/health", p.HealthHandler.Health)
	p.Echo.GET("/ready", p.HealthHandler.Ready)

	// API routes
	api := p.Echo.Group("/api/v1")
	api.Use(p.AuthMiddleware.OptionalJWT())
	api.GET("/flags", p.FlagsHandler.GetFlags)

	// Lifecycle hooks
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				p.Logger.Info("server started", slog.String("port", p.Config.ServerPort))
				if err := p.Echo.Start(":" + p.Config.ServerPort); err != nil {
					p.Logger.Error("server error", slog.String("error", err.Error()))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("shutting down server")
			if err := p.Echo.Shutdown(ctx); err != nil {
				p.Logger.Error("server shutdown error", slog.String("error", err.Error()))
			}
			p.Logger.Info("server stopped")
			return nil
		},
	})
}
