package gormrepo

import (
	"coffee-ranker/entity"
	"coffee-ranker/repository/gorm/model"
	"context"

	"gorm.io/gorm"
)

// usersテーブルのDB操作を担当する。
// 認証や権限の判断はUsecaseで行う。
type GormUserRepository struct {
	baseRepo
}

// ユーザー作成、取得、認証用検索、状態更新。
// 認証や権限判断はUsecaseの責務であり、Repositoryは永続化。
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uint64) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error
	IncrementTokenVersion(ctx context.Context, userID uint64) error
	UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error
	List(ctx context.Context, limit int, offset int) ([]*entity.User, error)
}

// usersテーブルを操作するRepositoryを生成。
// UsecaseからはInterface越しに呼び出す。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{baseRepo{db: db}}
}

// 新規ユーザーをDBに作成する。
// 保存するPasswordHashはUsecaseでhash化済みの値を使う。
func (r *GormUserRepository) Create(ctx context.Context, user *entity.User) error {
	//Usecaseから渡されたentity.UserをDB保存用のmodel.Userに変換
	m := toUserModel(user)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return mapDBError(err)
	}
	//保存後の結果を元のentity.Userに戻す
	*user = *toUserEntity(m)
	return nil
}

// userIDに一致するユーザーを取得する。
// ユーザー状態の判断はUsecaseで行う。
func (r *GormUserRepository) FindByID(ctx context.Context, id uint64) (*entity.User, error) {
	// DBからの取得結果を入れるもの
	var m model.User

	// DBからUserを1件取得
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, mapDBError(err)
	}
	// model.Userをentity.Userに変換
	return toUserEntity(&m), nil
}

// emailに一致するユーザーを取得する。
// emailの正規化はUsecaseまたはValidator側で行う。
func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var m model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&m).Error; err != nil {
		return nil, mapDBError(err)
	}
	return toUserEntity(&m), nil
}

// 同じemailのユーザーが存在するか確認する。
// Signup前の重複確認で使う。
func (r *GormUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
}
