package main

import (
	"better-feature-flag/src/internal/config"
	"better-feature-flag/src/internal/handlers"
	"better-feature-flag/src/internal/middleware"
	"better-feature-flag/src/internal/services"

	fxserver "better-feature-flag/src/internal/fx"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		config.Module,
		services.Module,
		handlers.Module,
		middleware.Module,
		fxserver.Module,
	).Run()
}
