package services_test

import (
	"better-feature-flag/internal/config"
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestFlagRegistry_LoadValid(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	flags, err := registry.GetFlagsForApp("flutter")
	require.NoError(t, err)
	assert.Len(t, flags, 2)
	assert.Equal(t, "dark_mode", flags[0].Name)
	assert.Equal(t, models.FlagValueTypeBool, flags[0].Type)
}

func TestFlagRegistry_UnknownApp(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	_, err = registry.GetFlagsForApp("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown app")
}

func TestFlagRegistry_InvalidType(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/invalid_type.yaml"}}
	_, err := services.NewFlagRegistryService(cfg, testLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}

func TestFlagRegistry_FileNotFound(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "nonexistent.yaml"}}
	_, err := services.NewFlagRegistryService(cfg, testLogger())
	assert.Error(t, err)
}

func TestFlagRegistry_GetAnyFlags(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	flags, err := registry.GetAnyFlags()
	require.NoError(t, err)
	assert.NotEmpty(t, flags)
}

func TestFlagRegistry_MultipleApps(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{FlagsFile: "../../testdata/valid_flags.yaml"}}
	registry, err := services.NewFlagRegistryService(cfg, testLogger())
	require.NoError(t, err)

	flutter, err := registry.GetFlagsForApp("flutter")
	require.NoError(t, err)
	assert.Len(t, flutter, 2)

	backend, err := registry.GetFlagsForApp("backend")
	require.NoError(t, err)
	assert.Len(t, backend, 1)
	assert.Equal(t, "cache_enabled", backend[0].Name)
}
