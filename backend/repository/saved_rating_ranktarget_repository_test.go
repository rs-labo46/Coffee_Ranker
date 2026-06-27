package repository

import (
	"context"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
)

// 保存・評価・ランキング対象など、rank_target_idが必要なテストの事前データとして使う。
func createTestRankTarget(t *testing.T, db *gorm.DB, contentType entity.ContentType, contentID uint64) model.RankTarget {
	t.Helper()

	// BeanやArticleなど、ランキング対象になるデータを作る。
	// content_typeとcontent_idの組み合わせで、どのコンテンツを指すかを表す。
	target := model.RankTarget{
		ContentType: contentType,
		ContentID:   contentID,
		IsActive:    true,
	}

	// rank_targetsテーブルへテスト用データを保存する。
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("create rank target: %v", err)
	}

	return target
}

// uint64の値をポインタにするhelper。
// user_idやguest_session_idなど、nilを許可する項目のテストで使う。
func ptrUint64(v uint64) *uint64 {
	return &v
}

// stringの値をポインタにするhelper。
// nilを許可する文字列項目のテストで使う。
func ptrString(v string) *string {
	return &v
}

// 同じcontent_typeとcontent_idで2回呼んでも、重複作成されず同じRankTargetが返ることを見る。
func TestRankTargetRepository_FindOrCreateDoesNotDuplicate(t *testing.T) {
	db := newRepositoryTestDB(t)
	ctx := context.Background()
	repo := NewRankTargetRepository(db)

	// Beanのcontent_id=2001に対応するRankTargetを取得または作成する。
	// 初回なので、DBに新規作成される想定。
	first, err := repo.FindOrCreate(ctx, entity.ContentTypeBean, 2001)
	if err != nil {
		t.Fatalf("find or create first: %v", err)
	}

	// 同じcontent_typeとcontent_idでもう一度FindOrCreateする。
	// すでに存在するため、新規作成ではなく既存のRankTargetが返る想定。
	second, err := repo.FindOrCreate(ctx, entity.ContentTypeBean, 2001)
	if err != nil {
		t.Fatalf("find or create second: %v", err)
	}

	// 1回目と2回目で同じIDが返ることを確認する。
	// ここでIDが違う場合、同じランキング対象が重複作成されている。
	if first.ID == 0 || second.ID == 0 || first.ID != second.ID {
		t.Fatalf("target ids = %d/%d, want same non-zero", first.ID, second.ID)
	}

	// activeなBeanのRankTarget一覧を取得する。
	list, err := repo.ListActiveByType(ctx, entity.ContentTypeBean)
	if err != nil {
		t.Fatalf("list active targets: %v", err)
	}

	// 同じBean対象は重複作成されていないため、activeなRankTargetは1件だけの想定。
	if len(list) != 1 {
		t.Fatalf("active targets = %d, want 1", len(list))
	}
}

// SavedItemRepositoryの保存・解除・再保存を確認する。
// 保存解除後に再保存したとき、新しい行を作らず、元の行を復活できることを見る。
func TestSavedItemRepository_SaveRemoveRestore(t *testing.T) {
	db := newRepositoryTestDB(t)
	ctx := context.Background()

	// 保存するUserを作成。
	user := createTestUser(t, db, "saved")

	// 保存対象になるRankTargetを作成。
	target := createTestRankTarget(t, db, entity.ContentTypeBean, 3001)

	// SavedItemRepositoryを作成。
	repo := NewSavedItemRepository(db)

	// テスト内で使う基準時刻を用意。
	now := time.Now().UTC()

	// UserがRankTargetを保存。
	item, err := repo.SaveOrRestore(ctx, user.ID, target.ID, now)
	if err != nil {
		t.Fatalf("save item: %v", err)
	}

	// 保存直後なのでIDがあり、removed_atがnilであることを確認する。
	// removed_atがnilなら、現在も保存中という意味。
	if item.ID == 0 || item.RemovedAt != nil {
		t.Fatalf("saved item = %+v, want active item", item)
	}

	// 保存済みとして存在するか確認する。
	exists, err := repo.ExistsActive(ctx, user.ID, target.ID)
	if err != nil {
		t.Fatalf("exists active after save: %v", err)
	}

	// 保存直後なのでactiveな保存データが存在する想定。
	if !exists {
		t.Fatal("saved item should exist after save")
	}

	// 保存を解除する時間を用意。
	removedAt := now.Add(time.Minute)

	// Userの保存を解除。
	// 実装側では物理削除ではなく、removed_atを入れる想定。
	if err := repo.Remove(ctx, user.ID, target.ID, removedAt); err != nil {
		t.Fatalf("remove item: %v", err)
	}

	// 保存解除後に、activeな保存データとして存在するか確認する。
	exists, err = repo.ExistsActive(ctx, user.ID, target.ID)
	if err != nil {
		t.Fatalf("exists active after remove: %v", err)
	}

	// removed_atが入ったため、activeな保存データとしては存在しない想定。
	if exists {
		t.Fatal("saved item should not be active after remove")
	}

	// 同じUser、同じRankTargetを再保存する。
	// 既存行のremoved_atをnilに戻して復活する想定。
	restored, err := repo.SaveOrRestore(ctx, user.ID, target.ID, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("restore item: %v", err)
	}

	// 再保存では新しい行を作らず、元のsaved_items行を復活することを確認。
	if restored.ID != item.ID {
		t.Fatalf("restored id = %d, want original id %d", restored.ID, item.ID)
	}

	// 復活後はremoved_atがnilに戻っていることを確認する。
	if restored.RemovedAt != nil {
		t.Fatalf("restored removed_at = %v, want nil", restored.RemovedAt)
	}
}

