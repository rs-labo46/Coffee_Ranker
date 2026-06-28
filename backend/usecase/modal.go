package usecase

import (
	"context"
	"strconv"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// Redis keyに使うactorをuser:IDまたはguest:IDで作成。
func modalActorKey(actor Actor) (string, error) {
	if err := requireActor(actor); err != nil {
		return "", err
	}
	if actor.UserID != nil {
		return "user:" + strconv.FormatUint(*actor.UserID, 10), nil
	}
	return "guest:" + strconv.FormatUint(*actor.GuestSessionID, 10), nil
}

// 推薦モーダルの表示可否、表示記録、クリック、閉じる処理。
type ModalUsecase struct {
	displays    repository.ModalDisplayLogRepository
	blocks      repository.ModalBlockLogRepository
	rankTargets repository.RankTargetRepository
	saved       repository.SavedItemRepository
	events      repository.ActionEventRepository
	suppression repository.ModalSuppressionRepository
}

// モーダル表示判断に必要なactor、候補、表示理由、表示場所。
// RankTargetIDは、すでに候補として選ばれた対象を想定。
type ShowModalInput struct {
	Actor        Actor
	RankTargetID uint64
	Trigger      entity.ModalTrigger
	PagePath     string
}

// モーダルクリック・クローズに必要なactorと表示ログID。
type ModalActionInput struct {
	Actor             Actor
	ModalDisplayLogID uint64
	PagePath          string
}

// モーダル制御に必要なRepositoryを受け取るコンストラクタ。
func NewModalUsecase(displays repository.ModalDisplayLogRepository, blocks repository.ModalBlockLogRepository, rankTargets repository.RankTargetRepository, saved repository.SavedItemRepository, events repository.ActionEventRepository, suppression repository.ModalSuppressionRepository) *ModalUsecase {
	return &ModalUsecase{
		displays:    displays,
		blocks:      blocks,
		rankTargets: rankTargets,
		saved:       saved,
		events:      events,
		suppression: suppression,
	}
}

// 表示条件を確認し、表示ログ・event・Redis抑制を記録する。
func (u *ModalUsecase) Show(ctx context.Context, input ShowModalInput) (*model.ModalDisplayLog, error) {
	// UserまたはGuestSessionの片方だけであることを確認。
	if err := requireActor(input.Actor); err != nil {
		return nil, err
	}

	// 表示候補、表示ページ、表示理由が不足している場合は不正入力。
	if input.RankTargetID == 0 || input.PagePath == "" || !validModalTrigger(input.Trigger) {
		return nil, entity.ErrInvalidInput
	}

	now := time.Now()
	cooldown := time.Duration(entity.ModalCooldownHours) * time.Hour
	since := now.Add(-cooldown)

	// Redisの抑制keyで使うactor識別子を作る。
	key, err := modalActorKey(input.Actor)
	if err != nil {
		return nil, err
	}

	// 表示候補が有効なランキング対象か確認。
	if err := ensureActiveRankTarget(ctx, u.rankTargets, input.RankTargetID); err != nil {
		return nil, err
	}

	// 同一ページでの表示上限を確認。
	pageCount, err := u.displays.CountShownOnPage(ctx, input.Actor.UserID, input.Actor.GuestSessionID, input.PagePath, since)
	if err != nil {
		return nil, err
	}
	if pageCount >= int64(entity.MaxModalPerPage) {
		return nil, u.blockModal(ctx, input.Actor, &input.RankTargetID, entity.ModalBlockPageLimitReached, input.PagePath, now)
	}

	// 同一セッションでの表示上限を確認。
	sessionCount, err := u.displays.CountShownInSession(ctx, input.Actor.UserID, input.Actor.GuestSessionID, since)
	if err != nil {
		return nil, err
	}
	if sessionCount >= int64(entity.MaxModalPerSession) {
		return nil, u.blockModal(ctx, input.Actor, &input.RankTargetID, entity.ModalBlockSessionLimitReached, input.PagePath, now)
	}

	// Userの場合、保存済みコンテンツは再推薦しない。
	if input.Actor.UserID != nil {
		saved, err := u.saved.ExistsActive(ctx, *input.Actor.UserID, input.RankTargetID)
		if err != nil {
			return nil, err
		}
		if saved {
			return nil, u.blockModal(ctx, input.Actor, &input.RankTargetID, entity.ModalBlockAlreadySaved, input.PagePath, now)
		}
	}

	// 同じ候補をクールダウン内に再表示しない。
	shown, err := u.suppression.WasShown(ctx, key, input.RankTargetID)
	if err != nil {
		return nil, err
	}
	if shown {
		return nil, u.blockModal(ctx, input.Actor, &input.RankTargetID, entity.ModalBlockRecentlyShown, input.PagePath, now)
	}

	// 直近で閉じられた候補をすぐ再表示しない。
	closed, err := u.suppression.WasClosed(ctx, key, input.RankTargetID)
	if err != nil {
		return nil, err
	}
	if closed {
		return nil, u.blockModal(ctx, input.Actor, &input.RankTargetID, entity.ModalBlockRecentlyClosed, input.PagePath, now)
	}

	// モーダルを表示した記録をDBに保存するためのデータを作る。
	log := &model.ModalDisplayLog{
		UserID:         input.Actor.UserID,
		GuestSessionID: input.Actor.GuestSessionID,
		RankTargetID:   input.RankTargetID,
		Trigger:        input.Trigger,
		PagePath:       input.PagePath,
		ShownAt:        now,
	}
	if err := u.displays.Create(ctx, log); err != nil {
		return nil, err
	}

	// 同じ候補をクールダウン内に再表示しないよう、Redisへ一時記録を残す。
	// 失敗しても表示ログ作成済みのため、モーダル表示自体は成功扱いにする。
	_ = u.suppression.SetShown(ctx, key, input.RankTargetID, cooldown)

	// modal_impression eventも補助記録。
	// event作成に失敗しても、表示本体は成功扱いに。
	if u.events != nil {
		_ = u.events.Create(ctx, &model.ActionEvent{
			UserID:            input.Actor.UserID,
			GuestSessionID:    input.Actor.GuestSessionID,
			EventType:         entity.EventTypeModalImpression,
			RankTargetID:      &input.RankTargetID,
			Placement:         entity.PlacementModal,
			ModalDisplayLogID: &log.ID,
			PagePath:          input.PagePath,
			OccurredAt:        now,
		})
	}

	return log, nil
}

// actor本人の表示ログにクリック時刻を入れ、modal_click eventをbest effortで記録する。
func (u *ModalUsecase) Click(ctx context.Context, input ModalActionInput) error {
	// UserまたはGuestSessionの片方だけであることを確認。
	if err := requireActor(input.Actor); err != nil {
		return err
	}

	// 表示ログIDと発生ページがない場合は不正入力。
	if input.ModalDisplayLogID == 0 || input.PagePath == "" {
		return entity.ErrInvalidInput
	}

	now := time.Now()

	// actor条件付きで表示ログを取得。
	// 他人のmodal_display_log_idを指定しても取得できない。
	log, err := u.displays.FindByIDForActor(ctx, input.ModalDisplayLogID, input.Actor.UserID, input.Actor.GuestSessionID)
	if err != nil {
		return err
	}

	// actor条件付きでクリック済みに更新。
	if err := u.displays.MarkClickedForActor(ctx, input.ModalDisplayLogID, input.Actor.UserID, input.Actor.GuestSessionID, now); err != nil {
		return err
	}

	// modal_click eventは補助記録。
	// event作成に失敗しても、クリック本体は成功扱いに。
	if u.events != nil {
		_ = u.events.Create(ctx, &model.ActionEvent{
			UserID:            input.Actor.UserID,
			GuestSessionID:    input.Actor.GuestSessionID,
			EventType:         entity.EventTypeModalClick,
			RankTargetID:      &log.RankTargetID,
			Placement:         entity.PlacementModal,
			ModalDisplayLogID: &input.ModalDisplayLogID,
			PagePath:          input.PagePath,
			OccurredAt:        now,
		})
	}

	return nil
}

// actor条件付きで表示ログへ閉じた時刻を入れ、modal_close eventと閉じた候補の抑制を保存。
func (u *ModalUsecase) Close(ctx context.Context, input ModalActionInput) error {
	// UserまたはGuestSessionの片方だけであることを確認。
	if err := requireActor(input.Actor); err != nil {
		return err
	}

	// 表示ログIDと発生ページがない場合は不正入力。
	if input.ModalDisplayLogID == 0 || input.PagePath == "" {
		return entity.ErrInvalidInput
	}

	now := time.Now()

	// actor条件付きで表示ログを取得。
	// 他人のmodal_display_log_idを指定しても取得できない。
	log, err := u.displays.FindByIDForActor(ctx, input.ModalDisplayLogID, input.Actor.UserID, input.Actor.GuestSessionID)
	if err != nil {
		return err
	}

	// actor条件付きでクローズ済みに更新。
	if err := u.displays.MarkClosedForActor(ctx, input.ModalDisplayLogID, input.Actor.UserID, input.Actor.GuestSessionID, now); err != nil {
		return err
	}

	// 閉じた候補はクールダウン内に再表示しない。
	key, err := modalActorKey(input.Actor)
	if err != nil {
		return err
	}
	_ = u.suppression.SetClosed(ctx, key, log.RankTargetID, time.Duration(entity.ModalCooldownHours)*time.Hour)

	// modal_close eventは補助記録。
	// event作成に失敗しても、クローズ本体は成功扱いに。
	if u.events != nil {
		_ = u.events.Create(ctx, &model.ActionEvent{
			UserID:            input.Actor.UserID,
			GuestSessionID:    input.Actor.GuestSessionID,
			EventType:         entity.EventTypeModalClose,
			RankTargetID:      &log.RankTargetID,
			Placement:         entity.PlacementModal,
			ModalDisplayLogID: &input.ModalDisplayLogID,
			PagePath:          input.PagePath,
			OccurredAt:        now,
		})
	}

	return nil
}

// 表示しない理由をDBに残し、呼び出し元へ候補なしエラーを返す。
func (u *ModalUsecase) blockModal(ctx context.Context, actor Actor, candidateRankTargetID *uint64, reason entity.ModalBlockReason, pagePath string, blockedAt time.Time) error {
	// block logは補助記録。
	// 保存に失敗しても、モーダルを出さない。判断自体は変えない。
	_ = u.blocks.Create(ctx, modalBlock(actor, candidateRankTargetID, reason, pagePath, blockedAt))
	return entity.ErrModalCandidateNotFound
}

// モーダルを出さなかった理由を保存するmodelを作成。
func modalBlock(actor Actor, candidateRankTargetID *uint64, reason entity.ModalBlockReason, pagePath string, blockedAt time.Time) *model.ModalBlockLog {
	return &model.ModalBlockLog{
		UserID:                actor.UserID,
		GuestSessionID:        actor.GuestSessionID,
		CandidateRankTargetID: candidateRankTargetID,
		Reason:                reason,
		PagePath:              pagePath,
		BlockedAt:             blockedAt,
	}
}

// ModalTriggerが許可された値か確認。
func validModalTrigger(trigger entity.ModalTrigger) bool {
	switch trigger {
	case entity.ModalTriggerFirstVisit,
		entity.ModalTriggerScrollEnd,
		entity.ModalTriggerBeanStay,
		entity.ModalTriggerArticleStay,
		entity.ModalTriggerSameOriginViewed,
		entity.ModalTriggerSameRoastClicked,
		entity.ModalTriggerSavedContent,
		entity.ModalTriggerGoodRating,
		entity.ModalTriggerReSearch:
		return true
	default:
		return false
	}
}
