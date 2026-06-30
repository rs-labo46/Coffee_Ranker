package repository

import (
	"context"
	"time"

	"coffee-ranker/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RefreshTokenの検索、rotation、失効、削除に必要なDB操作。
type IRefreshTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	MarkUsed(ctx context.Context, id uint64, usedAt time.Time, replacedByTokenID uint64) error
	Create(ctx context.Context, token *model.RefreshToken) error
	Revoke(ctx context.Context, id uint64, revokedAt time.Time) error
	RevokeByFamilyID(ctx context.Context, familyID string, revokedAt time.Time) error
	RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type GormRefreshTokenRepository struct {
	db *gorm.DB
}

// RefreshTokenRepositoryを作成する。
func NewRefreshTokenRepository(db *gorm.DB) IRefreshTokenRepository {
	return &GormRefreshTokenRepository{db}
}

// hash化済みRefreshTokenに一致するレコードを取得。
func (r *GormRefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &token, nil
}

// RefreshTokenと所有者ユーザーをまとめて取得。
func (r *GormRefreshTokenRepository) FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("token_hash = ?", tokenHash).
		First(&token).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &token, nil
}

// Refresh rotation中の二重使用を防ぐため行ロック付きで取得。
func (r *GormRefreshTokenRepository) FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken

	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("User").
		Where("token_hash = ?", tokenHash).
		First(&token).Error
	if err != nil {
		return nil, mapDBError(err)
	}

	return &token, nil
}

// 旧RefreshTokenを使用済みにし、置き換え後token IDを保存。
func (r *GormRefreshTokenRepository) MarkUsed(ctx context.Context, id uint64, usedAt time.Time, replacedByTokenID uint64) error {
	updates := model.RefreshToken{
		UsedAt:            &usedAt,
		ReplacedByTokenID: &replacedByTokenID,
	}

	res := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", id).
		Updates(&updates)

	return mapRowsAffected(res.RowsAffected, res.Error)
}

// hash化済みRefreshTokenをDBへ保存。
func (r *GormRefreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	return mapDBError(r.db.WithContext(ctx).Create(token).Error)
}

// RefreshToken単体に失効日時を設定。
func (r *GormRefreshTokenRepository) Revoke(ctx context.Context, id uint64, revokedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", revokedAt)

	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 同じfamily_idのRefreshTokenをまとめて失効。
func (r *GormRefreshTokenRepository) RevokeByFamilyID(ctx context.Context, familyID string, revokedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", revokedAt)

	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 指定ユーザーのRefreshTokenをすべて失効。
// 全端末logoutでは、すでに全RefreshTokenが失効済みでもtoken_version更新は続ける必要がある。
// そのため、更新件数0件はエラーにせず、DBエラーだけを返す。
func (r *GormRefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", revokedAt)

	return mapDBError(res.Error)
}

// 有効期限切れのRefreshTokenを削除。
func (r *GormRefreshTokenRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&model.RefreshToken{})

	if res.Error != nil {
		return 0, mapDBError(res.Error)
	}

	return res.RowsAffected, nil
}
