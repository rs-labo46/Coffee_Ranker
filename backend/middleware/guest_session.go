package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/usecase"

	"github.com/labstack/echo/v4"
)

const GuestSessionCookieName = "__Host-guest_session"

// MiddlewareはGuestの興味集計や行動ログ作成をせず、識別情報の確定だけをUsecaseへ委譲。
type GuestSessionGetter interface {
	GetOrCreateGuestSession(ctx context.Context, sessionKey string) (usecase.GuestSessionResult, error)
}

// GuestSession Cookieの設定値をまとめる。
// 本番で別site構成の場合はSameSiteNone + Secure=trueを検討。
type GuestSessionCookieConfig struct {
	Name     string
	MaxAge   time.Duration
	Secure   bool
	SameSite http.SameSite
}

// 未ログインの場合だけGuestSessionを取得または作成し、AppContextへGuestSessionIDを保存。
// ログイン済みUserの場合はGuestSessionを作らず、Actorが二重になる状態を防ぐ。
func GuestSessionMiddleware(guests GuestSessionGetter, config GuestSessionCookieConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := EnsureAppContext(c)
			if ctx.AuthUserID != nil {
				ctx.GuestSessionID = nil
				return next(c)
			}

			if guests == nil {
				return internalServerError(c)
			}

			rawKey := guestSessionKeyFromCookie(c, config.cookieName())
			result, err := guests.GetOrCreateGuestSession(c.Request().Context(), rawKey)
			if err != nil && shouldRetryGuestSession(err) {
				clearGuestSessionCookie(c, config)
				result, err = guests.GetOrCreateGuestSession(c.Request().Context(), "")
			}
			if err != nil {
				return internalServerError(c)
			}

			if result.Session == nil || result.Session.ID == 0 || result.SessionKey == "" {
				return internalServerError(c)
			}

			SetGuestSession(c, result.Session.ID)
			setGuestSessionCookie(c, result.SessionKey, config)

			return next(c)
		}
	}
}

// GuestSession Cookie名を決定。
// 未指定なら仕様上の__Host-guest_sessionを使う。
func (c GuestSessionCookieConfig) cookieName() string {
	if c.Name == "" {
		return GuestSessionCookieName
	}
	return c.Name
}

// CookieからGuestSessionの生keyを読み取る。
// 生keyはUsecaseへ渡す直前だけ扱い、AppContextやログには保存しない。
func guestSessionKeyFromCookie(c echo.Context, cookieName string) string {
	cookie, err := c.Cookie(cookieName)
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}

// 不正・期限切れ扱いのGuestSessionだけ、新規作成のために空keyで再試行。
// DB障害などは再試行せず500。
func shouldRetryGuestSession(err error) bool {
	return errors.Is(err, entity.ErrInvalidInput) || errors.Is(err, entity.ErrGuestSessionNotFound) || errors.Is(err, entity.ErrNotFound)
}

// GuestSession Cookieを設定。
// DBにはhashだけを保存し、Cookieにはクライアント識別用の生keyをHttpOnlyで保持。
func setGuestSessionCookie(c echo.Context, sessionKey string, config GuestSessionCookieConfig) {
	cookie := &http.Cookie{
		Name:     config.cookieName(),
		Value:    sessionKey,
		Path:     "/",
		HttpOnly: true,
		Secure:   config.Secure,
		SameSite: config.sameSite(),
	}

	if config.MaxAge > 0 {
		cookie.MaxAge = int(config.MaxAge.Seconds())
		cookie.Expires = time.Now().Add(config.MaxAge)
	}

	c.SetCookie(cookie)
}

// 不正または期限切れのGuestSession Cookieを削除。
// 古いkeyを残さず、次の処理で新規GuestSessionを作れる状態にする。
func clearGuestSessionCookie(c echo.Context, config GuestSessionCookieConfig) {
	c.SetCookie(&http.Cookie{
		Name:     config.cookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   config.Secure,
		SameSite: config.sameSite(),
	})
}

// SameSite未指定時のデフォルトをLaxに。
// 同一site構成ではLaxを優先し、別site + Cookie送信が必要な場合のみNoneを明示。
func (c GuestSessionCookieConfig) sameSite() http.SameSite {
	if c.SameSite == 0 {
		return http.SameSiteLaxMode
	}
	return c.SameSite
}
