package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

// panicを捕捉して500。
// panic内容をクライアントへ返さず、必要な情報はサーバーログ側へ残す。
func RecoverMiddleware(logger ILogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logMiddlewareError(c, logger, "panic", fmt.Sprint(recovered), debug.Stack())
					err = writeError(c, http.StatusInternalServerError, "internal_server_error")
				}
			}()

			return next(c)
		}
	}
}

// panicやMiddleware内エラーをLoggerへ渡す。
// token、Cookie、passwordなどの生値は渡さず、request_idやpathなど安全な情報だけを使う。
func logMiddlewareError(c echo.Context, logger ILogger, code string, message string, stack []byte) {
	if logger == nil {
		return
	}

	ctx, _ := GetAppContext(c)
	input := LogInput{
		RequestID: requestIDFromApp(ctx),
		Method:    c.Request().Method,
		Path:      c.Path(),
		Status:    http.StatusInternalServerError,
		ErrorCode: code,
		Message:   message,
		Stack:     string(stack),
	}

	if ctx != nil {
		input.UserID = ctx.AuthUserID
		input.GuestSessionID = ctx.GuestSessionID
	}

	logger.Error(c.Request().Context(), input)
}

// AppContextが存在する場合だけrequest_idを取り出す。
// Middleware適用順が崩れていてもpanicしないよう、空文字を許可。
func requestIDFromApp(ctx *AppContext) string {
	if ctx == nil {
		return ""
	}
	return ctx.RequestID
}
