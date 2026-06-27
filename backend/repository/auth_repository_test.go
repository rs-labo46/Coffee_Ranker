package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Repositoryテスト用のPostgreSQL接続を作成。
func newRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// 環境変数でRepository統合テストをスキップできるように。
	if os.Getenv("SKIP_REPOSITORY_INTEGRATION_TESTS") == "1" {
		t.Skip("SKIP_REPOSITORY_INTEGRATION_TESTS=1")
	}

	// 未指定の場合は、ローカルDockerのPostgreSQLに接続。
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "host=127.0.0.1 user=myuser password=mypassword dbname=mydb port=5435 sslmode=disable TimeZone=Asia/Tokyo"
	}

	// PostgreSQLへ接続。
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	// GORM内部のsql.DBを取得。
	// 接続数制御や最後のCloseで使う。
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	// search_pathはPostgreSQLの接続。
	// 接続数を1つに固定。
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	// テストごとになschema名を作る。
	// 同じDBを使っても、テスト同士のテーブルやデータが混ざらないように。
	schema := fmt.Sprintf("repo_test_%d", time.Now().UnixNano())

	// テスト専用schemaを作成。
	if err := db.Exec("CREATE SCHEMA " + quoteIdent(schema)).Error; err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	// この接続では、作成したschemaを優先して見るように。
	// 以降のAutoMigrateやRepository操作は、このschema内のテーブルを使う。
	if err := db.Exec("SET search_path TO " + quoteIdent(schema)).Error; err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	// テスト終了時に専用schemaを削除。
	// CASCADEにより、schema内に作られたテーブルもまとめて削除される。
	t.Cleanup(func() {
		_ = db.Exec("DROP SCHEMA IF EXISTS " + quoteIdent(schema) + " CASCADE").Error
		_ = sqlDB.Close()
	})

	// Repositoryテストで使う全テーブルを作成。
	//テスト専用schema内へテーブルを作る。
	if err := db.AutoMigrate(
		&model.User{},
		&model.GuestSession{},
		&model.Bean{},
		&model.Article{},
		&model.RankTarget{},
		&model.RefreshToken{},
		&model.BeanArticle{},
		&model.ModalDisplayLog{},
		&model.ModalBlockLog{},
		&model.ActionEvent{},
		&model.SavedItem{},
		&model.Rating{},
		&model.ContentMetric{},
		&model.InterestProfile{},
		&model.BatchRun{},
		&model.AuditLog{},
	); err != nil {
		t.Fatalf("auto migrate test database: %v", err)
	}

	return db
}

// Repositoryテスト用のUserを作成。
// RefreshTokenや保存、評価など、user_idが必要なテストの事前データとして使う。
func createTestUser(t *testing.T, db *gorm.DB, suffix string) model.User {
	t.Helper()

	user := model.User{
		Name:         "user-" + suffix,
		Email:        "user-" + suffix + "@example.com",
		PasswordHash: "hashed-password",
		Role:         entity.UserRoleUser,
		Status:       entity.UserStatusActive,
	}

	// usersテーブルへテスト用Userを保存。
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	return user
}

// schema名などをSQLに埋め込むとき、意図しないSQLにならないように。
func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// token_versionを2回増やしたとき、DB上の値が2になるか確認。
// AccessTokenの強制失効に使うtoken_versionが正しく加算されるかを見る。
func TestUserRepository_IncrementTokenVersion(t *testing.T) {
	// テスト専用DB。
	db := newRepositoryTestDB(t)

	// Repositoryへ渡すcontext。
	ctx := context.Background()

	// UserRepositoryを作成。
	repo := NewUserRepository(db)

	// token_version確認用のUserを作成。
	user := createTestUser(t, db, "token-version")

	// token_versionを1回増やす。
	if err := repo.IncrementTokenVersion(ctx, user.ID); err != nil {
		t.Fatalf("increment token version first: %v", err)
	}

	// token_versionをもう1回増やす。
	if err := repo.IncrementTokenVersion(ctx, user.ID); err != nil {
		t.Fatalf("increment token version second: %v", err)
	}

	// DBからUserを取得し直す。
	got, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}

	// 2回増やしたので、token_versionが2になっていることを確認。
	if got.TokenVersion != 2 {
		t.Fatalf("token_version = %d, want 2", got.TokenVersion)
	}
}

