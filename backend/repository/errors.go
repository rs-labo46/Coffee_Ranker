package repository

import "errors"

// 対象のレコードが見つからない状態。DBのテーブルを漏らさない。
var ErrNotFound = errors.New("repository not found")

// 重複の作成をさせない。
var ErrConflict = errors.New("repository conflict")

// アップデートする対象が0件
var ErrNoRowsAffected = errors.New("repository no rows affected")

// DBやRedis操作が失敗
var ErrRepositoryFailed = errors.New("repository failed")
