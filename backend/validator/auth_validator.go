package validator

import "coffee-ranker/entity"

type AuthValidator struct{}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewAuthValidatorを生成してDI層やRouterから使えるようにする。
func NewAuthValidator() *AuthValidator {
	return &AuthValidator{}
}

// Signup Requestのname/email/passwordを検証。
// email重複、password hash化、User作成可否はUsecaseで判断。
func (v *AuthValidator) Signup(input SignupRequest) (SignupRequest, error) {
	// nameは表示名として保存するため、空文字・HTML・制御文字を拒否。
	name, err := NormalizeText(input.Name, 1, 50)
	if err != nil {
		return SignupRequest{}, err
	}
	// emailはログインIDなので、小文字化・形式確認をここで行う。
	email, err := ValidateEmail(input.Email)
	if err != nil {
		return SignupRequest{}, err
	}
	// bcrypt等のhash処理に渡せる長さだけ許可。
	// 実際のhash化はUsecaseの責務。
	if err := ValidatePasswordForSignup(input.Password); err != nil {
		return SignupRequest{}, err
	}
	input.Name = name
	input.Email = email
	return input, nil
}

// Login Requestのemail/password形式を検証。
// 認証成功可否やアカウント状態確認はUsecaseで判断。
func (v *AuthValidator) Login(input LoginRequest) (LoginRequest, error) {
	// emailはログインIDなので、小文字化・形式確認をここで行う。
	email, err := ValidateEmail(input.Email)
	if err != nil {
		return LoginRequest{}, err
	}
	// Loginでは空文字と長すぎるpasswordだけを止める。
	// 正しいpasswordかどうかはUsecaseで照合。
	if err := ValidatePasswordForLogin(input.Password); err != nil {
		return LoginRequest{}, err
	}
	input.Email = email
	return input, nil
}

// Cookieから取得したRefreshToken生値が空でないかを検証。
// tokenの有効性、期限切れ、reuse検知はAuthUsecaseで判断。
func (v *AuthValidator) RefreshTokenFromCookie(token string) error {
	// RefreshTokenはbodyではなくCookieから来る前提。
	// 空ならUsecaseへ渡さず、認証不正として止める。
	if token == "" {
		return entity.ErrInvalidToken
	}
	return nil
}
