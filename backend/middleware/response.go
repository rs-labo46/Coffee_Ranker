package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// 内部エラー詳細やtoken情報を返さず、クライアントには安全なerror codeだけを返す。
type ErrorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// Middleware内で検出した失敗をHTTPレスポンスへ変換。
// raw token、Cookie値、DB詳細などの機密情報はレスポンスに含めない。
func writeError(c echo.Context, status int, code string) error {
	requestID := ""
	if ctx, ok := GetAppContext(c); ok {
		requestID = ctx.RequestID
	}

	return c.JSON(status, ErrorResponse{
		Error:     code,
		RequestID: requestID,
	})
}

// 認証が必要なAPIで、AccessTokenがない・不正・期限切れの場合に返す。
func unauthorized(c echo.Context) error {
	return writeError(c, http.StatusUnauthorized, "unauthorized")
}

// 認証はできているが、admin権限やCSRF条件を満たさない場合に返す。
func forbidden(c echo.Context, code string) error {
	return writeError(c, http.StatusForbidden, code)
}

// Middleware設定不備や外部依存障害など、クライアント入力では復旧できない場合に返す。
func internalServerError(c echo.Context) error {
	return writeError(c, http.StatusInternalServerError, "internal_server_error")
}

// RateLimit超過時に429を返す。
// Retry-AfterなどのヘッダーはRateLimitMiddleware側で設定。
func rateLimited(c echo.Context) error {
	return writeError(c, http.StatusTooManyRequests, "rate_limited")
}

// Bodyが許可サイズを超えた場合に413を返す。
func bodyTooLarge(c echo.Context) error {
	return writeError(c, http.StatusRequestEntityTooLarge, "body_too_large")
}
