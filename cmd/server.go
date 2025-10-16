package cmd

import (
	"better-feature-flag/internal/config"
	"better-feature-flag/internal/handlers"
	"better-feature-flag/internal/middleware"
	"better-feature-flag/internal/services"

	fxserver "better-feature-flag/internal/fx"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the feature flag proxy server",
	Long:  `Start the HTTP server that proxies feature flag requests with authentication.`,
	Run: func(cmd *cobra.Command, args []string) {
		fx.New(
			config.Module,
			services.Module,
			handlers.Module,
			middleware.Module,
			fxserver.Module,
		).Run()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}
