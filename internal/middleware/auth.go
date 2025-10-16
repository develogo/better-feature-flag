package middleware

import (
	"better-feature-flag/internal/models"
	"better-feature-flag/internal/services"
	"strings"

	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	keycloakService *services.KeycloakService
}

func NewAuthMiddleware(keycloakService *services.KeycloakService) *AuthMiddleware {
	return &AuthMiddleware{
		keycloakService: keycloakService,
	}
}

// OptionalJWT extrai o JWT se presente, mas não falha se ausente
func (m *AuthMiddleware) OptionalJWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientCtx := &models.ClientContext{}

			// Tenta extrair info do JWT se presente
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader != "" {
				tokenString := strings.TrimPrefix(authHeader, "Bearer ")
				if claims, err := m.keycloakService.ValidateToken(c.Request().Context(), tokenString); err == nil {
					clientCtx.UserID = claims.Sub
					clientCtx.Email = claims.Email
					clientCtx.Username = claims.Username
				}
			}

			// Extrai headers do Flutter
			clientCtx.AppVersion = c.Request().Header.Get("X-App-Version")
			clientCtx.Platform = c.Request().Header.Get("X-Platform")
			clientCtx.DeviceID = c.Request().Header.Get("X-Device-ID")

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
