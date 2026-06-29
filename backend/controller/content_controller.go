package controller

import (
	"net/http"

	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type ContentController struct {
	beans     *usecase.BeanUsecase
	articles  *usecase.ArticleUsecase
	validator *validator.ContentValidator
}

// NewContentControllerを生成してDI層やRouterから使えるようにする。
func NewContentController(beans *usecase.BeanUsecase, articles *usecase.ArticleUsecase, validator *validator.ContentValidator) *ContentController {
	// ControllerはRepositoryを持たず、UsecaseとValidatorだけを受け取る。
	return &ContentController{beans: beans, articles: articles, validator: validator}
}

// Bean一覧queryを検証して公開Bean一覧を取得。
func (h *ContentController) ListBeans(c echo.Context) error {
	page, err := h.listPage(c)
	if err != nil {
		return writeError(c, err)
	}
	beans, err := h.beans.List(c.Request().Context(), usecase.Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, beans)
}

// Bean詳細IDを検証して公開Bean詳細を取得。
func (h *ContentController) GetBean(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.DetailID(id); err != nil {
		return writeError(c, err)
	}
	bean, err := h.beans.GetDetail(c.Request().Context(), id)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, bean)
}

// Bean IDとlimitを検証して関連記事を取得。
// 関連が存在するかどうかはUsecaseで判断。
func (h *ContentController) RelatedArticles(c echo.Context) error {
	beanID, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	limit, err = h.validator.RelatedLimit(limit)
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.beans.RelatedArticles(c.Request().Context(), beanID, limit)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, items)
}

// Article一覧queryを検証して公開Article一覧を取得。
// 詳細本文の閲覧制御はUsecaseで判断。
func (h *ContentController) ListArticles(c echo.Context) error {
	page, err := h.listPage(c)
	if err != nil {
		return writeError(c, err)
	}
	articles, err := h.articles.List(c.Request().Context(), usecase.Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, articles)
}

// Article IDを検証してArticle詳細を取得。
// Guestへ本文を返すかどうかはUsecaseで判断。
func (h *ContentController) GetArticleByID(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.DetailID(id); err != nil {
		return writeError(c, err)
	}
	article, err := h.articles.GetDetailByID(c.Request().Context(), id, optionalUserID(c) != nil)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, article)
}

// slug形式を検証してArticle詳細を取得。
func (h *ContentController) GetArticleBySlug(c echo.Context) error {
	slug, err := h.validator.DetailSlug(c.Param("slug"))
	if err != nil {
		return writeError(c, err)
	}
	article, err := h.articles.GetDetailBySlug(c.Request().Context(), slug, optionalUserID(c) != nil)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, article)
}

// Article IDとlimitを検証して関連Beanを取得。
func (h *ContentController) RelatedBeans(c echo.Context) error {
	articleID, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	limit, err = h.validator.RelatedLimit(limit)
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.articles.RelatedBeans(c.Request().Context(), articleID, limit)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, items)
}

// 一覧系queryのlimit/offsetを共通で読み取る。
func (h *ContentController) listPage(c echo.Context) (validator.PageQuery, error) {
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return validator.PageQuery{}, err
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return validator.PageQuery{}, err
	}
	return h.validator.List(validator.ListQuery{Limit: limit, Offset: offset})
}
