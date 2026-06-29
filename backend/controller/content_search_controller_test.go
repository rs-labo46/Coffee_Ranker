package controller

import (
	"net/http"
	"testing"

	"coffee-ranker/usecase"
	"coffee-ranker/validator"
)

// Bean一覧でqueryのlimit/offsetを検証し、公開Bean一覧を200で返すことを確認。
func TestContentControllerListBeansSuccess(t *testing.T) {
	controller := NewContentController(usecase.NewBeanUsecase(&fakeBeanRepo{}, &fakeBeanArticleRepo{}), nil, validator.NewContentValidator())
	_, c, rec := newTestContext(http.MethodGet, "/beans?limit=10&offset=0", "")

	if err := controller.ListBeans(c); err != nil {
		t.Fatalf("ListBeans failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
}

// Bean詳細のpath paramが不正な場合、Usecaseへ到達せず400を返すことを確認。
func TestContentControllerGetBeanInvalidID(t *testing.T) {
	controller := NewContentController(nil, nil, validator.NewContentValidator())
	_, c, rec := newTestContext(http.MethodGet, "/beans/abc", "")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	if err := controller.GetBean(c); err != nil {
		t.Fatalf("GetBean failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// Article詳細は未認証ならUsecase側のLoginRequiredをControllerが403へ変換することを確認。
func TestContentControllerGetArticleRequiresLogin(t *testing.T) {
	articles := usecase.NewArticleUsecase(&fakeArticleRepo{}, &fakeBeanArticleRepo{})
	controller := NewContentController(nil, articles, validator.NewContentValidator())
	_, c, rec := newTestContext(http.MethodGet, "/articles/1", "")
	c.SetParamNames("id")
	c.SetParamValues("1")

	if err := controller.GetArticleByID(c); err != nil {
		t.Fatalf("GetArticleByID failed: %v", err)
	}

	assertStatus(t, rec, http.StatusForbidden)
}

// Article slugがURL用の安全な形式でない場合、400を返すことを確認。
func TestContentControllerGetArticleBySlugInvalid(t *testing.T) {
	controller := NewContentController(nil, nil, validator.NewContentValidator())
	_, c, rec := newTestContext(http.MethodGet, "/articles/bad%20slug", "")
	c.SetParamNames("slug")
	c.SetParamValues("bad slug")

	if err := controller.GetArticleBySlug(c); err != nil {
		t.Fatalf("GetArticleBySlug failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// Bean検索queryをValidatorで正規化し、SearchUsecaseへ渡して200を返すことを確認。
func TestSearchControllerSearchBeansSuccess(t *testing.T) {
	controller := NewSearchController(usecase.NewSearchUsecase(&fakeBeanRepo{}, &fakeArticleRepo{}), validator.NewSearchValidator())
	_, c, rec := newTestContext(http.MethodGet, "/search/beans?q=ethiopia&roast_level=medium&limit=10", "")

	if err := controller.SearchBeans(c); err != nil {
		t.Fatalf("SearchBeans failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
}

// Bean検索の数値queryが文字列の場合、Usecaseへ渡さず400を返すことを確認。
func TestSearchControllerSearchBeansInvalidNumber(t *testing.T) {
	controller := NewSearchController(nil, validator.NewSearchValidator())
	_, c, rec := newTestContext(http.MethodGet, "/search/beans?acidity=bad", "")

	if err := controller.SearchBeans(c); err != nil {
		t.Fatalf("SearchBeans failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// Article検索のcategoryが未定義の場合、Usecaseへ渡さず400を返すことを確認。
func TestSearchControllerSearchArticlesInvalidCategory(t *testing.T) {
	controller := NewSearchController(nil, validator.NewSearchValidator())
	_, c, rec := newTestContext(http.MethodGet, "/search/articles?category=hack", "")

	if err := controller.SearchArticles(c); err != nil {
		t.Fatalf("SearchArticles failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}
