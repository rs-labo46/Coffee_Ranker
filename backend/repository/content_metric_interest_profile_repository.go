package repository

import (
	"context"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ランキング計算後の指標を保存、取得、ランキング表示するDB操作。
type IContentMetricRepository interface {
	Upsert(ctx context.Context, metric *model.ContentMetric) error
	BulkUpsert(ctx context.Context, metrics []*model.ContentMetric) error
	FindByRankTargetID(ctx context.Context, rankTargetID uint64) (*model.ContentMetric, error)
	FindByRankTargetIDs(ctx context.Context, rankTargetIDs []uint64) ([]*model.ContentMetric, error)
	ListRanking(ctx context.Context, contentType *entity.ContentType, limit int, offset int) ([]*model.ContentMetric, error)
	ListTopByScore(ctx context.Context, limit int) ([]*model.ContentMetric, error)
	LatestCalculatedAt(ctx context.Context) (*time.Time, error)
}

// ユーザーまたはゲストの興味スコアを保存、取得、削除するDB操作。
type IInterestProfileRepository interface {
	Upsert(ctx context.Context, profile *model.InterestProfile) error
	BulkUpsert(ctx context.Context, profiles []*model.InterestProfile) error
	FindByUser(ctx context.Context, userID uint64) ([]*model.InterestProfile, error)
	FindByGuestSession(ctx context.Context, guestSessionID uint64) ([]*model.InterestProfile, error)
	ListTopByUser(ctx context.Context, userID uint64, limit int) ([]*model.InterestProfile, error)
	ListTopByGuest(ctx context.Context, guestSessionID uint64, limit int) ([]*model.InterestProfile, error)
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type GormContentMetricRepository struct {
	db *gorm.DB
}

type GormInterestProfileRepository struct {
	db *gorm.DB
}

func NewContentMetricRepository(db *gorm.DB) IContentMetricRepository {
	return &GormContentMetricRepository{db}
}

func NewInterestProfileRepository(db *gorm.DB) IInterestProfileRepository {
	return &GormInterestProfileRepository{db}
}

// ランキング指標を新規作成または更新
func (r *GormContentMetricRepository) Upsert(ctx context.Context, metric *model.ContentMetric) error {
	return mapDBError(r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "rank_target_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"score",
				"impression_count",
				"content_view_count",
				"click_count",
				"stay_total_ms",
				"save_count",
				"rating_count",
				"good_count",
				"bad_count",
				"re_search_count",
				"rating_avg",
				"good_rate",
				"bad_rate",
				"modal_impression_count",
				"modal_click_count",
				"modal_close_count",
				"click_rate",
				"save_rate",
				"re_search_rate",
				"modal_click_rate",
				"modal_close_rate",
				"period_start",
				"period_end",
				"calculated_at",
				"updated_at",
			}),
		}).
		Create(metric).Error)
}

// 複数のランキング指標をまとめて新規作成または更新。
func (r *GormContentMetricRepository) BulkUpsert(ctx context.Context, metrics []*model.ContentMetric) error {
	for _, metric := range metrics {
		if err := r.Upsert(ctx, metric); err != nil {
			return err
		}
	}
	return nil
}

// ランキング対象IDに一致する指標を取得。
func (r *GormContentMetricRepository) FindByRankTargetID(ctx context.Context, rankTargetID uint64) (*model.ContentMetric, error) {
	var metric model.ContentMetric
	if err := r.db.WithContext(ctx).Where("rank_target_id = ?", rankTargetID).First(&metric).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &metric, nil
}

// 複数のランキング対象IDに一致する指標を取得。
func (r *GormContentMetricRepository) FindByRankTargetIDs(ctx context.Context, rankTargetIDs []uint64) ([]*model.ContentMetric, error) {
	var metrics []*model.ContentMetric
	if len(rankTargetIDs) == 0 {
		return metrics, nil
	}
	if err := r.db.WithContext(ctx).Where("rank_target_id IN ?", rankTargetIDs).Find(&metrics).Error; err != nil {
		return nil, mapDBError(err)
	}
	return metrics, nil
}

// ランキング表示用に指標をスコア順で取得。
func (r *GormContentMetricRepository) ListRanking(ctx context.Context, contentType *entity.ContentType, limit int, offset int) ([]*model.ContentMetric, error) {
	var metrics []*model.ContentMetric
	db := r.db.WithContext(ctx).Preload("RankTarget").Joins("JOIN rank_targets ON rank_targets.id = content_metrics.rank_target_id").Where("rank_targets.is_active = true")
	if contentType != nil {
		db = db.Where("rank_targets.content_type = ?", *contentType)
	}
	if err := db.Order("content_metrics.score DESC").Order("content_metrics.rank_target_id DESC").Limit(limit).Offset(offset).Find(&metrics).Error; err != nil {
		return nil, mapDBError(err)
	}
	return metrics, nil
}