// RefreshTokenを一度だけ使用済みにできることを確認。
// さらに、RefreshToken取得時にUserが一緒に取得されていることも確認。
func TestRefreshTokenRepository_MarkUsedOnlyOnceAndPreloadUser(t *testing.T) {
	// テスト専用DB。
	db := newRepositoryTestDB(t)

	// Repositoryへ渡すcontext。
	ctx := context.Background()

	// RefreshTokenに紐づけるUserを作成。
	user := createTestUser(t, db, "refresh")

	// RefreshTokenRepositoryを作成。
	repo := NewRefreshTokenRepository(db)

	// テスト内で使う基準時刻。
	now := time.Now().UTC()

	// 使用済みに古いRefreshToken。
	oldToken := model.RefreshToken{
		UserID:    user.ID,
		TokenHash: "old-token-hash",
		FamilyID:  "family-1",
		ExpiresAt: now.Add(24 * time.Hour),
	}

	// 古いRefreshTokenの置き換え先になる新しいRefreshToken。
	newToken := model.RefreshToken{
		UserID:    user.ID,
		TokenHash: "new-token-hash",
		FamilyID:  "family-1",
		ExpiresAt: now.Add(48 * time.Hour),
	}

	// 古いRefreshTokenをDBに作成。
	if err := repo.Create(ctx, &oldToken); err != nil {
		t.Fatalf("create old token: %v", err)
	}

	// 新しいRefreshTokenをDBに作成。
	if err := repo.Create(ctx, &newToken); err != nil {
		t.Fatalf("create new token: %v", err)
	}

	// 古いRefreshTokenに設定使用済み日時。
	usedAt := now.Add(time.Minute)

	// 古いRefreshTokenを使用済みに。
	// used_atとreplaced_by_token_idが保存される。
	if err := repo.MarkUsed(ctx, oldToken.ID, usedAt, newToken.ID); err != nil {
		t.Fatalf("mark used: %v", err)
	}

	// token_hashでRefreshTokenを取得。
	// ForUpdate版なので、実装側ではSELECT FOR UPDATE相当の行ロックを使う。
	locked, err := repo.FindByTokenHashWithUserForUpdate(ctx, oldToken.TokenHash)
	if err != nil {
		t.Fatalf("find token for update: %v", err)
	}

	// RefreshTokenに紐づくUserが一緒に取得されていることを確認。
	if locked.User.ID != user.ID {
		t.Fatalf("preloaded user id = %d, want %d", locked.User.ID, user.ID)
	}

	// used_atが期待どおり保存されていることを確認。
	if locked.UsedAt == nil || !locked.UsedAt.Equal(usedAt) {
		t.Fatalf("used_at = %v, want %v", locked.UsedAt, usedAt)
	}

	// replaced_by_token_idに新しいRefreshTokenのIDが保存されていることを確認。
	if locked.ReplacedByTokenID == nil || *locked.ReplacedByTokenID != newToken.ID {
		t.Fatalf("replaced_by_token_id = %v, want %d", locked.ReplacedByTokenID, newToken.ID)
	}

	// 同じRefreshTokenをもう一度使用済みにしようと。
	// MarkUsedは未使用かつ未失効のtokenだけ更新ため、2回目は更新対象0件になる想定。
	err = repo.MarkUsed(ctx, oldToken.ID, now.Add(2*time.Minute), newToken.ID)

	// 2回目のMarkUsedが成功せず、ErrNoRowsAffectedになることを確認。
	// これにより、RefreshTokenの二重使用や競合を見る。
	if !errors.Is(err, entity.ErrNoRowsAffected) {
		t.Fatalf("second mark used error = %v, want ErrNoRowsAffected", err)
	}
}

