package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/segmentio/ksuid"
)

func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = ksuid.New().String()
			}
			c.Response().Header().Set("X-Request-ID", requestID)
			c.Set("request_id", requestID)
			return next(c)
		}
	}
}
