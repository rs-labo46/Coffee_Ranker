package middleware

import "github.com/labstack/echo/v4"

// UserまたはGuestSessionのどちらか片方だけが存在することを保証。
func ActorMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, ok := GetAppContext(c)
			if !ok {
				return unauthorized(c)
			}

			if !HasSingleActor(ctx) {
				return unauthorized(c)
			}

			return next(c)
		}
	}
}