// 全コンテンツの上位スコア指標を取得。
func (r *GormContentMetricRepository) ListTopByScore(ctx context.Context, limit int) ([]*model.ContentMetric, error) {
	var metrics []*model.ContentMetric
	err := r.db.WithContext(ctx).Preload("RankTarget").Joins("JOIN rank_targets ON rank_targets.id = content_metrics.rank_target_id").Where("rank_targets.is_active = true").Order("content_metrics.score DESC").Order("content_metrics.rank_target_id DESC").Limit(limit).Find(&metrics).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return metrics, nil
}

// 最後にランキング指標を計算した日時を取得。
func (r *GormContentMetricRepository) LatestCalculatedAt(ctx context.Context) (*time.Time, error) {
	var metric model.ContentMetric
	err := r.db.WithContext(ctx).Order("calculated_at DESC").First(&metric).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return &metric.CalculatedAt, nil
}

// 興味プロフィールを新規作成または更新。
func (r *GormInterestProfileRepository) Upsert(ctx context.Context, profile *model.InterestProfile) error {
	var existing model.InterestProfile
	db := r.db.WithContext(ctx).Model(&model.InterestProfile{})
	if profile.UserID != nil {
		db = db.Where("user_id = ? AND guest_session_id IS NULL AND dimension = ? AND value = ?", *profile.UserID, profile.Dimension, profile.Value)
	} else if profile.GuestSessionID != nil {
		db = db.Where("guest_session_id = ? AND user_id IS NULL AND dimension = ? AND value = ?", *profile.GuestSessionID, profile.Dimension, profile.Value)
	} else {
		return entity.ErrInvalidInput
	}
	err := db.Assign(*profile).FirstOrCreate(&existing).Error
	if err != nil {
		return mapDBError(err)
	}
	profile.ID = existing.ID
	return nil
}

// 複数の興味プロフィールをまとめて新規作成または更新。
func (r *GormInterestProfileRepository) BulkUpsert(ctx context.Context, profiles []*model.InterestProfile) error {
	for _, profile := range profiles {
		if err := r.Upsert(ctx, profile); err != nil {
			return err
		}
	}
	return nil
}

// 指定ユーザーの興味プロフィールを取得。
func (r *GormInterestProfileRepository) FindByUser(ctx context.Context, userID uint64) ([]*model.InterestProfile, error) {
	var profiles []*model.InterestProfile
	err := r.db.WithContext(ctx).Where("user_id = ? AND guest_session_id IS NULL", userID).Order("score DESC").Find(&profiles).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return profiles, nil
}

// 指定ゲストセッションの興味プロフィールを取得。
func (r *GormInterestProfileRepository) FindByGuestSession(ctx context.Context, guestSessionID uint64) ([]*model.InterestProfile, error) {
	var profiles []*model.InterestProfile
	err := r.db.WithContext(ctx).Where("guest_session_id = ? AND user_id IS NULL", guestSessionID).Order("score DESC").Find(&profiles).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return profiles, nil
}

// 指定ユーザーの興味スコア上位を取得。
func (r *GormInterestProfileRepository) ListTopByUser(ctx context.Context, userID uint64, limit int) ([]*model.InterestProfile, error) {
	var profiles []*model.InterestProfile
	err := r.db.WithContext(ctx).Where("user_id = ? AND guest_session_id IS NULL", userID).Order("score DESC").Limit(limit).Find(&profiles).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return profiles, nil
}

// 指定ゲストセッションの興味スコア上位を取得。
func (r *GormInterestProfileRepository) ListTopByGuest(ctx context.Context, guestSessionID uint64, limit int) ([]*model.InterestProfile, error) {
	var profiles []*model.InterestProfile
	err := r.db.WithContext(ctx).Where("guest_session_id = ? AND user_id IS NULL", guestSessionID).Order("score DESC").Limit(limit).Find(&profiles).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return profiles, nil
}

// 期限切れの興味プロフィールを削除。
func (r *GormInterestProfileRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at IS NOT NULL AND expires_at < ?", now).Delete(&model.InterestProfile{})
	if res.Error != nil {
		return 0, mapDBError(res.Error)
	}
	return res.RowsAffected, nil
}
