package validator

import (
	"testing"

	"coffee-ranker/entity"
)

// ランキング一覧のcontent_typeとページングを検証することを確認。
func TestRankingValidatorList(t *testing.T) {
	v := NewRankingValidator()

	got, err := v.List(RankingQuery{ContentType: "bean"})
	assertNoError(t, err)
	if got.ContentType == nil || *got.ContentType != entity.ContentTypeBean || got.Page.Limit != 20 {
		t.Fatalf("unexpected ranking query: %+v", got)
	}

	got, err = v.List(RankingQuery{})
	assertNoError(t, err)
	if got.ContentType != nil || got.Page.Limit != 20 {
		t.Fatalf("expected no content_type and default page, got %+v", got)
	}

	_, err = v.List(RankingQuery{ContentType: "coffee"})
	assertErrorIs(t, err, entity.ErrInvalidContentType)

	_, err = v.List(RankingQuery{Limit: 101})
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}

// TOPランキングlimitのデフォルト化と上限超過拒否を検証。
func TestRankingValidatorTop(t *testing.T) {
	v := NewRankingValidator()

	got, err := v.Top(0)
	assertNoError(t, err)
	if got != 10 {
		t.Fatalf("expected default top limit 10, got %d", got)
	}

	got, err = v.Top(100)
	assertNoError(t, err)
	if got != 100 {
		t.Fatalf("expected top limit 100, got %d", got)
	}

	_, err = v.Top(101)
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}

// 推薦一覧のcontent_typeとページング上限50を検証することを確認。
func TestRecommendationValidatorList(t *testing.T) {
	v := NewRecommendationValidator()

	got, err := v.List(RecommendationQuery{ContentType: "article"})
	assertNoError(t, err)
	if got.ContentType == nil || *got.ContentType != entity.ContentTypeArticle || got.Page.Limit != 20 {
		t.Fatalf("unexpected recommendation query: %+v", got)
	}

	_, err = v.List(RecommendationQuery{ContentType: "coffee"})
	assertErrorIs(t, err, entity.ErrInvalidContentType)

	_, err = v.List(RecommendationQuery{Limit: 51})
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}
