package controller

import (
	"net/http"

	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type SavedItemController struct {
	saved     *usecase.SavedItemUsecase
	validator *validator.SavedItemValidator
}

type RatingController struct {
	ratings   *usecase.RatingUsecase
	validator *validator.RatingValidator
}

// NewSavedItemControllerを生成してDI層やRouterから使えるようにする。
func NewSavedItemController(saved *usecase.SavedItemUsecase, validator *validator.SavedItemValidator) *SavedItemController {
	// 保存操作はログインUser専用です。UserIDはbodyではなく認証済みContextから取得します。
	return &SavedItemController{saved: saved, validator: validator}
}

// NewRatingControllerを生成してDI層やRouterから使えるようにする。
func NewRatingController(ratings *usecase.RatingUsecase, validator *validator.RatingValidator) *RatingController {
	// 評価操作もログインUser専用です。scoreはValidatorで+1/-1だけに絞る。
	return &RatingController{ratings: ratings, validator: validator}
}

// 認証済みUserの保存Requestを検証してSavedItemUsecaseへ渡す。
// Guest保存や対象RankTargetの有効性はUsecaseで判断。
func (h *SavedItemController) Save(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.SaveRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断は行わない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に正規化。
	valid, err := h.validator.Save(req)
	if err != nil {
		return writeError(c, err)
	}
	item, err := h.saved.Save(c.Request().Context(), userID, valid.RankTargetID, valid.Placement, valid.PagePath)
	if err != nil {
		return writeError(c, err)
	}
	// 作成成功としてHTTP 201で返す。
	return c.JSON(http.StatusCreated, item)
}

// 認証済みUserの保存解除対象rank_target_idを検証。
// bodyのuser_idは使わず、Contextのuser_idだけをUsecaseへ渡す。
func (h *SavedItemController) Remove(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	rankTargetID, err := parseUintParam(c, "rank_target_id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.RankTargetID(rankTargetID); err != nil {
		return writeError(c, err)
	}
	if err := h.saved.Remove(c.Request().Context(), userID, rankTargetID); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// 認証済みUserの保存一覧ページングを検証。
// 保存データの取得条件はUsecaseへ渡す。
func (h *SavedItemController) List(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
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
	page, err := h.validator.List(validator.PageQuery{Limit: limit, Offset: offset})
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.saved.List(c.Request().Context(), userID, usecase.Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, items)
}

// Exists APIのHTTP境界処理を担当します。
// Bind、Validator、認証済みID取得、Usecase呼び出し、HTTPレスポンス変換を行う。
func (h *SavedItemController) Exists(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	rankTargetID, err := parseUintParam(c, "rank_target_id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.RankTargetID(rankTargetID); err != nil {
		return writeError(c, err)
	}
	exists, err := h.saved.Exists(c.Request().Context(), userID, rankTargetID)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, map[string]bool{"saved": exists})
}

// 認証済みUserの評価Requestを検証してRatingUsecaseへ渡す。
// rating_scoreは+1/-1だけ許可し、Guest不可判定はUsecaseで行う。
func (h *RatingController) Rate(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.RatingRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断は行わない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に正規化。
	valid, err := h.validator.Rate(req)
	if err != nil {
		return writeError(c, err)
	}
	rating, err := h.ratings.Rate(c.Request().Context(), userID, valid.RankTargetID, valid.Score, valid.Placement, valid.PagePath)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, rating)
}

// 認証済みUserの評価削除対象rank_target_idを検証。
// bodyのuser_idは信用せず、Contextのuser_idだけを使う。
func (h *RatingController) Delete(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	rankTargetID, err := parseUintParam(c, "rank_target_id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.RankTargetID(rankTargetID); err != nil {
		return writeError(c, err)
	}
	if err := h.ratings.Delete(c.Request().Context(), userID, rankTargetID); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// 認証済みUserの評価取得対象rank_target_idを検証。
func (h *RatingController) Get(c echo.Context) error {
	// 認証済みUserIDはbodyから受け取らず、MiddlewareがContextへ入れた値だけを使う。
	userID, err := mustUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	rankTargetID, err := parseUintParam(c, "rank_target_id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.RankTargetID(rankTargetID); err != nil {
		return writeError(c, err)
	}
	rating, err := h.ratings.Get(c.Request().Context(), userID, rankTargetID)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, rating)
}
