package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"coffee-ranker/entity"

	"github.com/redis/go-redis/v9"
)

// Redisを使うRepositoryテスト用のclientを作成。
func newRepositoryTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	// 環境変数でRepositoryテストをスキップできるようにする。
	if os.Getenv("SKIP_REPOSITORY_INTEGRATION_TESTS") == "1" {
		t.Skip("SKIP_REPOSITORY_INTEGRATION_TESTS=1")
	}

	// 未指定の場合は、ローカルのRedisに接続。
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}

	// テスト用Redis clientを作成。
	// DBは15を使い、本番や開発用のRedisデータと混ざりにくくする。
	client := redis.NewClient(&redis.Options{Addr: addr, DB: 15})

	// Redis接続確認用のcontextを作る。
	// 3秒以内に接続できなければ、テストを失敗させる。
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// RedisへPINGを送り、接続できるか確認。
	// 接続できない場合はclientを閉じて、テストを失敗させる。
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("connect test redis: %v", err)
	}

	// このテスト専用のRedis key prefixを作る。
	// テストごとにkeyを分けることで、他のテスト結果と混ざらないように。
	prefix := testKeyPrefix(t)

	// テスト終了時に、このテストで作ったRedis keyだけ削除。
	// prefix + "*" に一致keyを消してから、Redis clientを閉じる。
	t.Cleanup(func() {
		cleanupRedisKeys(context.Background(), client, prefix+"*")
		_ = client.Close()
	})

	return client
}

// テストごとにRedis keyのprefixを作る。
// 同じRedis DBを使っても、テスト同士のkeyが衝突しないように。
func testKeyPrefix(t *testing.T) string {
	t.Helper()

	// t.Name()には "/" や空白などが入る場合がある。
	// Redis keyとして扱いやすい形に置き換える。
	name := strings.NewReplacer("/", ":", " ", "_", "-", "_").Replace(t.Name())

	// Repositoryテスト用だと分かるprefixを付ける。
	return "repo_test:" + name + ":"
}

// 指定したpatternに一致Redis keyを削除。
// テスト終了時の後片付けとして使う。
func cleanupRedisKeys(ctx context.Context, client *redis.Client, pattern string) {
	var cursor uint64

	for {
		// SCANでpatternに一致するkeyを少しずつ取得。
		// KEYSではなくSCANを使うことで、Redisに重い全件検索をかけにくく。
		keys, next, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}

		// 見つかったkeyがあれば削除。
		if len(keys) > 0 {
			_ = client.Del(ctx, keys...).Err()
		}

		// nextが0なら、SCANが最後まで終わった。
		if next == 0 {
			return
		}

		// 次のSCAN位置を保存。
		cursor = next
	}
}

// RateLimitRepositoryのTokenBucket処理を確認。
// 指定回数までは許可され、上限を超えると拒否され、時間経過で再補充されるかどうか。
func TestRateLimitRepository_TokenBucket(t *testing.T) {
	client := newRepositoryTestRedis(t)

	// Repositoryメソッドへ渡すcontext。
	ctx := context.Background()

	repo := NewRateLimitRepository(client)

	// テストごとに衝突しないRedis keyを作る。
	key := testKeyPrefix(t) + "rate_limit"

	now := time.Now().UTC()

	// 1回目のアクセスを実行。
	// capacity=2なので、最初は許可され、残りtokenは1になる想定。
	first, err := repo.Take(ctx, key, 2, 1, now)
	if err != nil {
		t.Fatalf("take first: %v", err)
	}
	if !first.Allowed || first.Remaining != 1 {
		t.Fatalf("first result = %+v, want allowed remaining=1", first)
	}

	// 2回目のアクセスを実行。
	// capacity=2なので、2回目も許可され、残りtokenは0になる想定。
	second, err := repo.Take(ctx, key, 2, 1, now)
	if err != nil {
		t.Fatalf("take second: %v", err)
	}
	if !second.Allowed || second.Remaining != 0 {
		t.Fatalf("second result = %+v, want allowed remaining=0", second)
	}

	// 3回目のアクセスを実行。
	// tokenを使い切っているため拒否され、RetryAfterが返る想定。
	third, err := repo.Take(ctx, key, 2, 1, now)
	if err != nil {
		t.Fatalf("take third: %v", err)
	}
	if third.Allowed || third.RetryAfter <= 0 {
		t.Fatalf("third result = %+v, want denied with retry_after", third)
	}

	// 1秒後の時刻でアクセス。
	// refillRate=1なので、1秒経過によりtokenが1つ補充され、再び許可される想定。
	refilled, err := repo.Take(ctx, key, 2, 1, now.Add(time.Second))
	if err != nil {
		t.Fatalf("take refilled: %v", err)
	}
	if !refilled.Allowed {
		t.Fatalf("refilled result = %+v, want allowed", refilled)
	}

	// RateLimit状態をリセット。
	// 管理者による制限解除やテスト後の状態確認を想定した処理。
	if err := repo.Reset(ctx, key); err != nil {
		t.Fatalf("reset rate limit: %v", err)
	}

	// reset後に再度アクセス。
	// bucketが初期状態に戻るため、許可され、残りtokenは1になる想定。
	afterReset, err := repo.Take(ctx, key, 2, 1, now)
	if err != nil {
		t.Fatalf("take after reset: %v", err)
	}
	if !afterReset.Allowed || afterReset.Remaining != 1 {
		t.Fatalf("after reset = %+v, want allowed remaining=1", afterReset)
	}
}

