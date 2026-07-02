package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
)

// Refresh成功時に行ロック取得・新Token作成・旧Token使用済み化がTx内で行われることを確認する。
// RefreshToken rotationの途中成功を防ぎ、同じRefreshTokenの二重使用を防ぐための重要テスト。
func TestAuthUsecaseRefresh_RotatesTokenWithinTransaction(t *testing.T) {
	ctx := context.Background()
	users := &fakeUserRepo{}
	refreshTokens := &fakeRefreshTokenRepo{forUpdate: activeRefreshToken(10, 7, "family-1")}
	tx := &fakeTxManager{repos: fakeTxRepos{user: users, refresh: refreshTokens}}
	tokens := &fakeTokenManager{hashValue: "old-hash", refreshPlain: "new-plain", refreshHash: "new-hash", accessToken: "new-access"}
	u := NewAuthUsecase(users, refreshTokens, &fakeAuditRepo{}, tx, &fakePasswordHasher{}, tokens, time.Hour)

	result, err := u.Refresh(ctx, "old-plain", AuditMeta{})
	assertNoError(t, err)

	if !tx.called {
		t.Fatal("transaction was not used")
	}
	if len(refreshTokens.created) != 1 {
		t.Fatalf("created refresh token count = %d, want 1", len(refreshTokens.created))
	}
	created := refreshTokens.created[0]
	if created.UserID != 7 || created.FamilyID != "family-1" || created.TokenHash != "new-hash" {
		t.Fatalf("created refresh token = %+v", created)
	}
	if refreshTokens.markedUsedID != 10 {
		t.Fatalf("marked used id = %d, want 10", refreshTokens.markedUsedID)
	}
	if result.AccessToken != "new-access" || result.RefreshToken != "new-plain" || result.User.ID != 7 {
		t.Fatalf("result = %+v", result)
	}
}

// 使用済みRefreshTokenの再利用を検知した時の失効処理を確認する。
// 盗難疑いのあるRefreshToken familyを止め、既存AccessTokenもtoken_versionで無効化する。
func TestAuthUsecaseRefresh_ReuseRevokesFamilyAndIncrementsTokenVersion(t *testing.T) {
	ctx := context.Background()
	usedAt := time.Now().Add(-time.Minute)
	users := &fakeUserRepo{}
	refreshTokens := &fakeRefreshTokenRepo{forUpdate: activeRefreshToken(10, 7, "family-1")}
	refreshTokens.forUpdate.UsedAt = &usedAt
	audits := &fakeAuditRepo{}
	tx := &fakeTxManager{repos: fakeTxRepos{user: users, refresh: refreshTokens}}
	u := NewAuthUsecase(users, refreshTokens, audits, tx, &fakePasswordHasher{}, &fakeTokenManager{hashValue: "old-hash"}, time.Hour)

	_, err := u.Refresh(ctx, "old-plain", AuditMeta{})
	if !errors.Is(err, entity.ErrRefreshTokenReuseDetected) {
		t.Fatalf("error = %v, want ErrRefreshTokenReuseDetected", err)
	}
	if !tx.called {
		t.Fatal("transaction was not used")
	}
	if refreshTokens.revokedFamily != "family-1" {
		t.Fatalf("revoked family = %q", refreshTokens.revokedFamily)
	}
	if users.incrementedID != 7 {
		t.Fatalf("incremented user id = %d, want 7", users.incrementedID)
	}
	if len(refreshTokens.created) != 0 || refreshTokens.markedUsedID != 0 {
		t.Fatalf("reuse path created=%d marked=%d, want no rotation", len(refreshTokens.created), refreshTokens.markedUsedID)
	}
	if len(audits.created) != 1 || audits.created[0].Action != entity.AuditActionRefreshReuseDetected {
		t.Fatalf("audit logs = %+v", audits.created)
	}
}

// 期限切れRefreshTokenでは新Token作成やMarkUsedが行われないことを確認する。
// 期限切れTokenを使ったAccessToken再発行を止め、認証状態の延命を防ぐ。
func TestAuthUsecaseRefresh_ExpiredTokenDoesNotRotate(t *testing.T) {
	ctx := context.Background()
	users := &fakeUserRepo{}
	refreshTokens := &fakeRefreshTokenRepo{forUpdate: activeRefreshToken(10, 7, "family-1")}
	refreshTokens.forUpdate.ExpiresAt = time.Now().Add(-time.Minute)
	tx := &fakeTxManager{repos: fakeTxRepos{user: users, refresh: refreshTokens}}
	u := NewAuthUsecase(users, refreshTokens, nil, tx, &fakePasswordHasher{}, &fakeTokenManager{hashValue: "old-hash"}, time.Hour)

	_, err := u.Refresh(ctx, "old-plain", AuditMeta{})
	if !errors.Is(err, entity.ErrRefreshTokenExpired) {
		t.Fatalf("error = %v, want ErrRefreshTokenExpired", err)
	}
	if len(refreshTokens.created) != 0 || refreshTokens.markedUsedID != 0 {
		t.Fatalf("expired path created=%d marked=%d, want no rotation", len(refreshTokens.created), refreshTokens.markedUsedID)
	}
}

