package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"coffee-ranker/entity"

	"github.com/labstack/echo/v4"
)

const appContextKey = "coffee_ranker_app_context"

// Controllerへ渡す最小限の共通情報だけを保持。
// 生AccessToken、生RefreshToken、CSRF token、password、Cookie headerは保存しません。
type AppContext struct {
	RequestID      string
	AuthUserID     *uint64
	AuthRole       *entity.UserRole
	TokenVersion   *int
	GuestSessionID *uint64
	ClientIPHash   string
}

// Echo Context内のAppContextを取得。
// まだ存在しない場合は新規作成し、以降のMiddlewareやControllerで同じ情報を使えるようにする。
func EnsureAppContext(c echo.Context) *AppContext {
	stored := c.Get(appContextKey)
	ctx, ok := stored.(*AppContext)
	if ok && ctx != nil {
		return ctx
	}

	ctx = &AppContext{}
	c.Set(appContextKey, ctx)
	return ctx
}

// Echo Context内のAppContextを取得。
// Middleware適用漏れなどで存在しない場合はfalseを返し、呼び出し側で認証失敗として扱えるようにする。
func GetAppContext(c echo.Context) (*AppContext, bool) {
	stored := c.Get(appContextKey)
	ctx, ok := stored.(*AppContext)
	if !ok || ctx == nil {
		return nil, false
	}

	return ctx, true
}

// 認証済みUser情報をAppContextへ保存。
// ログイン済みUserとして扱うため、GuestSessionIDは同時に保持しないよう消す。
func SetAuthUser(c echo.Context, userID uint64, role entity.UserRole, tokenVersion int) {
	ctx := EnsureAppContext(c)
	ctx.AuthUserID = &userID
	ctx.AuthRole = &role
	ctx.TokenVersion = &tokenVersion
	ctx.GuestSessionID = nil
}

// GuestSessionIDをAppContextへ保存。
// 未ログインGuestとして扱うため、認証済みUserが既にある場合は保存しない。
func SetGuestSession(c echo.Context, guestSessionID uint64) {
	ctx := EnsureAppContext(c)
	if ctx.AuthUserID != nil {
		ctx.GuestSessionID = nil
		return
	}

	ctx.GuestSessionID = &guestSessionID
}

// request_idをAppContextへ保存。
// Controller、ログ、エラーレスポンスで同じrequest_idを使えるようにする。
func SetRequestID(c echo.Context, requestID string) {
	ctx := EnsureAppContext(c)
	ctx.RequestID = requestID
}

// IPアドレスをHMAC-SHA256でhash化。
// 生IPをAppContextやログへ残さず、RateLimit keyに使える文字列へ変換。
func SetClientIPHash(c echo.Context, ip string, secret string) {
	ctx := EnsureAppContext(c)
	ctx.ClientIPHash = HashClientIP(ip, secret)
}

// 生IPをログやContextに残さないためのhash値を作る。
func HashClientIP(ip string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(ip))
	return hex.EncodeToString(mac.Sum(nil))
}

// UserまたはGuestSessionのどちらか片方だけが存在するか確認。
// 両方なし・両方ありはDB制約とも矛盾するため、不正なActor状態として扱う。
func HasSingleActor(ctx *AppContext) bool {
	if ctx == nil {
		return false
	}

	hasUser := ctx.AuthUserID != nil
	hasGuest := ctx.GuestSessionID != nil

	return (hasUser && !hasGuest) || (!hasUser && hasGuest)
}

// RateLimitやUsecase inputで使うActor keyを生成。
// UserとGuestを同じ数値IDだけで混ぜないよう、prefixを付けて区別。
func ActorKey(ctx *AppContext) (string, bool) {
	if !HasSingleActor(ctx) {
		return "", false
	}

	if ctx.AuthUserID != nil {
		return "user:" + strconv.FormatUint(*ctx.AuthUserID, 10), true
	}

	return "guest:" + strconv.FormatUint(*ctx.GuestSessionID, 10), true
}
