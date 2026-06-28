package validator

import (
	"testing"

	"coffee-ranker/entity"
)

// Bean検索queryのtrim、enum、味覚値、sort、ページング正規化を検証。
func TestSearchValidatorBeanSearch(t *testing.T) {
	v := NewSearchValidator()

	got, err := v.BeanSearch(BeanSearchQuery{
		Q:          "  fruity  ",
		Origin:     "  Ethiopia  ",
		RoastLevel: "medium",
		Acidity:    4,
		Bitterness: 2,
		Flavor:     5,
		Aroma:      3,
		Body:       1,
		Sort:       "popular",
	})
	assertNoError(t, err)
	if got.Keyword == nil || *got.Keyword != "fruity" {
		t.Fatalf("expected keyword fruity, got %v", got.Keyword)
	}
	if got.Origin == nil || *got.Origin != "Ethiopia" {
		t.Fatalf("expected origin Ethiopia, got %v", got.Origin)
	}
	if got.RoastLevel == nil || *got.RoastLevel != entity.RoastLevelMedium {
		t.Fatalf("expected medium roast level, got %v", got.RoastLevel)
	}
	if got.Acidity == nil || *got.Acidity != 4 || got.Sort != "popular" || got.Page.Limit != 20 {
		t.Fatalf("unexpected valid bean search: %+v", got)
	}

	_, err = v.BeanSearch(BeanSearchQuery{RoastLevel: "city"})
	assertErrorIs(t, err, entity.ErrInvalidSearchCondition)

	_, err = v.BeanSearch(BeanSearchQuery{Acidity: 6})
	assertErrorIs(t, err, entity.ErrInvalidSearchCondition)

	_, err = v.BeanSearch(BeanSearchQuery{Q: "<script>bad</script>"})
	assertErrorIs(t, err, entity.ErrInvalidSearchCondition)

	_, err = v.BeanSearch(BeanSearchQuery{Limit: 51})
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}

// Article検索queryのkeyword/category/sort/pageを検証。
func TestSearchValidatorArticleSearch(t *testing.T) {
	v := NewSearchValidator()

	got, err := v.ArticleSearch(ArticleSearchQuery{Q: "  drip  ", Category: "brewing", Sort: "newest"})
	assertNoError(t, err)
	if got.Keyword == nil || *got.Keyword != "drip" {
		t.Fatalf("expected keyword drip, got %v", got.Keyword)
	}
	if got.Category == nil || *got.Category != "brewing" || got.Sort != "newest" || got.Page.Limit != 20 {
		t.Fatalf("unexpected valid article search: %+v", got)
	}

	_, err = v.ArticleSearch(ArticleSearchQuery{Category: "security"})
	assertErrorIs(t, err, entity.ErrInvalidSearchCondition)

	_, err = v.ArticleSearch(ArticleSearchQuery{Sort: "random"})
	assertErrorIs(t, err, entity.ErrInvalidSearchCondition)

	_, err = v.ArticleSearch(ArticleSearchQuery{Offset: 501})
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}
