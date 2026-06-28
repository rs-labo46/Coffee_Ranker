package validator

import (
	"testing"

	"coffee-ranker/entity"
)

// Signup入力のname trim、email小文字化、password長さ検証を行うことを確認。
func TestAuthValidatorSignup(t *testing.T) {
	v := NewAuthValidator()

	got, err := v.Signup(SignupRequest{Name: "  Shoma  ", Email: " USER@EXAMPLE.COM ", Password: "password123"})
	assertNoError(t, err)
	if got.Name != "Shoma" || got.Email != "user@example.com" || got.Password != "password123" {
		t.Fatalf("unexpected normalized signup request: %+v", got)
	}

	_, err = v.Signup(SignupRequest{Name: "<script>bad</script>", Email: "user@example.com", Password: "password123"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Signup(SignupRequest{Name: "Shoma", Email: "invalid", Password: "password123"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Signup(SignupRequest{Name: "Shoma", Email: "user@example.com", Password: "short"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// Login入力ではemail形式とpassword処理可能長だけを確認し、照合はしないことを検証。
func TestAuthValidatorLogin(t *testing.T) {
	v := NewAuthValidator()

	got, err := v.Login(LoginRequest{Email: " USER@EXAMPLE.COM ", Password: "x"})
	assertNoError(t, err)
	if got.Email != "user@example.com" || got.Password != "x" {
		t.Fatalf("unexpected normalized login request: %+v", got)
	}

	_, err = v.Login(LoginRequest{Email: "invalid", Password: "x"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Login(LoginRequest{Email: "user@example.com", Password: ""})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// RefreshTokenをBodyではなくCookieから受け取り、空値だけを止めることを検証。
func TestAuthValidatorRefreshTokenFromCookie(t *testing.T) {
	v := NewAuthValidator()

	assertNoError(t, v.RefreshTokenFromCookie("raw-refresh-token"))
	assertErrorIs(t, v.RefreshTokenFromCookie(""), entity.ErrInvalidToken)
}
