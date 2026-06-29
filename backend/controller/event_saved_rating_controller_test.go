package controller

import (
	"net/http"
	"testing"

	"coffee-ranker/entity"
	"coffee-ranker/usecase"
	"coffee-ranker/validator"
)

// POST /eventsはUser/GuestのactorをContextから必須取得し、bodyのactor情報を信用しないことを確認。
func TestEventControllerRecordRequiresActor(t *testing.T) {
	controller := NewEventController(nil, validator.NewEventValidator())
	body := jsonBody(t, validator.EventRequest{EventType: entity.EventTypeClick, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, PagePath: "/"})
	_, c, rec := newTestContext(http.MethodPost, "/events", body)

	if err := controller.Record(c); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
}

// POST /eventsでsave/rating/modal系イベントを直接受け付けず、400を返すことを確認。
func TestEventControllerRecordRejectsIndirectEvent(t *testing.T) {
	controller := NewEventController(nil, validator.NewEventValidator())
	body := jsonBody(t, validator.EventRequest{EventType: entity.EventTypeSave, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, PagePath: "/"})
	_, c, rec := newTestContext(http.MethodPost, "/events", body)
	setGuest(c, 2)

	if err := controller.Record(c); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// POST /eventsの正常入力でEventUsecaseへ渡され、作成成功時に201を返すことを確認。
func TestEventControllerRecordSuccess(t *testing.T) {
	events := &fakeEventRepo{}
	uc := usecase.NewEventUsecase(events, &fakeRankTargetRepo{}, &fakeDedupRepo{})
	controller := NewEventController(uc, validator.NewEventValidator())
	body := jsonBody(t, validator.EventRequest{EventType: entity.EventTypeClick, RankTargetID: uint64Ptr(1), Placement: entity.PlacementTop, PagePath: "/beans/1"})
	_, c, rec := newTestContext(http.MethodPost, "/events", body)
	setGuest(c, 2)
	setMeta(c)

	if err := controller.Record(c); err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	assertStatus(t, rec, http.StatusCreated)
	if events.created == nil || events.created.GuestSessionID == nil || *events.created.GuestSessionID != 2 {
		t.Fatalf("created event = %#v", events.created)
	}
}

// 保存APIはbodyのuser_idではなくContextの認証済みUserIDを必須にすることを確認。
func TestSavedItemControllerSaveRequiresUserContext(t *testing.T) {
	controller := NewSavedItemController(nil, validator.NewSavedItemValidator())
	body := `{ "user_id": 1, "rank_target_id": 1, "placement": "top", "page_path": "/" }`
	_, c, rec := newTestContext(http.MethodPost, "/saved", body)

	if err := controller.Save(c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
}

// 保存APIの正常入力でContextのUserIDだけをUsecaseへ渡し、201を返すことを確認。
func TestSavedItemControllerSaveSuccess(t *testing.T) {
	uc := usecase.NewSavedItemUsecase(&fakeSavedRepo{}, &fakeRankTargetRepo{}, &fakeEventRepo{})
	controller := NewSavedItemController(uc, validator.NewSavedItemValidator())
	body := jsonBody(t, validator.SaveRequest{RankTargetID: 1, Placement: entity.PlacementTop, PagePath: "/beans/1"})
	_, c, rec := newTestContext(http.MethodPost, "/saved", body)
	setUser(c, 5)

	if err := controller.Save(c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	assertStatus(t, rec, http.StatusCreated)
}

// 保存解除のrank_target_idが不正な場合、Usecaseへ渡さず400を返すことを確認。
func TestSavedItemControllerRemoveInvalidRankTargetID(t *testing.T) {
	controller := NewSavedItemController(nil, validator.NewSavedItemValidator())
	_, c, rec := newTestContext(http.MethodDelete, "/saved/abc", "")
	setUser(c, 5)
	c.SetParamNames("rank_target_id")
	c.SetParamValues("abc")

	if err := controller.Remove(c); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// 評価APIはGood(+1)/Bad(-1)以外をValidatorで拒否し、400を返すことを確認。
func TestRatingControllerRateInvalidScore(t *testing.T) {
	controller := NewRatingController(nil, validator.NewRatingValidator())
	body := jsonBody(t, validator.RatingRequest{RankTargetID: 1, Score: entity.RatingScore(5), Placement: entity.PlacementTop, PagePath: "/beans/1"})
	_, c, rec := newTestContext(http.MethodPost, "/ratings", body)
	setUser(c, 5)

	if err := controller.Rate(c); err != nil {
		t.Fatalf("Rate failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// 評価APIの正常入力でContextのUserIDとrating_scoreをUsecaseへ渡し、200を返すことを確認。
func TestRatingControllerRateSuccess(t *testing.T) {
	uc := usecase.NewRatingUsecase(&fakeRatingRepo{}, &fakeRankTargetRepo{}, &fakeEventRepo{})
	controller := NewRatingController(uc, validator.NewRatingValidator())
	body := jsonBody(t, validator.RatingRequest{RankTargetID: 1, Score: entity.RatingScoreGood, Placement: entity.PlacementTop, PagePath: "/beans/1"})
	_, c, rec := newTestContext(http.MethodPost, "/ratings", body)
	setUser(c, 5)

	if err := controller.Rate(c); err != nil {
		t.Fatalf("Rate failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
}

// 評価取得はrank_target_idの形式を検証し、不正IDなら400を返すことを確認。
func TestRatingControllerGetInvalidRankTargetID(t *testing.T) {
	controller := NewRatingController(nil, validator.NewRatingValidator())
	_, c, rec := newTestContext(http.MethodGet, "/ratings/0", "")
	setUser(c, 5)
	c.SetParamNames("rank_target_id")
	c.SetParamValues("0")

	if err := controller.Get(c); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}
