package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// Echo middlewareの単体テストで使うEcho、Context、ResponseRecorderを作成する。
func newMiddlewareTestContext(method string, target string) (*echo.Echo, echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath(target)
	return e, c, rec
}

// 指定したmiddlewareをhandlerへ適用し、テスト用Contextで実行する。
func runMiddleware(t *testing.T, c echo.Context, handler echo.HandlerFunc, middlewares ...echo.MiddlewareFunc) error {
	t.Helper()

	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	return wrapped(c)
}

// middleware通過後の正常終了handlerとしてHTTP 200を返す。
func okHandler(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}
