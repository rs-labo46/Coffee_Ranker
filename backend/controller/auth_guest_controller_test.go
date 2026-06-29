package controller

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/usecase"
	"coffee-ranker/validator"
)

// SignupでJSON bodyをBindし、Validator通過後にUser作成結果を201で返すことを確認。
func TestAuthControllerSignupSuccess(t *testing.T) {
	users := &fakeUserRepo{}
	auth := usecase.NewAuthUsecase(users, &fakeRefreshTokenRepo{}, &fakeAuditRepo{}, nil, &fakePasswordHasher{}, &fakeTokenManager{}, 14*24*time.Hour)
	controller := NewAuthController(auth, validator.NewAuthValidator(), CookieConfig{})
	_, c, rec := newTestContext(http.MethodPost, "/signup", jsonBody(t, validator.SignupRequest{Name: "Taro", Email: "TARO@example.com", Password: "password123"}))

	if err := controller.Signup(c); err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	assertStatus(t, rec, http.StatusCreated)
	if users.created == nil || users.created.Email != "taro@example.com" {
		t.Fatalf("created user = %#v", users.created)
	}
	if strings.Contains(rec.Body.String(), "password") || strings.Contains(rec.Body.String(), "hashed") {
		t.Fatalf("response leaked password data: %s", rec.Body.String())
	}
}

// Signupの入力がValidatorで落ちた場合、Usecaseに到達せず400を返すことを確認。
func TestAuthControllerSignupInvalidInput(t *testing.T) {
	controller := NewAuthController(nil, validator.NewAuthValidator(), CookieConfig{})
	_, c, rec := newTestContext(http.MethodPost, "/signup", jsonBody(t, validator.SignupRequest{Name: "", Email: "bad", Password: "short"}))

	if err := controller.Signup(c); err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// Login成功時にAccessTokenはbodyへ、RefreshTokenはCookieへ設定されることを確認。
func TestAuthControllerLoginSetsRefreshCookie(t *testing.T) {
	auth := usecase.NewAuthUsecase(&fakeUserRepo{}, &fakeRefreshTokenRepo{}, &fakeAuditRepo{}, nil, &fakePasswordHasher{}, &fakeTokenManager{refreshPlain: "new-refresh", accessToken: "access-1"}, 14*24*time.Hour)
	controller := NewAuthController(auth, validator.NewAuthValidator(), CookieConfig{RefreshTokenTTL: time.Hour})
	_, c, rec := newTestContext(http.MethodPost, "/login", jsonBody(t, validator.LoginRequest{Email: "user@example.com", Password: "password123"}))
	setMeta(c)

	if err := controller.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "access-1") {
		t.Fatalf("access token was not returned: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), RefreshTokenCookieName+"=new-refresh") {
		t.Fatalf("refresh cookie was not set: %s", rec.Header().Get("Set-Cookie"))
	}
	if strings.Contains(rec.Body.String(), "new-refresh") {
		t.Fatalf("refresh token leaked in body: %s", rec.Body.String())
	}
}

// RefreshToken Cookieがない場合、認証Cookieを削除し401を返すことを確認。
func TestAuthControllerRefreshWithoutCookieClearsCookies(t *testing.T) {
	controller := NewAuthController(nil, validator.NewAuthValidator(), CookieConfig{})
	_, c, rec := newTestContext(http.MethodPost, "/refresh", "")

	if err := controller.Refresh(c); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
	cookies := rec.Header().Values("Set-Cookie")
	joined := strings.Join(cookies, ";")
	if !strings.Contains(joined, RefreshTokenCookieName+"=") || !strings.Contains(joined, CSRFCookieName+"=") {
		t.Fatalf("auth cookies were not cleared: %v", cookies)
	}
}

// Logoutはbodyのuser_idではなくContextのUserIDを必須にし、未認証なら401を返すことを確認。
func TestAuthControllerLogoutRequiresContextUser(t *testing.T) {
	controller := NewAuthController(nil, validator.NewAuthValidator(), CookieConfig{})
	_, c, rec := newTestContext(http.MethodPost, "/logout", `{ "user_id": 1 }`)

	if err := controller.Logout(c); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
}

// MeはContextの認証済みUserIDだけで自分の情報を取得し、password hashを返さないことを確認。
func TestAuthControllerMeSuccess(t *testing.T) {
	auth := usecase.NewAuthUsecase(&fakeUserRepo{byID: &modelUserMe}, &fakeRefreshTokenRepo{}, &fakeAuditRepo{}, nil, &fakePasswordHasher{}, &fakeTokenManager{}, time.Hour)
	controller := NewAuthController(auth, validator.NewAuthValidator(), CookieConfig{})
	_, c, rec := newTestContext(http.MethodGet, "/me", "")
	setUser(c, modelUserMe.ID)

	if err := controller.Me(c); err != nil {
		t.Fatalf("Me failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
	if strings.Contains(rec.Body.String(), "secret-hash") {
		t.Fatalf("password hash leaked: %s", rec.Body.String())
	}
}

// GuestSession新規作成時にDB IDではなくsession keyをCookie/Headerで返すことを確認。
func TestGuestSessionControllerGetOrCreateSetsCookieAndHeader(t *testing.T) {
	guest := usecase.NewGuestSessionUsecase(&fakeGuestSessionRepo{}, &fakeGuestKeys{}, time.Hour)
	controller := NewGuestSessionController(guest, CookieConfig{GuestSessionTTL: time.Hour})
	_, c, rec := newTestContext(http.MethodPost, "/guest-session", "")

	if err := controller.GetOrCreate(c); err != nil {
		t.Fatalf("GetOrCreate failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get(GuestSessionHeaderName); got != "guest-plain" {
		t.Fatalf("guest session header = %q", got)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), GuestSessionCookieName+"=guest-plain") {
		t.Fatalf("guest session cookie was not set: %s", rec.Header().Get("Set-Cookie"))
	}
}

// TouchはbodyではなくContextのguest_session_idを使い、Contextにない場合は401を返すことを確認。
func TestGuestSessionControllerTouchRequiresContextGuest(t *testing.T) {
	controller := NewGuestSessionController(nil, CookieConfig{})
	_, c, rec := newTestContext(http.MethodPost, "/guest-session/touch", `{ "guest_session_id": 1 }`)

	if err := controller.Touch(c); err != nil {
		t.Fatalf("Touch failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
}

var modelUserMe = model.User{ID: 7, Name: "me", Email: "me@example.com", PasswordHash: "secret-hash", Role: entity.UserRoleUser, Status: entity.UserStatusActive}
