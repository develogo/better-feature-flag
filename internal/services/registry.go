package services

import (
	"better-feature-flag/internal/config"
	"better-feature-flag/internal/models"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type FlagRegistryService struct {
	apps   map[string][]models.FlagDefinition
	logger *slog.Logger
}

type flagsFileSchema struct {
	Apps map[string]struct {
		Flags []models.FlagDefinition `yaml:"flags"`
	} `yaml:"apps"`
}

func NewFlagRegistryService(cfg *config.Config, logger *slog.Logger) (*FlagRegistryService, error) {
	filePath := cfg.App.FlagsFile
	if filePath == "" {
		filePath = "config/flags.yaml"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read flags file %s: %w", filePath, err)
	}

	var schema flagsFileSchema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse flags file: %w", err)
	}

	apps := make(map[string][]models.FlagDefinition, len(schema.Apps))
	for appName, appDef := range schema.Apps {
		for i, flag := range appDef.Flags {
			if flag.Name == "" {
				return nil, fmt.Errorf("flag at index %d in app %q has no name", i, appName)
			}
			if !isValidFlagType(flag.Type) {
				return nil, fmt.Errorf("flag %q in app %q has invalid type %q", flag.Name, appName, flag.Type)
			}
		}
		apps[appName] = appDef.Flags
	}

	logger.Info("flag registry loaded", slog.Int("apps", len(apps)))
	for appName, flags := range apps {
		logger.Info("registered app flags", slog.String("app", appName), slog.Int("flags", len(flags)))
	}

	return &FlagRegistryService{apps: apps, logger: logger}, nil
}

func (r *FlagRegistryService) GetFlagsForApp(appName string) ([]models.FlagDefinition, error) {
	flags, ok := r.apps[appName]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", appName)
	}
	return flags, nil
}

func (r *FlagRegistryService) GetAnyFlags() ([]models.FlagDefinition, error) {
	for _, flags := range r.apps {
		if len(flags) > 0 {
			return flags, nil
		}
	}
	return nil, fmt.Errorf("no flags registered")
}

func isValidFlagType(t models.FlagValueType) bool {
	switch t {
	case models.FlagValueTypeBool, models.FlagValueTypeString,
		models.FlagValueTypeInt, models.FlagValueTypeFloat:
		return true
	}
	return false
}
