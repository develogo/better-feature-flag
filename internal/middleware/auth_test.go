package middleware_test

import (
	"better-feature-flag/internal/middleware"
	"better-feature-flag/internal/models"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenValidator struct {
	claims *models.TokenClaims
	err    error
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, token string) (*models.TokenClaims, error) {
	return m.claims, m.err
}

func TestOptionalJWT_WithValidToken(t *testing.T) {
	validator := &mockTokenValidator{
		claims: &models.TokenClaims{Sub: "user-123", Email: "test@test.com", Username: "testuser"},
	}
	auth := middleware.NewAuthMiddleware(validator)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	req.Header.Set("Device-ID", "device-abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedCtx *models.ClientContext
	handler := auth.OptionalJWT()(func(c echo.Context) error {
		capturedCtx = middleware.GetClientContext(c)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, "user-123", capturedCtx.UserID)
	assert.Equal(t, "test@test.com", capturedCtx.Email)
	assert.Equal(t, "device-abc", capturedCtx.DeviceID)
}

func TestOptionalJWT_WithoutToken(t *testing.T) {
	validator := &mockTokenValidator{}
	auth := middleware.NewAuthMiddleware(validator)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Device-ID", "device-abc")
	req.Header.Set("Platform", "android")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedCtx *models.ClientContext
	handler := auth.OptionalJWT()(func(c echo.Context) error {
		capturedCtx = middleware.GetClientContext(c)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Empty(t, capturedCtx.UserID)
	assert.Equal(t, "device-abc", capturedCtx.DeviceID)
	assert.Equal(t, "android", capturedCtx.Platform)
}

func TestOptionalJWT_WithInvalidToken(t *testing.T) {
	validator := &mockTokenValidator{err: fmt.Errorf("token expired")}
	auth := middleware.NewAuthMiddleware(validator)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	req.Header.Set("Device-ID", "device-abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedCtx *models.ClientContext
	handler := auth.OptionalJWT()(func(c echo.Context) error {
		capturedCtx = middleware.GetClientContext(c)
		return c.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Empty(t, capturedCtx.UserID)
	assert.Equal(t, "device-abc", capturedCtx.DeviceID)
}

func TestGetClientContext_NoContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	ctx := middleware.GetClientContext(c)
	assert.NotNil(t, ctx)
	assert.Empty(t, ctx.UserID)
}
