package middleware

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"coffee-ranker/entity"

	"github.com/labstack/echo/v4"
)

// AccessToken検証結果を任意に返すAuthMiddleware用のテストfake。
type fakeAccessTokenValidator struct {
	userID       uint64
	role         entity.UserRole
	tokenVersion int
	err          error
	rawToken     string
	called       bool
}

// AuthMiddlewareから渡された生AccessTokenを記録し、設定済みの検証結果を返す。
func (f *fakeAccessTokenValidator) ValidateAccessToken(ctx context.Context, rawToken string) (uint64, entity.UserRole, int, error) {
	f.called = true
	f.rawToken = rawToken
	return f.userID, f.role, f.tokenVersion, f.err
}

// Authorization headerからBearer tokenだけを安全に取り出すことを検証する。
func TestBearerToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
		ok     bool
	}{
		{name: "valid", header: "Bearer abc.def", want: "abc.def", ok: true},
		{name: "missing", header: "", ok: false},
		{name: "wrong prefix", header: "Token abc", ok: false},
		{name: "empty token", header: "Bearer   ", ok: false},
		{name: "contains space", header: "Bearer abc def", ok: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := bearerToken(tt.header)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("bearerToken() = %q, %v, want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

// 正常なAccessTokenでAuthUser情報がAppContextへ保存されることを検証する。
func TestAuthMiddleware_ValidTokenStoresAuthUser(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/private")
	c.Request().Header.Set("Authorization", "Bearer access-token")
	auth := &fakeAccessTokenValidator{userID: 123, role: entity.UserRoleUser, tokenVersion: 4}

	err := runMiddleware(t, c, func(c echo.Context) error {
		ctx, ok := GetAppContext(c)
		if !ok {
			t.Fatal("app context was not created")
		}
		if ctx.AuthUserID == nil || *ctx.AuthUserID != 123 {
			t.Fatalf("auth user id = %#v, want 123", ctx.AuthUserID)
		}
		if ctx.AuthRole == nil || *ctx.AuthRole != entity.UserRoleUser {
			t.Fatalf("auth role = %#v, want user", ctx.AuthRole)
		}
		if ctx.TokenVersion == nil || *ctx.TokenVersion != 4 {
			t.Fatalf("token version = %#v, want 4", ctx.TokenVersion)
		}
		return c.NoContent(http.StatusOK)
	}, AuthMiddleware(auth))

	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !auth.called || auth.rawToken != "access-token" {
		t.Fatalf("validator called=%v raw=%q, want true access-token", auth.called, auth.rawToken)
	}
}

// AccessTokenがない場合や検証失敗時に401を返すことを検証する。
func TestAuthMiddleware_RejectsMissingOrInvalidToken(t *testing.T) {
	cases := []struct {
		name   string
		header string
		err    error
		want   int
	}{
		{name: "missing header", header: "", want: http.StatusUnauthorized},
		{name: "invalid format", header: "Bearer token with-space", want: http.StatusUnauthorized},
		{name: "validator unauthorized", header: "Bearer token", err: entity.ErrUnauthorized, want: http.StatusUnauthorized},
		{name: "suspended", header: "Bearer token", err: entity.ErrUserSuspended, want: http.StatusForbidden},
		{name: "deleted", header: "Bearer token", err: entity.ErrUserDeleted, want: http.StatusForbidden},
		{name: "forbidden", header: "Bearer token", err: entity.ErrForbidden, want: http.StatusForbidden},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, c, rec := newMiddlewareTestContext(http.MethodGet, "/private")
			if tt.header != "" {
				c.Request().Header.Set("Authorization", tt.header)
			}
			auth := &fakeAccessTokenValidator{userID: 123, role: entity.UserRoleUser, tokenVersion: 1, err: tt.err}

			err := runMiddleware(t, c, okHandler, AuthMiddleware(auth))
			if err != nil {
				t.Fatalf("middleware error: %v", err)
			}
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

// token検証結果のuserIDが0の場合に不正として拒否することを検証する。
func TestAuthMiddleware_RejectsZeroUserID(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/private")
	c.Request().Header.Set("Authorization", "Bearer token")
	auth := &fakeAccessTokenValidator{userID: 0, role: entity.UserRoleUser, tokenVersion: 1}

	err := runMiddleware(t, c, okHandler, AuthMiddleware(auth))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// Authorization headerがない場合でもOptionalAuthが処理を通すことを検証する。
func TestOptionalAuthMiddleware_PassesWithoutHeader(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/optional")
	auth := &fakeAccessTokenValidator{err: errors.New("should not be called")}

	err := runMiddleware(t, c, okHandler, OptionalAuthMiddleware(auth))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if auth.called {
		t.Fatal("validator should not be called when Authorization header is missing")
	}
}

// OptionalAuthで有効なAccessTokenがある場合にAuthUser情報を保存することを検証する。
func TestOptionalAuthMiddleware_ValidHeaderStoresAuthUser(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/optional")
	c.Request().Header.Set("Authorization", "Bearer optional-token")
	auth := &fakeAccessTokenValidator{userID: 77, role: entity.UserRoleAdmin, tokenVersion: 8}

	err := runMiddleware(t, c, okHandler, OptionalAuthMiddleware(auth))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	ctx, ok := GetAppContext(c)
	if !ok || ctx.AuthUserID == nil || *ctx.AuthUserID != 77 {
		t.Fatalf("auth user was not stored: %#v", ctx)
	}
}

// Admin権限のみ通し、未認証や一般Userを拒否することを検証する。
func TestAdminMiddleware(t *testing.T) {
	t.Run("admin passes", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/admin")
		SetAuthUser(c, 1, entity.UserRoleAdmin, 1)

		err := runMiddleware(t, c, okHandler, AdminMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("normal user is forbidden", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/admin")
		SetAuthUser(c, 1, entity.UserRoleUser, 1)

		err := runMiddleware(t, c, okHandler, AdminMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("missing auth is unauthorized", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodGet, "/admin")

		err := runMiddleware(t, c, okHandler, AdminMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}
