package middleware

import (
	"testing"

	"coffee-ranker/entity"
)

// 認証Userを保存したとき、GuestSessionが消え、User actor keyが作られることを検証する。
func TestAppContext_SetAuthUserClearsGuestAndBuildsActorKey(t *testing.T) {
	_, c, _ := newMiddlewareTestContext("GET", "/test")

	SetGuestSession(c, 55)
	ctx, ok := GetAppContext(c)
	if !ok || ctx.GuestSessionID == nil || *ctx.GuestSessionID != 55 {
		t.Fatalf("guest session was not stored: %#v", ctx)
	}

	SetAuthUser(c, 10, entity.UserRoleAdmin, 3)
	ctx, ok = GetAppContext(c)
	if !ok {
		t.Fatal("app context was not found")
	}
	if ctx.AuthUserID == nil || *ctx.AuthUserID != 10 {
		t.Fatalf("auth user id = %#v, want 10", ctx.AuthUserID)
	}
	if ctx.AuthRole == nil || *ctx.AuthRole != entity.UserRoleAdmin {
		t.Fatalf("auth role = %#v, want admin", ctx.AuthRole)
	}
	if ctx.TokenVersion == nil || *ctx.TokenVersion != 3 {
		t.Fatalf("token version = %#v, want 3", ctx.TokenVersion)
	}
	if ctx.GuestSessionID != nil {
		t.Fatalf("guest session should be cleared when auth user is set: %#v", ctx.GuestSessionID)
	}

	actorKey, ok := ActorKey(ctx)
	if !ok || actorKey != "user:10" {
		t.Fatalf("actor key = %s, %v, want user:10 true", actorKey, ok)
	}
}

// 認証済みUserが存在する場合、GuestSessionで上書きされないことを検証する。
func TestAppContext_SetGuestSessionDoesNotOverrideAuthUser(t *testing.T) {
	_, c, _ := newMiddlewareTestContext("GET", "/test")

	SetAuthUser(c, 20, entity.UserRoleUser, 1)
	SetGuestSession(c, 99)

	ctx, ok := GetAppContext(c)
	if !ok {
		t.Fatal("app context was not found")
	}
	if ctx.AuthUserID == nil || *ctx.AuthUserID != 20 {
		t.Fatalf("auth user id changed: %#v", ctx.AuthUserID)
	}
	if ctx.GuestSessionID != nil {
		t.Fatalf("guest session should not be stored when auth user exists: %#v", ctx.GuestSessionID)
	}
}

// UserまたはGuestのどちらか一方だけがactorとして成立することを検証する。
func TestAppContext_HasSingleActor(t *testing.T) {
	userID := uint64(1)
	guestID := uint64(2)

	cases := []struct {
		name string
		ctx  *AppContext
		want bool
	}{
		{name: "nil context", ctx: nil, want: false},
		{name: "no actor", ctx: &AppContext{}, want: false},
		{name: "user only", ctx: &AppContext{AuthUserID: &userID}, want: true},
		{name: "guest only", ctx: &AppContext{GuestSessionID: &guestID}, want: true},
		{name: "both", ctx: &AppContext{AuthUserID: &userID, GuestSessionID: &guestID}, want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasSingleActor(tt.ctx); got != tt.want {
				t.Fatalf("HasSingleActor() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Client IP hashが同じsecretでは同じ値、異なるsecretでは別値になることを検証する。
func TestHashClientIP_IsDeterministicAndSecretDependent(t *testing.T) {
	first := HashClientIP("192.0.2.10", "secret-a")
	second := HashClientIP("192.0.2.10", "secret-a")
	third := HashClientIP("192.0.2.10", "secret-b")

	if first == "" {
		t.Fatal("hash should not be empty")
	}
	if first != second {
		t.Fatalf("same ip and secret should produce same hash: %s != %s", first, second)
	}
	if first == third {
		t.Fatal("different secret should produce different hash")
	}
}
