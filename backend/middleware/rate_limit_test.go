package middleware

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/repository"
	"coffee-ranker/usecase"
)

// RateLimitRepositoryの結果を任意に返すRateLimitMiddleware用のテストfake。
type fakeRateLimiter struct {
	result repository.RateLimitResult
	err    error
	rules  []usecase.RateLimitRule
}

// middlewareが生成したRateLimitRuleを記録し、設定済みのRateLimit結果を返す。
func (f *fakeRateLimiter) Take(ctx context.Context, rule usecase.RateLimitRule, now time.Time) (repository.RateLimitResult, error) {
	f.rules = append(f.rules, rule)
	return f.result, f.err
}

// PublicReadのRateLimit keyがClient IP hashから作られることを検証する。
func TestRateLimitMiddleware_PublicReadUsesIPHash(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/public")
	SetClientIPHash(c, "203.0.113.10", "secret")
	ctx, _ := GetAppContext(c)
	limiter := &fakeRateLimiter{result: repository.RateLimitResult{Allowed: true, Remaining: 11}}

	err := runMiddleware(t, c, okHandler, RateLimitMiddleware(limiter, RateLimitPublicRead))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if len(limiter.rules) != 1 {
		t.Fatalf("rules len = %d, want 1", len(limiter.rules))
	}
	wantKey := "rate:public_read:ip:" + ctx.ClientIPHash
	if limiter.rules[0].Key != wantKey {
		t.Fatalf("rate key = %q, want %q", limiter.rules[0].Key, wantKey)
	}
	if limiter.rules[0].Capacity != 120 || limiter.rules[0].RefillRate != 2.0 {
		t.Fatalf("rule = %#v, want public read defaults", limiter.rules[0])
	}
	if rec.Header().Get("X-RateLimit-Name") != string(RateLimitPublicRead) {
		t.Fatalf("rate limit name header = %q", rec.Header().Get("X-RateLimit-Name"))
	}
	if rec.Header().Get("X-RateLimit-Remaining") != "11" {
		t.Fatalf("remaining header = %q, want 11", rec.Header().Get("X-RateLimit-Remaining"))
	}
}

// User actorとGuest actorでRateLimit keyが分かれることを検証する。
func TestRateLimitMiddleware_ActorRules(t *testing.T) {
	t.Run("user actor", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/events")
		SetAuthUser(c, 9, entity.UserRoleUser, 1)
		limiter := &fakeRateLimiter{result: repository.RateLimitResult{Allowed: true, Remaining: 5}}

		err := runMiddleware(t, c, okHandler, RateLimitMiddleware(limiter, RateLimitEvent))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if limiter.rules[0].Key != "rate:event:user:9" {
			t.Fatalf("key = %q, want rate:event:user:9", limiter.rules[0].Key)
		}
	})

	t.Run("guest actor", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/events")
		SetGuestSession(c, 10)
		limiter := &fakeRateLimiter{result: repository.RateLimitResult{Allowed: true, Remaining: 5}}

		err := runMiddleware(t, c, okHandler, RateLimitMiddleware(limiter, RateLimitSearch))
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if limiter.rules[0].Key != "rate:search:guest:10" {
			t.Fatalf("key = %q, want rate:search:guest:10", limiter.rules[0].Key)
		}
	})
}

// AdminBatchのRateLimit keyが安全化されたpathを含むことを検証する。
func TestRateLimitMiddleware_AdminBatchUsesPath(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodPost, "/admin/batch/ranking-run")
	c.SetPath("/admin/batch/:name")
	SetAuthUser(c, 1, entity.UserRoleAdmin, 1)
	limiter := &fakeRateLimiter{result: repository.RateLimitResult{Allowed: true, Remaining: 2}}

	err := runMiddleware(t, c, okHandler, RateLimitMiddleware(limiter, RateLimitAdminBatch))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if limiter.rules[0].Key != "rate:admin_batch:admin:1:/admin/batch/:name" {
		t.Fatalf("key = %q", limiter.rules[0].Key)
	}
	if limiter.rules[0].Capacity != 3 {
		t.Fatalf("capacity = %d, want 3", limiter.rules[0].Capacity)
	}
}

// RateLimit rule生成に必要なactorやIP hashがない場合に401を返すことを検証する。
func TestRateLimitMiddleware_RejectsWhenRuleCannotBeBuilt(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodPost, "/events")
	limiter := &fakeRateLimiter{result: repository.RateLimitResult{Allowed: true}}

	err := runMiddleware(t, c, okHandler, RateLimitMiddleware(limiter, RateLimitEvent))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if len(limiter.rules) != 0 {
		t.Fatalf("limiter should not be called when rule cannot be built: %#v", limiter.rules)
	}
}

// RateLimit超過時に429とRetry-Afterを返すことを検証する。
func TestRateLimitMiddleware_Returns429AndRetryAfter(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/public")
	SetClientIPHash(c, "203.0.113.11", "secret")
	limiter := &fakeRateLimiter{
		result: repository.RateLimitResult{Allowed: false, Remaining: 0, RetryAfter: 1500 * time.Millisecond},
		err:    entity.ErrRateLimited,
	}

	err := runMiddleware(t, c, okHandler, RateLimitMiddleware(limiter, RateLimitPublicRead))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("Retry-After = %q, want 1", rec.Header().Get("Retry-After"))
	}
}

// RateLimiter依存先エラー時に503を返すことを検証する。
func TestRateLimitMiddleware_Returns503OnLimiterError(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/public")
	SetClientIPHash(c, "203.0.113.12", "secret")
	limiter := &fakeRateLimiter{err: errors.New("redis down")}

	err := runMiddleware(t, c, okHandler, RateLimitMiddleware(limiter, RateLimitPublicRead))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

// RateLimiter未設定時に500を返すことを検証する。
func TestRateLimitMiddleware_NilLimiterReturns500(t *testing.T) {
	_, c, rec := newMiddlewareTestContext(http.MethodGet, "/public")
	SetClientIPHash(c, "203.0.113.13", "secret")

	err := runMiddleware(t, c, okHandler, RateLimitMiddleware(nil, RateLimitPublicRead))
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

// pathをRateLimit keyに使える安全な文字列へ変換することを検証する。
func TestSafePathKey(t *testing.T) {
	if got := safePathKey(""); got != "unknown" {
		t.Fatalf("safePathKey empty = %q, want unknown", got)
	}
	if got := safePathKey("/admin/batch ranking!run"); got != "/admin/batch_ranking_run" {
		t.Fatalf("safePathKey = %q, want sanitized path", got)
	}
}

// AppContext内のactor状態がUser単独またはGuest単独の場合だけ通ることを検証する。
func TestActorMiddleware(t *testing.T) {
	t.Run("single actor passes", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/events")
		SetGuestSession(c, 12)

		err := runMiddleware(t, c, okHandler, ActorMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("no actor is unauthorized", func(t *testing.T) {
		_, c, rec := newMiddlewareTestContext(http.MethodPost, "/events")
		EnsureAppContext(c)

		err := runMiddleware(t, c, okHandler, ActorMiddleware())
		if err != nil {
			t.Fatalf("middleware error: %v", err)
		}
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}
