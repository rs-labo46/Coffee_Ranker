package middleware

import "github.com/labstack/echo/v4"

// client IPをhash化してAppContextへ保存。
// RateLimitでIP単位の制限を行うために使い、生IPはContextやログに残さない。
func ClientIPHashMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			SetClientIPHash(c, c.RealIP(), secret)
			return next(c)
		}
	}
}
