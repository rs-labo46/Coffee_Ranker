package middleware

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type ILogger interface {
	Info(ctx context.Context, input LogInput)
	Error(ctx context.Context, input LogInput)
}

// Middlewareログに載せてよい安全な情報だけをまとめる。
// raw token、Cookie、password、CSRF token、Authorization headerは含めない。
type LogInput struct {
	RequestID      string
	Method         string
	Path           string
	Status         int
	LatencyMs      int64
	UserID         *uint64
	GuestSessionID *uint64
	ErrorCode      string
	Message        string
	Stack          string
}

// 標準logへ安全なアクセスログを出す。
// 本番で構造化ログへ差し替える場合も、Middleware側のinterfaceは変えない。
type StdLogger struct {
	logger *log.Logger
}

// 標準logを使うLoggerを作る。
// nilを渡した場合はlog.Default()を使い、呼び出し側の初期化漏れでpanicしないようにする。
func NewStdLogger(logger *log.Logger) *StdLogger {
	if logger == nil {
		logger = log.Default()
	}
	return &StdLogger{logger: logger}
}

// 正常系アクセスログを出力。
// 個人情報やtokenを含めず、request_id、method、path、status、latencyだけを中心に残す。
func (l *StdLogger) Info(ctx context.Context, input LogInput) {
	if l == nil || l.logger == nil {
		return
	}

	l.logger.Printf("level=info request_id=%s method=%s path=%s status=%d latency_ms=%d user_id=%s guest_session_id=%s",
		input.RequestID,
		input.Method,
		input.Path,
		input.Status,
		input.LatencyMs,
		logUint64Ptr(input.UserID),
		logUint64Ptr(input.GuestSessionID),
	)
}

// エラー系ログを出力。
// エラー内容は運用確認用に残が、tokenやCookie値は渡さない前提。
func (l *StdLogger) Error(ctx context.Context, input LogInput) {
	if l == nil || l.logger == nil {
		return
	}

	if input.Stack != "" {
		l.logger.Printf("level=error request_id=%s method=%s path=%s status=%d error_code=%s message=%s user_id=%s guest_session_id=%s stack=%s",
			input.RequestID,
			input.Method,
			input.Path,
			input.Status,
			input.ErrorCode,
			input.Message,
			logUint64Ptr(input.UserID),
			logUint64Ptr(input.GuestSessionID),
			input.Stack,
		)
		return
	}

	l.logger.Printf("level=error request_id=%s method=%s path=%s status=%d error_code=%s message=%s user_id=%s guest_session_id=%s",
		input.RequestID,
		input.Method,
		input.Path,
		input.Status,
		input.ErrorCode,
		input.Message,
		logUint64Ptr(input.UserID),
		logUint64Ptr(input.GuestSessionID),
	)
}

// ログ出力用にuint64 pointerを文字列へ変換。
// nilの場合は未設定を示すハイフンにして、0と未設定を区別。
func logUint64Ptr(value *uint64) string {
	if value == nil {
		return "-"
	}
	return strconv.FormatUint(*value, 10)
}

// request/responseの概要を記録。
// 監査ログの代替ではなく、API運用時のアクセス確認用ログとして使う。
func AccessLogMiddleware(logger ILogger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)

			status := c.Response().Status
			if status == 0 {
				status = 200
			}

			ctx, _ := GetAppContext(c)
			input := LogInput{
				RequestID: requestIDFromApp(ctx),
				Method:    c.Request().Method,
				Path:      c.Path(),
				Status:    status,
				LatencyMs: latency.Milliseconds(),
			}
			if ctx != nil {
				input.UserID = ctx.AuthUserID
				input.GuestSessionID = ctx.GuestSessionID
			}

			if err != nil {
				input.ErrorCode = "handler_error"
				input.Message = err.Error()
				if logger != nil {
					logger.Error(c.Request().Context(), input)
				}
				return err
			}

			if logger != nil {
				logger.Info(c.Request().Context(), input)
			}
			return nil
		}
	}
}
