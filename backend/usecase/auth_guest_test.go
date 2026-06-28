package usecase

import (
	"context"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
)

// Signupでemailが小文字・trimに正規化され、passwordがhash化された値でUser作成されることを確認
func TestAuthUsecaseSignup_NormalizesEmailAndHashesPassword(t *testing.T) {
	ctx := context.Background()
	users := &fakeUserRepo{}
	hasher := &fakePasswordHasher{hash: "hashed-password"}
	u := NewAuthUsecase(users, &fakeRefreshTokenRepo{}, nil, nil, hasher, &fakeTokenManager{}, 24*time.Hour)

	user, err := u.Signup(ctx, SignupInput{Name: "Rin", Email: " USER@Example.COM ", Password: "password123"})
	assertNoError(t, err)

	if user.Email != "user@example.com" {
		t.Fatalf("email = %q, want normalized", user.Email)
	}
	if user.PasswordHash != "hashed-password" {
		t.Fatalf("password hash = %q", user.PasswordHash)
	}
	if users.created == nil {
		t.Fatal("user was not created")
	}
}

// Login成功時にAccessToken/RefreshTokenが発行され、RefreshTokenがDB保存用情報で作成され、login監査ログが残ることを確認。
func TestAuthUsecaseLogin_CreatesRefreshTokenAccessTokenAndAudit(t *testing.T) {
	ctx := context.Background()
	users := &fakeUserRepo{byEmail: &model.User{ID: 7, Email: "user@example.com", PasswordHash: "hash", Status: entity.UserStatusActive}}
	refreshTokens := &fakeRefreshTokenRepo{}
	audits := &fakeAuditRepo{}
	tokens := &fakeTokenManager{refreshPlain: "plain", refreshHash: "hash", familyID: "family", accessToken: "access"}
	u := NewAuthUsecase(users, refreshTokens, audits, nil, &fakePasswordHasher{}, tokens, 24*time.Hour)

	result, err := u.Login(ctx, LoginInput{Email: " USER@example.com ", Password: "password"})
	assertNoError(t, err)

	if result.AccessToken != "access" || result.RefreshToken != "plain" {
		t.Fatalf("tokens = %+v", result)
	}
	if len(refreshTokens.created) != 1 {
		t.Fatalf("refresh tokens created = %d, want 1", len(refreshTokens.created))
	}
	if refreshTokens.created[0].UserID != 7 || refreshTokens.created[0].TokenHash != "hash" || refreshTokens.created[0].FamilyID != "family" {
		t.Fatalf("refresh token = %+v", refreshTokens.created[0])
	}
	if len(audits.created) != 1 || audits.created[0].Action != entity.AuditActionLogin {
		t.Fatalf("audit = %+v", audits.created)
	}
}

// GuestSession keyがない場合、新しいsession keyを発行し、hashだけをDB保存し、作成済みとして返すことを確認。
func TestGuestSessionUsecaseGetOrCreate_CreatesNewSessionWhenKeyMissing(t *testing.T) {
	ctx := context.Background()
	sessions := &fakeGuestSessionRepo{}
	keys := &fakeGuestKeys{plain: "plain-session", hash: "hash-session"}
	u := NewGuestSessionUsecase(sessions, keys, time.Hour)

	result, err := u.GetOrCreateGuestSession(ctx, "")
	assertNoError(t, err)

	if !result.Created || result.SessionKey != "plain-session" {
		t.Fatalf("result = %+v", result)
	}
	if sessions.created == nil || sessions.created.SessionKeyHash != "hash-session" {
		t.Fatalf("created session = %+v", sessions.created)
	}
}

// 有効なGuestSession keyがある場合、新規作成せず既存sessionを延長することを確認。
func TestGuestSessionUsecaseGetOrCreate_TouchesExistingSession(t *testing.T) {
	ctx := context.Background()
	sessions := &fakeGuestSessionRepo{byHash: &model.GuestSession{ID: 3, SessionKeyHash: "hash-session", ExpiresAt: time.Now().Add(time.Hour)}}
	keys := &fakeGuestKeys{hash: "hash-session"}
	u := NewGuestSessionUsecase(sessions, keys, time.Hour)

	result, err := u.GetOrCreateGuestSession(ctx, "plain-session")
	assertNoError(t, err)

	if result.Created {
		t.Fatal("Created = true, want existing session")
	}
	if sessions.touchedID != 3 {
		t.Fatalf("touched id = %d, want 3", sessions.touchedID)
	}
}
