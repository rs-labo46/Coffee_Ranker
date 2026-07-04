package repository

import (
	"coffee-ranker/entity"
	"coffee-ranker/model"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 行動ログの保存、取得、集計に必要なDB操作。
type IActionEventRepository interface {
	Create(ctx context.Context, event *model.ActionEvent) error
	BulkCreate(ctx context.Context, events []*model.ActionEvent) error
	FindRecentByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ActionEvent, error)
	FindLastSearchHash(ctx context.Context, userID *uint64, guestSessionID *uint64) (*string, error)
	AggregateContentMetrics(ctx context.Context, periodStart time.Time, periodEnd time.Time) ([]ContentMetricAggregate, error)
	AggregateUserInterest(ctx context.Context, userID uint64, periodStart time.Time, periodEnd time.Time) ([]InterestAggregate, error)
	AggregateGuestInterest(ctx context.Context, guestSessionID uint64, periodStart time.Time, periodEnd time.Time) ([]InterestAggregate, error)
	CountByActorAndTypeSince(ctx context.Context, userID *uint64, guestSessionID *uint64, eventType entity.EventType, since time.Time) (int64, error)
	CountByTargetAndTypeSince(ctx context.Context, rankTargetID uint64, eventType entity.EventType, since time.Time) (int64, error)
}

// ユーザーの保存済みコンテンツを作成、解除、取得するDB操作。
type ISavedItemRepository interface {
	Create(ctx context.Context, item *model.SavedItem) error
	FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.SavedItem, error)
	SaveOrRestore(ctx context.Context, userID uint64, rankTargetID uint64, now time.Time) (*model.SavedItem, error)
	Remove(ctx context.Context, userID uint64, rankTargetID uint64, removedAt time.Time) error
	ListActiveByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*model.SavedItem, error)
	ListActiveRankTargetIDsByUser(ctx context.Context, userID uint64, rankTargetIDs []uint64) (map[uint64]bool, error)
	ExistsActive(ctx context.Context, userID uint64, rankTargetID uint64) (bool, error)
	CountActiveByTarget(ctx context.Context, rankTargetID uint64) (int64, error)
}

// GoodやBad評価の登録、更新、削除、集計に必要なDB操作。
type IRatingRepository interface {
	Create(ctx context.Context, rating *model.Rating) error
	FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.Rating, error)
	ListByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*model.Rating, error)
	Upsert(ctx context.Context, userID uint64, rankTargetID uint64, score entity.RatingScore, now time.Time) (*model.Rating, error)
	Delete(ctx context.Context, userID uint64, rankTargetID uint64) error
	CountByTarget(ctx context.Context, rankTargetID uint64) (int64, error)
	AggregateByTarget(ctx context.Context, rankTargetID uint64) (RatingAggregate, error)
}

type GormActionEventRepository struct {
	db *gorm.DB
}

type GormSavedItemRepository struct {
	db *gorm.DB
}

type GormRatingRepository struct {
	db *gorm.DB
}

// 行動ログからランキング指標を作るための集計結果。
type ContentMetricAggregate struct {
	RankTargetID         uint64
	ImpressionCount      int64
	ContentViewCount     int64
	ClickCount           int64
	StayTotalMs          int64
	SaveCount            int64
	RatingCount          int64
	GoodCount            int64
	BadCount             int64
	ReSearchCount        int64
	ModalImpressionCount int64
	ModalClickCount      int64
	ModalCloseCount      int64
}

// 対象コンテンツごとの評価件数と評価割合。
type RatingAggregate struct {
	RatingCount int64
	GoodCount   int64
	BadCount    int64
	RatingAvg   float64
	GoodRate    float64
	BadRate     float64
}

// 行動ログから興味プロフィールを作るための集計結果。
type InterestAggregate struct {
	UserID         *uint64
	GuestSessionID *uint64
	Dimension      entity.InterestDimension
	Value          string
	ScoreDelta     float64
	LastEventAt    time.Time
}

