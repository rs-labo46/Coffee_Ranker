package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/labstack/echo/v4"
)

const requestIDHeader = "X-Request-ID"

// request_idを生成または引き継ぎ、AppContextとResponse Headerに保存。
// 以降のログ、エラーレスポンス、監査メタ情報で同じIDを使えるようにする。
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get(requestIDHeader)
			if requestID == "" || !isSafeRequestID(requestID) {
				requestID = newRequestID()
			}

			SetRequestID(c, requestID)
			c.Response().Header().Set(requestIDHeader, requestID)

			return next(c)
		}
	}
}

// 外部から渡されたrequest_idを使ってよい形式か確認。
// 制御文字や長すぎる値をログに混ぜないため、英数字・ハイフン・アンダースコアだけ許可。
func isSafeRequestID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}

	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}

	return true
}

// 予測されにくいrequest_idを生成。
// 生成に失敗した場合でも空にせず、最小限の固定prefix付きIDを返す。
func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "req_fallback"
	}

	return "req_" + hex.EncodeToString(buf)
}
