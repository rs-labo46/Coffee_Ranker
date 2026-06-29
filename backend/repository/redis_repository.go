package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"coffee-ranker/entity"

	"github.com/redis/go-redis/v9"
)

// Redisで行動ログの重複送信を防ぐ操作。
type IEventDedupRepository interface {
	SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
}

// RedisでAPI利用回数を制限するTokenBucket操作。
type IRateLimitRepository interface {
	Take(ctx context.Context, key string, capacity int, refillRate float64, now time.Time) (RateLimitResult, error)
	Reset(ctx context.Context, key string) error
}

// Redisで推薦モーダルの表示済み、閉じた状態、表示回数を管理する操作。
type IModalSuppressionRepository interface {
	SetShown(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error
	WasShown(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error)
	IncrementPageCount(ctx context.Context, actorKey string, pagePath string, ttl time.Duration) (int64, error)
	IncrementSessionCount(ctx context.Context, actorKey string, ttl time.Duration) (int64, error)
	SetClosed(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error
	WasClosed(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error)
}

type IBatchLockRepository interface {
	Acquire(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string, owner string) error
	Extend(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error)
}

type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

type RedisIRateLimitRepository struct {
	client *redis.Client
}

type RedisEventDedupRepository struct {
	client *redis.Client
}

type RedisModalSuppressionRepository struct {
	client *redis.Client
}

func NewIRateLimitRepository(client *redis.Client) IRateLimitRepository {
	return &RedisIRateLimitRepository{client}
}

func NewEventDedupRepository(client *redis.Client) IEventDedupRepository {
	return &RedisEventDedupRepository{client: client}
}

func NewModalSuppressionRepository(client *redis.Client) IModalSuppressionRepository {
	return &RedisModalSuppressionRepository{client}
}

// TokenBucketを補充してから1回分を消費できるか判定。
func (r *RedisIRateLimitRepository) Take(ctx context.Context, key string, capacity int, refillRate float64, now time.Time) (RateLimitResult, error) {
	if capacity <= 0 || refillRate <= 0 || key == "" {
		return RateLimitResult{}, entity.ErrInvalidInput
	}

	ttl := tokenBucketTTL(capacity, refillRate)
	res, err := r.client.Eval(ctx, rateLimitTakeScript, []string{key}, capacity, refillRate, now.UnixMilli(), int(ttl.Seconds())).Result()
	if err != nil {
		return RateLimitResult{}, mapRedisError(err)
	}

	values, ok := res.([]interface{})
	if !ok || len(values) != 3 {
		return RateLimitResult{}, entity.ErrRepositoryFailed
	}

	allowed, err := redisInt(values[0])
	if err != nil {
		return RateLimitResult{}, entity.ErrRepositoryFailed
	}

	remaining, err := redisInt(values[1])
	if err != nil {
		return RateLimitResult{}, entity.ErrRepositoryFailed
	}

	retryAfterMs, err := redisInt(values[2])
	if err != nil {
		return RateLimitResult{}, entity.ErrRepositoryFailed
	}

	return RateLimitResult{
		Allowed:    allowed == 1,
		Remaining:  int(remaining),
		RetryAfter: time.Duration(retryAfterMs) * time.Millisecond,
	}, nil
}

// 指定したRateLimitキーを削除して制限状態を初期化。
func (r *RedisIRateLimitRepository) Reset(ctx context.Context, key string) error {
	if key == "" {
		return entity.ErrInvalidInput
	}

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return mapRedisError(err)
	}

	return nil
}

// 重複防止キーがなければTTL付きで保存。
func (r *RedisEventDedupRepository) SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if key == "" || ttl <= 0 {
		return false, entity.ErrInvalidInput
	}

	ok, err := r.client.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, mapRedisError(err)
	}

	return ok, nil
}

// 重複防止キーが存在するか確認。
func (r *RedisEventDedupRepository) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, entity.ErrInvalidInput
	}

	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, mapRedisError(err)
	}

	return count > 0, nil
}

// 重複防止キーを削除。
func (r *RedisEventDedupRepository) Delete(ctx context.Context, key string) error {
	if key == "" {
		return entity.ErrInvalidInput
	}

	if err := r.client.Del(ctx, key).Err(); err != nil {
		return mapRedisError(err)
	}

	return nil
}

