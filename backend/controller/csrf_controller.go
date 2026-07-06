package controller

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type CSRFController struct {
	cookies CookieConfig
}

type CSRFResponse struct {
	Token string `json:"csrf_token"`
}

func NewCSRFController(cookies CookieConfig) *CSRFController {
	return &CSRFController{cookies: cookies}
}

func (h *CSRFController) Issue(c echo.Context) error {
	token, err := newCSRFToken()
	if err != nil {
		return writeError(c, err)
	}

	expires := time.Now().Add(time.Hour)
	c.SetCookie(&http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		Domain:   h.cookies.Domain,
		MaxAge:   int(time.Hour.Seconds()),
		Expires:  expires,
		HttpOnly: false,
		Secure:   h.cookies.Secure,
		SameSite: h.cookies.SameSite,
	})

	return c.JSON(http.StatusOK, CSRFResponse{Token: token})
}

func newCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
