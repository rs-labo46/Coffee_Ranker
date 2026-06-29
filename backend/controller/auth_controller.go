package controller

import (
	"net/http"

	"coffee-ranker/model"
	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type AuthController struct {
	auth      *usecase.AuthUsecase
	validator *validator.AuthValidator
	cookies   CookieConfig
}

type UserResponse struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type AuthResponse struct {
	User        UserResponse `json:"user"`
	AccessToken string       `json:"access_token"`
}

// NewAuthControllerを生成してDI層やRouterから使えるようにする。
// 検証処理そのものは行わず、依存関係をまとめるだけです。
func NewAuthController(auth *usecase.AuthUsecase, validator *validator.AuthValidator, cookies CookieConfig) *AuthController {
	// DIで受け取ったUsecaseとValidatorだけを保持。
	return &AuthController{auth: auth, validator: validator, cookies: cookies}
}

// SignupのJSON bodyを検証してAuthUsecaseへ渡す。
// 成功時も自動ログインせず、User作成結果だけを201で返す。
func (h *AuthController) Signup(c echo.Context) error {
	var req validator.SignupRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断はしない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// Validatorで入力形式・enum・文字数・危険文字だけを検証。
	input, err := h.validator.Signup(req)
	if err != nil {
		return writeError(c, err)
	}
	user, err := h.auth.Signup(c.Request().Context(), usecase.SignupInput{Name: input.Name, Email: input.Email, Password: input.Password})
	if err != nil {
		return writeError(c, err)
	}
	// 作成成功としてHTTP 201で返す。
	return c.JSON(http.StatusCreated, toUserResponse(user))
}

// LoginのJSON bodyを検証してAuthUsecaseへ渡す。
// 成功時はRefreshTokenをCookieへ設定し、token hashは返しません。
func (h *AuthController) Login(c echo.Context) error {
	var req validator.LoginRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断はしない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// Validatorで入力形式・enum・文字数・危険文字だけを検証。
	input, err := h.validator.Login(req)
	if err != nil {
		return writeError(c, err)
	}
	result, err := h.auth.Login(c.Request().Context(), usecase.LoginInput{Email: input.Email, Password: input.Password, Meta: auditMeta(c)})
	if err != nil {
		return writeError(c, err)
	}
	setRefreshCookie(c, h.cookies, result.RefreshToken)
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, AuthResponse{User: toUserResponse(result.User), AccessToken: result.AccessToken})
}

// CookieのRefreshTokenを検証してAccessToken再発行を行う。
// Bodyでtokenを受け取らず、成功時だけ新RefreshToken Cookieを設定。
func (h *AuthController) Refresh(c echo.Context) error {
	token := refreshTokenFromCookie(c)
	if err := h.validator.RefreshTokenFromCookie(token); err != nil {
		clearAuthCookies(c, h.cookies)
		return writeError(c, err)
	}
	result, err := h.auth.Refresh(c.Request().Context(), token, auditMeta(c))
	if err != nil {
		clearAuthCookies(c, h.cookies)
		return writeError(c, err)
	}
	setRefreshCookie(c, h.cookies, result.RefreshToken)
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, AuthResponse{User: toUserResponse(result.User), AccessToken: result.AccessToken})
}

// 認証済みUserとCookieのRefreshTokenを使って現在familyをlogout。
// 成功・失敗に関係なく認証Cookieを削除する境界処理を担当。
func (h *AuthController) Logout(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		clearAuthCookies(c, h.cookies)
		return writeError(c, err)
	}
	if err := h.auth.LogoutCurrentFamily(c.Request().Context(), userID, refreshTokenFromCookie(c), auditMeta(c)); err != nil {
		clearAuthCookies(c, h.cookies)
		return writeError(c, err)
	}
	clearAuthCookies(c, h.cookies)
	return c.NoContent(http.StatusNoContent)
}

// 認証済みUserとCookieのRefreshTokenを使って現在familyをlogout。
// 成功・失敗に関係なく認証Cookieを削除する境界処理を担当。
func (h *AuthController) LogoutAllDevices(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		clearAuthCookies(c, h.cookies)
		return writeError(c, err)
	}
	if err := h.auth.LogoutAllDevices(c.Request().Context(), userID, auditMeta(c)); err != nil {
		clearAuthCookies(c, h.cookies)
		return writeError(c, err)
	}
	clearAuthCookies(c, h.cookies)
	return c.NoContent(http.StatusNoContent)
}

// 認証済みUserIDから自分の情報を取得。
// bodyのuser_idは受け取らず、password hashなどはResponseに含めない。
func (h *AuthController) Me(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	user, err := h.auth.Me(c.Request().Context(), userID)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, toUserResponse(user))
}

// User modelを外部公開用Response DTOへ変換。
// PasswordHashやTokenVersionなど返してはいけない値。
func toUserResponse(user *model.User) UserResponse {
	if user == nil {
		return UserResponse{}
	}
	return UserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: string(user.Role), Status: string(user.Status), CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), UpdatedAt: user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")}
}
