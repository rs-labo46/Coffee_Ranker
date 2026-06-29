package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/usecase"

	"github.com/labstack/echo/v4"
)

const (
	ContextUserIDKey         = "user_id"
	ContextGuestSessionIDKey = "guest_session_id"
	ContextRequestIDKey      = "request_id"
	ContextIPHashKey         = "ip_hash"

	RefreshTokenCookieName = "refresh_token"
	CSRFCookieName         = "csrf_token"
	GuestSessionCookieName = "guest_session_key"
	GuestSessionHeaderName = "X-Guest-Session-Key"
)

type CookieConfig struct {
	Secure          bool
	Domain          string
	RefreshTokenTTL time.Duration
	GuestSessionTTL time.Duration
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Usecase/Validatorのエラーを共通Error Responseへ変換。
// ControllerごとにHTTP変換が散らばらないようにする。
func writeError(c echo.Context, err error) error {
	status, code, message := mapError(err)
	return c.JSON(status, ErrorResponse{Code: code, Message: message})
}

// domain errorをHTTP status/code/messageへ対応付ける。
// GORMや内部実装の詳細をレスポンスに漏らさないための境界。
func mapError(err error) (int, string, string) {
	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		if echoErr.Code >= 400 && echoErr.Code < 500 {
			return echoErr.Code, "invalid_request", "リクエストが不正です"
		}
	}
	switch {
	case errors.Is(err, entity.ErrInvalidInput), errors.Is(err, entity.ErrInvalidPagination), errors.Is(err, entity.ErrInvalidContentType), errors.Is(err, entity.ErrInvalidEventType), errors.Is(err, entity.ErrInvalidRatingScore), errors.Is(err, entity.ErrInvalidSearchCondition):
		return http.StatusBadRequest, "invalid_input", "入力内容が不正です"
	case errors.Is(err, entity.ErrUnauthorized), errors.Is(err, entity.ErrInvalidToken), errors.Is(err, entity.ErrInvalidCredentials), errors.Is(err, entity.ErrRefreshTokenExpired), errors.Is(err, entity.ErrRefreshTokenRevoked), errors.Is(err, entity.ErrRefreshTokenReuseDetected):
		return http.StatusUnauthorized, "unauthorized", "認証が必要です"
	case errors.Is(err, entity.ErrForbidden), errors.Is(err, entity.ErrLoginRequired), errors.Is(err, entity.ErrUserSuspended), errors.Is(err, entity.ErrUserDeleted):
		return http.StatusForbidden, "forbidden", "操作が許可されていません"
	case errors.Is(err, entity.ErrBeanNotFound), errors.Is(err, entity.ErrArticleNotFound), errors.Is(err, entity.ErrRankTargetNotFound), errors.Is(err, entity.ErrGuestSessionNotFound), errors.Is(err, entity.ErrSavedItemNotFound), errors.Is(err, entity.ErrRatingNotFound), errors.Is(err, entity.ErrModalCandidateNotFound), errors.Is(err, entity.ErrModalDisplayLogNotFound), errors.Is(err, entity.ErrNotFound):
		return http.StatusNotFound, "not_found", "対象が見つかりません"
	case errors.Is(err, entity.ErrEmailAlreadyExists), errors.Is(err, entity.ErrSavedItemAlreadyExists), errors.Is(err, entity.ErrConflict):
		return http.StatusConflict, "conflict", "既に存在します"
	case errors.Is(err, entity.ErrRateLimited):
		return http.StatusTooManyRequests, "rate_limited", "リクエスト回数が上限に達しました"
	case errors.Is(err, entity.ErrBatchAlreadyRunning), errors.Is(err, entity.ErrBatchLockFailed):
		return http.StatusConflict, "batch_running", "バッチが既に実行中です"
	default:
		return http.StatusInternalServerError, "internal_error", "サーバー内部でエラーが発生しました"
	}
}

// path paramをuint64へ変換できるか確認。
// 0や文字列を弾き、IDのDB存在確認はUsecaseへ任せる。
func parseUintParam(c echo.Context, name string) (uint64, error) {
	value := strings.TrimSpace(c.Param(name))
	if value == "" {
		return 0, entity.ErrInvalidInput
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, entity.ErrInvalidInput
	}
	return id, nil
}

// query paramをintへ変換できるか確認。
// 未指定は0として扱い、範囲検証は各Validatorで行う。
func parseIntQuery(c echo.Context, name string) (int, error) {
	value := strings.TrimSpace(c.QueryParam(name))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, entity.ErrInvalidInput
	}
	return parsed, nil
}

// 認証MiddlewareがContextへ入れたuser_idを必須取得。
// bodyのuser_idを信用せず、未認証ならErrUnauthorizedにする。
func mustUserID(c echo.Context) (uint64, error) {
	id, ok := uint64FromContext(c, ContextUserIDKey)
	if !ok || id == 0 {
		return 0, entity.ErrUnauthorized
	}
	return id, nil
}

// Contextにuser_idがある場合だけ取得。
// Guest/User両対応APIでactorを作るために使う。
func optionalUserID(c echo.Context) *uint64 {
	id, ok := uint64FromContext(c, ContextUserIDKey)
	if !ok || id == 0 {
		return nil
	}
	return &id
}

// Contextにguest_session_idがある場合だけ取得。
// bodyのguest_session_idを信用しないための共通処理。
func optionalGuestSessionID(c echo.Context) *uint64 {
	id, ok := uint64FromContext(c, ContextGuestSessionIDKey)
	if !ok || id == 0 {
		return nil
	}
	return &id
}

