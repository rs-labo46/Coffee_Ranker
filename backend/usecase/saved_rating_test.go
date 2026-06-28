package usecase

import (
	"context"
	"testing"

	"coffee-ranker/entity"
)

// 保存処理で、RankTargetが有効な場合に保存本体を実行し、成功後にsave eventを記録することを確認。
func TestSavedItemUsecaseSave_ChecksActiveTargetAndWritesEvent(t *testing.T) {
	ctx := context.Background()
	saved := &fakeSavedRepo{}
	ranks := &fakeRankTargetRepo{existsActive: true}
	events := &fakeActionEventRepo{}
	u := NewSavedItemUsecase(saved, ranks, events)

	item, err := u.Save(ctx, 1, 10, entity.PlacementSearchResult, "/search")
	assertNoError(t, err)

	if item == nil || item.UserID != 1 || item.RankTargetID != 10 {
		t.Fatalf("item = %+v", item)
	}
	if len(events.created) != 1 || events.created[0].EventType != entity.EventTypeSave {
		t.Fatalf("events = %+v", events.created)
	}
}

// 保存処理で、RankTargetが無効または存在しない場合は保存せずErrRankTargetNotFoundを返すことを確認。
func TestSavedItemUsecaseSave_RejectsInactiveRankTarget(t *testing.T) {
	ctx := context.Background()
	u := NewSavedItemUsecase(&fakeSavedRepo{}, &fakeRankTargetRepo{existsActive: false}, &fakeActionEventRepo{})

	_, err := u.Save(ctx, 1, 10, entity.PlacementSearchResult, "/search")
	assertErrorIs(t, err, entity.ErrRankTargetNotFound)
}

// 評価処理で、Good(+1)/Bad(-1)以外のscoreをErrInvalidRatingScoreで弾くことを確認。
func TestRatingUsecaseRate_RejectsInvalidScore(t *testing.T) {
	ctx := context.Background()
	u := NewRatingUsecase(&fakeRatingRepo{}, &fakeRankTargetRepo{existsActive: true}, &fakeActionEventRepo{})

	_, err := u.Rate(ctx, 1, 10, entity.RatingScore(5), entity.PlacementBeanDetail, "/beans/10")
	assertErrorIs(t, err, entity.ErrInvalidRatingScore)
}

// 評価処理で、rating event作成が失敗しても評価本体は成功扱いになるbest effort動作を確認。
func TestRatingUsecaseRate_WritesRatingEventBestEffort(t *testing.T) {
	ctx := context.Background()
	ratings := &fakeRatingRepo{}
	ranks := &fakeRankTargetRepo{existsActive: true}
	events := &fakeActionEventRepo{createErr: entity.ErrRepositoryFailed}
	u := NewRatingUsecase(ratings, ranks, events)

	rating, err := u.Rate(ctx, 1, 10, entity.RatingScoreGood, entity.PlacementBeanDetail, "/beans/10")
	assertNoError(t, err)

	if rating == nil || rating.Score != entity.RatingScoreGood {
		t.Fatalf("rating = %+v", rating)
	}
	if len(events.created) != 1 || events.created[0].EventType != entity.EventTypeRating {
		t.Fatalf("events = %+v", events.created)
	}
}
