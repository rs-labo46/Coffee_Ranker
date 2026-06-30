package usecase

import (
	"context"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// ランキング取得で、ContentMetricに含まれるRankTargetをBean/Articleに振り分け、それぞれの実体を取得することを確認。
func TestRankingUsecaseList_SplitsMetricTargetsAndFetchesEntities(t *testing.T) {
	ctx := context.Background()
	metrics := &fakeContentMetricRepo{ranking: []*model.ContentMetric{
		{RankTargetID: 1, RankTarget: model.RankTarget{ID: 1, ContentType: entity.ContentTypeBean, ContentID: 100}},
		{RankTargetID: 2, RankTarget: model.RankTarget{ID: 2, ContentType: entity.ContentTypeArticle, ContentID: 200}},
	}}
	beans := &fakeBeanRepo{}
	articles := &fakeArticleRepo{}
	u := NewRankingUsecase(metrics, beans, articles)

	result, err := u.List(ctx, nil, Page{})
	assertNoError(t, err)

	if len(result.Beans) != 1 || result.Beans[0].ID != 100 {
		t.Fatalf("beans = %+v", result.Beans)
	}
	if len(result.Articles) != 1 || result.Articles[0].ID != 200 {
		t.Fatalf("articles = %+v", result.Articles)
	}
}

// ログインUser向け推薦で、保存済みRankTargetを推薦結果から除外し、取得件数を多めに確保することを確認。
func TestRecommendationUsecaseList_UserFiltersSavedMetrics(t *testing.T) {
	ctx := context.Background()
	metrics := &fakeContentMetricRepo{ranking: []*model.ContentMetric{{RankTargetID: 1}, {RankTargetID: 2}}}
	saved := &fakeSavedRepo{saved: true}
	u := NewRecommendationUsecase(metrics, &fakeInterestRepo{}, saved, &fakeActionEventRepo{}, &fakeBeanRepo{}, &fakeArticleRepo{})
	userID := uint64(1)

	result, err := u.List(ctx, RecommendationInput{Actor: Actor{UserID: &userID}, Page: Page{Limit: 10}})
	assertNoError(t, err)
	if len(result) != 0 {
		t.Fatalf("recommendations = %d, want saved items filtered out", len(result))
	}
	if metrics.listLimit != 20 {
		t.Fatalf("fetch limit = %d, want doubled 20", metrics.listLimit)
	}
}

// 興味プロフィールとBean属性が一致した候補のscoreが加算され、推薦順が上がることを確認。
// 以前はテストが保存済み除外しか見ておらず、InterestProfileを使わない実装漏れを検出できなかった。
func TestRecommendationUsecaseList_UsesInterestProfilesToRescoreBeanCandidates(t *testing.T) {
	ctx := context.Background()
	userID := uint64(1)
	ethiopia := "Ethiopia"
	brazil := "Brazil"

	metrics := &fakeContentMetricRepo{ranking: []*model.ContentMetric{
		{
			RankTargetID: 1,
			Score:        10,
			RankTarget: model.RankTarget{
				ID:          1,
				ContentType: entity.ContentTypeBean,
				ContentID:   101,
			},
		},
		{
			RankTargetID: 2,
			Score:        20,
			RankTarget: model.RankTarget{
				ID:          2,
				ContentType: entity.ContentTypeBean,
				ContentID:   202,
			},
		},
	}}

	interests := &fakeInterestRepo{
		topUser: []*model.InterestProfile{
			{
				UserID:    &userID,
				Dimension: entity.InterestDimensionOrigin,
				Value:     "Ethiopia",
				Score:     30,
			},
		},
	}

	beans := &fakeBeanRepo{
		byIDs: map[uint64]*model.Bean{
			101: {ID: 101, Origin: &ethiopia, RoastLevel: entity.RoastLevelLight},
			202: {ID: 202, Origin: &brazil, RoastLevel: entity.RoastLevelDark},
		},
	}

	u := NewRecommendationUsecase(
		metrics,
		interests,
		&fakeSavedRepo{},
		&fakeActionEventRepo{},
		beans,
		&fakeArticleRepo{},
	)

	result, err := u.List(ctx, RecommendationInput{
		Actor: Actor{UserID: &userID},
		Page:  Page{Limit: 10},
	})
	assertNoError(t, err)

	if len(result) != 2 {
		t.Fatalf("recommendations = %d, want 2", len(result))
	}
	if result[0].RankTargetID != 1 {
		t.Fatalf("first rank_target_id = %d, want 1", result[0].RankTargetID)
	}
	if result[0].Score <= result[1].Score {
		t.Fatalf("scores = %f, %f, want personalized first score higher", result[0].Score, result[1].Score)
	}
}

// 興味プロフィールに一致した候補には、フロント表示用の推薦理由DTOが付くことを確認。
// score順だけを見るテストでは、理由生成の実装漏れを検出できないため。
func TestRecommendationUsecaseList_ReturnsRecommendationReasons(t *testing.T) {
	ctx := context.Background()
	userID := uint64(1)
	ethiopia := "Ethiopia"

	metrics := &fakeContentMetricRepo{ranking: []*model.ContentMetric{
		{
			RankTargetID: 1,
			Score:        10,
			RankTarget: model.RankTarget{
				ID:          1,
				ContentType: entity.ContentTypeBean,
				ContentID:   101,
			},
		},
	}}
	interests := &fakeInterestRepo{topUser: []*model.InterestProfile{
		{UserID: &userID, Dimension: entity.InterestDimensionOrigin, Value: "Ethiopia", Score: 3},
	}}
	beans := &fakeBeanRepo{byIDs: map[uint64]*model.Bean{
		101: {ID: 101, Origin: &ethiopia, RoastLevel: entity.RoastLevelLight},
	}}

	u := NewRecommendationUsecase(metrics, interests, &fakeSavedRepo{}, &fakeActionEventRepo{}, beans, &fakeArticleRepo{})

	result, err := u.List(ctx, RecommendationInput{Actor: Actor{UserID: &userID}, Page: Page{Limit: 10}})
	assertNoError(t, err)

	if len(result) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(result))
	}
	if len(result[0].Reasons) != 1 {
		t.Fatalf("reasons = %+v, want one origin reason", result[0].Reasons)
	}
	if result[0].Reasons[0].Dimension != entity.InterestDimensionOrigin || result[0].Reasons[0].Value != "Ethiopia" {
		t.Fatalf("reason = %+v, want Ethiopia origin reason", result[0].Reasons[0])
	}
	if result[0].InterestScore <= 0 {
		t.Fatalf("interest score = %f, want positive", result[0].InterestScore)
	}
}

