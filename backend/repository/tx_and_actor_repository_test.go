package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
)

// ActionEventRepositoryのactor条件と、最後の検索条件hash取得を確認する。
// userとguestを正しく分け、actor指定が不正な場合はfail closedで0件になることを見る。
func TestActionEventRepository_ActorFilterFailClosedAndLastSearchHash(t *testing.T) {
	db := newRepositoryTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db, "actor")
	// Guest actor用のテストGuestSessionを作成する。
	guest := createTestGuestSession(t, db, "actor")
	// ActionEventRepositoryを作成する。
	repo := NewActionEventRepository(db)
	now := time.Now().UTC()

	// 古い検索条件hashと新しい検索条件hashを用意する。
	oldHash := "old-search-hash"
	newHash := "new-search-hash"

	// Userのclick、Guestのclick、Userの再検索eventを2件作る。
	// FindLastSearchHashでは、Userの最新re_searchであるnewHashが返る想定。
	events := []*model.ActionEvent{
		{
			UserID:     ptrUint64(user.ID),
			EventType:  entity.EventTypeClick,
			Placement:  entity.PlacementTop,
			PagePath:   "/",
			OccurredAt: now.Add(-3 * time.Minute),
		},
		{
			GuestSessionID: ptrUint64(guest.ID),
			EventType:      entity.EventTypeClick,
			Placement:      entity.PlacementTop,
			PagePath:       "/",
			OccurredAt:     now.Add(-2 * time.Minute),
		},
		{
			UserID:              ptrUint64(user.ID),
			EventType:           entity.EventTypeReSearch,
			Placement:           entity.PlacementSearchResult,
			SearchConditionHash: &oldHash,
			PagePath:            "/search",
			OccurredAt:          now.Add(-time.Minute),
		},
		{
			UserID:              ptrUint64(user.ID),
			EventType:           entity.EventTypeReSearch,
			Placement:           entity.PlacementSearchResult,
			SearchConditionHash: &newHash,
			PagePath:            "/search",
			OccurredAt:          now,
		},
	}

	// ActionEventをまとめてDBに作成する。
	if err := repo.BulkCreate(ctx, events); err != nil {
		t.Fatalf("bulk create events: %v", err)
	}

	// User actorのclick数を数える。
	userCount, err := repo.CountByActorAndTypeSince(ctx, ptrUint64(user.ID), nil, entity.EventTypeClick, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("count user events: %v", err)
	}

	// Userのclickは1件だけなので、1件になる想定。
	if userCount != 1 {
		t.Fatalf("user count = %d, want 1", userCount)
	}

	// Guest actorのclick数を数える。
	guestCount, err := repo.CountByActorAndTypeSince(ctx, nil, ptrUint64(guest.ID), entity.EventTypeClick, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("count guest events: %v", err)
	}

	// Guestのclickは1件だけなので、1件になる想定。
	if guestCount != 1 {
		t.Fatalf("guest count = %d, want 1", guestCount)
	}

	// user_idとguest_session_idを両方指定して数える。
	// actorはUserかGuestのどちらか一方なので、両方指定は不正扱いにする想定。
	bothCount, err := repo.CountByActorAndTypeSince(ctx, ptrUint64(user.ID), ptrUint64(guest.ID), entity.EventTypeClick, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("count both actor events: %v", err)
	}

	// fail closedとして0件になることを確認する。
	if bothCount != 0 {
		t.Fatalf("both actor count = %d, want 0", bothCount)
	}

	// user_idもguest_session_idも指定せずに数える。
	// actor不明のまま集計すると危険なので、これも不正扱いにする想定。
	noActorCount, err := repo.CountByActorAndTypeSince(ctx, nil, nil, entity.EventTypeClick, now.Add(-time.Hour))
	if err != nil {
		t.Fatalf("count no actor events: %v", err)
	}

	// fail closedとして0件になることを確認する。
	if noActorCount != 0 {
		t.Fatalf("no actor count = %d, want 0", noActorCount)
	}

	// User actorの最後の検索条件hashを取得する。
	lastHash, err := repo.FindLastSearchHash(ctx, ptrUint64(user.ID), nil)
	if err != nil {
		t.Fatalf("find last search hash: %v", err)
	}

	// Userのre_searchはoldHash、新hashの順で作成しているため、最新のnewHashが返る想定。
	if lastHash == nil || *lastHash != newHash {
		t.Fatalf("last hash = %v, want %s", lastHash, newHash)
	}

	// Guest actorの最後の検索条件hashを取得する。
	// Guestにはre_search eventを作っていないのでnilが返る想定。
	missingHash, err := repo.FindLastSearchHash(ctx, nil, ptrUint64(guest.ID))
	if err != nil {
		t.Fatalf("find missing search hash: %v", err)
	}

	// 検索履歴がないためnilであることを確認する。
	if missingHash != nil {
		t.Fatalf("missing hash = %v, want nil", *missingHash)
	}
}

