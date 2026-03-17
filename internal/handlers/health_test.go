package handlers_test

import (
	"better-feature-flag/internal/handlers"
	"better-feature-flag/internal/models"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth_AlwaysOK(t *testing.T) {
	eval := &mockEvaluator{}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "test", Type: models.FlagValueTypeBool}},
	}}
	h := handlers.NewHealthHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Health(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ok")
}

func TestReady_Success(t *testing.T) {
	eval := &mockEvaluator{}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "test", Type: models.FlagValueTypeBool}},
	}}
	h := handlers.NewHealthHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Ready(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ready")
}

func TestReady_GOFFUnavailable(t *testing.T) {
	eval := &mockEvaluator{err: fmt.Errorf("connection refused")}
	reg := &mockRegistry{apps: map[string][]models.FlagDefinition{
		"flutter": {{Name: "test", Type: models.FlagValueTypeBool}},
	}}
	h := handlers.NewHealthHandler(eval, reg, testLogger())

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Ready(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
