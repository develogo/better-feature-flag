package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

func Logger(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			req := c.Request()
			res := c.Response()

			requestID, _ := c.Get("request_id").(string)

			logger.Info("request",
				slog.String("request_id", requestID),
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.Duration("latency", time.Since(start)),
				slog.String("ip", c.RealIP()),
				slog.String("user_agent", req.UserAgent()),
			)

			return err
		}
	}
}
