package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func CORS() echo.MiddlewareFunc {
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Em produção, especificar domínios permitidos
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"X-App-Version",
			"X-Platform",
			"X-Device-ID",
		},
		ExposeHeaders: []string{echo.HeaderContentLength},
		MaxAge:        3600,
	})
}