// 初回だけ保存でき、2回目は保存できず、存在確認と削除ができることを見る。
func TestEventDedupRepository_SetExistsDelete(t *testing.T) {

	client := newRepositoryTestRedis(t)

	ctx := context.Background()
	repo := NewEventDedupRepository(client)

	// テストごとに衝突しないRedis keyを作る。
	key := testKeyPrefix(t) + "dedup"

	// dedup keyを初回保存。
	// まだ存在しないkeyなのでtrueが返る想定。
	ok, err := repo.SetIfNotExists(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("set first dedup: %v", err)
	}
	if !ok {
		t.Fatal("first dedup set should be true")
	}

	// 同じdedup keyをもう一度保存。
	// すでに存在keyなのでfalseが返る想定。
	ok, err = repo.SetIfNotExists(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("set second dedup: %v", err)
	}
	if ok {
		t.Fatal("second dedup set should be false")
	}

	// dedup keyがRedis上に存在ことを確認。
	exists, err := repo.Exists(ctx, key)
	if err != nil {
		t.Fatalf("exists dedup: %v", err)
	}
	if !exists {
		t.Fatal("dedup key should exist")
	}

	// dedup keyを削除。
	if err := repo.Delete(ctx, key); err != nil {
		t.Fatalf("delete dedup: %v", err)
	}

	// 削除後はdedup keyが存在しないことを確認。
	exists, err = repo.Exists(ctx, key)
	if err != nil {
		t.Fatalf("exists after delete: %v", err)
	}
	if exists {
		t.Fatal("dedup key should not exist after delete")
	}
}

// ModalSuppressionRepositoryの表示抑制状態と表示回数を確認。
// shownとclosedの一時フラグ、ページ単位回数、セッション単位回数がRedisで管理できることを見る。
func TestModalSuppressionRepository_FlagsAndCounts(t *testing.T) {
	client := newRepositoryTestRedis(t)
	ctx := context.Background()

	// ModalSuppressionRepositoryを作成。
	repo := NewModalSuppressionRepository(client)

	// UserまたはGuestSessionを表すテスト用actor keyを作る。
	actorKey := testKeyPrefix(t) + "actor"

	// ページ単位の表示回数を確認ためのpath。
	pagePath := "/beans/1?query=long-value"

	// モーダル候補になったRankTargetのID。
	var rankTargetID uint64 = 123

	// まだSetShownしていないため、表示済みではないことを確認。
	shown, err := repo.WasShown(ctx, actorKey, rankTargetID)
	if err != nil {
		t.Fatalf("was shown before set: %v", err)
	}
	if shown {
		t.Fatal("shown should be false before set")
	}

	// 指定actorに対して、このRankTargetを表示済みとして保存。
	if err := repo.SetShown(ctx, actorKey, rankTargetID, time.Minute); err != nil {
		t.Fatalf("set shown: %v", err)
	}

	// SetShown後は、表示済みとして判定されることを確認。
	shown, err = repo.WasShown(ctx, actorKey, rankTargetID)
	if err != nil {
		t.Fatalf("was shown after set: %v", err)
	}
	if !shown {
		t.Fatal("shown should be true after set")
	}

	// まだSetClosedしていないため、閉じた状態ではないことを確認。
	closed, err := repo.WasClosed(ctx, actorKey, rankTargetID)
	if err != nil {
		t.Fatalf("was closed before set: %v", err)
	}
	if closed {
		t.Fatal("closed should be false before set")
	}

	// 指定actorに対して、このRankTargetを閉じた状態として保存。
	if err := repo.SetClosed(ctx, actorKey, rankTargetID, time.Minute); err != nil {
		t.Fatalf("set closed: %v", err)
	}

	// SetClosed後は、閉じた状態として判定されることを確認。
	closed, err = repo.WasClosed(ctx, actorKey, rankTargetID)
	if err != nil {
		t.Fatalf("was closed after set: %v", err)
	}
	if !closed {
		t.Fatal("closed should be true after set")
	}

	// 同じactor、同じpagePathでページ内表示回数を1増やす。
	pageCount1, err := repo.IncrementPageCount(ctx, actorKey, pagePath, time.Minute)
	if err != nil {
		t.Fatalf("increment page count first: %v", err)
	}

	// もう一度増やす。
	pageCount2, err := repo.IncrementPageCount(ctx, actorKey, pagePath, time.Minute)
	if err != nil {
		t.Fatalf("increment page count second: %v", err)
	}

	// 1回目は1、2回目は2になることを確認。
	// これにより、同一ページ内のモーダル表示上限判定に使える。
	if pageCount1 != 1 || pageCount2 != 2 {
		t.Fatalf("page counts = %d/%d, want 1/2", pageCount1, pageCount2)
	}

	// actor単位でセッション内表示回数を1増やす。
	sessionCount1, err := repo.IncrementSessionCount(ctx, actorKey, time.Minute)
	if err != nil {
		t.Fatalf("increment session count first: %v", err)
	}

	// もう一度増やす。
	sessionCount2, err := repo.IncrementSessionCount(ctx, actorKey, time.Minute)
	if err != nil {
		t.Fatalf("increment session count second: %v", err)
	}

	// 1回目は1、2回目は2になることを確認。
	// これにより、同一セッション内のモーダル表示上限判定に使える。
	if sessionCount1 != 1 || sessionCount2 != 2 {
		t.Fatalf("session counts = %d/%d, want 1/2", sessionCount1, sessionCount2)
	}
}