// ModalDisplayLogRepositoryのactor制限付き更新を確認する。
// 他人のモーダル表示ログをクリック済みにできず、本人だけが更新できることを見る。
func TestModalDisplayLogRepository_ActorScopedUpdate(t *testing.T) {

	db := newRepositoryTestDB(t)
	ctx := context.Background()
	user := createTestUser(t, db, "modal-owner")

	// 別Userを作成する。
	// このUserでは、ownerのモーダルログを更新できないことを確認する。
	other := createTestUser(t, db, "modal-other")

	// モーダル表示対象になるRankTargetを作成する。
	target := createTestRankTarget(t, db, entity.ContentTypeBean, 1001)

	// ModalDisplayLogRepositoryを作成する。
	repo := NewModalDisplayLogRepository(db)
	now := time.Now().UTC()

	// ownerのモーダル表示ログを作成する。
	log := model.ModalDisplayLog{
		UserID:       ptrUint64(user.ID),
		RankTargetID: target.ID,
		Trigger:      entity.ModalTriggerFirstVisit,
		PagePath:     "/beans/1001",
		ShownAt:      now,
	}

	// modal_display_logsテーブルへ表示ログを保存する。
	if err := repo.Create(ctx, &log); err != nil {
		t.Fatalf("create modal display log: %v", err)
	}

	// 別Userがownerのモーダルログをクリック済みにしようとする。
	// actor条件が合わないため、更新対象0件になる想定。
	err := repo.MarkClickedForActor(ctx, log.ID, ptrUint64(other.ID), nil, now.Add(time.Minute))
	if !errors.Is(err, entity.ErrNoRowsAffected) {
		t.Fatalf("other actor click error = %v, want ErrNoRowsAffected", err)
	}

	// ownerのactor条件でログを取得する。
	got, err := repo.FindByIDForActor(ctx, log.ID, ptrUint64(user.ID), nil)
	if err != nil {
		t.Fatalf("find owner log: %v", err)
	}

	// 別Userによる更新は失敗しているため、clicked_atはnilのままであることを確認する。
	if got.ClickedAt != nil {
		t.Fatalf("clicked_at changed by other actor: %v", got.ClickedAt)
	}

	// owner本人がクリックした時刻を用意する。
	clickedAt := now.Add(2 * time.Minute)

	// owner本人がモーダルログをクリック済みにする。
	if err := repo.MarkClickedForActor(ctx, log.ID, ptrUint64(user.ID), nil, clickedAt); err != nil {
		t.Fatalf("owner click: %v", err)
	}

	// ownerのactor条件で、クリック後のログを取得する。
	got, err = repo.FindByIDForActor(ctx, log.ID, ptrUint64(user.ID), nil)
	if err != nil {
		t.Fatalf("find clicked owner log: %v", err)
	}

	// clicked_atがowner本人のクリック時刻で保存されていることを確認する。
	if got.ClickedAt == nil || !got.ClickedAt.Equal(clickedAt) {
		t.Fatalf("clicked_at = %v, want %v", got.ClickedAt, clickedAt)
	}

	// 同じログをもう一度クリック済みにしようとする。
	// すでにclicked_atが入っているため、2回目は更新対象0件になる想定。
	err = repo.MarkClickedForActor(ctx, log.ID, ptrUint64(user.ID), nil, now.Add(3*time.Minute))
	if !errors.Is(err, entity.ErrNoRowsAffected) {
		t.Fatalf("second owner click error = %v, want ErrNoRowsAffected", err)
	}

	// user_idとguest_session_idを両方指定して取得しようとする。
	// actor指定として不正なため、fail closedでErrNotFoundになる想定。
	_, err = repo.FindByIDForActor(ctx, log.ID, ptrUint64(user.ID), ptrUint64(999))
	if !errors.Is(err, entity.ErrNotFound) {
		t.Fatalf("both actor find error = %v, want ErrNotFound", err)
	}
}

// Repositoryテスト用のGuestSessionを作成する。
// ActionEventなど、guest_session_idが必要なテストの事前データとして使う。
func createTestGuestSession(t *testing.T, db *gorm.DB, suffix string) model.GuestSession {
	t.Helper()

	// テスト内で使う基準時刻を用意する。
	now := time.Now().UTC()

	// 有効なGuestSessionを作る。
	session := model.GuestSession{
		SessionKeyHash: "guest-hash-" + suffix,
		FirstSeenAt:    now,
		LastSeenAt:     now,
		ExpiresAt:      now.Add(24 * time.Hour),
	}

	// guest_sessionsテーブルへテスト用GuestSessionを保存する。
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("create guest session: %v", err)
	}

	return session
}
