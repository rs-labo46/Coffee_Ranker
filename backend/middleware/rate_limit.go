package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/repository"
	"coffee-ranker/usecase"

	"github.com/labstack/echo/v4"
)

// RateLimitの種類を明示。
// Router側でAPI分類ごとに指定し、Middleware内でRequest Bodyを読まないようにする。
type RateLimitName string

const (
	RateLimitPublicRead     RateLimitName = "public_read"
	RateLimitAuthIP         RateLimitName = "auth_ip"
	RateLimitRefresh        RateLimitName = "refresh"
	RateLimitUser           RateLimitName = "user"
	RateLimitSearch         RateLimitName = "search"
	RateLimitEvent          RateLimitName = "event"
	RateLimitRecommendation RateLimitName = "recommendation"
	RateLimitModal          RateLimitName = "modal"
	RateLimitAdmin          RateLimitName = "admin"
	RateLimitAdminBatch     RateLimitName = "admin_batch"
)

// MiddlewareはRedis Clientを直接触らず、Usecase経由でTokenBucket判定を行う。
type RateLimiter interface {
	Take(ctx context.Context, rule usecase.RateLimitRule, now time.Time) (repository.RateLimitResult, error)
}

// API分類ごとのRateLimitを実行。
// key材料はAppContextのUserID、GuestSessionID、IP hashなどから作り、Request Bodyは読まない。
func RateLimitMiddleware(limiter RateLimiter, name RateLimitName) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if limiter == nil {
				return internalServerError(c)
			}

			rule, ok := rateLimitRule(c, name)
			if !ok {
				return unauthorized(c)
			}

			result, err := limiter.Take(c.Request().Context(), rule, time.Now())
			setRateLimitHeaders(c, name, result)
			if err != nil {
				if errors.Is(err, entity.ErrRateLimited) {
					return rateLimited(c)
				}
				return writeError(c, http.StatusServiceUnavailable, "rate_limit_unavailable")
			}

			return next(c)
		}
	}
}

// RateLimit名に応じてTokenBucketのkeyと制限値を作る。
// signup/loginのemail単位制限はRequest Bodyを読まないため、Usecase側で別途行う前提。
func rateLimitRule(c echo.Context, name RateLimitName) (usecase.RateLimitRule, bool) {
	ctx, _ := GetAppContext(c)

	switch name {
	case RateLimitPublicRead:
		return ipRule(ctx, "public_read", 120, 2.0)
	case RateLimitAuthIP:
		return ipRule(ctx, "auth_ip", 5, 1.0/120.0)
	case RateLimitRefresh:
		return ipRule(ctx, "refresh", 30, 1.0/20.0)
	case RateLimitUser:
		return userRule(ctx, "user", 30, 0.5)
	case RateLimitSearch:
		return actorRule(ctx, "search", 30, 0.5)
	case RateLimitEvent:
		return actorRule(ctx, "event", 60, 1.0)
	case RateLimitRecommendation:
		return actorRule(ctx, "recommendation", 20, 1.0/3.0)
	case RateLimitModal:
		return actorRule(ctx, "modal", 20, 1.0/3.0)
	case RateLimitAdmin:
		return adminRule(ctx, "admin", 60, 1.0)
	case RateLimitAdminBatch:
		return adminBatchRule(c, ctx)
	default:
		return usecase.RateLimitRule{}, false
	}
}

// IP単位のRateLimit ruleを作る。
// 未認証のsignup/login/refresh/public readなど、UserIDがないAPIで使う。
func ipRule(ctx *AppContext, prefix string, capacity int, refillRate float64) (usecase.RateLimitRule, bool) {
	if ctx == nil || ctx.ClientIPHash == "" {
		return usecase.RateLimitRule{}, false
	}

	return usecase.RateLimitRule{
		Key:        "rate:" + prefix + ":ip:" + ctx.ClientIPHash,
		Capacity:   capacity,
		RefillRate: refillRate,
	}, true
}

// 認証済みUser単位のRateLimit ruleを作る。
// 保存、評価、logoutなど、ログイン済みUserだけが使うAPIで使う。
func userRule(ctx *AppContext, prefix string, capacity int, refillRate float64) (usecase.RateLimitRule, bool) {
	if ctx == nil || ctx.AuthUserID == nil {
		return usecase.RateLimitRule{}, false
	}

	return usecase.RateLimitRule{
		Key:        "rate:" + prefix + ":user:" + strconv.FormatUint(*ctx.AuthUserID, 10),
		Capacity:   capacity,
		RefillRate: refillRate,
	}, true
}

// UserまたはGuestSession単位のRateLimit ruleを作る。
// Event、Search、Recommendation、Modalなど、Actorが必要なAPIで使う。
func actorRule(ctx *AppContext, prefix string, capacity int, refillRate float64) (usecase.RateLimitRule, bool) {
	actorKey, ok := ActorKey(ctx)
	if !ok {
		return usecase.RateLimitRule{}, false
	}

	return usecase.RateLimitRule{
		Key:        "rate:" + prefix + ":" + actorKey,
		Capacity:   capacity,
		RefillRate: refillRate,
	}, true
}

// admin単位のRateLimit ruleを作る。
// AdminGuard後に呼ぶ前提で、管理操作の連打を抑える。
func adminRule(ctx *AppContext, prefix string, capacity int, refillRate float64) (usecase.RateLimitRule, bool) {
	if ctx == nil || ctx.AuthUserID == nil || ctx.AuthRole == nil || *ctx.AuthRole != entity.UserRoleAdmin {
		return usecase.RateLimitRule{}, false
	}

	return usecase.RateLimitRule{
		Key:        "rate:" + prefix + ":admin:" + strconv.FormatUint(*ctx.AuthUserID, 10),
		Capacity:   capacity,
		RefillRate: refillRate,
	}, true
}

// Admin batch用のRateLimit ruleを作る。
// Request Bodyを読まず、admin_user_idとpathで手動実行の連打を抑える。
func adminBatchRule(c echo.Context, ctx *AppContext) (usecase.RateLimitRule, bool) {
	rule, ok := adminRule(ctx, "admin_batch", 3, 1.0/1200.0)
	if !ok {
		return usecase.RateLimitRule{}, false
	}

	rule.Key = rule.Key + ":" + safePathKey(c.Path())
	return rule, true
}

// pathをRateLimit keyに使える安全な文字列へ変換。
// 空白や制御文字を含めず、未知文字はアンダースコアに置き換える。
func safePathKey(path string) string {
	if path == "" {
		return "unknown"
	}

	buf := make([]rune, 0, len(path))
	for _, r := range path {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '/' || r == ':' {
			buf = append(buf, r)
			continue
		}
		buf = append(buf, '_')
	}

	return string(buf)
}

// RateLimit結果をResponse Headerへ設定。
// 429時にクライアントが再試行間隔を判断できるようRetry-Afterも付ける。
func setRateLimitHeaders(c echo.Context, name RateLimitName, result repository.RateLimitResult) {
	h := c.Response().Header()
	h.Set("X-RateLimit-Name", string(name))
	if result.Remaining >= 0 {
		h.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	}
	if result.RetryAfter > 0 {
		retryAfterSec := int(result.RetryAfter.Seconds())
		if retryAfterSec <= 0 {
			retryAfterSec = 1
		}
		h.Set("Retry-After", strconv.Itoa(retryAfterSec))
	}
}
