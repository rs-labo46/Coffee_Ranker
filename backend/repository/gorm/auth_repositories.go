package gormrepo

import (
	"coffee-ranker/entity"
	"coffee-ranker/repository/gorm/model"
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// usersテーブルのDB操作を担当する。
// 認証や権限の判断はUsecaseで行う。
type GormUserRepository struct {
	baseRepo
}

// refresh_tokensテーブルのDB操作。
// 生RefreshTokenではなくhash化済みTokenHashだけを扱う。
type GormRefreshTokenRepository struct {
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

// RefreshTokenの作成、取得、使用済み化、失効。
// 生RefreshTokenは扱わず、Usecaseでhash化された値だけを保存・検索する。
type RefreshTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	MarkUsed(ctx context.Context, id uint64, usedAt time.Time, replacedByTokenID uint64) error
	Create(ctx context.Context, token *entity.RefreshToken) error
	Revoke(ctx context.Context, id uint64, revokedAt time.Time) error
	RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) error
	RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
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
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, mapDBError(err)
	}
	return existsFromCount(count), nil
}

// 指定ユーザーのtoken_versionを指定値へ更新。
// Usecaseが決めた値をDBへ反映する。
func (r *GormUserRepository) UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error {
	//token_versionを指定し更新
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("token_version", tokenVersion)
	return mapRowsAffected(res.RowsAffected, res.Error) //
}

// 指定ユーザーのtoken_versionを1増やす。
// 既存AccessTokenを無効化するときに使う。
func (r *GormUserRepository) IncrementTokenVersion(ctx context.Context, userID uint64) error {
	//自動更新を避けるためにカラムを指定して＋１する。
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).UpdateColumn("token_version", gorm.Expr("token_version + 1"))
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 指定ユーザーの状態を更新する。
// 管理者権限の確認はUsecaseで行う。
func (r *GormUserRepository) UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("status", string(status))
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// ユーザー一覧を新しい順で取得。
// 管理画面の一覧表示で使う。
func (r *GormUserRepository) List(ctx context.Context, limit int, offset int) ([]*entity.User, error) {
	var models []model.User
	//usersテーブルから、指定位置から指定件数だけ、IDが新しい順に取得して、modelsに詰める。
	if err := r.db.WithContext(ctx).Order("id DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, mapDBError(err)
	}
	//DBから取得した一覧を入れる
	//1件ずつentityへ変換
	users := make([]*entity.User, 0, len(models))
	for i := range models {
		//DBのmodelをdomain entityへ変換したものをusersに。
		users = append(users, toUserEntity(&models[i]))
	}
	return users, nil
}

// token_hashに一致するRefreshTokenを取得する。
// Cookieの生TokenはUsecase側でhash化してから渡す。
func (r *GormRefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var m model.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m).Error; err != nil {
		return nil, mapDBError(err)
	}
	return toRefreshTokenEntity(&m), nil
}

// token_hashに一致するRefreshTokenをUser付きで取得する。
// LogoutやRefresh前の確認で使う。
func (r *GormRefreshTokenRepository) FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var m model.RefreshToken
	if err := r.db.WithContext(ctx).Preload("User").Where("token_hash = ?", tokenHash).First(&m).Error; err != nil {
		return nil, mapDBError(err)
	}
	return toRefreshTokenEntity(&m), nil
}

// RefreshTokenを行ロック付きで取得する。
// 同じRefreshTokenが同時に使われることを防ぐ。
func (r *GormRefreshTokenRepository) FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var m model.RefreshToken
	err := r.db.WithContext(ctx).
		//Clauses:取得した行を他の処理が同時に更新できないようにする
		//Pread:RefreshTokenに紐づくUserも一緒に取得
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("User").
		Where("token_hash = ?", tokenHash).
		First(&m).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return toRefreshTokenEntity(&m), nil
}