// 指定候補を表示済みとしてTTL付きで保存。
func (r *RedisModalSuppressionRepository) SetShown(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error {
	if actorKey == "" || rankTargetID == 0 {
		return entity.ErrInvalidInput
	}

	return r.setFlag(ctx, modalShownKey(actorKey, rankTargetID), ttl)
}

// 指定候補を表示済みか確認。
func (r *RedisModalSuppressionRepository) WasShown(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error) {
	if actorKey == "" || rankTargetID == 0 {
		return false, entity.ErrInvalidInput
	}

	return r.exists(ctx, modalShownKey(actorKey, rankTargetID))
}

// 同一ページ内のモーダル表示回数を1増やす。
func (r *RedisModalSuppressionRepository) IncrementPageCount(ctx context.Context, actorKey string, pagePath string, ttl time.Duration) (int64, error) {
	if actorKey == "" || pagePath == "" {
		return 0, entity.ErrInvalidInput
	}

	return r.increment(ctx, modalPageCountKey(actorKey, pagePath), ttl)
}

// 同一セッション内のモーダル表示回数を1増やす。
func (r *RedisModalSuppressionRepository) IncrementSessionCount(ctx context.Context, actorKey string, ttl time.Duration) (int64, error) {
	if actorKey == "" {
		return 0, entity.ErrInvalidInput
	}

	return r.increment(ctx, modalSessionCountKey(actorKey), ttl)
}

// 指定候補を閉じた候補としてTTL付きで保存。
func (r *RedisModalSuppressionRepository) SetClosed(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error {
	if actorKey == "" || rankTargetID == 0 {
		return entity.ErrInvalidInput
	}

	return r.setFlag(ctx, modalClosedKey(actorKey, rankTargetID), ttl)
}

// 指定候補を直近で閉じたか確認。
func (r *RedisModalSuppressionRepository) WasClosed(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error) {
	if actorKey == "" || rankTargetID == 0 {
		return false, entity.ErrInvalidInput
	}

	return r.exists(ctx, modalClosedKey(actorKey, rankTargetID))
}

// TTL付きのフラグ値をRedisへ保存する共通処理。
func (r *RedisModalSuppressionRepository) setFlag(ctx context.Context, key string, ttl time.Duration) error {
	if key == "" || ttl <= 0 {
		return entity.ErrInvalidInput
	}

	if err := r.client.Set(ctx, key, "1", ttl).Err(); err != nil {
		return mapRedisError(err)
	}

	return nil
}

// Redisキーが存在するか確認する共通処理。
func (r *RedisModalSuppressionRepository) exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, entity.ErrInvalidInput
	}

	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, mapRedisError(err)
	}

	return count > 0, nil
}

// Redisキーを加算し、初回だけTTLを設定する共通処理。
func (r *RedisModalSuppressionRepository) increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if key == "" || ttl <= 0 {
		return 0, entity.ErrInvalidInput
	}

	res, err := r.client.Eval(ctx, incrementWithTTLScript, []string{key}, int(ttl.Seconds())).Int64()
	if err != nil {
		return 0, mapRedisError(err)
	}

	return res, nil
}

// Redisでバッチlockを扱う実装。
type RedisBatchLockRepository struct {
	client *redis.Client
}

// バッチlock RepositoryのRedis実装。
func NewBatchLockRepository(client *redis.Client) IBatchLockRepository {
	return &RedisBatchLockRepository{client: client}
}

// 指定ownerでバッチlockを取得。
func (r *RedisBatchLockRepository) Acquire(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error) {
	if key == "" || owner == "" || ttl <= 0 {
		return false, entity.ErrInvalidInput
	}

	ok, err := r.client.SetNX(ctx, key, owner, ttl).Result()
	if err != nil {
		return false, mapRedisError(err)
	}

	return ok, nil
}

// ownerが一致する場合だけバッチlockを解除。
func (r *RedisBatchLockRepository) Release(ctx context.Context, key string, owner string) error {
	if key == "" || owner == "" {
		return entity.ErrInvalidInput
	}

	res, err := r.client.Eval(ctx, releaseLockScript, []string{key}, owner).Int64()
	if err != nil {
		return mapRedisError(err)
	}

	if res == 0 {
		return entity.ErrBatchLockFailed
	}

	return nil
}

