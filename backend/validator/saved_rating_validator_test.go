package validator

import (
	"testing"

	"coffee-ranker/entity"
)

// 保存Requestでrank_target_id、placement、page_pathだけを検証することを確認。
func TestSavedItemValidatorSave(t *testing.T) {
	v := NewSavedItemValidator()

	got, err := v.Save(SaveRequest{RankTargetID: 1, Placement: entity.PlacementSearchResult, PagePath: " /beans/1 "})
	assertNoError(t, err)
	if got.RankTargetID != 1 || got.PagePath != "/beans/1" {
		t.Fatalf("unexpected save request: %+v", got)
	}

	_, err = v.Save(SaveRequest{RankTargetID: 0, Placement: entity.PlacementSearchResult, PagePath: "/beans/1"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Save(SaveRequest{RankTargetID: 1, Placement: entity.PlacementModal, PagePath: "/beans/1"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Save(SaveRequest{RankTargetID: 1, Placement: entity.PlacementSearchResult, PagePath: "https://example.com"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// 保存解除・確認対象のIDが0でないことを検証。
func TestSavedItemValidatorRankTargetID(t *testing.T) {
	v := NewSavedItemValidator()

	assertNoError(t, v.RankTargetID(1))
	assertErrorIs(t, v.RankTargetID(0), entity.ErrInvalidInput)
}

// 保存一覧ページングを安全な範囲へ正規化することを検証。
func TestSavedItemValidatorList(t *testing.T) {
	v := NewSavedItemValidator()

	got, err := v.List(PageQuery{})
	assertNoError(t, err)
	if got.Limit != 20 || got.Offset != 0 {
		t.Fatalf("expected default saved list page, got %+v", got)
	}

	_, err = v.List(PageQuery{Limit: 101})
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}

// 評価Requestでrank_target_id、Good/Bad score、placement、page_pathを検証することを確認。
func TestRatingValidatorRate(t *testing.T) {
	v := NewRatingValidator()

	got, err := v.Rate(RatingRequest{RankTargetID: 1, Score: entity.RatingScoreGood, Placement: entity.PlacementBeanDetail, PagePath: " /beans/1 "})
	assertNoError(t, err)
	if got.Score != entity.RatingScoreGood || got.PagePath != "/beans/1" {
		t.Fatalf("unexpected rating request: %+v", got)
	}

	_, err = v.Rate(RatingRequest{RankTargetID: 0, Score: entity.RatingScoreGood, Placement: entity.PlacementBeanDetail, PagePath: "/beans/1"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Rate(RatingRequest{RankTargetID: 1, Score: entity.RatingScore(5), Placement: entity.PlacementBeanDetail, PagePath: "/beans/1"})
	assertErrorIs(t, err, entity.ErrInvalidRatingScore)

	_, err = v.Rate(RatingRequest{RankTargetID: 1, Score: entity.RatingScoreBad, Placement: entity.PlacementModal, PagePath: "/beans/1"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// 評価取得・削除対象のrank_target_idが0でないことを検証。
func TestRatingValidatorRankTargetID(t *testing.T) {
	v := NewRatingValidator()

	assertNoError(t, v.RankTargetID(1))
	assertErrorIs(t, v.RankTargetID(0), entity.ErrInvalidInput)
}
