package middleware

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// Logger middlewareの出力内容をメモリ上に保存するテストfake。
type fakeMiddlewareLogger struct {
	infos  []LogInput
	errors []LogInput
}

// Info log入力を保存する。
func (f *fakeMiddlewareLogger) Info(ctx context.Context, input LogInput) {
	f.infos = append(f.infos, input)
}

// Error log入力を保存する。
func (f *fakeMiddlewareLogger) Error(ctx context.Context, input LogInput) {
	f.errors = append(f.errors, input)
}

// 安全なRequest IDは再利用し、不正なRequest IDは生成し直すことを検証する。
func TestRequestIDMiddleware_UsesSafeHeaderAndGeneratesUnsafe(t *testing.T) {
	t.Run("safe request id is reused", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/request-id")
		c.Request().Header.Set(requestIDHeader, "req_12345678")

		err := runMiddleware(t, c, okHandler, RequestIDMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Header().Get(requestIDHeader) != "req_12345678" {
			t.Fatalf("response request id = %q", rec.Header().Get(requestIDHeader))
		}
		ctx, ok := GetAppContext(c)
		if !ok || ctx.RequestID != "req_12345678" {
			t.Fatalf("context request id = %#v", ctx)
		}
	})

	t.Run("unsafe request id is replaced", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/request-id")
		c.Request().Header.Set(requestIDHeader, "bad id with space")

		err := runMiddleware(t, c, okHandler, RequestIDMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		got := rec.Header().Get(requestIDHeader)
		if got == "" {
			t.Fatal("request id should be generated")
		}
		if got == "bad id with space" {
			t.Fatal("unsafe request id should not be reused")
		}
		if !strings.HasPrefix(got, "req_") {
			t.Fatalf("generated request id = %q, want req_ prefix", got)
		}
	})
}

// Client IPをhash化してAppContextへ保存することを検証する。
func TestClientIPHashMiddleware(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/ip")
	c.Request().RemoteAddr = "203.0.113.20:12345"

	err := runMiddleware(t, c, okHandler, ClientIPHashMiddleware("secret"))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	ctx, ok := GetAppContext(c)
	if !ok || ctx.ClientIPHash == "" {
		t.Fatalf("client ip hash was not stored: %#v", ctx)
	}
	if strings.Contains(ctx.ClientIPHash, "203.0.113.20") {
		t.Fatalf("client ip hash should not contain raw ip: %s", ctx.ClientIPHash)
	}
}

// handler panicをRecoverMiddlewareが捕捉し、500とerror logを返すことを検証する。
func TestRecoverMiddleware_CatchesPanicAndLogsError(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/panic")
	SetRequestID(c, "req_12345678")
	logger := &fakeMiddlewareLogger{}

	err := runMiddleware(t, c, func(c echo.Context) error {
		panic("boom")
	}, RecoverMiddleware(logger))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 body=%s", rec.Code, rec.Body.String())
	}
	if len(logger.errors) != 1 {
		t.Fatalf("logger errors len = %d, want 1", len(logger.errors))
	}
	input := logger.errors[0]
	if input.RequestID != "req_12345678" {
		t.Fatalf("logged request id = %q, want req_12345678", input.RequestID)
	}
	if input.ErrorCode != "panic" || input.Message != "boom" {
		t.Fatalf("logged panic = %s %s, want panic boom", input.ErrorCode, input.Message)
	}
	if input.Stack == "" {
		t.Fatal("panic stack should be logged")
	}
}

// 正常応答とエラー応答のaccess logが記録されることを検証する。
func TestAccessLogMiddleware_LogsInfoAndErrors(t *testing.T) {
	t.Run("info log on success", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/access")
		SetRequestID(c, "req_87654321")
		logger := &fakeMiddlewareLogger{}

		err := runMiddleware(t, c, okHandler, AccessLogMiddleware(logger))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if len(logger.infos) != 1 {
			t.Fatalf("info logs len = %d, want 1", len(logger.infos))
		}
		if logger.infos[0].RequestID != "req_87654321" || logger.infos[0].Status != http.StatusOK {
			t.Fatalf("info log = %#v", logger.infos[0])
		}
	})

	t.Run("error log on handler error", func(t *testing.T) {
		_, c, _ := newMiddlewareTestContext(http.MethodGet, "/access-error")
		SetRequestID(c, "req_99999999")
		logger := &fakeMiddlewareLogger{}
		handlerErr := errors.New("handler failed")

		err := runMiddleware(t, c, func(c echo.Context) error {
			return handlerErr
		}, AccessLogMiddleware(logger))
		if !errors.Is(err, handlerErr) {
			t.Fatalf("error = %v, want handler failed", err)
		}
		if len(logger.errors) != 1 {
			t.Fatalf("error logs len = %d, want 1", len(logger.errors))
		}
		if logger.errors[0].ErrorCode != "handler_error" || logger.errors[0].Message != "handler failed" {
			t.Fatalf("error log = %#v", logger.errors[0])
		}
	})
}

// StdLoggerがtokenやcookieを含めず、許可された項目とstackを出力することを検証する。
func TestStdLogger_OutputsSafeFieldsAndStack(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewStdLogger(log.New(buf, "", 0))
	userID := uint64(7)
	guestID := uint64(8)

	logger.Info(context.Background(), LogInput{
		RequestID:      "req_12345678",
		Method:         http.MethodGet,
		Path:           "/items",
		Status:         http.StatusOK,
		LatencyMs:      12,
		UserID:         &userID,
		GuestSessionID: &guestID,
	})

	logger.Error(context.Background(), LogInput{
		RequestID: "req_12345678",
		Method:    http.MethodPost,
		Path:      "/panic",
		Status:    http.StatusInternalServerError,
		ErrorCode: "panic",
		Message:   "boom",
		Stack:     "stack trace",
	})

	out := buf.String()
	if !strings.Contains(out, "level=info") || !strings.Contains(out, "level=error") {
		t.Fatalf("log output missing levels: %s", out)
	}
	if !strings.Contains(out, "stack=stack trace") {
		t.Fatalf("log output should include stack: %s", out)
	}
	if strings.Contains(out, "Authorization") || strings.Contains(out, "Cookie") {
		t.Fatalf("log output should not contain sensitive headers: %s", out)
	}
}
