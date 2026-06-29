package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// 安全なHTTP methodは通し、更新系methodではCSRF cookieとheaderの一致を要求することを検証する。
func TestCSRFMiddleware(t *testing.T) {
	t.Run("safe method passes without token", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/csrf")

		err := runMiddleware(t, c, okHandler, CSRFMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("post passes when cookie and header match", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/csrf")
		c.Request().AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "token-123"})
		c.Request().Header.Set(CSRFHeaderName, "token-123")

		err := runMiddleware(t, c, okHandler, CSRFMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("post rejects missing cookie", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/csrf")
		c.Request().Header.Set(CSRFHeaderName, "token-123")

		err := runMiddleware(t, c, okHandler, CSRFMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("post rejects mismatch", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/csrf")
		c.Request().AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "cookie-token"})
		c.Request().Header.Set(CSRFHeaderName, "header-token")

		err := runMiddleware(t, c, okHandler, CSRFMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})
}

// Request bodyが上限を超えた場合にhandler実行前に413を返すことを検証する。
func TestBodyLimitMiddleware(t *testing.T) {
	t.Run("content length over limit is rejected before handler", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("123456"))
		req.ContentLength = 6
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		called := false

		err := runMiddleware(t, c, func(c echo.Context) error {
			called = true
			return c.NoContent(http.StatusOK)
		}, BodyLimitMiddleware(5))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if called {
			t.Fatal("handler should not be called when Content-Length exceeds limit")
		}
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", rec.Code)
		}
	})

	t.Run("read over limit is converted to 413", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("123456"))
		req.ContentLength = -1
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := runMiddleware(t, c, func(c echo.Context) error {
			_, readErr := io.ReadAll(c.Request().Body)
			return readErr
		}, BodyLimitMiddleware(5))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413 body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("zero limit disables limit", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("123456"))
		req.ContentLength = 6
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := runMiddleware(t, c, okHandler, BodyLimitMiddleware(0))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})
}

// セキュリティ系HTTP headerがresponseへ設定されることを検証する。
func TestSecureHeadersMiddleware(t *testing.T) {
	t.Run("sets common security headers", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/beans")

		err := runMiddleware(t, c, okHandler, SecureHeadersMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}

		headers := rec.Header()
		if headers.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("X-Content-Type-Options = %q, want nosniff", headers.Get("X-Content-Type-Options"))
		}
		if headers.Get("X-Frame-Options") != "DENY" {
			t.Fatalf("X-Frame-Options = %q, want DENY", headers.Get("X-Frame-Options"))
		}
		if headers.Get("Referrer-Policy") == "" {
			t.Fatal("Referrer-Policy should be set")
		}
	})

	t.Run("auth path gets no-store", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/auth/login")

		err := runMiddleware(t, c, okHandler, SecureHeadersMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("Cache-Control = %q, want no-store", rec.Header().Get("Cache-Control"))
		}
		if rec.Header().Get("Pragma") != "no-cache" {
			t.Fatalf("Pragma = %q, want no-cache", rec.Header().Get("Pragma"))
		}
	})
}

// 許可originだけCORS headerを返し、未許可originを拒否することを検証する。
func TestCORSMiddleware(t *testing.T) {
	t.Run("allowed origin gets cors headers", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodOptions, "/cors")
		c.Request().Header.Set("Origin", "http://localhost:3000")

		err := runMiddleware(t, c, okHandler, CORSMiddleware(CORSConfig{
			AllowOrigins:     []string{"http://localhost:3000"},
			AllowCredentials: true,
		}))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
			t.Fatalf("Allow-Origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Fatalf("Allow-Credentials = %q", rec.Header().Get("Access-Control-Allow-Credentials"))
		}
	})

	t.Run("wildcard is ignored when credentials are allowed", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodOptions, "/cors")
		c.Request().Header.Set("Origin", "http://evil.example")

		err := runMiddleware(t, c, okHandler, CORSMiddleware(CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowCredentials: true,
		}))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatalf("unexpected Allow-Origin = %q", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("wildcard works without credentials", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/cors")
		c.Request().Header.Set("Origin", "http://any.example")

		err := runMiddleware(t, c, okHandler, CORSMiddleware(CORSConfig{AllowOrigins: []string{"*"}}))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://any.example" {
			t.Fatalf("Allow-Origin = %q, want request origin", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}

// Echoのbody size errorをBodyLimit用の判定関数で検出できることを検証する。
func TestIsBodyTooLargeError(t *testing.T) {
	if !isBodyTooLargeError(&http.MaxBytesError{Limit: 1}) {
		t.Fatal("http.MaxBytesError should be detected")
	}
	if isBodyTooLargeError(errors.New("other")) {
		t.Fatal("normal error should not be detected as MaxBytesError")
	}
}
