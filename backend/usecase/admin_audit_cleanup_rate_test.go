package usecase

import (
	"context"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/repository"
)

// 監査ログ一覧取得で、limit未指定時にデフォルト値が入ること、過大limit/offsetが不正ページングとして弾かれることを確認。
func TestAdminAuditUsecaseList_DefaultLimitAndRejectsInvalidPagination(t *testing.T) {
	ctx := context.Background()
	audits := &fakeAuditRepo{}
	u := NewAdminAuditUsecase(audits)

	_, err := u.List(ctx, repository.AuditLogFilter{})
	assertNoError(t, err)
	if audits.lastFilter.Limit != 20 {
		t.Fatalf("default limit = %d, want 20", audits.lastFilter.Limit)
	}

	_, err = u.List(ctx, repository.AuditLogFilter{Limit: 101})
	assertErrorIs(t, err, entity.ErrInvalidPagination)

	_, err = u.List(ctx, repository.AuditLogFilter{Limit: 20, Offset: 10001})
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}

// 期限切れRefreshToken・GuestSession・InterestProfileを順に削除し、それぞれの削除件数を結果として返すことを確認。
func TestCleanupUsecaseDeleteExpired_ReturnsDeletedCounts(t *testing.T) {
	ctx := context.Background()
	refreshTokens := &fakeRefreshTokenRepo{deleteCount: 2}
	guestSessions := &fakeGuestSessionRepo{deleteCount: 3}
	interests := &fakeInterestRepo{deleteCount: 4}
	u := NewCleanupUsecase(refreshTokens, guestSessions, interests)

	result, err := u.DeleteExpired(ctx)
	assertNoError(t, err)

	if result.RefreshTokensDeleted != 2 || result.GuestSessionsDeleted != 3 || result.InterestDeleted != 4 {
		t.Fatalf("cleanup result = %+v, want 2/3/4", result)
	}
}

// Redis RepositoryがAllowed=falseを返した場合、UsecaseがErrRateLimitedへ変換し、制限keyも正しく渡すことを確認。
func TestRateLimitUsecaseTake_MapsDeniedToRateLimited(t *testing.T) {
	ctx := context.Background()
	rates := &fakeRateLimitRepo{result: repository.RateLimitResult{Allowed: false, Remaining: 0, RetryAfter: time.Second}}
	u := NewRateLimitUsecase(rates)

	result, err := u.Take(ctx, RateLimitRule{Key: "login:ip", Capacity: 3, RefillRate: 0.1}, time.Time{})
	assertErrorIs(t, err, entity.ErrRateLimited)
	if result.Allowed {
		t.Fatal("result.Allowed = true, want false")
	}
	if rates.takeKey != "login:ip" {
		t.Fatalf("rate key = %q, want login:ip", rates.takeKey)
	}
}

// 管理者がRateLimitをresetしたとき、reset本体は成功し、監査ログ作成が失敗しても本体を失敗にしないことを確認。
func TestAdminRateLimitUsecaseReset_WritesAuditBestEffort(t *testing.T) {
	ctx := context.Background()
	rates := &fakeRateLimitRepo{}
	audits := &fakeAuditRepo{createErr: entity.ErrRepositoryFailed}
	u := NewAdminRateLimitUsecase(rates, audits)

	err := u.Reset(ctx, "login:ip:127.0.0.1", AdminMeta{AdminUserID: 99})
	assertNoError(t, err)

	if rates.resetKey != "login:ip:127.0.0.1" {
		t.Fatalf("reset key = %q", rates.resetKey)
	}
	if len(audits.created) != 1 {
		t.Fatalf("audit logs = %d, want 1", len(audits.created))
	}
	if audits.created[0].Action != entity.AuditActionRateLimitReset {
		t.Fatalf("audit action = %q, want %q", audits.created[0].Action, entity.AuditActionRateLimitReset)
	}
}
