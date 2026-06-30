package controller

import (
	"net/http"
	"testing"

	"coffee-ranker/entity"
	"coffee-ranker/usecase"
	"coffee-ranker/validator"
)

// ランキング一覧のcontent_typeが未定義の場合、Usecaseへ渡さず400を返すことを確認。
func TestRankingControllerListInvalidContentType(t *testing.T) {
	controller := NewRankingController(nil, validator.NewRankingValidator())
	_, c, rec := newTestContext(http.MethodGet, "/ranking?content_type=coffee", "")

	if err := controller.List(c); err != nil {
		t.Fatalf("List failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// TOPランキングのlimitが範囲内ならUsecaseへ渡され、200を返すことを確認。
func TestRankingControllerTopSuccess(t *testing.T) {
	uc := usecase.NewRankingUsecase(
		&fakeMetricRepo{},
		&fakeBeanRepo{},
		&fakeArticleRepo{},
	)
	controller := NewRankingController(uc, validator.NewRankingValidator())
	_, c, rec := newTestContext(http.MethodGet, "/ranking/top?limit=5", "")

	if err := controller.Top(c); err != nil {
		t.Fatalf("Top failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
}

// 推薦一覧はUser/Guest actorをContextから必須取得し、未設定なら401を返すことを確認。
func TestRecommendationControllerListRequiresActor(t *testing.T) {
	controller := NewRecommendationController(nil, validator.NewRecommendationValidator())
	_, c, rec := newTestContext(http.MethodGet, "/recommendations?content_type=bean", "")

	if err := controller.List(c); err != nil {
		t.Fatalf("List failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
}

// 推薦一覧の正常入力でGuest actorをUsecaseへ渡し、200を返すことを確認。
func TestRecommendationControllerListGuestSuccess(t *testing.T) {
	uc := usecase.NewRecommendationUsecase(&fakeMetricRepo{},
		&fakeInterestRepo{},
		&fakeSavedRepo{},
		&fakeEventRepo{},
		&fakeBeanRepo{},
		&fakeArticleRepo{})
	controller := NewRecommendationController(uc, validator.NewRecommendationValidator())
	_, c, rec := newTestContext(http.MethodGet, "/recommendations?content_type=bean&limit=5", "")
	setGuest(c, 11)

	if err := controller.List(c); err != nil {
		t.Fatalf("List failed: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
}

// モーダル表示はactorをContextから取得し、未設定なら401を返すことを確認。
func TestModalControllerShowRequiresActor(t *testing.T) {
	controller := NewModalController(nil, validator.NewModalValidator())
	body := jsonBody(t, validator.ShowModalRequest{RankTargetID: 1, Trigger: entity.ModalTriggerFirstVisit, PagePath: "/"})
	_, c, rec := newTestContext(http.MethodPost, "/modal/show", body)

	if err := controller.Show(c); err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
}

// モーダルクリックはmodal_display_log_idの形式を検証し、不正IDなら400を返すことを確認。
func TestModalControllerClickInvalidID(t *testing.T) {
	controller := NewModalController(nil, validator.NewModalValidator())
	body := jsonBody(t, validator.ModalActionRequest{ModalDisplayLogID: 0, PagePath: "/"})
	_, c, rec := newTestContext(http.MethodPost, "/modal/click", body)
	setUser(c, 5)

	if err := controller.Click(c); err != nil {
		t.Fatalf("Click failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// モーダル表示の正常入力で表示ログを作成し、201を返すことを確認。
func TestModalControllerShowSuccess(t *testing.T) {
	uc := usecase.NewModalUsecase(&fakeModalDisplayRepo{}, &fakeModalBlockRepo{}, &fakeRankTargetRepo{}, &fakeSavedRepo{}, &fakeEventRepo{}, &fakeSuppressionRepo{})
	controller := NewModalController(uc, validator.NewModalValidator())
	body := jsonBody(t, validator.ShowModalRequest{RankTargetID: 1, Trigger: entity.ModalTriggerFirstVisit, PagePath: "/beans/1"})
	_, c, rec := newTestContext(http.MethodPost, "/modal/show", body)
	setGuest(c, 11)

	if err := controller.Show(c); err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	assertStatus(t, rec, http.StatusCreated)
}

// 管理者Bean作成はContextの管理者IDを必須にし、bodyのadmin_user_idを信用しないことを確認。
func TestAdminBeanControllerCreateRequiresAdminContext(t *testing.T) {
	controller := NewAdminBeanController(nil, validator.NewAdminBeanValidator())
	body := `{ "admin_user_id": 1, "name": "Bean", "roast_level": "medium" }`
	_, c, rec := newTestContext(http.MethodPost, "/admin/beans", body)

	if err := controller.Create(c); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	assertStatus(t, rec, http.StatusUnauthorized)
}

// 管理者Bean作成の入力が不正な場合、Usecaseへ渡さず400を返すことを確認。
func TestAdminBeanControllerCreateInvalidBody(t *testing.T) {
	controller := NewAdminBeanController(nil, validator.NewAdminBeanValidator())
	body := jsonBody(t, validator.AdminBeanRequest{Name: "", RoastLevel: entity.RoastLevelMedium})
	_, c, rec := newTestContext(http.MethodPost, "/admin/beans", body)
	setUser(c, 1)

	if err := controller.Create(c); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// 管理者監査ログ詳細はpath paramのIDを検証し、不正IDなら400を返すことを確認。
func TestAdminAuditControllerFindByIDInvalid(t *testing.T) {
	controller := NewAdminAuditController(nil, validator.NewAdminAuditValidator())
	_, c, rec := newTestContext(http.MethodGet, "/admin/audits/abc", "")
	c.SetParamNames("id")
	c.SetParamValues("abc")

	if err := controller.FindByID(c); err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}

// RateLimit resetはkeyの形式を検証し、空文字なら400を返すことを確認。
func TestAdminRateLimitControllerResetInvalidKey(t *testing.T) {
	controller := NewAdminRateLimitController(nil, validator.NewAdminRateLimitValidator())
	body := jsonBody(t, validator.RateLimitResetRequest{Key: ""})
	_, c, rec := newTestContext(http.MethodPost, "/admin/rate-limit/reset", body)
	setUser(c, 1)

	if err := controller.Reset(c); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	assertStatus(t, rec, http.StatusBadRequest)
}
