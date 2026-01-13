package services

import (
	"better-feature-flag/internal/config"
	"better-feature-flag/internal/models"
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

func NewFeatureFlagService(cfg *config.Config, logger *slog.Logger) (*FeatureFlagService, error) {
	provider, err := gofeatureflag.NewProvider(
		gofeatureflag.ProviderOptions{
			Endpoint:     cfg.Goff.Endpoint,
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

func (s *FeatureFlagService) EvaluateAllFlags(ctx context.Context, clientCtx *models.ClientContext) (map[string]interface{}, error) {
	evalCtx := s.buildEvaluationContext(clientCtx)
	flags := make(map[string]interface{}, len(FrontendFlagDefinitions))
	defaultedCount := 0

	for _, def := range FrontendFlagDefinitions {
		switch def.Type {
		case FlagValueTypeBool:
			defaultValue, ok := def.Default.(bool)
			if !ok {
				defaultedCount++
				flags[def.Name] = false
				s.logger.Warn("invalid bool flag default; using false",
					slog.String("flag", def.Name),
				)
				continue
			}
			value, err := s.client.BooleanValue(ctx, def.Name, defaultValue, evalCtx)
			if err != nil {
				defaultedCount++
				flags[def.Name] = defaultValue
				s.logger.Warn("flag evaluation failed; using default",
					slog.String("flag", def.Name),
					slog.String("type", string(def.Type)),
					slog.String("error", err.Error()),
				)
				continue
			}
			flags[def.Name] = value
		case FlagValueTypeString:
			defaultValue, ok := def.Default.(string)
			if !ok {
				defaultedCount++
				flags[def.Name] = ""
				s.logger.Warn("invalid string flag default; using empty string",
					slog.String("flag", def.Name),
				)
				continue
			}
			value, err := s.client.StringValue(ctx, def.Name, defaultValue, evalCtx)
			if err != nil {
				defaultedCount++
				flags[def.Name] = defaultValue
				s.logger.Warn("flag evaluation failed; using default",
					slog.String("flag", def.Name),
					slog.String("type", string(def.Type)),
					slog.String("error", err.Error()),
				)
				continue
			}
			flags[def.Name] = value
		default:
			defaultedCount++
			flags[def.Name] = def.Default
			s.logger.Warn("unknown flag type; using default",
				slog.String("flag", def.Name),
				slog.String("type", string(def.Type)),
			)
		}
	}

	s.logger.Info("all flags evaluated",
		slog.Int("count", len(flags)),
		slog.Int("defaulted", defaultedCount),
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

func (s *FeatureFlagService) HealthCheck(ctx context.Context) error {
	evalCtx := of.NewEvaluationContext("health-check", map[string]interface{}{})
	_, err := s.client.BooleanValue(ctx, "maintenance_mode", false, evalCtx)
	return err
}