// RatingRepositoryの評価登録・評価更新・集計を確認する。
// 同じUserの再評価は同じ行を更新し、別Userの評価は別行として集計されることを見る。
func TestRatingRepository_UpsertAndAggregate(t *testing.T) {
	db := newRepositoryTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db, "rating-1")

	// 別Userの評価も集計に含めるため、もう1人Userを作成。
	other := createTestUser(t, db, "rating-2")

	// 評価対象になるArticleのRankTargetを作成。
	target := createTestRankTarget(t, db, entity.ContentTypeArticle, 4001)
	repo := NewRatingRepository(db)
	now := time.Now().UTC()

	// 1人目のUserがGood評価を登録。
	first, err := repo.Upsert(ctx, user.ID, target.ID, entity.RatingScoreGood, now)
	if err != nil {
		t.Fatalf("upsert first rating: %v", err)
	}

	// 同じUserが同じ対象にBad評価を登録。
	// Upsertなので、新規行を作らず既存の評価行を更新する想定。
	second, err := repo.Upsert(ctx, user.ID, target.ID, entity.RatingScoreBad, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("upsert second rating: %v", err)
	}

	// 同じUserの再評価なので、1回目と2回目のrating IDは同じになる想定。
	if first.ID != second.ID {
		t.Fatalf("rating ids = %d/%d, want same row", first.ID, second.ID)
	}

	// 2回目でBadに更新したため、現在のscoreがBadになっていることを確認。
	if second.Score != entity.RatingScoreBad {
		t.Fatalf("score = %d, want bad", second.Score)
	}

	// 別Userが同じ対象にGood評価を登録。
	// 別Userなので、別のrating行として保存される想定。
	if _, err := repo.Upsert(ctx, other.ID, target.ID, entity.RatingScoreGood, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("upsert other rating: %v", err)
	}

	// 対象RankTargetに紐づく評価数を数える。
	count, err := repo.CountByTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("count ratings: %v", err)
	}

	// userの評価1件とotherの評価1件で、合計2件になる想定。
	if count != 2 {
		t.Fatalf("rating count = %d, want 2", count)
	}

	// Good数、Bad数、Good率、Bad率を集計する。
	agg, err := repo.AggregateByTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("aggregate ratings: %v", err)
	}

	// userは最終的にBad、otherはGoodなので、
	// 評価数2、Good1、Bad1になる想定。
	if agg.RatingCount != 2 || agg.GoodCount != 1 || agg.BadCount != 1 {
		t.Fatalf("aggregate = %+v, want rating=2 good=1 bad=1", agg)
	}

	// Good1件、Bad1件、合計2件なので、それぞれ0.5になる想定。
	if agg.GoodRate != 0.5 || agg.BadRate != 0.5 {
		t.Fatalf("rates = good %.2f bad %.2f, want 0.50/0.50", agg.GoodRate, agg.BadRate)
	}
}
