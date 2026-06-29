package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// 基本的なセキュリティヘッダーを全レスポンスへ付与。
// HTML sanitizeではなく、ブラウザ側の誤解釈や不要機能を抑えるためのHTTP境界対策。
func SecureHeadersMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// 認証・token系レスポンスがブラウザや中間キャッシュに保存されないように。
			if isAuthPath(c.Request().URL.Path) {
				h.Set("Cache-Control", "no-store")
				// 古いHTTPキャッシュ互換のためPragmaも付ける。
				h.Set("Pragma", "no-cache")
			}

			return next(c)
		}
	}
}

// 認証系レスポンスかどうかをpathだけで判定。
// tokenやCookieを扱うAPIのキャッシュを防ぐために使う。
func isAuthPath(path string) bool {
	return len(path) >= len("/auth") && path[:len("/auth")] == "/auth"
}

// CORSの許可条件をまとめる。
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
}

// FrontendからのCORS通信を許可Originに限定。
// Cookieを使う場合に任意Originへ許可しないよう、Origin一致時だけAccess-Control-Allow-Originを返す。
func CORSMiddleware(config CORSConfig) echo.MiddlewareFunc {
	methods := config.AllowMethods
	if len(methods) == 0 {
		methods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions}
	}

	headers := config.AllowHeaders
	if len(headers) == 0 {
		headers = []string{"Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID"}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			if origin == "" {
				return next(c)
			}

			if !isAllowedOrigin(origin, config.AllowOrigins, config.AllowCredentials) {
				if c.Request().Method == http.MethodOptions {
					return c.NoContent(http.StatusNoContent)
				}
				return next(c)
			}

			h := c.Response().Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Allow-Methods", joinHeaderValues(methods))
			h.Set("Access-Control-Allow-Headers", joinHeaderValues(headers))
			if config.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}

			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}

			return next(c)
		}
	}
}

// Originが許可リストに含まれるか確認。
// AllowCredentials=trueの場合は*を許可せず、Cookie付き通信の開放事故を防ぐ。
func isAllowedOrigin(origin string, allowed []string, allowCredentials bool) bool {
	for _, item := range allowed {
		if item == "*" && !allowCredentials {
			return true
		}
		if item == origin {
			return true
		}
	}

	return false
}

// 複数Header値をカンマ区切り文字列へ変換。
// CORSのAllow-Methods / Allow-Headersのレスポンス生成だけに使う。
func joinHeaderValues(values []string) string {
	if len(values) == 0 {
		return ""
	}

	joined := values[0]
	for i := 1; i < len(values); i++ {
		joined += ", " + values[i]
	}
	return joined
}
