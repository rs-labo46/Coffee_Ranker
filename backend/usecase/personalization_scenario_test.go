package usecase

import (
	"context"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// Userごとの行動集計が混ざらず、別々のInterestProfileとして保存されることを確認。
// 例:
// - User A は Ethiopia / light に強く反応した想定。
// - User B は Brazil / dark に強く反応した想定。
// このテストではHTTPは通さず、InterestBatchUsecaseがUser別に集計Repositoryを呼び分け、
// user_id付きのinterest_profilesを別々に作れるかを確認。
func TestPersonalizationScenario_InterestBatchSeparatesUserInterests(t *testing.T) {
	ctx := context.Background()
	userA := uint64(101)
	userB := uint64(202)
	now := time.Now()

	events := &personalizationActionEventRepo{userInterestByUserID: map[uint64][]repository.InterestAggregate{
		userA: {
			{UserID: &userA, Dimension: entity.InterestDimensionOrigin, Value: "Ethiopia", ScoreDelta: 12.5, LastEventAt: now},
			{UserID: &userA, Dimension: entity.InterestDimensionRoastLevel, Value: string(entity.RoastLevelLight), ScoreDelta: 8.0, LastEventAt: now},
		},
		userB: {
			{UserID: &userB, Dimension: entity.InterestDimensionOrigin, Value: "Brazil", ScoreDelta: 10.0, LastEventAt: now},
			{UserID: &userB, Dimension: entity.InterestDimensionRoastLevel, Value: string(entity.RoastLevelDark), ScoreDelta: 7.5, LastEventAt: now},
		},
	}}
	profiles := &fakeInterestRepo{}
	runs := &fakeBatchRunRepo{}
	locks := &fakeBatchLockRepo{locked: true}
	audits := &fakeAuditRepo{}
	u := NewInterestBatchUsecase(events, profiles, runs, locks, audits, 0)

	run, err := u.Recalculate(ctx, InterestBatchInput{
		BatchInput: BatchInput{Owner: "personalization-test"},
		UserIDs:    []uint64{userA, userB},
	})
	assertNoError(t, err)

	if run.Status != entity.BatchStatusSuccess {
		t.Fatalf("batch status = %s, want %s", run.Status, entity.BatchStatusSuccess)
	}
	if run.RowsProcessed != 4 {
		t.Fatalf("rows processed = %d, want 4", run.RowsProcessed)
	}
	if len(profiles.bulk) != 4 {
		t.Fatalf("profiles count = %d, want 4", len(profiles.bulk))
	}

	// User A側にはEthiopia/lightの興味が保存されることを確認。
	assertProfileExists(t, profiles.bulk, userA, entity.InterestDimensionOrigin, "Ethiopia")
	assertProfileExists(t, profiles.bulk, userA, entity.InterestDimensionRoastLevel, string(entity.RoastLevelLight))

	// User B側にはBrazil/darkの興味が保存されることを確認。
	assertProfileExists(t, profiles.bulk, userB, entity.InterestDimensionOrigin, "Brazil")
	assertProfileExists(t, profiles.bulk, userB, entity.InterestDimensionRoastLevel, string(entity.RoastLevelDark))

	// 逆方向の組み合わせが存在しないことも確認。
	// ここが通ることで、User A/Bの興味プロフィールが混ざっていないと判断できる。
	assertProfileNotExists(t, profiles.bulk, userA, entity.InterestDimensionOrigin, "Brazil")
	assertProfileNotExists(t, profiles.bulk, userB, entity.InterestDimensionOrigin, "Ethiopia")
}

// Userごとに保存済み除外が分かれることを確認。
// RecommendationUsecaseはInterestProfileで候補を再スコアリングするが、
// User向け推薦では保存済みRankTargetを必ず除外する。
// そのため、User Aでは保存済みなので除外、User Bでは未保存なので表示という個人差を確認する。
func TestPersonalizationScenario_RecommendationSeparatesSavedExclusionByUser(t *testing.T) {
	ctx := context.Background()
	userA := uint64(101)
	userB := uint64(202)
	targetID := uint64(10)

	metrics := &fakeContentMetricRepo{ranking: []*model.ContentMetric{
		{RankTargetID: targetID, Score: 100},
	}}
	saved := &personalizationSavedRepo{savedByUserAndTarget: map[uint64]map[uint64]bool{
		userA: {targetID: true},
		userB: {targetID: false},
	}}
	u := NewRecommendationUsecase(metrics, &fakeInterestRepo{}, saved, &fakeActionEventRepo{}, &fakeBeanRepo{}, &fakeArticleRepo{})

	// User AはtargetIDを保存済みなので、推薦候補から除外される。
	resultA, err := u.List(ctx, RecommendationInput{Actor: Actor{UserID: &userA}, Page: Page{Limit: 10}})
	assertNoError(t, err)
	if len(resultA) != 0 {
		t.Fatalf("user A recommendations = %d, want 0 because target is saved", len(resultA))
	}

	// User Bは同じtargetIDを保存していないため、推薦候補として残る。
	resultB, err := u.List(ctx, RecommendationInput{Actor: Actor{UserID: &userB}, Page: Page{Limit: 10}})
	assertNoError(t, err)
	if len(resultB) != 1 || resultB[0].RankTargetID != targetID {
		t.Fatalf("user B recommendations = %+v, want target %d", resultB, targetID)
	}
}

// Userごとの興味プロフィールによって、同じランキング候補でも推薦順が変わることを確認。
// 以前は保存済み除外だけを見ていたため、InterestProfileを使わない業務ロジック漏れを検出できなかった。
func TestPersonalizationScenario_RecommendationUsesUserInterestProfile(t *testing.T) {
	ctx := context.Background()
	userA := uint64(101)
	userB := uint64(202)
	ethiopia := "Ethiopia"
	brazil := "Brazil"

	metrics := &fakeContentMetricRepo{ranking: []*model.ContentMetric{
		{
			RankTargetID: 1,
			Score:        20,
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

	interests := &personalizationInterestRepo{topByUserID: map[uint64][]*model.InterestProfile{
		userA: {
			{UserID: &userA, Dimension: entity.InterestDimensionOrigin, Value: "Ethiopia", Score: 15},
		},
		userB: {
			{UserID: &userB, Dimension: entity.InterestDimensionOrigin, Value: "Brazil", Score: 15},
		},
	}}

	beans := &fakeBeanRepo{byIDs: map[uint64]*model.Bean{
		101: {ID: 101, Origin: &ethiopia, RoastLevel: entity.RoastLevelLight},
		202: {ID: 202, Origin: &brazil, RoastLevel: entity.RoastLevelDark},
	}}

	u := NewRecommendationUsecase(metrics, interests, &personalizationSavedRepo{}, &fakeActionEventRepo{}, beans, &fakeArticleRepo{})

	resultA, err := u.List(ctx, RecommendationInput{Actor: Actor{UserID: &userA}, Page: Page{Limit: 10}})
	assertNoError(t, err)
	if len(resultA) != 2 || resultA[0].RankTargetID != 1 {
		t.Fatalf("user A recommendations = %+v, want Ethiopia target first", resultA)
	}

	resultB, err := u.List(ctx, RecommendationInput{Actor: Actor{UserID: &userB}, Page: Page{Limit: 10}})
	assertNoError(t, err)
	if len(resultB) != 2 || resultB[0].RankTargetID != 2 {
		t.Fatalf("user B recommendations = %+v, want Brazil target first", resultB)
	}
}

// ModalUsecaseの表示抑制がUserごとに分かれることを確認。
// 現在のModalUsecaseは、候補選定済みのRankTargetIDを受け取って表示可否を判定。
// そのため、このテストでは「User Aには最近表示済みなのでブロック、User Bには未表示なので表示可能」という、actor別のモーダル抑制が正しく効くかを確認する。
func TestPersonalizationScenario_ModalSuppressionIsSeparatedByUser(t *testing.T) {
	ctx := context.Background()
	userA := uint64(101)
	userB := uint64(202)
	targetID := uint64(10)

	suppression := &personalizationModalSuppressionRepo{shownByActorAndTarget: map[string]map[uint64]bool{
		"user:101": {targetID: true},
		"user:202": {targetID: false},
	}}
	blocks := &fakeModalBlockRepo{}
	displays := &fakeModalDisplayRepo{}
	u := NewModalUsecase(
		displays,
		blocks,
		&fakeRankTargetRepo{existsActive: true},
		&personalizationSavedRepo{},
		&fakeActionEventRepo{},
		suppression,
	)

	// User Aには同じ候補が直近表示済みなので、recently_shownとしてブロックされる。
	_, err := u.Show(ctx, ShowModalInput{
		Actor:        Actor{UserID: &userA},
		RankTargetID: targetID,
		Trigger:      entity.ModalTriggerGoodRating,
		PagePath:     "/beans/10",
	})
	assertErrorIs(t, err, entity.ErrModalCandidateNotFound)
	if len(blocks.created) != 1 || blocks.created[0].Reason != entity.ModalBlockRecentlyShown {
		t.Fatalf("block logs = %+v, want recently_shown", blocks.created)
	}

	// User Bには同じ候補をまだ表示していないため、表示ログが作成される。
	log, err := u.Show(ctx, ShowModalInput{
		Actor:        Actor{UserID: &userB},
		RankTargetID: targetID,
		Trigger:      entity.ModalTriggerGoodRating,
		PagePath:     "/beans/10",
	})
	assertNoError(t, err)
	if log == nil || log.UserID == nil || *log.UserID != userB || log.RankTargetID != targetID {
		t.Fatalf("modal log = %+v, want user B target %d", log, targetID)
	}

	// 表示成功後はUser B用のRedis keyに表示済み情報が保存される。
	if !suppression.shownByActorAndTarget["user:202"][targetID] {
		t.Fatalf("user B target %d was not stored as shown", targetID)
	}
}

// fakeActionEventRepoを拡張し、userIDごとに違う興味集計結果を返せるようにする。
// これにより、InterestBatchUsecaseがUserごとに集計を呼び分けているかどうか。
type personalizationActionEventRepo struct {
	fakeActionEventRepo
	userInterestByUserID map[uint64][]repository.InterestAggregate
}

func (f *personalizationActionEventRepo) AggregateUserInterest(ctx context.Context, userID uint64, periodStart time.Time, periodEnd time.Time) ([]repository.InterestAggregate, error) {
	return f.userInterestByUserID[userID], nil
}

// fakeInterestRepoを拡張し、UserIDごとに異なる上位InterestProfileを返せるようにする。
// RecommendationUsecaseがactorごとの興味を本当に使って推薦順を変えているかを確認するため。
type personalizationInterestRepo struct {
	fakeInterestRepo
	topByUserID map[uint64][]*model.InterestProfile
}

func (f *personalizationInterestRepo) ListTopByUser(ctx context.Context, userID uint64, limit int) ([]*model.InterestProfile, error) {
	if f.topByUserID == nil {
		return nil, nil
	}
	return f.topByUserID[userID], nil
}

// fakeSavedRepoを拡張し、UserIDとRankTargetIDの組み合わせごとに保存済み状態を返せるようにする。
// User Aでは除外されるがUser Bでは除外されない、という個人差をテスト。
type personalizationSavedRepo struct {
	fakeSavedRepo
	savedByUserAndTarget map[uint64]map[uint64]bool
}

func (f *personalizationSavedRepo) ListActiveRankTargetIDsByUser(ctx context.Context, userID uint64, rankTargetIDs []uint64) (map[uint64]bool, error) {
	ids := make(map[uint64]bool)
	if f.savedByUserAndTarget == nil {
		return ids, nil
	}
	savedByTarget, ok := f.savedByUserAndTarget[userID]
	if !ok {
		return ids, nil
	}
	for _, id := range rankTargetIDs {
		if savedByTarget[id] {
			ids[id] = true
		}
	}
	return ids, nil
}

func (f *personalizationSavedRepo) ExistsActive(ctx context.Context, userID uint64, rankTargetID uint64) (bool, error) {
	if f.savedByUserAndTarget == nil {
		return false, nil
	}
	savedByTarget, ok := f.savedByUserAndTarget[userID]
	if !ok {
		return false, nil
	}
	return savedByTarget[rankTargetID], nil
}

// fakeModalSuppressionRepoを拡張し、actorKeyごとに表示済み状態を分ける。
// ModalUsecaseではuser:ID / guest:IDのRedis keyを作るため、このfakeでも同じkey単位で状態を持たせる。
type personalizationModalSuppressionRepo struct {
	fakeModalSuppressionRepo
	shownByActorAndTarget map[string]map[uint64]bool
}

func (f *personalizationModalSuppressionRepo) WasShown(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error) {
	if f.shownByActorAndTarget == nil {
		return false, nil
	}
	shownByTarget, ok := f.shownByActorAndTarget[actorKey]
	if !ok {
		return false, nil
	}
	return shownByTarget[rankTargetID], nil
}

func (f *personalizationModalSuppressionRepo) SetShown(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error {
	if f.shownByActorAndTarget == nil {
		f.shownByActorAndTarget = map[string]map[uint64]bool{}
	}
	if f.shownByActorAndTarget[actorKey] == nil {
		f.shownByActorAndTarget[actorKey] = map[uint64]bool{}
	}
	f.shownByActorAndTarget[actorKey][rankTargetID] = true
	return nil
}

// 指定UserのInterestProfileが保存されていることを確認する。
func assertProfileExists(t *testing.T, profiles []*model.InterestProfile, userID uint64, dimension entity.InterestDimension, value string) {
	t.Helper()
	if !profileExists(profiles, userID, dimension, value) {
		t.Fatalf("profile for user=%d dimension=%s value=%s was not found in %+v", userID, dimension, value, profiles)
	}
}

// 指定UserのInterestProfileが保存されていないことを確認する。
func assertProfileNotExists(t *testing.T, profiles []*model.InterestProfile, userID uint64, dimension entity.InterestDimension, value string) {
	t.Helper()
	if profileExists(profiles, userID, dimension, value) {
		t.Fatalf("profile for user=%d dimension=%s value=%s should not exist in %+v", userID, dimension, value, profiles)
	}
}

func profileExists(profiles []*model.InterestProfile, userID uint64, dimension entity.InterestDimension, value string) bool {
	for _, profile := range profiles {
		if profile == nil || profile.UserID == nil {
			continue
		}
		if *profile.UserID == userID && profile.Dimension == dimension && profile.Value == value {
			return true
		}
	}
	return false
}
