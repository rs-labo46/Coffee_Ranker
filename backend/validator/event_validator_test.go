package validator

import (
	"testing"
	"time"

	"coffee-ranker/entity"
)

// content_viewにrank_target_id、placement、page_pathが必要なことを検証。
func TestEventValidatorRecordContentView(t *testing.T) {
	v := NewEventValidator()

	got, err := v.Record(EventRequest{
		EventType:    entity.EventTypeContentView,
		RankTargetID: uint64Ptr(1),
		Placement:    entity.PlacementBeanDetail,
		PagePath:     " /beans/1 ",
	})
	assertNoError(t, err)
	if got.PagePath != "/beans/1" || got.RankTargetID == nil || *got.RankTargetID != 1 {
		t.Fatalf("unexpected content_view event: %+v", got)
	}

	_, err = v.Record(EventRequest{EventType: entity.EventTypeContentView, Placement: entity.PlacementBeanDetail, PagePath: "/beans/1"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// stayでrank_target_idとdwell_ms必須、滞在時間3秒〜30分だけ許可を検証。
func TestEventValidatorRecordStay(t *testing.T) {
	v := NewEventValidator()

	got, err := v.Record(EventRequest{
		EventType:    entity.EventTypeStay,
		RankTargetID: uint64Ptr(1),
		Placement:    entity.PlacementArticleDetail,
		DwellMs:      int64Ptr(3000),
		PagePath:     "/articles/light-roast-guide",
	})
	assertNoError(t, err)
	if got.DwellMs == nil || *got.DwellMs != 3000 {
		t.Fatalf("expected dwell_ms 3000, got %+v", got)
	}

	_, err = v.Record(EventRequest{EventType: entity.EventTypeStay, RankTargetID: uint64Ptr(1), Placement: entity.PlacementArticleDetail, DwellMs: int64Ptr(2999), PagePath: "/articles/a"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Record(EventRequest{EventType: entity.EventTypeStay, RankTargetID: uint64Ptr(1), Placement: entity.PlacementArticleDetail, PagePath: "/articles/a"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// re_searchで検索条件hashを必須にし、placementをsearch_resultへ寄せることを検証。
func TestEventValidatorRecordReSearch(t *testing.T) {
	v := NewEventValidator()

	got, err := v.Record(EventRequest{
		EventType:           entity.EventTypeReSearch,
		SearchConditionHash: stringPtr(" hash-new "),
		SearchKeyword:       stringPtr("  fruity  "),
		SearchRoastLevel:    roastLevelPtr(entity.RoastLevelLight),
		SearchAcidity:       intPtr(4),
		SearchCategory:      stringPtr("brewing"),
		PagePath:            "/search/beans",
	})
	assertNoError(t, err)
	if got.Placement != entity.PlacementSearchResult {
		t.Fatalf("expected search_result placement, got %q", got.Placement)
	}
	if got.SearchKeyword == nil || *got.SearchKeyword != "fruity" {
		t.Fatalf("expected normalized keyword fruity, got %v", got.SearchKeyword)
	}

	_, err = v.Record(EventRequest{EventType: entity.EventTypeReSearch, PagePath: "/search/beans"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Record(EventRequest{EventType: entity.EventTypeReSearch, SearchConditionHash: stringPtr("hash"), SearchAcidity: intPtr(6), PagePath: "/search/beans"})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// save/rating/modal系をPOST /eventsで直接受け付けないことを検証。
func TestEventValidatorRejectsIndirectEvents(t *testing.T) {
	v := NewEventValidator()

	indirectTypes := []entity.EventType{entity.EventTypeSave, entity.EventTypeRating, entity.EventTypeModalImpression, entity.EventTypeModalClick, entity.EventTypeModalClose}
	for _, eventType := range indirectTypes {
		_, err := v.Record(EventRequest{EventType: eventType, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, PagePath: "/"})
		assertErrorIs(t, err, entity.ErrInvalidEventType)
	}
}

// rating_score混入、不正page_path、負のdedup TTLを拒否することを検証。
func TestEventValidatorRejectsUnsafeFields(t *testing.T) {
	v := NewEventValidator()

	_, err := v.Record(EventRequest{EventType: entity.EventTypeClick, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, RatingScore: ratingScorePtr(entity.RatingScoreGood), PagePath: "/"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Record(EventRequest{EventType: entity.EventTypeClick, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, PagePath: "https://example.com"})
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = v.Record(EventRequest{EventType: entity.EventTypeClick, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, PagePath: "/", DedupTTLSeconds: -1})
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// dedup_ttl_secondsをtime.Durationへ変換することを検証。
func TestEventValidatorDedupTTL(t *testing.T) {
	v := NewEventValidator()

	got, err := v.Record(EventRequest{EventType: entity.EventTypeImpression, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, PagePath: "/", DedupTTLSeconds: 60})
	assertNoError(t, err)
	if got.DedupTTL != 60*time.Second {
		t.Fatalf("expected 60s dedup ttl, got %s", got.DedupTTL)
	}
}
