package services

import (
	"better-feature-flag/internal/models"
	"context"
)

type FeatureFlagEvaluator interface {
	EvaluateFlags(ctx context.Context, flags []models.FlagDefinition, clientCtx *models.ClientContext) (map[string]interface{}, error)
	HealthCheck(ctx context.Context, flags []models.FlagDefinition) error
}

type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*models.TokenClaims, error)
}

type FlagRegistry interface {
	GetFlagsForApp(appName string) ([]models.FlagDefinition, error)
	GetAnyFlags() ([]models.FlagDefinition, error)
}
