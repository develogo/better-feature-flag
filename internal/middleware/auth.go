package middleware

import (
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"strings"

	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	tokenValidator services.TokenValidator
}

func NewAuthMiddleware(tokenValidator services.TokenValidator) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: tokenValidator,
	}
}

// OptionalJWT extrai o JWT se presente, mas nao falha se ausente
func (m *AuthMiddleware) OptionalJWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientCtx := &models.ClientContext{}

			// Tenta extrair info do JWT se presente
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				if claims, err := m.tokenValidator.ValidateToken(c.Request().Context(), tokenString); err == nil {
					clientCtx.UserID = claims.Sub
					clientCtx.Email = claims.Email
					clientCtx.Username = claims.Username
				}
			}

			// Extrai headers do Flutter
			clientCtx.DeviceID = c.Request().Header.Get("Device-ID")
			clientCtx.Platform = c.Request().Header.Get("Platform")
			clientCtx.PlatformVersion = c.Request().Header.Get("Platform-Version")
			clientCtx.DeviceModel = c.Request().Header.Get("Device-Model")
			clientCtx.Architecture = c.Request().Header.Get("Device-Architecture")
			clientCtx.DeviceBrand = c.Request().Header.Get("Device-Brand")
			clientCtx.Mobile = c.Request().Header.Get("Mobile")
			clientCtx.Device = c.Request().Header.Get("Device")
			clientCtx.AppName = c.Request().Header.Get("App-Name")
			clientCtx.AppVersion = c.Request().Header.Get("App-Version")
			clientCtx.PackageName = c.Request().Header.Get("Package-Name")
			clientCtx.BuildNumber = c.Request().Header.Get("Build-Number")

			// Armazena no contexto do Echo
			c.Set("client_context", clientCtx)

			return next(c)
		}
	}
}

func GetClientContext(c echo.Context) *models.ClientContext {
	if ctx, ok := c.Get("client_context").(*models.ClientContext); ok {
		return ctx
	}
	return &models.ClientContext{}
}