// ownerが一致する場合だけバッチlockのTTLを延長。
func (r *RedisBatchLockRepository) Extend(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error) {
	if key == "" || owner == "" || ttl <= 0 {
		return false, entity.ErrInvalidInput
	}

	res, err := r.client.Eval(ctx, extendLockScript, []string{key}, owner, int(ttl.Milliseconds())).Int64()
	if err != nil {
		return false, mapRedisError(err)
	}

	return res == 1, nil
}

// TokenBucketのRedisキーが自然に消えるTTLを計算。
func tokenBucketTTL(capacity int, refillRate float64) time.Duration {
	seconds := int(float64(capacity)/refillRate*2) + 1
	if seconds < 60 {
		seconds = 60
	}

	return time.Duration(seconds) * time.Second
}

// 表示済みモーダル候補を識別するRedisキーを作る。
func modalShownKey(actorKey string, rankTargetID uint64) string {
	return fmt.Sprintf("modal:shown:%s:%d", actorKey, rankTargetID)
}

// 閉じられたモーダル候補を識別するRedisキーを作る。
func modalClosedKey(actorKey string, rankTargetID uint64) string {
	return fmt.Sprintf("modal:closed:%s:%d", actorKey, rankTargetID)
}

// ページ単位のモーダル表示回数を識別するRedisキーを作る。
func modalPageCountKey(actorKey string, pagePath string) string {
	return fmt.Sprintf("modal:page_count:%s:%s", actorKey, hashString(pagePath))
}

// セッション単位のモーダル表示回数を識別するRedisキーを作る。
func modalSessionCountKey(actorKey string) string {
	return fmt.Sprintf("modal:session_count:%s", actorKey)
}

// Redisキーに入れる長い文字列をsha256で短く固定化。
func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Redis由来のエラーをアプリ共通エラーへ変換。
func mapRedisError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, redis.Nil) {
		return entity.ErrNotFound
	}

	return entity.ErrRepositoryFailed
}

// Lua実行結果の数値をint64へ変換。
func redisInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, entity.ErrRepositoryFailed
	}
}

// TokenBucketの補充、消費、保存をRedis Luaで1回の処理に。
const rateLimitTakeScript = `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local ttl_seconds = tonumber(ARGV[4])

local tokens = tonumber(redis.call("HGET", key, "tokens"))
local updated_at = tonumber(redis.call("HGET", key, "updated_at"))

if tokens == nil then
  tokens = capacity
end

if updated_at == nil then
  updated_at = now_ms
end

local elapsed_ms = now_ms - updated_at
if elapsed_ms < 0 then
  elapsed_ms = 0
end

local refill = (elapsed_ms / 1000) * refill_rate
tokens = math.min(capacity, tokens + refill)

local allowed = 0
local retry_after_ms = 0

if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
else
  retry_after_ms = math.ceil(((1 - tokens) / refill_rate) * 1000)
end

redis.call("HSET", key, "tokens", tokens, "updated_at", now_ms)
redis.call("EXPIRE", key, ttl_seconds)

return {allowed, math.floor(tokens), retry_after_ms}
`

// カウント増加と初回TTL設定をRedis Luaで1回の原子的処理に。
const incrementWithTTLScript = `
local key = KEYS[1]
local ttl_seconds = tonumber(ARGV[1])
local count = redis.call("INCR", key)

if count == 1 then
  redis.call("EXPIRE", key, ttl_seconds)
end

return count
`

// ownerが一致する場合だけlockを削除。
const releaseLockScript = `
local key = KEYS[1]
local owner = ARGV[1]

if redis.call("GET", key) == owner then
  return redis.call("DEL", key)
end

return 0
`

// ownerが一致する場合だけlockの有効期限を延長。
const extendLockScript = `
local key = KEYS[1]
local owner = ARGV[1]
local ttl_ms = tonumber(ARGV[2])

if redis.call("GET", key) == owner then
  redis.call("PEXPIRE", key, ttl_ms)
  return 1
end

return 0
`
