package controller

import (
	"net/http"

	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type EventController struct {
	events    *usecase.EventUsecase
	validator *validator.EventValidator
}

// NewEventControllerを生成してDI層やRouterから使えるようにする。
func NewEventController(events *usecase.EventUsecase, validator *validator.EventValidator) *EventController {
	// 行動ログは入力の形が崩れると集計が壊れるため、必ずValidatorを通してからUsecaseへ渡す。
	return &EventController{events: events, validator: validator}
}

// POST /eventsのbodyとContext actorを検証して行動ログを記録す。
// user_id/guest_session_idはbodyから受け取らず、Contextだけを信用。
func (h *EventController) Record(c echo.Context) error {
	// User/Guestのactorはbodyから受け取らず、Contextから取得。
	// user_id/guest_session_idのなりすましを防ぐ。
	actor, err := actorFromContext(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.EventRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Record(req)
	if err != nil {
		return writeError(c, err)
	}
	userAgent := c.Request().UserAgent()
	meta := auditMeta(c)
	input := usecase.RecordEventInput{
		Actor:                 actor,
		EventType:             valid.EventType,
		RankTargetID:          valid.RankTargetID,
		Placement:             valid.Placement,
		DwellMs:               valid.DwellMs,
		SearchConditionHash:   valid.SearchConditionHash,
		PreviousConditionHash: valid.PreviousConditionHash,
		SearchKeyword:         valid.SearchKeyword,
		SearchOrigin:          valid.SearchOrigin,
		SearchRoastLevel:      valid.SearchRoastLevel,
		SearchAcidity:         valid.SearchAcidity,
		SearchBitterness:      valid.SearchBitterness,
		SearchAroma:           valid.SearchAroma,
		SearchFlavor:          valid.SearchFlavor,
		SearchBody:            valid.SearchBody,
		SearchCategory:        valid.SearchCategory,
		PagePath:              valid.PagePath,
		ReferrerPath:          valid.ReferrerPath,
		UserAgent:             &userAgent,
		IPAddressHash:         meta.IPAddressHash,
		RequestID:             meta.RequestID,
		DedupKey:              valid.DedupKey,
		DedupTTL:              valid.DedupTTL,
	}
	event, err := h.events.Record(c.Request().Context(), input)
	if err != nil {
		return writeError(c, err)
	}
	if event == nil {
		return c.NoContent(http.StatusNoContent)
	}
	// 作成成功としてHTTP 201で返す。
	return c.JSON(http.StatusCreated, event)
}
