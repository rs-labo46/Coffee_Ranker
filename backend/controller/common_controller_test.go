package controller

import (
	"errors"
	"net/http"
	"testing"

	"coffee-ranker/entity"
)

// 業務エラーが外部へ漏らしてよいHTTP status/codeへ変換されることを確認。
func TestMapError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid input", err: entity.ErrInvalidInput, want: http.StatusBadRequest},
		{name: "unauthorized", err: entity.ErrUnauthorized, want: http.StatusUnauthorized},
		{name: "forbidden", err: entity.ErrForbidden, want: http.StatusForbidden},
		{name: "not found", err: entity.ErrBeanNotFound, want: http.StatusNotFound},
		{name: "conflict", err: entity.ErrEmailAlreadyExists, want: http.StatusConflict},
		{name: "rate limit", err: entity.ErrRateLimited, want: http.StatusTooManyRequests},
		{name: "unknown", err: errors.New("db detail"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, _ := mapError(tt.err)
			if status != tt.want {
				t.Fatalf("status = %d, want %d", status, tt.want)
			}
		})
	}
}

// path paramのIDが正の整数だけ許可され、0や文字列をUsecaseへ渡さないことを確認。
func TestParseUintParam(t *testing.T) {
	e, c, _ := newTestContext(http.MethodGet, "/beans/12", "")
	_ = e
	c.SetParamNames("id")
	c.SetParamValues("12")
	id, err := parseUintParam(c, "id")
	if err != nil || id != 12 {
		t.Fatalf("id = %d, err = %v", id, err)
	}

	c.SetParamValues("0")
	if _, err := parseUintParam(c, "id"); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}

	c.SetParamValues("abc")
	if _, err := parseUintParam(c, "id"); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

// query paramの整数変換で未指定は0、文字列は不正入力になることを確認。
func TestParseIntQuery(t *testing.T) {
	_, c, _ := newTestContext(http.MethodGet, "/items?limit=20", "")
	value, err := parseIntQuery(c, "limit")
	if err != nil || value != 20 {
		t.Fatalf("value = %d, err = %v", value, err)
	}

	_, c, _ = newTestContext(http.MethodGet, "/items?limit=abc", "")
	if _, err := parseIntQuery(c, "limit"); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}

	_, c, _ = newTestContext(http.MethodGet, "/items", "")
	value, err = parseIntQuery(c, "limit")
	if err != nil || value != 0 {
		t.Fatalf("value = %d, err = %v", value, err)
	}
}

// 認証済みUserIDはContextからのみ取得し、未設定ならUnauthorizedになることを確認。
func TestMustUserID(t *testing.T) {
	_, c, _ := newTestContext(http.MethodGet, "/me", "")
	if _, err := mustUserID(c); !errors.Is(err, entity.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}

	setUser(c, 9)
	id, err := mustUserID(c)
	if err != nil || id != 9 {
		t.Fatalf("id = %d, err = %v", id, err)
	}
}

// User/Guestのactorが片方だけContextにある場合だけ許可されることを確認。
func TestActorFromContext(t *testing.T) {
	_, c, _ := newTestContext(http.MethodPost, "/events", "")
	if _, err := actorFromContext(c); !errors.Is(err, entity.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}

	setUser(c, 1)
	actor, err := actorFromContext(c)
	if err != nil || actor.UserID == nil || *actor.UserID != 1 || actor.GuestSessionID != nil {
		t.Fatalf("actor = %#v, err = %v", actor, err)
	}

	setGuest(c, 2)
	if _, err := actorFromContext(c); !errors.Is(err, entity.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized when both actor IDs exist", err)
	}
}

// 監査ログ用メタ情報がContextから取得されることを確認。
func TestAuditMeta(t *testing.T) {
	_, c, _ := newTestContext(http.MethodPost, "/events", "")
	setMeta(c)
	meta := auditMeta(c)
	if meta.RequestID == nil || *meta.RequestID != "req-1" {
		t.Fatalf("request id = %#v", meta.RequestID)
	}
	if meta.IPAddressHash == nil || *meta.IPAddressHash != "ip-hash-1" {
		t.Fatalf("ip hash = %#v", meta.IPAddressHash)
	}
}

// ErrorResponseがJSONで返り、内部error文字列を直接返さないことを確認。
func TestWriteError(t *testing.T) {
	_, c, rec := newTestContext(http.MethodGet, "/x", "")
	if err := writeError(c, entity.ErrInvalidInput); err != nil {
		t.Fatalf("writeError failed: %v", err)
	}
	assertStatus(t, rec, http.StatusBadRequest)
	if rec.Body.String() == "" {
		t.Fatal("empty error body")
	}
}

// 型変換補助がuint、int、stringのContext値をuint64として扱えることを確認。
func TestUint64FromContext(t *testing.T) {
	_, c, _ := newTestContext(http.MethodGet, "/x", "")
	c.Set("a", uint(3))
	c.Set("b", int(4))
	c.Set("c", "5")
	for key, want := range map[string]uint64{"a": 3, "b": 4, "c": 5} {
		got, ok := uint64FromContext(c, key)
		if !ok || got != want {
			t.Fatalf("%s = %d/%v, want %d/true", key, got, ok, want)
		}
	}
	c.Set("bad", -1)
	if _, ok := uint64FromContext(c, "bad"); ok {
		t.Fatal("negative int should not be accepted")
	}
}

// adminMetaがbodyではなくContextの管理者IDを監査情報として使うことを確認。
func TestAdminMeta(t *testing.T) {
	_, c, _ := newTestContext(http.MethodPost, "/admin", "")
	setUser(c, 10)
	setMeta(c)
	meta, err := adminMeta(c)
	if err != nil {
		t.Fatalf("adminMeta failed: %v", err)
	}
	if meta.AdminUserID != 10 {
		t.Fatalf("admin user id = %d, want 10", meta.AdminUserID)
	}
	if meta.AuditMeta.RequestID == nil || *meta.AuditMeta.RequestID != "req-1" {
		t.Fatalf("request id = %#v", meta.AuditMeta.RequestID)
	}
}
