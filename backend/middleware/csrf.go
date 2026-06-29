package middleware

import (
	"crypto/hmac"
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	CSRFHeaderName = "X-CSRF-Token"
	CSRFCookieName = "csrf_token"
)

// Double Submit Cookie方式でCSRF tokenを確認。
// Cookie値とHeader値が一致するかだけを確認し、RefreshToken rotationなどの業務処理は行わない。
func CSRFMiddleware() echo.MiddlewareFunc {
	return CSRFMiddlewareWithNames(CSRFCookieName, CSRFHeaderName)
}

// CSRF用のCookie名とHeader名を指定して検証。
// Refresh/logout/event/modal操作など、Cookieが自動送信される更新系APIに適用。
func CSRFMiddlewareWithNames(cookieName string, headerName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(c)
			}

			cookie, err := c.Cookie(cookieName)
			if err != nil || cookie == nil || cookie.Value == "" {
				return forbidden(c, "csrf_required")
			}

			headerValue := c.Request().Header.Get(headerName)
			if headerValue == "" {
				return forbidden(c, "csrf_required")
			}

			if !sameToken(cookie.Value, headerValue) {
				return forbidden(c, "csrf_mismatch")
			}

			return next(c)
		}
	}
}

// CSRF tokenを比較。
// 通常の文字列比較ではなくhmac.Equalを使い、比較時間差による推測を避ける。
func sameToken(cookieValue string, headerValue string) bool {
	return hmac.Equal([]byte(cookieValue), []byte(headerValue))
}
