package middleware

import (
	"better-feature-flag/src/internal/models"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type Claims struct {
	Sub      string `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"preferred_username"`
	jwt.RegisteredClaims
}

type AuthMiddleware struct {
	jwtSecret string
}

func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
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
				if claims, err := m.extractAndValidateJWT(authHeader); err == nil {
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

func (m *AuthMiddleware) extractAndValidateJWT(authHeader string) (*Claims, error) {
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

func GetClientContext(c echo.Context) *models.ClientContext {
	if ctx, ok := c.Get("client_context").(*models.ClientContext); ok {
		return ctx
	}
	return &models.ClientContext{}
}
