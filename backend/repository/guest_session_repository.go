package repository

import (
	"context"
	"time"

	"coffee-ranker/model"

	"gorm.io/gorm"
)

// 未ログインユーザーの一時識別情報を作成、取得、延長するDB操作。
type IGuestSessionRepository interface {
	Create(ctx context.Context, session *model.GuestSession) error
	FindByID(ctx context.Context, id uint64) (*model.GuestSession, error)
	FindBySessionKeyHash(ctx context.Context, hash string) (*model.GuestSession, error)
	Touch(ctx context.Context, id uint64, lastSeenAt time.Time, expiresAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type GormGuestSessionRepository struct {
	db *gorm.DB
}

// GuestSessionRepositoryを作成する。
func NewGuestSessionRepository(db *gorm.DB) IGuestSessionRepository {
	return &GormGuestSessionRepository{db}
}

// ゲストセッションを新規作成。
func (r *GormGuestSessionRepository) Create(ctx context.Context, session *model.GuestSession) error {
	return mapDBError(r.db.WithContext(ctx).Create(session).Error)
}

// ゲストセッションIDに一致するレコードを取得。
func (r *GormGuestSessionRepository) FindByID(ctx context.Context, id uint64) (*model.GuestSession, error) {
	var session model.GuestSession
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &session, nil
}

// hash化済みsession keyに一致するゲストセッションを取得。
func (r *GormGuestSessionRepository) FindBySessionKeyHash(ctx context.Context, hash string) (*model.GuestSession, error) {
	var session model.GuestSession
	if err := r.db.WithContext(ctx).Where("session_key_hash = ?", hash).First(&session).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &session, nil
}

// 最終アクセス日時と有効期限を更新。
func (r *GormGuestSessionRepository) Touch(ctx context.Context, id uint64, lastSeenAt time.Time, expiresAt time.Time) error {
	updates := model.GuestSession{
		LastSeenAt: lastSeenAt,
		ExpiresAt:  expiresAt,
	}

	res := r.db.WithContext(ctx).
		Model(&model.GuestSession{}).
		Where("id = ?", id).
		UpdateColumns(&updates)

	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 有効期限切れのゲストセッションを削除。
func (r *GormGuestSessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&model.GuestSession{})

	if res.Error != nil {
		return 0, mapDBError(res.Error)
	}

	return res.RowsAffected, nil
}