// UserまたはGuestSessionのどちらのactorかをContextから確定。
// 両方なし・両方ありの状態を拒否し、actor偽装を防ぐ。
func actorFromContext(c echo.Context) (usecase.Actor, error) {
	actor := usecase.Actor{UserID: optionalUserID(c), GuestSessionID: optionalGuestSessionID(c)}
	if (actor.UserID == nil && actor.GuestSessionID == nil) || (actor.UserID != nil && actor.GuestSessionID != nil) {
		return usecase.Actor{}, entity.ErrUnauthorized
	}
	return actor, nil
}

// Admin操作の監査ログに使う管理者情報をContextから取得。
// AdminGuard後のuser_idだけを信用し、bodyの管理者IDは使わない。
func adminMeta(c echo.Context) (usecase.AdminMeta, error) {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return usecase.AdminMeta{}, err
	}
	return usecase.AdminMeta{AdminUserID: userID, AuditMeta: auditMeta(c)}, nil
}

// 監査・イベント用のIP hashとrequest_idをContextから取得。
// ログ追跡用の補助情報であり、業務判断には使わない。
func auditMeta(c echo.Context) usecase.AuditMeta {
	return usecase.AuditMeta{RequestID: optionalStringContext(c, ContextRequestIDKey), IPAddressHash: optionalStringContext(c, ContextIPHashKey)}
}

// Echo Context内の値をuint64として安全に取り出す。
// uint/int/stringの可能性を吸収し、変換できない値は無効扱いにする。
func uint64FromContext(c echo.Context, key string) (uint64, bool) {
	value := c.Get(key)
	switch v := value.(type) {
	case uint64:
		return v, true
	case uint:
		return uint64(v), true
	case int:
		if v <= 0 {
			return 0, false
		}
		return uint64(v), true
	case string:
		id, err := strconv.ParseUint(v, 10, 64)
		return id, err == nil
	default:
		return 0, false
	}
}

// 空文字をnilに変換し、任意条件をUsecase inputへ渡しやすく。
// 文字列の安全性検証は呼び出し元で済ませます。
func optionalStringContext(c echo.Context, key string) *string {
	value := c.Get(key)
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(text)
	return &trimmed
}

// RefreshTokenをHttpOnly Cookieとしてレスポンスへ設定。
// token生値はレスポンスbodyに出さず、Cookie境界で扱う。
func setRefreshCookie(c echo.Context, cfg CookieConfig, token string) {
	maxAge := int(cfg.RefreshTokenTTL.Seconds())
	if maxAge <= 0 {
		maxAge = int((14 * 24 * time.Hour).Seconds())
	}
	cookie := &http.Cookie{Name: RefreshTokenCookieName, Value: token, Path: "/", Domain: cfg.Domain, MaxAge: maxAge, HttpOnly: true, Secure: cfg.Secure, SameSite: http.SameSiteStrictMode}
	c.SetCookie(cookie)
}

// GuestSession keyをHttpOnly Cookieとしてレスポンスへ設定。
// DB上のguest_session_idはクライアントへ直接渡さない。
func setGuestSessionCookie(c echo.Context, cfg CookieConfig, sessionKey string) {
	maxAge := int(cfg.GuestSessionTTL.Seconds())
	if maxAge <= 0 {
		maxAge = int((7 * 24 * time.Hour).Seconds())
	}
	cookie := &http.Cookie{Name: GuestSessionCookieName, Value: sessionKey, Path: "/", Domain: cfg.Domain, MaxAge: maxAge, HttpOnly: true, Secure: cfg.Secure, SameSite: http.SameSiteLaxMode}
	c.SetCookie(cookie)
	c.Response().Header().Set(GuestSessionHeaderName, sessionKey)
}

// logout/refresh失敗時に認証系Cookieを削除。
// refresh_tokenとcsrf_tokenの残存による誤動作を防ぐ。
func clearAuthCookies(c echo.Context, cfg CookieConfig) {
	clearCookie(c, cfg, RefreshTokenCookieName, true)
	clearCookie(c, cfg, CSRFCookieName, false)
}

// 指定Cookieを期限切れにして削除。
// Cookie削除処理を共通化し、設定漏れを防ぐ。
func clearCookie(c echo.Context, cfg CookieConfig, name string, httpOnly bool) {
	c.SetCookie(&http.Cookie{Name: name, Value: "", Path: "/", Domain: cfg.Domain, MaxAge: -1, HttpOnly: httpOnly, Secure: cfg.Secure, SameSite: http.SameSiteStrictMode})
}

// refresh_token Cookieから生RefreshTokenを取り出す。
// tokenの有効性はAuthUsecaseで判断。
func refreshTokenFromCookie(c echo.Context) string {
	cookie, err := c.Cookie(RefreshTokenCookieName)
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}

// GuestSession keyをHeader優先、なければCookieから取り出す。
// DB上のguest_session_idをbodyから受け取らないための入口。
func guestSessionKeyFromRequest(c echo.Context) string {
	if value := strings.TrimSpace(c.Request().Header.Get(GuestSessionHeaderName)); value != "" {
		return value
	}
	cookie, err := c.Cookie(GuestSessionCookieName)
	if err != nil || cookie == nil {
		return ""
	}
	return cookie.Value
}