// 行動ログRepositoryのGORM実装。
func NewActionEventRepository(db *gorm.DB) IActionEventRepository {
	return &GormActionEventRepository{db}
}

// 保存済みRepositoryのGORM実装。
func NewSavedItemRepository(db *gorm.DB) ISavedItemRepository {
	return &GormSavedItemRepository{db}
}

// 評価RepositoryのGORM実装。
func NewRatingRepository(db *gorm.DB) IRatingRepository {
	return &GormRatingRepository{db}
}

// 行動ログを1件保存。
func (r *GormActionEventRepository) Create(ctx context.Context, event *model.ActionEvent) error {
	return mapDBError(r.db.WithContext(ctx).Create(event).Error)
}

// 複数の行動ログをまとめて保存。
func (r *GormActionEventRepository) BulkCreate(ctx context.Context, events []*model.ActionEvent) error {
	if len(events) == 0 {
		return nil
	}
	return mapDBError(r.db.WithContext(ctx).Create(&events).Error)
}

// 指定されたユーザーまたはゲストの直近行動ログを取得。
func (r *GormActionEventRepository) FindRecentByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ActionEvent, error) {
	var events []*model.ActionEvent
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ActionEvent{}), userID, guestSessionID)
	if err := db.Order("occurred_at DESC").Limit(limit).Find(&events).Error; err != nil {
		return nil, mapDBError(err)
	}
	return events, nil
}

// 前回検索条件のhashを取得し、初回検索ではnilを返。
func (r *GormActionEventRepository) FindLastSearchHash(ctx context.Context, userID *uint64, guestSessionID *uint64) (*string, error) {
	var event model.ActionEvent
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ActionEvent{}), userID, guestSessionID)

	err := db.
		Where("event_type = ? AND search_condition_hash IS NOT NULL", entity.EventTypeReSearch).
		Order("occurred_at DESC").
		First(&event).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, mapDBError(err)
	}

	return event.SearchConditionHash, nil
}

// 指定期間の行動ログをランキング指標用に集計。
func (r *GormActionEventRepository) AggregateContentMetrics(ctx context.Context, periodStart time.Time, periodEnd time.Time) ([]ContentMetricAggregate, error) {
	var rows []ContentMetricAggregate
	err := r.db.WithContext(ctx).Model(&model.ActionEvent{}).
		Select(`
			rank_target_id,
			COUNT(*) FILTER (WHERE event_type = ?) AS impression_count,
			COUNT(*) FILTER (WHERE event_type = ?) AS content_view_count,
			COUNT(*) FILTER (WHERE event_type = ?) AS click_count,
			COALESCE(SUM(dwell_ms) FILTER (WHERE event_type = ?), 0) AS stay_total_ms,
			COUNT(*) FILTER (WHERE event_type = ?) AS save_count,
			COUNT(*) FILTER (WHERE event_type = ?) AS rating_count,
			COUNT(*) FILTER (WHERE event_type = ? AND rating_score = 1) AS good_count,
			COUNT(*) FILTER (WHERE event_type = ? AND rating_score = -1) AS bad_count,
			COUNT(*) FILTER (WHERE event_type = ?) AS re_search_count,
			COUNT(*) FILTER (WHERE event_type = ?) AS modal_impression_count,
			COUNT(*) FILTER (WHERE event_type = ?) AS modal_click_count,
			COUNT(*) FILTER (WHERE event_type = ?) AS modal_close_count
		`, entity.EventTypeImpression, entity.EventTypeContentView, entity.EventTypeClick, entity.EventTypeStay, entity.EventTypeSave, entity.EventTypeRating, entity.EventTypeRating, entity.EventTypeRating, entity.EventTypeReSearch, entity.EventTypeModalImpression, entity.EventTypeModalClick, entity.EventTypeModalClose).
		Where("occurred_at >= ? AND occurred_at < ? AND rank_target_id IS NOT NULL", periodStart, periodEnd).
		Group("rank_target_id").
		Scan(&rows).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return rows, nil
}

