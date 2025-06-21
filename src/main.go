package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	gofeatureflag "github.com/open-feature/go-sdk-contrib/providers/go-feature-flag/pkg"
	of "github.com/open-feature/go-sdk/openfeature"
)

// Claims representa a estrutura dos dados do JWT
type Claims struct {
	Sub      string `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"preferred_username"`
	jwt.RegisteredClaims
}

// JWT_SECRET - Em produção, use uma variável de ambiente
const JWT_SECRET = "minha-chave-secreta-super-segura"

// extractJWTClaims extrai as claims do JWT SEM validar assinatura (apenas para testes)
func extractJWTClaims(authHeader string) (*Claims, error) {
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header vazio")
	}

	// Remove "Bearer " do início
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return nil, fmt.Errorf("formato do token inválido - deve começar com 'Bearer '")
	}

	// Parse SEM validar assinatura (apenas para testes)
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do token: %v", err)
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims, nil
	}

	return nil, fmt.Errorf("formato das claims inválido")
}

func main() {
	fmt.Printf("🚀 Iniciando Better Feature Flag...\n")

	provider, err := gofeatureflag.NewProvider(
		gofeatureflag.ProviderOptions{
			Endpoint:     "http://localhost:1031",
			DisableCache: true,
		})
	if err != nil {
		panic(fmt.Sprintf("Erro ao criar provider: %v", err))
	}

	of.SetProvider(provider)
	client := of.NewClient("better-feature-flag")

	e := echo.New()

	e.GET("/api/v1/flags", func(c echo.Context) error {
		ctx := c.Request().Context()

		// Extrai JWT do Header Authorization (OPCIONAL)
		userToken := c.Request().Header.Get("Authorization")

		userInfo := make(map[string]interface{})
		deviceInfo := make(map[string]interface{})
		appInfo := make(map[string]interface{})

		var targetingKey string

		// Extrai informações do dispositivo/app dos headers
		deviceInfo["platform"] = c.Request().Header.Get("sec-ch-ua-platform")
		deviceInfo["platform_version"] = c.Request().Header.Get("sec-ch-ua-platform-version")
		deviceInfo["architecture"] = c.Request().Header.Get("sec-ch-ua-arch")
		deviceInfo["model"] = c.Request().Header.Get("sec-ch-ua-model")
		deviceInfo["is_mobile"] = c.Request().Header.Get("sec-ch-ua-mobile") == "?1"
		deviceInfo["user_agent"] = c.Request().Header.Get("user-agent")
		deviceInfo["host"] = c.Request().Header.Get("host")

		appInfo["name_version"] = c.Request().Header.Get("sec-ch-ua")
		appInfo["full_version"] = c.Request().Header.Get("sec-ch-ua-full-version")

		if userToken != "" {
			// Se há token, tenta decodificar
			claims, err := extractJWTClaims(userToken)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": fmt.Sprintf("Token inválido: %v", err),
				})
			}

			// Usa informações do JWT
			userInfo["sub"] = claims.Sub
			userInfo["email"] = claims.Email
			userInfo["username"] = claims.Username

			// Usa sub (ID do usuário) como targetingKey, com fallbacks
			targetingKey = claims.Sub
			if targetingKey == "" {
				targetingKey = claims.Username
			}
			if targetingKey == "" {
				targetingKey = claims.Email
			}
			if targetingKey == "" {
				targetingKey = "anonymous"
			}
		} else {
			// Sem token = usuário anônimo
			userInfo["sub"] = "anonymous"
			userInfo["email"] = ""
			userInfo["username"] = "anonymous"
			targetingKey = "anonymous"
		}

		// Cria contexto de avaliação com todas as informações
		evalCtx := of.NewEvaluationContext(targetingKey, map[string]interface{}{
			"user":   userInfo,
			"device": deviceInfo,
			"app":    appInfo,
		})

		flags := make(map[string]interface{})

		// Avalia todos os flags com o contexto do usuário
		frontDarkMode, _ := client.BooleanValue(ctx, "front-dark-mode", false, evalCtx)
		forceUpdateEnabled, _ := client.BooleanValue(ctx, "force_update_enabled", false, evalCtx)
		maintenanceMessage, _ := client.StringValue(ctx, "maintenance_message", "Mensagem da manutenção", evalCtx)
		maintenanceTitle, _ := client.StringValue(ctx, "maintenance_title", "Título da Manutenção", evalCtx)
		minimumAppVersion, _ := client.StringValue(ctx, "minimum_app_version", "1.2.0", evalCtx)
		feedbackEnabled, _ := client.BooleanValue(ctx, "feedback_enabled", true, evalCtx)
		maintenanceMode, _ := client.BooleanValue(ctx, "maintenance_mode", false, evalCtx)
		updateTitle, _ := client.StringValue(ctx, "update_title", "Atualização Necessária", evalCtx)
		updateMessage, _ := client.StringValue(ctx, "update_message", "Atualize para continuar", evalCtx)

		flags["front-dark-mode"] = frontDarkMode
		flags["force_update_enabled"] = forceUpdateEnabled
		flags["maintenance_message"] = maintenanceMessage
		flags["maintenance_title"] = maintenanceTitle
		flags["minimum_app_version"] = minimumAppVersion
		flags["feedback_enabled"] = feedbackEnabled
		flags["maintenance_mode"] = maintenanceMode
		flags["update_title"] = updateTitle
		flags["update_message"] = updateMessage

		return c.JSON(http.StatusOK, map[string]interface{}{
			"context": map[string]interface{}{
				"targeting_key": targetingKey,
				"user":          userInfo,
				"device":        deviceInfo,
				"app":           appInfo,
			},
			"flags": flags,
		})
	})

	e.Start(":1324")
}
