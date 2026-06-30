package repository

import (
	"context"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
)

// ユーザー作成、認証用検索、状態更新に必要なDB操作。
type IUserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uint64) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error
	IncrementTokenVersion(ctx context.Context, userID uint64) error
	UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error
	List(ctx context.Context, limit int, offset int) ([]*model.User, error)
}

type GormUserRepository struct {
	db *gorm.DB
}

// UserRepositoryを作成する。
func NewUserRepository(db *gorm.DB) IUserRepository {
	return &GormUserRepository{db}
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
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("email = ?", email).
		Count(&count).Error

	return existsFromCount(count, err)
}

// AccessToken強制失効用のtoken_versionを指定値に更新。
func (r *GormUserRepository) UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error {
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("token_version", tokenVersion)

	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 既存AccessTokenを無効化するためtoken_versionを1増やす。
func (r *GormUserRepository) IncrementTokenVersion(ctx context.Context, userID uint64) error {
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumn("token_version", gorm.Expr("token_version + 1"))

	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 管理者操作でユーザー状態をactive/suspended/deletedに更新。
func (r *GormUserRepository) UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error {
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("status", status)

	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 管理画面用にユーザー一覧を作成日時の新しい順で取得する。
func (r *GormUserRepository) List(ctx context.Context, limit int, offset int) ([]*model.User, error) {
	var users []*model.User

	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, mapDBError(err)
	}

	return users, nil
}