// 指定ユーザーの行動ログを興味プロフィール用に集計。
func (r *GormActionEventRepository) AggregateUserInterest(ctx context.Context, userID uint64, periodStart time.Time, periodEnd time.Time) ([]InterestAggregate, error) {
	return r.aggregateInterest(ctx, &userID, nil, periodStart, periodEnd)
}

// 指定ゲストセッションの行動ログを興味プロフィール用に集計。
func (r *GormActionEventRepository) AggregateGuestInterest(ctx context.Context, guestSessionID uint64, periodStart time.Time, periodEnd time.Time) ([]InterestAggregate, error) {
	return r.aggregateInterest(ctx, nil, &guestSessionID, periodStart, periodEnd)
}

// ユーザーまたはゲストの検索条件を興味スコアへ変換する共通の集計。
// 1つの検索イベントにoriginとroast_levelなど複数条件が含まれる場合、
// それぞれを別dimensionとして集計し、推薦ロジックで使える興味プロフィールを欠落させない。
func (r *GormActionEventRepository) aggregateInterest(ctx context.Context, userID *uint64, guestSessionID *uint64, periodStart time.Time, periodEnd time.Time) ([]InterestAggregate, error) {
	actorWhere, actorArg, err := interestActorCondition(userID, guestSessionID)
	if err != nil {
		return nil, err
	}

	query := `
WITH scoped AS (
	SELECT
		user_id,
		guest_session_id,
		search_origin,
		search_roast_level,
		search_acidity,
		search_bitterness,
		search_flavor,
		search_aroma,
		search_body,
		search_category,
		occurred_at
	FROM action_events
	WHERE occurred_at >= ?
	  AND occurred_at < ?
	  AND ` + actorWhere + `
)
SELECT user_id, guest_session_id, ? AS dimension, search_origin AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_origin IS NOT NULL AND search_origin <> ''
GROUP BY user_id, guest_session_id, value
UNION ALL
SELECT user_id, guest_session_id, ? AS dimension, CAST(search_roast_level AS TEXT) AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_roast_level IS NOT NULL AND CAST(search_roast_level AS TEXT) <> ''
GROUP BY user_id, guest_session_id, value
UNION ALL
SELECT user_id, guest_session_id, ? AS dimension, CAST(search_acidity AS TEXT) AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_acidity IS NOT NULL
GROUP BY user_id, guest_session_id, value
UNION ALL
SELECT user_id, guest_session_id, ? AS dimension, CAST(search_bitterness AS TEXT) AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_bitterness IS NOT NULL
GROUP BY user_id, guest_session_id, value
UNION ALL
SELECT user_id, guest_session_id, ? AS dimension, CAST(search_flavor AS TEXT) AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_flavor IS NOT NULL
GROUP BY user_id, guest_session_id, value
UNION ALL
SELECT user_id, guest_session_id, ? AS dimension, CAST(search_aroma AS TEXT) AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_aroma IS NOT NULL
GROUP BY user_id, guest_session_id, value
UNION ALL
SELECT user_id, guest_session_id, ? AS dimension, CAST(search_body AS TEXT) AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_body IS NOT NULL
GROUP BY user_id, guest_session_id, value
UNION ALL
SELECT user_id, guest_session_id, ? AS dimension, search_category AS value, COUNT(*)::float AS score_delta, MAX(occurred_at) AS last_event_at
FROM scoped
WHERE search_category IS NOT NULL AND search_category <> ''
GROUP BY user_id, guest_session_id, value
`

	var rows []InterestAggregate
	err = r.db.WithContext(ctx).Raw(
		query,
		periodStart,
		periodEnd,
		actorArg,
		entity.InterestDimensionOrigin,
		entity.InterestDimensionRoastLevel,
		entity.InterestDimensionAcidity,
		entity.InterestDimensionBitterness,
		entity.InterestDimensionFlavor,
		entity.InterestDimensionAroma,
		entity.InterestDimensionBody,
		entity.InterestDimensionArticleCategory,
	).Scan(&rows).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return rows, nil
}

