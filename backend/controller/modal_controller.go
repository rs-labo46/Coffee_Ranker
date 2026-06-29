package controller

import (
	"net/http"

	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type ModalController struct {
	modal     *usecase.ModalUsecase
	validator *validator.ModalValidator
}

// NewModalControllerを生成してDI層やRouterから使えるようにする。
func NewModalController(modal *usecase.ModalUsecase, validator *validator.ModalValidator) *ModalController {
	// モーダル操作はactor条件付き更新が重要なので、UserID/GuestSessionIDはContextから取得。
	return &ModalController{modal: modal, validator: validator}
}

// モーダル表示Requestとactorを検証してModalUsecaseへ渡す。
func (h *ModalController) Show(c echo.Context) error {
	// User/Guestのactorはbodyから受け取らず、Contextから取得。
	// user_id/guest_session_idのなりすましを防ぐ。
	actor, err := actorFromContext(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.ShowModalRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断は行わない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形にす。
	valid, err := h.validator.Show(req)
	if err != nil {
		return writeError(c, err)
	}
	log, err := h.modal.Show(c.Request().Context(), usecase.ShowModalInput{Actor: actor, RankTargetID: valid.RankTargetID, Trigger: valid.Trigger, PagePath: valid.PagePath})
	if err != nil {
		return writeError(c, err)
	}
	// 作成成功としてHTTP 201で返す。
	return c.JSON(http.StatusCreated, log)
}

// モーダルクリックRequestとactorを検証してModalUsecaseへ渡す。
// modal_display_log_id単体を信用せず、actor条件付き更新でIDORを防ぐ。
func (h *ModalController) Click(c echo.Context) error {
	// User/Guestのactorはbodyから受け取らず、Contextから取得。
	// user_id/guest_session_idのなりすましを防ぐ。
	actor, err := actorFromContext(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.ModalActionRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断は行わない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Action(req)
	if err != nil {
		return writeError(c, err)
	}
	if err := h.modal.Click(c.Request().Context(), usecase.ModalActionInput{Actor: actor, ModalDisplayLogID: valid.ModalDisplayLogID, PagePath: valid.PagePath}); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// モーダルクローズRequestとactorを検証してModalUsecaseへ渡す。
// 閉じた履歴や抑制保存はUsecase/Repositoryで処理。
func (h *ModalController) Close(c echo.Context) error {
	// User/Guestのactorはbodyから受け取らず、Contextから取得。
	// user_id/guest_session_idのなりすましを防ぐため。
	actor, err := actorFromContext(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.ModalActionRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断は行わない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Action(req)
	if err != nil {
		return writeError(c, err)
	}
	if err := h.modal.Close(c.Request().Context(), usecase.ModalActionInput{Actor: actor, ModalDisplayLogID: valid.ModalDisplayLogID, PagePath: valid.PagePath}); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
