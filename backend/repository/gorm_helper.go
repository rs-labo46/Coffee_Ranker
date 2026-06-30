package repository

import (
	"errors"
	"strings"

	"coffee-ranker/entity"

	"gorm.io/gorm"
)

// GORMやDB制約由来のエラーをアプリ共通エラーへ変換。
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.ErrNotFound
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
		return entity.ErrConflict
	}
	return entity.ErrRepositoryFailed
}

// 更新件数0件を見つからない更新。
func mapRowsAffected(rows int64, err error) error {
	if err != nil {
		return mapDBError(err)
	}
	if rows == 0 {
		return entity.ErrNoRowsAffected
	}
	return nil
}

// COUNT結果とDBエラーから存在可否を返す。
func existsFromCount(count int64, err error) (bool, error) {
	if err != nil {
		return false, mapDBError(err)
	}
	return count > 0, nil
}

// user_idとguest_session_idの片方だけを許可する検索条件を付ける。
func applyActorFilter(db *gorm.DB, userID *uint64, guestSessionID *uint64) *gorm.DB {
	if userID != nil && guestSessionID == nil {
		return db.Where("user_id = ? AND guest_session_id IS NULL", *userID)
	}

	if userID == nil && guestSessionID != nil {
		return db.Where("guest_session_id = ? AND user_id IS NULL", *guestSessionID)
	}

	return db.Where("1 = 0")
}

// ランキング指標を含む検索で使う並び順を付ける。
func applyMetricSort(db *gorm.DB, sort string, idColumn string) *gorm.DB {
	switch sort {
	case "popular":
		return db.
			Order("COALESCE(content_metrics.click_count, 0) DESC").
			Order("COALESCE(content_metrics.content_view_count, 0) DESC").
			Order(idColumn + " DESC")
	case "newest":
		return db.Order(idColumn + " DESC")
	default:
		return db.
			Order("COALESCE(content_metrics.score, 0) DESC").
			Order(idColumn + " DESC")
	}
}
