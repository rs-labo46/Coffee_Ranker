package controller

import (
	"net/http"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/usecase"

	"github.com/labstack/echo/v4"
)

type GuestSessionController struct {
	guest   *usecase.GuestSessionUsecase
	cookies CookieConfig
}

type GuestSessionResponse struct {
	ID        uint64 `json:"id"`
	Created   bool   `json:"created"`
	ExpiresAt string `json:"expires_at"`
}

// NewGuestSessionControllerを生成してDI層やRouterから使えるようにする。
func NewGuestSessionController(guest *usecase.GuestSessionUsecase, cookies CookieConfig) *GuestSessionController {
	return &GuestSessionController{guest: guest, cookies: cookies}
}

// Header/Cookieのsession keyからGuestSessionを取得または作成。
// DB上のguest_session_idはクライアントから受け取らない。
func (h *GuestSessionController) GetOrCreate(c echo.Context) error {
	result, err := h.guest.GetOrCreateGuestSession(c.Request().Context(), guestSessionKeyFromRequest(c))
	if err != nil {
		return writeError(c, err)
	}
	setGuestSessionCookie(c, h.cookies, result.SessionKey)
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, guestSessionResponse(result.Session, result.Created))
}

// Touch APIのHTTP境界処理。
// Bind、Validator、認証済みID取得、Usecase呼び出し、HTTPレスポンス変換を行う。
func (h *GuestSessionController) Touch(c echo.Context) error {
	guestID, ok := uint64FromContext(c, ContextGuestSessionIDKey)
	if !ok || guestID == 0 {
		return writeError(c, entity.ErrUnauthorized)
	}
	session, err := h.guest.TouchGuestSession(c.Request().Context(), guestID)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, guestSessionResponse(session, false))
}

// guestSessionResponseで使うController共通処理。
// HTTP境界の補助処理であり、RepositoryやDB操作は行わない。
func guestSessionResponse(session *model.GuestSession, created bool) GuestSessionResponse {
	if session == nil {
		return GuestSessionResponse{}
	}
	return GuestSessionResponse{ID: session.ID, Created: created, ExpiresAt: session.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")}
}