// BatchLockRepositoryのlock所有者チェックを確認。
// lockを取ったownerだけが延長・解除でき、別ownerは操作できないことを見る。
func TestBatchLockRepository_OwnerScopedReleaseAndExtend(t *testing.T) {
	client := newRepositoryTestRedis(t)
	ctx := context.Background()
	repo := NewBatchLockRepository(client)
	key := testKeyPrefix(t) + "batch_lock"

	// owner-1がlockを取得。
	// まだlockは存在しないため、取得成功想定。
	acquired, err := repo.Acquire(ctx, key, "owner-1", time.Minute)
	if err != nil {
		t.Fatalf("acquire first owner: %v", err)
	}
	if !acquired {
		t.Fatal("first owner should acquire lock")
	}

	// owner-2が同じlockを取得。
	// すでにowner-1が取得済みなので、取得できない想定。
	acquired, err = repo.Acquire(ctx, key, "owner-2", time.Minute)
	if err != nil {
		t.Fatalf("acquire second owner: %v", err)
	}
	if acquired {
		t.Fatal("second owner should not acquire existing lock")
	}

	// owner-2がlockを延長。
	// lock所有者ではないため、延長できない想定。
	extended, err := repo.Extend(ctx, key, "owner-2", time.Minute)
	if err != nil {
		t.Fatalf("extend wrong owner: %v", err)
	}
	if extended {
		t.Fatal("wrong owner should not extend lock")
	}

	// owner-2がlockを解除。
	// lock所有者ではないため、ErrBatchLockFailedになる想定。
	err = repo.Release(ctx, key, "owner-2")
	if !errors.Is(err, entity.ErrBatchLockFailed) {
		t.Fatalf("wrong owner release error = %v, want ErrBatchLockFailed", err)
	}

	// 正しいownerであるowner-1がlockを延長。
	// 自分が取得したlockなので、延長できる想定。
	extended, err = repo.Extend(ctx, key, "owner-1", time.Minute)
	if err != nil {
		t.Fatalf("extend owner: %v", err)
	}
	if !extended {
		t.Fatal("owner should extend lock")
	}

	// 正しいownerであるowner-1がlockを解除。
	// 自分が取得したlockなので、解除できる想定。
	if err := repo.Release(ctx, key, "owner-1"); err != nil {
		t.Fatalf("release owner: %v", err)
	}

	// lock解除後にowner-2がlockを取得。
	// すでにlockは解放済みなので、今度は取得成功想定。
	acquired, err = repo.Acquire(ctx, key, "owner-2", time.Minute)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if !acquired {
		t.Fatal("second owner should acquire after release")
	}
}