// 通常logoutで現在familyだけを失効することを確認する。
// 通常logoutではtoken_versionを増やさず、他端末のAccessTokenまで止めない。
func TestAuthUsecaseLogoutCurrentFamily_RevokesOnlyCurrentFamily(t *testing.T) {
	ctx := context.Background()
	users := &fakeUserRepo{}
	refreshTokens := &fakeRefreshTokenRepo{byHashUser: activeRefreshToken(10, 7, "family-1")}
	audits := &fakeAuditRepo{}
	u := NewAuthUsecase(users, refreshTokens, audits, nil, &fakePasswordHasher{}, &fakeTokenManager{hashValue: "hash-refresh"}, time.Hour)

	err := u.LogoutCurrentFamily(ctx, 7, "plain-refresh", AuditMeta{})
	assertNoError(t, err)

	if refreshTokens.revokedFamily != "family-1" {
		t.Fatalf("revoked family = %q, want family-1", refreshTokens.revokedFamily)
	}
	if users.incrementedID != 0 {
		t.Fatalf("incremented user id = %d, want 0", users.incrementedID)
	}
	if len(audits.created) != 1 || audits.created[0].Action != entity.AuditActionLogout {
		t.Fatalf("audit logs = %+v", audits.created)
	}
}

// CookieのRefreshToken所有者と認証Userが違う場合に拒否することを確認する。
// bodyのuser_idではなく認証済みUserIDを信頼し、他人のRefreshToken familyを失効できないようにする。
func TestAuthUsecaseLogoutCurrentFamily_RejectsTokenOwnerMismatch(t *testing.T) {
	ctx := context.Background()
	refreshTokens := &fakeRefreshTokenRepo{byHashUser: activeRefreshToken(10, 99, "family-1")}
	u := NewAuthUsecase(&fakeUserRepo{}, refreshTokens, nil, nil, &fakePasswordHasher{}, &fakeTokenManager{hashValue: "hash-refresh"}, time.Hour)

	err := u.LogoutCurrentFamily(ctx, 7, "plain-refresh", AuditMeta{})
	if !errors.Is(err, entity.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if refreshTokens.revokedFamily != "" {
		t.Fatalf("revoked family = %q, want empty", refreshTokens.revokedFamily)
	}
}

// 全端末logoutのTx内処理を確認する。
// 全RefreshToken失効とtoken_version更新を同じTxに入れ、RefreshTokenとAccessTokenの失効状態を一致させる。
func TestAuthUsecaseLogoutAllDevices_RevokesAllRefreshTokensAndIncrementsTokenVersion(t *testing.T) {
	ctx := context.Background()
	users := &fakeUserRepo{}
	refreshTokens := &fakeRefreshTokenRepo{}
	audits := &fakeAuditRepo{}
	tx := &fakeTxManager{repos: fakeTxRepos{user: users, refresh: refreshTokens}}
	u := NewAuthUsecase(users, refreshTokens, audits, tx, &fakePasswordHasher{}, &fakeTokenManager{}, time.Hour)

	err := u.LogoutAllDevices(ctx, 7, AuditMeta{})
	assertNoError(t, err)

	if !tx.called {
		t.Fatal("transaction was not used")
	}
	if refreshTokens.revokedUserID != 7 {
		t.Fatalf("revoked user id = %d, want 7", refreshTokens.revokedUserID)
	}
	if users.incrementedID != 7 {
		t.Fatalf("incremented user id = %d, want 7", users.incrementedID)
	}
	if len(audits.created) != 1 || audits.created[0].Action != entity.AuditActionLogout {
		t.Fatalf("audit logs = %+v", audits.created)
	}
}

// Refresh/Logoutテストで使う有効なRefreshToken modelを作る。
func activeRefreshToken(id uint64, userID uint64, familyID string) *model.RefreshToken {
	return &model.RefreshToken{
		ID:        id,
		UserID:    userID,
		TokenHash: "old-hash",
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(time.Hour),
		User: model.User{
			ID:           userID,
			Role:         entity.UserRoleUser,
			Status:       entity.UserStatusActive,
			TokenVersion: 0,
			PasswordHash: "hash",
			Email:        "user@example.com",
			Name:         "User",
		},
	}
}
