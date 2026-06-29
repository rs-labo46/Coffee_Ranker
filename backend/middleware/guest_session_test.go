package middleware

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/usecase"

	"github.com/labstack/echo/v4"
)

// GuestSessionUsecaseの結果を任意に返すGuestSessionMiddleware用のテストfake。
type fakeGuestSessionGetter struct {
	results []usecase.GuestSessionResult
	errs    []error
	keys    []string
}

// middlewareから渡されたsession keyを記録し、設定済みのGuestSession結果を返す。
func (f *fakeGuestSessionGetter) GetOrCreateGuestSession(ctx context.Context, sessionKey string) (usecase.GuestSessionResult, error) {
	f.keys = append(f.keys, sessionKey)
	idx := len(f.keys) - 1
	if idx < len(f.errs) && f.errs[idx] != nil {
		return usecase.GuestSessionResult{}, f.errs[idx]
	}
	if idx < len(f.results) {
		return f.results[idx], nil
	}
	return usecase.GuestSessionResult{}, nil
}

// GuestSessionMiddlewareテスト用のGuestSessionResultを作成する。
func guestResult(id uint64, key string) usecase.GuestSessionResult {
	return usecase.GuestSessionResult{
		Session:    &model.GuestSession{ID: id},
		SessionKey: key,
		Created:    true,
	}
}

// GuestSessionがないRequestで新規GuestSessionを作成し、cookieを返すことを検証する。
func TestGuestSessionMiddleware_CreatesGuestAndSetsCookie(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/guest")
	getter := &fakeGuestSessionGetter{results: []usecase.GuestSessionResult{guestResult(44, "new-key")}}
	config := GuestSessionCookieConfig{Name: "guest_session", MaxAge: time.Hour, Secure: false, SameSite: http.SameSiteLaxMode}

	err := runMiddleware(t, c, func(c echo.Context) error {
		ctx, ok := GetAppContext(c)
		if !ok || ctx.GuestSessionID == nil || *ctx.GuestSessionID != 44 {
			t.Fatalf("guest session was not stored: %#v", ctx)
		}
		return c.NoContent(http.StatusOK)
	}, GuestSessionMiddleware(getter, config))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(getter.keys) != 1 || getter.keys[0] != "" {
		t.Fatalf("guest keys = %#v, want empty key call", getter.keys)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies len = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "guest_session" || cookie.Value != "new-key" {
		t.Fatalf("cookie = %s:%s, want guest_session:new-key", cookie.Name, cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatal("guest session cookie should be HttpOnly")
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie path = %q, want /", cookie.Path)
	}
	if cookie.MaxAge <= 0 {
		t.Fatalf("cookie MaxAge = %d, want positive", cookie.MaxAge)
	}
}

// 既存cookieのsession keyを使ってGuestSessionを取得することを検証する。
func TestGuestSessionMiddleware_UsesExistingCookieKey(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/guest")
	c.Request().AddCookie(&http.Cookie{Name: "guest_session", Value: "existing-key"})
	getter := &fakeGuestSessionGetter{results: []usecase.GuestSessionResult{guestResult(45, "existing-key")}}

	err := runMiddleware(t, c, okHandler, GuestSessionMiddleware(getter, GuestSessionCookieConfig{Name: "guest_session"}))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(getter.keys) != 1 || getter.keys[0] != "existing-key" {
		t.Fatalf("guest keys = %#v, want existing-key", getter.keys)
	}
}

// 既存cookieが無効な場合、空keyで再作成を試みることを検証する。
func TestGuestSessionMiddleware_RetriesInvalidCookieWithEmptyKey(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/guest")
	c.Request().AddCookie(&http.Cookie{Name: "guest_session", Value: "invalid-key"})
	getter := &fakeGuestSessionGetter{
		errs:    []error{entity.ErrGuestSessionNotFound, nil},
		results: []usecase.GuestSessionResult{{}, guestResult(46, "replacement-key")},
	}

	err := runMiddleware(t, c, okHandler, GuestSessionMiddleware(getter, GuestSessionCookieConfig{Name: "guest_session"}))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(getter.keys) != 2 || getter.keys[0] != "invalid-key" || getter.keys[1] != "" {
		t.Fatalf("guest keys = %#v, want invalid-key then empty", getter.keys)
	}
	if len(rec.Result().Cookies()) < 2 {
		t.Fatalf("cookies len = %d, want clear cookie and replacement cookie", len(rec.Result().Cookies()))
	}
}

// 認証済みUserではGuestSessionを新規作成しないことを検証する。
func TestGuestSessionMiddleware_DoesNotCreateGuestForAuthUser(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/guest")
	SetAuthUser(c, 99, entity.UserRoleUser, 1)
	getter := &fakeGuestSessionGetter{errs: []error{errors.New("should not be called")}}

	err := runMiddleware(t, c, okHandler, GuestSessionMiddleware(getter, GuestSessionCookieConfig{Name: "guest_session"}))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(getter.keys) != 0 {
		t.Fatalf("guest getter should not be called for auth user: %#v", getter.keys)
	}
}

// GuestSession取得結果が不正な場合や依存先エラー時に500を返すことを検証する。
func TestGuestSessionMiddleware_Returns500ForInvalidResultOrDependencyError(t *testing.T) {
	cases := []struct {
		name   string
		getter *fakeGuestSessionGetter
	}{
		{name: "dependency error", getter: &fakeGuestSessionGetter{errs: []error{errors.New("db down")}}},
		{name: "nil session", getter: &fakeGuestSessionGetter{results: []usecase.GuestSessionResult{{Session: nil, SessionKey: "key"}}}},
		{name: "empty session key", getter: &fakeGuestSessionGetter{results: []usecase.GuestSessionResult{{Session: &model.GuestSession{ID: 1}, SessionKey: ""}}}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, c, rec := newMiddlewareTestContext(http.MethodGet, "/guest")

			err := runMiddleware(t, c, okHandler, GuestSessionMiddleware(tt.getter, GuestSessionCookieConfig{Name: "guest_session"}))
			if err != nil {
				t.Fatalf("middleware error: %v", err)
			}
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rec.Code)
			}
		})
	}
}