// 興味集計ではUserまたはGuestSessionの片方だけに絞る。
// Raw SQLに渡す条件文字列は固定文字列だけにし、値はplaceholderで渡す。
func interestActorCondition(userID *uint64, guestSessionID *uint64) (string, uint64, error) {
	if userID != nil && guestSessionID == nil {
		return "user_id = ? AND guest_session_id IS NULL", *userID, nil
	}
	if userID == nil && guestSessionID != nil {
		return "guest_session_id = ? AND user_id IS NULL", *guestSessionID, nil
	}
	return "", 0, entity.ErrInvalidInput
}

// 指定されたユーザーまたはゲストの特定イベント数を期間内で数える。
func (r *GormActionEventRepository) CountByActorAndTypeSince(ctx context.Context, userID *uint64, guestSessionID *uint64, eventType entity.EventType, since time.Time) (int64, error) {
	var count int64
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ActionEvent{}), userID, guestSessionID)
	err := db.Where("event_type = ? AND occurred_at >= ?", eventType, since).Count(&count).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

// 指定されたランキング対象の特定イベント数を期間内で数える。
func (r *GormActionEventRepository) CountByTargetAndTypeSince(ctx context.Context, rankTargetID uint64, eventType entity.EventType, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ActionEvent{}).Where("rank_target_id = ? AND event_type = ? AND occurred_at >= ?", rankTargetID, eventType, since).Count(&count).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

// 保存済みコンテンツを新規作成。
func (r *GormSavedItemRepository) Create(ctx context.Context, item *model.SavedItem) error {
	return mapDBError(r.db.WithContext(ctx).Create(item).Error)
}

// 指定ユーザーとランキング対象の保存状態を取得。
func (r *GormSavedItemRepository) FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.SavedItem, error) {
	var item model.SavedItem
	if err := r.db.WithContext(ctx).Where("user_id = ? AND rank_target_id = ?", userID, rankTargetID).First(&item).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &item, nil
}

// 未保存なら作成し、削除済みなら復元。
func (r *GormSavedItemRepository) SaveOrRestore(ctx context.Context, userID uint64, rankTargetID uint64, now time.Time) (*model.SavedItem, error) {
	item := model.SavedItem{UserID: userID, RankTargetID: rankTargetID, UpdatedAt: now}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "rank_target_id"}},
			DoUpdates: clause.Set{
				{Column: clause.Column{Name: "removed_at"}, Value: gorm.Expr("NULL")},
				{Column: clause.Column{Name: "updated_at"}, Value: now},
			},
		}).
		Create(&item).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return r.FindByUserAndTarget(ctx, userID, rankTargetID)
}

// 保存済みコンテンツに削除日時を入れて解除状態に。
func (r *GormSavedItemRepository) Remove(ctx context.Context, userID uint64, rankTargetID uint64, removedAt time.Time) error {
	updates := model.SavedItem{RemovedAt: &removedAt, UpdatedAt: removedAt}
	res := r.db.WithContext(ctx).Model(&model.SavedItem{}).
		Where("user_id = ? AND rank_target_id = ? AND removed_at IS NULL", userID, rankTargetID).
		UpdateColumns(&updates)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 指定ユーザーの有効な保存一覧を取得。
func (r *GormSavedItemRepository) ListActiveByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*model.SavedItem, error) {
	var items []*model.SavedItem
	err := r.db.WithContext(ctx).Preload("RankTarget").Where("user_id = ? AND removed_at IS NULL", userID).Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return items, nil
}

