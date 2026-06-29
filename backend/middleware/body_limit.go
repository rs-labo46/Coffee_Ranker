package middleware

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// 大きすぎるRequest BodyをController到達前に拒否。
// JSON Bind前に制限し、メモリ消費やDoSリスクを下げる。
func BodyLimitMiddleware(maxBytes int64) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if maxBytes <= 0 {
				return next(c)
			}

			if c.Request().ContentLength > maxBytes {
				return bodyTooLarge(c)
			}

			// Content-Lengthがないchunked requestでも読み取り時に上限を超えたら止める。
			c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxBytes)

			err := next(c)
			if isBodyTooLargeError(err) {
				return bodyTooLarge(c)
			}

			return err
		}
	}
}

// MaxBytesReader由来のエラーか確認。
// EchoのBind中に発生したbody過大を413へ寄せる
func isBodyTooLargeError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}
