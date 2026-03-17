package middleware

import (
	"go.uber.org/fx"
)

var Module = fx.Module("middleware",
	fx.Provide(
		NewAuthMiddleware,
		fx.Annotate(
			CORS,
			fx.ResultTags(`name:"cors"`),
		),
		fx.Annotate(
			Logger,
			fx.ResultTags(`name:"logger"`),
		),
		fx.Annotate(
			RequestID,
			fx.ResultTags(`name:"requestid"`),
		),
	),
)