// 同じfamily_idのRefreshTokenをまとめて失効できることを確認。
// 有効期限切れRefreshTokenだけを削除できることも確認。
func TestRefreshTokenRepository_RevokeFamilyAndDeleteExpired(t *testing.T) {
	// テスト専用DB。
	db := newRepositoryTestDB(t)

	// Repositoryへ渡すcontext。
	ctx := context.Background()

	// RefreshTokenに紐づけるUserを作成。
	user := createTestUser(t, db, "revoke-family")

	// RefreshTokenRepositoryを作成。
	repo := NewRefreshTokenRepository(db)

	now := time.Now().UTC()

	// 同じfamily_idのtokenを2件、期限切れtokenを1件用意。
	tokens := []model.RefreshToken{
		{UserID: user.ID, TokenHash: "family-token-1", FamilyID: "family-revoke", ExpiresAt: now.Add(time.Hour)},
		{UserID: user.ID, TokenHash: "family-token-2", FamilyID: "family-revoke", ExpiresAt: now.Add(time.Hour)},
		{UserID: user.ID, TokenHash: "expired-token", FamilyID: "family-expired", ExpiresAt: now.Add(-time.Hour)},
	}

	// 用意したRefreshTokenをDBに作成。
	for i := range tokens {
		if err := repo.Create(ctx, &tokens[i]); err != nil {
			t.Fatalf("create token %d: %v", i, err)
		}
	}

	// family失効時に設定失効日時。
	revokedAt := now.Add(time.Minute)

	// family_idがfamily-revokeのRefreshTokenをまとめて失効。
	if err := repo.RevokeFamily(ctx, "family-revoke", revokedAt); err != nil {
		t.Fatalf("revoke family: %v", err)
	}

	// 同じfamily_idだった2件のRefreshTokenにrevoked_atが入ったことを確認。
	for _, hash := range []string{"family-token-1", "family-token-2"} {
		token, err := repo.FindByTokenHash(ctx, hash)
		if err != nil {
			t.Fatalf("find revoked token %s: %v", hash, err)
		}

		// revoked_atが期待した日時になっていることを確認。
		if token.RevokedAt == nil || !token.RevokedAt.Equal(revokedAt) {
			t.Fatalf("revoked_at for %s = %v, want %v", hash, token.RevokedAt, revokedAt)
		}
	}

	// nowより前にexpires_atを過ぎているRefreshTokenを削除。
	deleted, err := repo.DeleteExpired(ctx, now)
	if err != nil {
		t.Fatalf("delete expired: %v", err)
	}

	// 期限切れtokenは1件だけなので、削除件数が1件であることを確認。
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
}

// GuestSessionの最終アクセス日時と有効期限を更新できることを確認。
// 有効期限切れGuestSessionだけを削除できることも確認。
func TestGuestSessionRepository_TouchAndDeleteExpired(t *testing.T) {
	// テスト専用DB。
	db := newRepositoryTestDB(t)

	// Repositoryへ渡すcontext。
	ctx := context.Background()

	// GuestSessionRepositoryを作成。
	repo := NewGuestSessionRepository(db)

	now := time.Now().UTC()

	// 有効なGuestSession。
	active := model.GuestSession{
		SessionKeyHash: "active-session",
		FirstSeenAt:    now,
		LastSeenAt:     now,
		ExpiresAt:      now.Add(time.Hour),
	}

	// すでに期限切れのGuestSession。
	expired := model.GuestSession{
		SessionKeyHash: "expired-session",
		FirstSeenAt:    now.Add(-2 * time.Hour),
		LastSeenAt:     now.Add(-2 * time.Hour),
		ExpiresAt:      now.Add(-time.Hour),
	}

	// 有効なGuestSessionをDBに作成。
	if err := repo.Create(ctx, &active); err != nil {
		t.Fatalf("create active session: %v", err)
	}

	// 期限切れGuestSessionをDBに作成。
	if err := repo.Create(ctx, &expired); err != nil {
		t.Fatalf("create expired session: %v", err)
	}

	// Touchで更新最終アクセス日時と有効期限。
	lastSeen := now.Add(10 * time.Minute)
	expiresAt := now.Add(2 * time.Hour)

	// 有効なGuestSessionの最終アクセス日時と有効期限を更新。
	if err := repo.Touch(ctx, active.ID, lastSeen, expiresAt); err != nil {
		t.Fatalf("touch session: %v", err)
	}

	// session_key_hashでGuestSessionを取得し直す。
	got, err := repo.FindBySessionKeyHash(ctx, active.SessionKeyHash)
	if err != nil {
		t.Fatalf("find session: %v", err)
	}

	// LastSeenAtとExpiresAtがTouchで指定した値に更新されていることを確認。
	if !got.LastSeenAt.Equal(lastSeen) || !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("session times = %v/%v, want %v/%v", got.LastSeenAt, got.ExpiresAt, lastSeen, expiresAt)
	}

	// nowより前にexpires_atを過ぎているGuestSessionを削除。
	deleted, err := repo.DeleteExpired(ctx, now)
	if err != nil {
		t.Fatalf("delete expired sessions: %v", err)
	}

	// 期限切れGuestSessionは1件だけなので、削除件数が1件であることを確認。
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
}