// 指定ユーザーが保存済みのRankTargetIDを一括取得する。
// 推薦候補の保存済み除外でExistsActiveを1件ずつ呼ぶN+1を避けるために使う。
func (r *GormSavedItemRepository) ListActiveRankTargetIDsByUser(ctx context.Context, userID uint64, rankTargetIDs []uint64) (map[uint64]bool, error) {
	ids := make(map[uint64]bool, len(rankTargetIDs))
	if len(rankTargetIDs) == 0 {
		return ids, nil
	}

	var savedIDs []uint64
	err := r.db.WithContext(ctx).
		Model(&model.SavedItem{}).
		Where("user_id = ? AND rank_target_id IN ? AND removed_at IS NULL", userID, rankTargetIDs).
		Pluck("rank_target_id", &savedIDs).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	for _, id := range savedIDs {
		ids[id] = true
	}
	return ids, nil
}

// 指定ユーザーが対象を現在保存しているか確認。
func (r *GormSavedItemRepository) ExistsActive(ctx context.Context, userID uint64, rankTargetID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.SavedItem{}).Where("user_id = ? AND rank_target_id = ? AND removed_at IS NULL", userID, rankTargetID).Count(&count).Error
	return existsFromCount(count, err)
}

// 指定されたランキング対象が保存されている件数を数える。。
func (r *GormSavedItemRepository) CountActiveByTarget(ctx context.Context, rankTargetID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.SavedItem{}).Where("rank_target_id = ? AND removed_at IS NULL", rankTargetID).Count(&count).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

// 評価を新規作成。
func (r *GormRatingRepository) Create(ctx context.Context, rating *model.Rating) error {
	return mapDBError(r.db.WithContext(ctx).Create(rating).Error)
}

// 指定ユーザーが対象へ付けた評価を取得。
func (r *GormRatingRepository) FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.Rating, error) {
	var rating model.Rating
	if err := r.db.WithContext(ctx).Where("user_id = ? AND rank_target_id = ?", userID, rankTargetID).First(&rating).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &rating, nil
}

// 指定Userの評価一覧を新しい順に取得。
func (r *GormRatingRepository) ListByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*model.Rating, error) {
	var ratings []*model.Rating
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&ratings).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return ratings, nil
}

// 未評価なら作成し、評価済みならGoodやBadを更新。
func (r *GormRatingRepository) Upsert(ctx context.Context, userID uint64, rankTargetID uint64, score entity.RatingScore, now time.Time) (*model.Rating, error) {
	rating := model.Rating{UserID: userID, RankTargetID: rankTargetID, Score: score, UpdatedAt: now}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "rank_target_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"score", "updated_at"}),
		}).
		Create(&rating).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return r.FindByUserAndTarget(ctx, userID, rankTargetID)
}

// 指定ユーザーの対象評価を削除。
func (r *GormRatingRepository) Delete(ctx context.Context, userID uint64, rankTargetID uint64) error {
	res := r.db.WithContext(ctx).Where("user_id = ? AND rank_target_id = ?", userID, rankTargetID).Delete(&model.Rating{})
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 指定されたランキング対象の評価件数を数える。
func (r *GormRatingRepository) CountByTarget(ctx context.Context, rankTargetID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Rating{}).Where("rank_target_id = ?", rankTargetID).Count(&count).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

// 指定されたランキング対象のGood/Bad件数と割合を集計。
func (r *GormRatingRepository) AggregateByTarget(ctx context.Context, rankTargetID uint64) (RatingAggregate, error) {
	var agg RatingAggregate
	err := r.db.WithContext(ctx).Model(&model.Rating{}).
		Select(`
			COUNT(*) AS rating_count,
			COUNT(*) FILTER (WHERE score = 1) AS good_count,
			COUNT(*) FILTER (WHERE score = -1) AS bad_count,
			COALESCE(AVG(score), 0) AS rating_avg,
			CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(*) FILTER (WHERE score = 1)::float / COUNT(*) END AS good_rate,
			CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(*) FILTER (WHERE score = -1)::float / COUNT(*) END AS bad_rate
		`).
		Where("rank_target_id = ?", rankTargetID).
		Scan(&agg).Error
	if err != nil {
		return RatingAggregate{}, mapDBError(err)
	}
	return agg, nil
}
