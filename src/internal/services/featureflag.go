package services

import (
	"better-feature-flag/src/internal/models"
	"context"
	"fmt"
	"log/slog"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
)

type FeatureFlagService struct {
	client *of.Client
	logger *slog.Logger
}

func NewFeatureFlagService(goffEndpoint string, logger *slog.Logger) (*FeatureFlagService, error) {
	provider, err := gofeatureflag.NewProvider(
		gofeatureflag.ProviderOptions{
			Endpoint:     goffEndpoint,
			DisableCache: true,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	of.SetProvider(provider)
	client := of.NewClient("better-feature-flag")

	return &FeatureFlagService{
		client: client,
		logger: logger,
	}, nil
}

// EvaluateAllFlags retorna todas as flags para o frontend (bulk)
func (s *FeatureFlagService) EvaluateAllFlags(ctx context.Context, clientCtx *models.ClientContext) (map[string]interface{}, error) {
	evalCtx := s.buildEvaluationContext(clientCtx)
	flags := make(map[string]interface{})

	// Frontend flags
	flags["front-dark-mode"], _ = s.client.BooleanValue(ctx, "front-dark-mode", false, evalCtx)
	flags["maintenance_mode"], _ = s.client.BooleanValue(ctx, "maintenance_mode", false, evalCtx)
	flags["maintenance_title"], _ = s.client.StringValue(ctx, "maintenance_title", "Estamos em manutenção", evalCtx)
	flags["maintenance_message"], _ = s.client.StringValue(ctx, "maintenance_message", "Mensagem da manutenção", evalCtx)
	flags["feedback_enabled"], _ = s.client.BooleanValue(ctx, "feedback_enabled", true, evalCtx)
	flags["force_update_enabled"], _ = s.client.BooleanValue(ctx, "force_update_enabled", false, evalCtx)
	flags["minimum_app_version"], _ = s.client.StringValue(ctx, "minimum_app_version", "1.0.0", evalCtx)
	flags["update_title"], _ = s.client.StringValue(ctx, "update_title", "Atualização Necessária", evalCtx)
	flags["update_message"], _ = s.client.StringValue(ctx, "update_message", "Atualize para continuar", evalCtx)
	flags["new_dashboard"], _ = s.client.BooleanValue(ctx, "new_dashboard", false, evalCtx)

	s.logger.Info("all flags evaluated",
		slog.Int("count", len(flags)),
		slog.String("targeting_key", clientCtx.GetTargetingKey()),
	)

	return flags, nil
}

func (s *FeatureFlagService) buildEvaluationContext(clientCtx *models.ClientContext) of.EvaluationContext {
	attributes := map[string]interface{}{
		"app_version": clientCtx.AppVersion,
		"platform":    clientCtx.Platform,
	}

	if clientCtx.IsAuthenticated() {
		attributes["user_id"] = clientCtx.UserID
		attributes["email"] = clientCtx.Email
		attributes["username"] = clientCtx.Username
	} else {
		attributes["device_id"] = clientCtx.DeviceID
	}

	return of.NewEvaluationContext(
		clientCtx.GetTargetingKey(),
		attributes,
	)
}

// HealthCheck verifica se o serviço está disponível
func (s *FeatureFlagService) HealthCheck(ctx context.Context) error {
	// Tenta avaliar uma flag simples para verificar conectividade
	evalCtx := of.NewEvaluationContext("health-check", map[string]interface{}{})
	_, err := s.client.BooleanValue(ctx, "maintenance_mode", false, evalCtx)
	return err
}
