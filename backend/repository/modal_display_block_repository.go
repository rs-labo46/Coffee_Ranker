package repository

import (
	"context"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
)

// 推薦モーダルの表示、クリック、クローズ履歴を扱うDB操作。
type ModalDisplayLogRepository interface {
	Create(ctx context.Context, log *model.ModalDisplayLog) error
	FindByIDForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64) (*model.ModalDisplayLog, error)
	MarkClickedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, clickedAt time.Time) error
	MarkClosedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, closedAt time.Time) error
	CountShownInSession(ctx context.Context, userID *uint64, guestSessionID *uint64, since time.Time) (int64, error)
	CountShownOnPage(ctx context.Context, userID *uint64, guestSessionID *uint64, pagePath string, since time.Time) (int64, error)
	WasTargetShownRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error)
	WasTargetClosedRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error)
}

// 推薦モーダルを出さなかった理由を記録、取得するDB操作。
type ModalBlockLogRepository interface {
	Create(ctx context.Context, log *model.ModalBlockLog) error
	ListByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ModalBlockLog, error)
	CountByReasonSince(ctx context.Context, reason entity.ModalBlockReason, since time.Time) (int64, error)
	ListByCandidate(ctx context.Context, rankTargetID uint64, limit int) ([]*model.ModalBlockLog, error)
}

type GormModalDisplayLogRepository struct {
	baseRepo
}

type GormModalBlockLogRepository struct {
	baseRepo
}

func NewModalDisplayLogRepository(db *gorm.DB) ModalDisplayLogRepository {
	return &GormModalDisplayLogRepository{baseRepo{db}}
}

func NewModalBlockLogRepository(db *gorm.DB) ModalBlockLogRepository {
	return &GormModalBlockLogRepository{baseRepo{db}}
}

// 推薦モーダルの表示履歴を保存。
func (r *GormModalDisplayLogRepository) Create(ctx context.Context, log *model.ModalDisplayLog) error {
	return mapDBError(r.db.WithContext(ctx).Create(log).Error)
}

// 指定ユーザーまたはゲスト本人に紐づく表示履歴だけを取得。
func (r *GormModalDisplayLogRepository) FindByIDForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64) (*model.ModalDisplayLog, error) {
	var log model.ModalDisplayLog
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalDisplayLog{}), userID, guestSessionID)
	if err := db.Where("id = ?", id).First(&log).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &log, nil
}

// 本人に紐づく表示履歴だけをクリック済みに更新。
func (r *GormModalDisplayLogRepository) MarkClickedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, clickedAt time.Time) error {
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalDisplayLog{}), userID, guestSessionID)
	res := db.Where("id = ? AND clicked_at IS NULL", id).Update("clicked_at", clickedAt)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 本人に紐づく表示履歴だけをクローズ済みに更新。
func (r *GormModalDisplayLogRepository) MarkClosedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, closedAt time.Time) error {
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalDisplayLog{}), userID, guestSessionID)
	res := db.Where("id = ? AND closed_at IS NULL", id).Update("closed_at", closedAt)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 指定期間内に同じユーザーまたはゲストへ表示した回数を数える。
func (r *GormModalDisplayLogRepository) CountShownInSession(ctx context.Context, userID *uint64, guestSessionID *uint64, since time.Time) (int64, error) {
	var count int64
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalDisplayLog{}), userID, guestSessionID)
	err := db.Where("shown_at >= ?", since).Count(&count).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

// 指定ページで同じユーザーまたはゲストへ表示した回数を数える。
func (r *GormModalDisplayLogRepository) CountShownOnPage(ctx context.Context, userID *uint64, guestSessionID *uint64, pagePath string, since time.Time) (int64, error) {
	var count int64
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalDisplayLog{}), userID, guestSessionID)
	err := db.Where("page_path = ? AND shown_at >= ?", pagePath, since).Count(&count).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

// 同じ候補を直近で表示したか確認。
func (r *GormModalDisplayLogRepository) WasTargetShownRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error) {
	var count int64
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalDisplayLog{}), userID, guestSessionID)
	err := db.Where("rank_target_id = ? AND shown_at >= ?", rankTargetID, since).Count(&count).Error
	return existsFromCount(count, err)
}

// 同じ候補を直近で閉じられたか確認。
func (r *GormModalDisplayLogRepository) WasTargetClosedRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error) {
	var count int64
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalDisplayLog{}), userID, guestSessionID)
	err := db.Where("rank_target_id = ? AND closed_at IS NOT NULL AND closed_at >= ?", rankTargetID, since).Count(&count).Error
	return existsFromCount(count, err)
}

// 推薦モーダルを出さなかった理由を保存。
func (r *GormModalBlockLogRepository) Create(ctx context.Context, log *model.ModalBlockLog) error {
	return mapDBError(r.db.WithContext(ctx).Create(log).Error)
}

// 指定ユーザーまたはゲストの非表示理由ログを取得。
func (r *GormModalBlockLogRepository) ListByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ModalBlockLog, error) {
	var logs []*model.ModalBlockLog
	db := applyActorFilter(r.db.WithContext(ctx).Model(&model.ModalBlockLog{}), userID, guestSessionID)
	if err := db.Order("blocked_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, mapDBError(err)
	}
	return logs, nil
}

// 指定期間内に特定理由で非表示にした回数を数える。
func (r *GormModalBlockLogRepository) CountByReasonSince(ctx context.Context, reason entity.ModalBlockReason, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ModalBlockLog{}).Where("reason = ? AND blocked_at >= ?", reason, since).Count(&count).Error
	if err != nil {
		return 0, mapDBError(err)
	}
	return count, nil
}

// 指定候補に関する非表示理由ログを取得。
func (r *GormModalBlockLogRepository) ListByCandidate(ctx context.Context, rankTargetID uint64, limit int) ([]*model.ModalBlockLog, error) {
	var logs []*model.ModalBlockLog
	err := r.db.WithContext(ctx).Where("candidate_rank_target_id = ?", rankTargetID).Order("blocked_at DESC").Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return logs, nil
}
