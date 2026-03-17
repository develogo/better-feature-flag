package handlers_test

import (
	"better-feature-flag/internal/handlers"
	"better-feature-flag/internal/models"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockEvaluator struct {
	result map[string]interface{}
	err    error
}

func (m *mockEvaluator) EvaluateFlags(ctx context.Context, flags []models.FlagDefinition, clientCtx *models.ClientContext) (map[string]interface{}, error) {
	return m.result, m.err
}

func (m *mockEvaluator) HealthCheck(ctx context.Context, flags []models.FlagDefinition) error {
	return m.err
}

type mockRegistry struct {
	apps map[string][]models.FlagDefinition
}

func (m *mockRegistry) GetFlagsForApp(appName string) ([]models.FlagDefinition, error) {
	flags, ok := m.apps[appName]
	if !ok {
		return nil, fmt.Errorf("unknown app: %s", appName)
	}
	return flags, nil
}

func (m *mockRegistry) GetAnyFlags() ([]models.FlagDefinition, error) {
	for _, flags := range m.apps {
		return flags, nil
	}
	return nil, fmt.Errorf("no flags")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestGetFlags_Success(t *testing.T) {
	eval := &mockEvaluator{result: map[string]interface{}{"dark_mode": true}}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "dark_mode", Type: models.FlagValueTypeBool, Default: false}},
	}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags?app=flutter", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{DeviceID: "dev-1"})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "dark_mode")
}

func TestGetFlags_DefaultApp(t *testing.T) {
	eval := &mockEvaluator{result: map[string]interface{}{"dark_mode": false}}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "dark_mode", Type: models.FlagValueTypeBool, Default: false}},
	}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetFlags_UnknownApp(t *testing.T) {
	eval := &mockEvaluator{}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags?app=unknown", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetFlags_EvaluationError(t *testing.T) {
	eval := &mockEvaluator{err: fmt.Errorf("goff down")}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "dark_mode", Type: models.FlagValueTypeBool, Default: false}},
	}}
	h := handlers.NewFlagsHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/flags?app=flutter", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("client_context", &models.ClientContext{})

	err := h.GetFlags(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
