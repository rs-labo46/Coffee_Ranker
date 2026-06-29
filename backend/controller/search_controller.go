package controller

import (
	"net/http"

	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type SearchController struct {
	search    *usecase.SearchUsecase
	validator *validator.SearchValidator
}

// NewSearchControllerを生成してDI層やRouterから使えるようにする。
func NewSearchController(search *usecase.SearchUsecase, validator *validator.SearchValidator) *SearchController {
	// 検索ControllerはQueryを読み取り、Validatorで安全な検索条件にしてUsecaseへ渡す。
	return &SearchController{search: search, validator: validator}
}

// Bean検索queryを読み取り、Validatorで検証してSearchUsecaseへ渡す。
// re_searchイベントはここでは作らず、POST /eventsで記録。
func (h *SearchController) SearchBeans(c echo.Context) error {
	query, err := h.beanSearchQuery(c)
	if err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に正規化。
	valid, err := h.validator.BeanSearch(query)
	if err != nil {
		return writeError(c, err)
	}
	beans, err := h.search.SearchBeans(c.Request().Context(), usecase.BeanSearchInput{Keyword: valid.Keyword, Origin: valid.Origin, RoastLevel: valid.RoastLevel, Acidity: valid.Acidity, Bitterness: valid.Bitterness, Flavor: valid.Flavor, Aroma: valid.Aroma, Body: valid.Body, Sort: valid.Sort, Page: usecase.Page{Limit: valid.Page.Limit, Offset: valid.Page.Offset}})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, beans)
}

// Article検索queryを読み取り、Validatorで検証してSearchUsecaseへ渡す。
// 検索結果の有無やランキング順の取得はUsecase/Repositoryで行う。
func (h *SearchController) SearchArticles(c echo.Context) error {
	query, err := h.articleSearchQuery(c)
	if err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に正規化。
	valid, err := h.validator.ArticleSearch(query)
	if err != nil {
		return writeError(c, err)
	}
	articles, err := h.search.SearchArticles(c.Request().Context(), usecase.ArticleSearchInput{Keyword: valid.Keyword, Category: valid.Category, Sort: valid.Sort, Page: usecase.Page{Limit: valid.Page.Limit, Offset: valid.Page.Offset}})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, articles)
}

// beanSearchQuery APIのHTTP境界処理。
// Bind、Validator、認証済みID取得、Usecase呼び出し、HTTPレスポンス変換を行う。
func (h *SearchController) beanSearchQuery(c echo.Context) (validator.BeanSearchQuery, error) {
	acidity, err := parseIntQuery(c, "acidity")
	if err != nil {
		return validator.BeanSearchQuery{}, err
	}
	bitterness, err := parseIntQuery(c, "bitterness")
	if err != nil {
		return validator.BeanSearchQuery{}, err
	}
	flavor, err := parseIntQuery(c, "flavor")
	if err != nil {
		return validator.BeanSearchQuery{}, err
	}
	aroma, err := parseIntQuery(c, "aroma")
	if err != nil {
		return validator.BeanSearchQuery{}, err
	}
	body, err := parseIntQuery(c, "body")
	if err != nil {
		return validator.BeanSearchQuery{}, err
	}
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return validator.BeanSearchQuery{}, err
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return validator.BeanSearchQuery{}, err
	}
	return validator.BeanSearchQuery{Q: c.QueryParam("q"), Origin: c.QueryParam("origin"), RoastLevel: c.QueryParam("roast_level"), Acidity: acidity, Bitterness: bitterness, Flavor: flavor, Aroma: aroma, Body: body, Sort: c.QueryParam("sort"), Limit: limit, Offset: offset}, nil
}

// articleSearchQuery APIのHTTP境界処理。
// Bind、Validator、認証済みID取得、Usecase呼び出し、HTTPレスポンス変換を行う。
func (h *SearchController) articleSearchQuery(c echo.Context) (validator.ArticleSearchQuery, error) {
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return validator.ArticleSearchQuery{}, err
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return validator.ArticleSearchQuery{}, err
	}
	return validator.ArticleSearchQuery{Q: c.QueryParam("q"), Category: c.QueryParam("category"), Sort: c.QueryParam("sort"), Limit: limit, Offset: offset}, nil
}
