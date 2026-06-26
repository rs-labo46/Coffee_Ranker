package repository

import (
	"context"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ユーザー作成、認証用検索、状態更新に必要なDB操作。
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uint64) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error
	IncrementTokenVersion(ctx context.Context, userID uint64) error
	UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error
	List(ctx context.Context, limit int, offset int) ([]*model.User, error)
}

// RefreshTokenの検索、rotation、失効、削除に必要なDB操作。
type RefreshTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	MarkUsed(ctx context.Context, id uint64, usedAt time.Time, replacedByTokenID uint64) error
	Create(ctx context.Context, token *model.RefreshToken) error
	Revoke(ctx context.Context, id uint64, revokedAt time.Time) error
	RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) error
	RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// 未ログインユーザーの一時識別情報を作成、取得、延長するDB操作。
type GuestSessionRepository interface {
	Create(ctx context.Context, session *model.GuestSession) error
	FindByID(ctx context.Context, id uint64) (*model.GuestSession, error)
	FindBySessionKeyHash(ctx context.Context, hash string) (*model.GuestSession, error)
	Touch(ctx context.Context, id uint64, lastSeenAt time.Time, expiresAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type GormUserRepository struct {
	baseRepo
}

type GormRefreshTokenRepository struct {
	baseRepo
}

type GormGuestSessionRepository struct {
	baseRepo
}

// ユーザーRepository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{baseRepo{db}}
}

// RefreshToken Repository
func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &GormRefreshTokenRepository{baseRepo{db}}
}

// ゲストセッションRepository
func NewGuestSessionRepository(db *gorm.DB) GuestSessionRepository {
	return &GormGuestSessionRepository{baseRepo{db}}
}

// ユーザーを新規作成。
func (r *GormUserRepository) Create(ctx context.Context, user *model.User) error {
	return mapDBError(r.db.WithContext(ctx).Create(user).Error)
}

// ユーザーIDに一致するユーザーを取得。
func (r *GormUserRepository) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &user, nil
}

// ログイン用メールアドレスに一致するユーザーを取得。
func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &user, nil
}

// 同じメールアドレスのユーザーが存在するか確認。
func (r *GormUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return existsFromCount(count, err)
}

// AccessToken強制失効用のtoken_versionを指定値に更新。
func (r *GormUserRepository) UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("token_version", tokenVersion)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 既存AccessTokenを無効化するためtoken_versionを1増やす。
func (r *GormUserRepository) IncrementTokenVersion(ctx context.Context, userID uint64) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).UpdateColumn("token_version", gorm.Expr("token_version + 1"))
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 管理者操作でユーザー状態をactive/suspended/deletedに更新。
func (r *GormUserRepository) UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("status", status)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 管理画面用にユーザー一覧を新しい順で取得。
func (r *GormUserRepository) List(ctx context.Context, limit int, offset int) ([]*model.User, error) {
	var users []*model.User
	if err := r.db.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, mapDBError(err)
	}
	return users, nil
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
	if err := r.db.WithContext(ctx).Preload("User").Where("token_hash = ?", tokenHash).First(&token).Error; err != nil {
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
	updates := model.RefreshToken{UsedAt: &usedAt, ReplacedByTokenID: &replacedByTokenID}
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
	res := r.db.WithContext(ctx).Model(&model.RefreshToken{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", revokedAt)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 同じfamily_idのRefreshTokenをまとめて失効。
func (r *GormRefreshTokenRepository) RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.RefreshToken{}).Where("family_id = ? AND revoked_at IS NULL", familyID).Update("revoked_at", revokedAt)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 指定ユーザーのRefreshTokenをすべて失効。
func (r *GormRefreshTokenRepository) RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", revokedAt)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 有効期限切れのRefreshTokenを削除。
func (r *GormRefreshTokenRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&model.RefreshToken{})
	if res.Error != nil {
		return 0, mapDBError(res.Error)
	}
	return res.RowsAffected, nil
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
	updates := model.GuestSession{LastSeenAt: lastSeenAt, ExpiresAt: expiresAt}
	res := r.db.WithContext(ctx).Model(&model.GuestSession{}).Where("id = ?", id).UpdateColumns(&updates)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 有効期限切れのゲストセッションを削除。
func (r *GormGuestSessionRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("expires_at < ?", now).Delete(&model.GuestSession{})
	if res.Error != nil {
		return 0, mapDBError(res.Error)
	}
	return res.RowsAffected, nil
}
