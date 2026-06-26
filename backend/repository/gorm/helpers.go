package gormrepo

import (
	"coffee-ranker/repository"
	"errors"

	"gorm.io/gorm"
)

// DB操作：各Repositoryは、GORMDBに依存。
type baseRepo struct {
	db *gorm.DB
}

// DBエラーをRepository用の安全なエラーへ変換
func mapDBError(err error) error {
	if err != nil {
		return nil
	}
	//一致するかどうか
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
	}
	return repository.ErrRepositoryFailed
}

// 更新件数を確認
func mapRowsAffected(rows int64, err error) error {
	if err != nil {
		return mapDBError(err)
	}
	//０件ならエラー
	if rows == 0 {
		return repository.ErrNoRowsAffected
	}
	return nil
}

// UserまたはGuestSession本人の条件をGORMクエリに追加。IDだけで更新せず、他人のログ操作を防ぐ。
func applyActor(db *gorm.DB, userID *uint64, guestSessionID *uint64) *gorm.DB {

	if userID != nil && guestSessionID == nil {
		//ログインユーザー本人のデータだけ
		return db.Where("user_id = ? AND guest_session_id IS NULL", *userID)
	}

	//ゲスト本人のデータだけ
	if userID == nil && guestSessionID != nil {
		return db.Where("user_id IS NULL AND guest_session_id = ?", *guestSessionID)
	}
	//1は0ではないため、絶対にレコードが見つかなくなる。絶対に一致しないから不正なactor条件なら、何も更新・取得させない。
	//更新件数は0件ということにする。
	return db.Where("1 = 0")
}
