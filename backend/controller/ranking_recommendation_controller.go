package controller

import (
	"net/http"

	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type RankingController struct {
	ranking   *usecase.RankingUsecase
	validator *validator.RankingValidator
}

type RecommendationController struct {
	recommendation *usecase.RecommendationUsecase
	validator      *validator.RecommendationValidator
}

// NewRankingControllerを生成してDI層やRouterから使えるようにする。
func NewRankingController(ranking *usecase.RankingUsecase, validator *validator.RankingValidator) *RankingController {
	// ランキングは表示系で、limit/content_typeの入力境界はValidatorで守る。
	return &RankingController{ranking: ranking, validator: validator}
}

// NewRecommendationControllerを生成してDI層やRouterから使えるように。
func NewRecommendationController(recommendation *usecase.RecommendationUsecase, validator *validator.RecommendationValidator) *RecommendationController {
	// 推薦はUser/Guestどちらでも使うため、actorはContextから取得してUsecaseへ渡す。
	return &RecommendationController{recommendation: recommendation, validator: validator}
}

// ランキング一覧queryを検証してRankingUsecaseへ渡す。
// content_typeとページングだけをController/Validator境界で確認。
func (h *RankingController) List(c echo.Context) error {
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に正規化。
	valid, err := h.validator.List(validator.RankingQuery{ContentType: c.QueryParam("content_type"), Limit: limit, Offset: offset})
	if err != nil {
		return writeError(c, err)
	}
	result, err := h.ranking.List(c.Request().Context(), valid.ContentType, usecase.Page{Limit: valid.Page.Limit, Offset: valid.Page.Offset})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, result)
}

// TOPランキングのlimitを検証して取得。
func (h *RankingController) Top(c echo.Context) error {
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	limit, err = h.validator.Top(limit)
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.ranking.Top(c.Request().Context(), limit)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返。
	return c.JSON(http.StatusOK, items)
}

// User/Guest actorと推薦queryを検証してRecommendationUsecaseへ渡す。
// 保存済み除外や興味スコア反映はUsecaseで判断。
func (h *RecommendationController) List(c echo.Context) error {
	// User/Guestのactorはbodyから受け取らず、Contextから取得。
	// user_id/guest_session_idのなりすましを防ぐため。
	actor, err := actorFromContext(c)
	if err != nil {
		return writeError(c, err)
	}
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に正規化。
	valid, err := h.validator.List(validator.RecommendationQuery{ContentType: c.QueryParam("content_type"), Limit: limit, Offset: offset})
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.recommendation.List(c.Request().Context(), usecase.RecommendationInput{Actor: actor, ContentType: valid.ContentType, Page: usecase.Page{Limit: valid.Page.Limit, Offset: valid.Page.Offset}})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, items)
}
