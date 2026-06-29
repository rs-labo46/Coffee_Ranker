package middleware

import (
	"context"
	"errors"
	"strings"

	"coffee-ranker/entity"

	"github.com/labstack/echo/v4"
)

// Middlewareは署名検証、token_version確認、User状態確認の中身を持たず、Usecase側へ委譲。
type AccessTokenValidator interface {
	ValidateAccessToken(ctx context.Context, rawToken string) (userID uint64, role entity.UserRole, tokenVersion int, err error)
}

// Authorization HeaderのBearer tokenを検証し、認証済みUser情報をAppContextへ保存。
// RefreshTokenやCookieは扱わず、AccessToken認証だけを担当。
func AuthMiddleware(auth AccessTokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if auth == nil {
				return internalServerError(c)
			}

			rawToken, ok := bearerToken(c.Request().Header.Get("Authorization"))
			if !ok {
				return unauthorized(c)
			}

			userID, role, tokenVersion, err := auth.ValidateAccessToken(c.Request().Context(), rawToken)
			if err != nil {
				if errors.Is(err, entity.ErrUserSuspended) || errors.Is(err, entity.ErrUserDeleted) || errors.Is(err, entity.ErrForbidden) {
					return forbidden(c, "forbidden")
				}
				return unauthorized(c)
			}

			if userID == 0 {
				return unauthorized(c)
			}

			SetAuthUser(c, userID, role, tokenVersion)
			return next(c)
		}
	}
}

// Authorization Headerがある場合だけAccessTokenを検証。
// HeaderがなければGuest候補として通し、不正tokenがある場合は401。
func OptionalAuthMiddleware(auth AccessTokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return next(c)
			}

			if auth == nil {
				return internalServerError(c)
			}

			rawToken, ok := bearerToken(header)
			if !ok {
				return unauthorized(c)
			}

			userID, role, tokenVersion, err := auth.ValidateAccessToken(c.Request().Context(), rawToken)
			if err != nil {
				if errors.Is(err, entity.ErrUserSuspended) || errors.Is(err, entity.ErrUserDeleted) || errors.Is(err, entity.ErrForbidden) {
					return forbidden(c, "forbidden")
				}
				return unauthorized(c)
			}

			if userID == 0 {
				return unauthorized(c)
			}

			SetAuthUser(c, userID, role, tokenVersion)
			return next(c)
		}
	}
}

// AuthMiddleware後にadmin権限を確認。
// roleをRequest Bodyから受け取らず、検証済みAccessToken由来のroleだけを信用。
func AdminMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx, ok := GetAppContext(c)
			if !ok || ctx.AuthUserID == nil || ctx.AuthRole == nil {
				return unauthorized(c)
			}

			if *ctx.AuthRole != entity.UserRoleAdmin {
				return forbidden(c, "admin_required")
			}

			return next(c)
		}
	}
}

// Authorization HeaderからBearer tokenだけを取り出す。
// Bearer形式以外や空tokenは拒否し、生tokenはAppContextやログには保存しない。
func bearerToken(header string) (string, bool) {
	if header == "" {
		return "", false
	}

	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" || strings.Contains(token, " ") {
		return "", false
	}

	return token, true
}
