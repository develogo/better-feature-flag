package services

import (
	"go.uber.org/fx"
)

var Module = fx.Module("services",
	fx.Provide(
		fx.Annotate(
			NewFeatureFlagService,
			fx.As(new(FeatureFlagEvaluator)),
		),
		fx.Annotate(
			NewKeycloakService,
			fx.As(new(TokenValidator)),
		),
		fx.Annotate(
			NewFlagRegistryService,
			fx.As(new(FlagRegistry)),
		),
	),
)
