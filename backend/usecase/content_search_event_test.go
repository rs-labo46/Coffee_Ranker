package usecase

import (
	"context"
	"testing"
	"time"

	"coffee-ranker/entity"
)

// Bean一覧取得で、ページング未指定時にlimit=20/offset=0へ補正されることを確認。
func TestBeanUsecaseList_DefaultsPagination(t *testing.T) {
	ctx := context.Background()
	beans := &fakeBeanRepo{}
	u := NewBeanUsecase(beans, &fakeBeanArticleRepo{})

	_, err := u.List(ctx, Page{})
	assertNoError(t, err)

	if beans.listLimit != 20 || beans.listOffset != 0 {
		t.Fatalf("pagination = limit %d offset %d, want 20/0", beans.listLimit, beans.listOffset)
	}
}

// Article詳細取得で、未ログインの場合は本文を返さずErrLoginRequiredでログイン要求することを確認。
func TestArticleUsecaseGetDetailByID_RequiresAuthentication(t *testing.T) {
	ctx := context.Background()
	u := NewArticleUsecase(&fakeArticleRepo{}, &fakeBeanArticleRepo{})

	_, err := u.GetDetailByID(ctx, 1, false)
	assertErrorIs(t, err, entity.ErrLoginRequired)
}

// Bean検索で、検索キーワードがtrimされ、sort未指定時にscore、limit未指定時に20へ補正されることを確認。
func TestSearchUsecaseSearchBeans_NormalizesStringAndSort(t *testing.T) {
	ctx := context.Background()
	beans := &fakeBeanRepo{}
	u := NewSearchUsecase(beans, &fakeArticleRepo{})
	keyword := "  latte  "

	_, err := u.SearchBeans(ctx, BeanSearchInput{Keyword: &keyword, Sort: "", Page: Page{}})
	assertNoError(t, err)

	if beans.searchFilter.Keyword == nil || *beans.searchFilter.Keyword != "latte" {
		t.Fatalf("keyword = %#v, want trimmed latte", beans.searchFilter.Keyword)
	}
	if beans.searchFilter.Sort != "score" || beans.searchFilter.Limit != 20 {
		t.Fatalf("filter = %+v", beans.searchFilter)
	}
}

// impressionなどの重複防止対象eventで、dedup済みの場合はActionEventを作成せずnilを返すことを確認。
func TestEventUsecaseRecord_DedupDuplicateReturnsNilWithoutCreate(t *testing.T) {
	ctx := context.Background()
	events := &fakeActionEventRepo{}
	dedup := &fakeDedupRepo{setOK: false}
	ranks := &fakeRankTargetRepo{existsActive: true}
	u := NewEventUsecase(events, ranks, dedup)
	userID := uint64(1)
	targetID := uint64(10)

	event, err := u.Record(ctx, RecordEventInput{
		Actor:        Actor{UserID: &userID},
		EventType:    entity.EventTypeImpression,
		RankTargetID: &targetID,
		Placement:    entity.PlacementTop,
		PagePath:     "/",
		DedupKey:     "impression:1",
		DedupTTL:     time.Minute,
	})
	assertNoError(t, err)
	if event != nil {
		t.Fatalf("event = %+v, want nil duplicate", event)
	}
	if len(events.created) != 0 {
		t.Fatalf("created events = %d, want 0", len(events.created))
	}
}

// re_search記録時に、直前の検索条件hashを取得しPreviousConditionHashへ入れて保存することを確認。
func TestEventUsecaseRecord_ReSearchLoadsPreviousHash(t *testing.T) {
	ctx := context.Background()
	previous := "previous-hash"
	events := &fakeActionEventRepo{lastSearch: &previous}
	u := NewEventUsecase(events, &fakeRankTargetRepo{}, &fakeDedupRepo{setOK: true})
	guestID := uint64(9)
	current := "current-hash"

	event, err := u.Record(ctx, RecordEventInput{
		Actor:               Actor{GuestSessionID: &guestID},
		EventType:           entity.EventTypeReSearch,
		SearchConditionHash: &current,
		PagePath:            "/search",
	})
	assertNoError(t, err)
	if event == nil || event.PreviousConditionHash == nil || *event.PreviousConditionHash != previous {
		t.Fatalf("previous hash = %+v, want %q", event, previous)
	}
}