// 直近閲覧済みのRankTargetを推薦候補から除外することを確認。
// これにより、閲覧済み除外用に注入したActionEventRepositoryが使われない実装漏れを防ぐ。
func TestRecommendationUsecaseList_ExcludesRecentlyViewedTargets(t *testing.T) {
	ctx := context.Background()
	userID := uint64(1)
	viewedTargetID := uint64(1)

	metrics := &fakeContentMetricRepo{ranking: []*model.ContentMetric{
		{RankTargetID: 1, Score: 20},
		{RankTargetID: 2, Score: 10},
	}}
	events := &fakeActionEventRepo{recent: []*model.ActionEvent{
		{EventType: entity.EventTypeContentView, RankTargetID: &viewedTargetID},
	}}
	u := NewRecommendationUsecase(metrics, &fakeInterestRepo{}, &fakeSavedRepo{}, events, &fakeBeanRepo{}, &fakeArticleRepo{})

	result, err := u.List(ctx, RecommendationInput{Actor: Actor{UserID: &userID}, Page: Page{Limit: 10}})
	assertNoError(t, err)

	if len(result) != 1 {
		t.Fatalf("recommendations = %d, want viewed target excluded", len(result))
	}
	if result[0].RankTargetID != 2 {
		t.Fatalf("rank_target_id = %d, want 2", result[0].RankTargetID)
	}
}

