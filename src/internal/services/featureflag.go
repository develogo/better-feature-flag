package services

import (
	"better-feature-flag/src/internal/models"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
	"gopkg.in/yaml.v3"
)

type FeatureFlagService struct {
	client      *of.Client
	logger      *slog.Logger
	flagsDir    string
	cachedFlags map[string]FlagMetadata
}

type FlagMetadata struct {
	DefaultValue interface{}
	Type         string // "bool", "string", "int", "float"
}

type YAMLFlag struct {
	Variations  map[string]interface{} `yaml:"variations"`
	DefaultRule map[string]interface{} `yaml:"defaultRule"`
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

	service := &FeatureFlagService{
		client:      client,
		logger:      logger,
		flagsDir:    "./flags",
		cachedFlags: make(map[string]FlagMetadata),
	}

	// Carrega metadados das flags dos arquivos YAML
	if err := service.loadFlagsMetadata(); err != nil {
		logger.Warn("failed to load flags metadata", slog.String("error", err.Error()))
	}

	return service, nil
}

func (s *FeatureFlagService) loadFlagsMetadata() error {
	files, err := filepath.Glob(filepath.Join(s.flagsDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to list yaml files: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			s.logger.Warn("failed to read yaml file", slog.String("file", file), slog.String("error", err.Error()))
			continue
		}

		var flags map[string]YAMLFlag
		if err := yaml.Unmarshal(data, &flags); err != nil {
			s.logger.Warn("failed to parse yaml file", slog.String("file", file), slog.String("error", err.Error()))
			continue
		}

		for flagKey, flagDef := range flags {
			// Determina o tipo e valor padrão da flag
			if len(flagDef.Variations) > 0 {
				// Pega o primeiro valor para determinar o tipo
				for _, value := range flagDef.Variations {
					metadata := FlagMetadata{
						DefaultValue: value,
						Type:         inferType(value),
					}
					s.cachedFlags[flagKey] = metadata
					break
				}
			}
		}
	}

	s.logger.Info("loaded flags metadata", slog.Int("count", len(s.cachedFlags)))
	return nil
}

func inferType(value interface{}) string {
	switch value.(type) {
	case bool:
		return "bool"
	case int, int64, float64:
		return "number"
	case string:
		return "string"
	default:
		return "unknown"
	}
}

func (s *FeatureFlagService) EvaluateFlags(ctx context.Context, clientCtx *models.ClientContext) (map[string]interface{}, error) {
	evalCtx := s.buildEvaluationContext(clientCtx)
	flags := make(map[string]interface{})

	// Avalia todas as flags descobertas dos arquivos YAML
	for flagKey, metadata := range s.cachedFlags {
		var value interface{}
		var err error

		switch metadata.Type {
		case "bool":
			defaultBool, _ := metadata.DefaultValue.(bool)
			value, err = s.client.BooleanValue(ctx, flagKey, defaultBool, evalCtx)
		case "string":
			defaultStr, _ := metadata.DefaultValue.(string)
			value, err = s.client.StringValue(ctx, flagKey, defaultStr, evalCtx)
		case "number":
			switch v := metadata.DefaultValue.(type) {
			case int:
				value, err = s.client.IntValue(ctx, flagKey, int64(v), evalCtx)
			case int64:
				value, err = s.client.IntValue(ctx, flagKey, v, evalCtx)
			case float64:
				value, err = s.client.FloatValue(ctx, flagKey, v, evalCtx)
			default:
				value, err = s.client.FloatValue(ctx, flagKey, 0, evalCtx)
			}
		default:
			// Tenta como object/JSON
			value, err = s.client.ObjectValue(ctx, flagKey, map[string]interface{}{}, evalCtx)
		}

		if err != nil {
			s.logger.Warn("failed to evaluate flag",
				slog.String("flag", flagKey),
				slog.String("error", err.Error()),
			)
			value = metadata.DefaultValue
		}

		flags[flagKey] = value

		s.logger.Debug("flag evaluated",
			slog.String("flag", flagKey),
			slog.Any("value", value),
			slog.String("targeting_key", clientCtx.GetTargetingKey()),
		)
	}

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