// ランキングバッチで、行動ログ集計結果をContentMetricへ保存し、BatchRun成功更新をTx内で行い、lock解放と監査ログ作成も行うことを確認。
func TestRankingBatchUsecaseRecalculate_SavesMetricsAndMarksRunSuccessInTx(t *testing.T) {
	ctx := context.Background()
	events := &fakeActionEventRepo{contentAggregates: []repository.ContentMetricAggregate{{RankTargetID: 10, ImpressionCount: 10, ClickCount: 2, RatingCount: 2, GoodCount: 1, BadCount: 1}}}
	metrics := &fakeContentMetricRepo{}
	runs := &fakeBatchRunRepo{}
	locks := &fakeBatchLockRepo{locked: true}
	audits := &fakeAuditRepo{}
	tx := &fakeTxManager{repos: fakeTxRepos{metric: metrics, run: runs}}
	u := NewRankingBatchUsecase(events, runs, locks, audits, tx)

	run, err := u.Recalculate(ctx, BatchInput{Owner: "tester"})
	assertNoError(t, err)

	if run.Status != entity.BatchStatusSuccess || run.RowsProcessed != 1 {
		t.Fatalf("run = %+v", run)
	}
	if len(metrics.bulk) != 1 || metrics.bulk[0].RankTargetID != 10 {
		t.Fatalf("metrics = %+v", metrics.bulk)
	}
	if runs.markSuccessID != run.ID {
		t.Fatalf("markSuccessID = %d, want %d", runs.markSuccessID, run.ID)
	}
	if !locks.released {
		t.Fatal("batch lock was not released")
	}
	if len(audits.created) != 1 || audits.created[0].Action != entity.AuditActionRankingBatchRun {
		t.Fatalf("audit = %+v", audits.created)
	}
}

// モーダル表示で、Userがすでに保存済みの候補は表示せず、already_savedのblock logを残すことを確認。
func TestModalUsecaseShow_BlocksSavedTarget(t *testing.T) {
	ctx := context.Background()
	userID := uint64(1)
	blocks := &fakeModalBlockRepo{}
	u := NewModalUsecase(
		&fakeModalDisplayRepo{},
		blocks,
		&fakeRankTargetRepo{existsActive: true},
		&fakeSavedRepo{saved: true},
		&fakeActionEventRepo{},
		&fakeModalSuppressionRepo{},
	)

	_, err := u.Show(ctx, ShowModalInput{Actor: Actor{UserID: &userID}, RankTargetID: 10, Trigger: entity.ModalTriggerFirstVisit, PagePath: "/"})
	assertErrorIs(t, err, entity.ErrModalCandidateNotFound)
	if len(blocks.created) != 1 || blocks.created[0].Reason != entity.ModalBlockAlreadySaved {
		t.Fatalf("block logs = %+v", blocks.created)
	}
}

// モーダルクリックで、表示ログをクリック済みに更新し、modal_click eventを記録することを確認。
func TestModalUsecaseClick_UpdatesDisplayAndWritesEvent(t *testing.T) {
	ctx := context.Background()
	userID := uint64(1)
	displays := &fakeModalDisplayRepo{}
	events := &fakeActionEventRepo{}
	u := NewModalUsecase(displays, &fakeModalBlockRepo{}, &fakeRankTargetRepo{existsActive: true}, &fakeSavedRepo{}, events, &fakeModalSuppressionRepo{})

	err := u.Click(ctx, ModalActionInput{Actor: Actor{UserID: &userID}, ModalDisplayLogID: 5, PagePath: "/"})
	assertNoError(t, err)
	if !displays.clicked {
		t.Fatal("display was not marked clicked")
	}
	if len(events.created) != 1 || events.created[0].EventType != entity.EventTypeModalClick {
		t.Fatalf("events = %+v", events.created)
	}
}

// 興味プロフィール変換で、dimensionが空の集計値を保存対象から除外し、有効期限を設定することを確認。
func TestProfilesFromAggregates_SkipsEmptyDimensionAndSetsExpiry(t *testing.T) {
	now := time.Now()
	userID := uint64(1)
	profiles := profilesFromAggregates([]repository.InterestAggregate{
		{UserID: &userID, Dimension: entity.InterestDimensionOrigin, Value: "Ethiopia", ScoreDelta: 1.2, LastEventAt: now},
		{UserID: &userID, Dimension: "", Value: "ignored", ScoreDelta: 3, LastEventAt: now},
	}, time.Hour, now)

	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(profiles))
	}
	if profiles[0].ExpiresAt == nil {
		t.Fatal("ExpiresAt = nil, want expiry")
	}
}
