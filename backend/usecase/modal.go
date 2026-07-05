package usecase

import (
	"context"
	"errors"
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
	displays    repository.IModalDisplayLogRepository
	blocks      repository.IModalBlockLogRepository
	rankTargets repository.IRankTargetRepository
	metrics     repository.IContentMetricRepository
	beans       repository.IBeanRepository
	articles    repository.IArticleRepository
	saved       repository.ISavedItemRepository
	events      repository.IActionEventRepository
	suppression repository.IModalSuppressionRepository
}

// モーダル表示判断に必要なactor、表示理由、表示場所。
// RankTargetIDが指定された場合はその候補を検証する。未指定の場合はBackendで候補を選ぶ。
type ShowModalInput struct {
	Actor              Actor
	RankTargetID       uint64
	SourceRankTargetID uint64
	Trigger            entity.ModalTrigger
	PagePath           string
}

// モーダル表示結果。
// 表示ログだけでなく、Frontendが実際に表示できるようRankTargetと本文データも返す。
type ModalShowResult struct {
	model.ModalDisplayLog
	Target  *model.RankTarget `json:"target,omitempty"`
	Bean    *model.Bean       `json:"bean,omitempty"`
	Article *model.Article    `json:"article,omitempty"`
}

// モーダルクリック・クローズに必要なactorと表示ログID。
type ModalActionInput struct {
	Actor             Actor
	ModalDisplayLogID uint64
	PagePath          string
}

// モーダル制御に必要なRepositoryを受け取るコンストラクタ。
func NewModalUsecase(displays repository.IModalDisplayLogRepository, blocks repository.IModalBlockLogRepository, rankTargets repository.IRankTargetRepository, saved repository.ISavedItemRepository, events repository.IActionEventRepository, suppression repository.IModalSuppressionRepository) *ModalUsecase {
	return &ModalUsecase{
		displays:    displays,
		blocks:      blocks,
		rankTargets: rankTargets,
		saved:       saved,
		events:      events,
		suppression: suppression,
	}
}

// 推薦候補選定にランキング指標を使うModalUsecaseを生成する。
func NewModalUsecaseWithMetrics(displays repository.IModalDisplayLogRepository, blocks repository.IModalBlockLogRepository, rankTargets repository.IRankTargetRepository, metrics repository.IContentMetricRepository, beans repository.IBeanRepository, articles repository.IArticleRepository, saved repository.ISavedItemRepository, events repository.IActionEventRepository, suppression repository.IModalSuppressionRepository) *ModalUsecase {
	u := NewModalUsecase(displays, blocks, rankTargets, saved, events, suppression)
	u.metrics = metrics
	u.beans = beans
	u.articles = articles
	return u
}

// 表示条件を確認し、候補選定・表示ログ・event・Redis抑制を記録する。
func (u *ModalUsecase) Show(ctx context.Context, input ShowModalInput) (*ModalShowResult, error) {
	// UserまたはGuestSessionの片方だけであることを確認。
	if err := requireActor(input.Actor); err != nil {
		return nil, err
	}

	// 表示ページ、表示理由が不足している場合は不正入力。
	if input.PagePath == "" || !validModalTrigger(input.Trigger) {
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

	// 同一ページでの表示上限を確認。
	pageCount, err := u.displays.CountShownOnPage(ctx, input.Actor.UserID, input.Actor.GuestSessionID, input.PagePath, since)
	if err != nil {
		return nil, err
	}
	if pageCount >= int64(entity.MaxModalPerPage) {
		return nil, u.blockModal(ctx, input.Actor, nil, entity.ModalBlockPageLimitReached, input.PagePath, now)
	}

	// 同一セッションでの表示上限を確認。
	sessionCount, err := u.displays.CountShownInSession(ctx, input.Actor.UserID, input.Actor.GuestSessionID, since)
	if err != nil {
		return nil, err
	}
	if sessionCount >= int64(entity.MaxModalPerSession) {
		return nil, u.blockModal(ctx, input.Actor, nil, entity.ModalBlockSessionLimitReached, input.PagePath, now)
	}

	rankTargetID := input.RankTargetID
	if rankTargetID == 0 {
		rankTargetID, err = u.selectCandidate(ctx, input.Actor, input.SourceRankTargetID, key, input.PagePath)
		if err != nil {
			return nil, err
		}
	}

	// 互換用に明示候補が送られた場合も、Backend側で表示してよい候補か再検証する。
	if err := u.validateCandidate(ctx, input.Actor, rankTargetID, key); err != nil {
		return nil, u.blockModal(ctx, input.Actor, &rankTargetID, blockReasonForCandidateError(err), input.PagePath, now)
	}

	// モーダルを表示した記録をDBに保存するためのデータを作る。
	log := &model.ModalDisplayLog{
		UserID:         input.Actor.UserID,
		GuestSessionID: input.Actor.GuestSessionID,
		RankTargetID:   rankTargetID,
		Trigger:        input.Trigger,
		PagePath:       input.PagePath,
		ShownAt:        now,
	}
	if err := u.displays.Create(ctx, log); err != nil {
		return nil, err
	}

	// 同じ候補をクールダウン内に再表示しないよう、Redisへ一時記録を残す。
	// 失敗しても表示ログ作成済みのため、モーダル表示自体は成功扱いにする。
	_ = u.suppression.SetShown(ctx, key, rankTargetID, cooldown)

	// modal_impression eventも補助記録。
	// event作成に失敗しても、表示本体は成功扱いに。
	if u.events != nil {
		_ = u.events.Create(ctx, &model.ActionEvent{
			UserID:            input.Actor.UserID,
			GuestSessionID:    input.Actor.GuestSessionID,
			EventType:         entity.EventTypeModalImpression,
			RankTargetID:      &rankTargetID,
			Placement:         entity.PlacementModal,
			ModalDisplayLogID: &log.ID,
			PagePath:          input.PagePath,
			OccurredAt:        now,
		})
	}

	return u.modalShowResult(ctx, log)
}

func (u *ModalUsecase) modalShowResult(ctx context.Context, log *model.ModalDisplayLog) (*ModalShowResult, error) {
	if log == nil {
		return nil, entity.ErrModalDisplayLogNotFound
	}

	result := &ModalShowResult{ModalDisplayLog: *log}
	if u.rankTargets == nil {
		return result, nil
	}

	target, err := u.rankTargets.FindByID(ctx, log.RankTargetID)
	if err != nil {
		return result, nil
	}
	result.Target = target

	switch target.ContentType {
	case entity.ContentTypeBean:
		if u.beans != nil {
			bean, err := u.beans.FindByID(ctx, target.ContentID)
			if err == nil {
				result.Bean = bean
			}
		}
	case entity.ContentTypeArticle:
		if u.articles != nil {
			article, err := u.articles.FindByID(ctx, target.ContentID)
			if err == nil {
				result.Article = article
			}
		}
	}

	return result, nil
}

const modalCandidateFetchLimit = 50

// Backend側で表示候補を選ぶ。表示済み・閉じた候補・保存済み・現在閲覧中の候補は除外する。
func (u *ModalUsecase) selectCandidate(ctx context.Context, actor Actor, sourceRankTargetID uint64, actorKey string, pagePath string) (uint64, error) {
	ids, err := u.modalCandidateIDs(ctx)
	if err != nil {
		return 0, err
	}

	for _, id := range ids {
		if id == 0 || id == sourceRankTargetID {
			continue
		}
		if err := u.validateCandidate(ctx, actor, id, actorKey); err != nil {
			continue
		}
		return id, nil
	}

	_ = u.blocks.Create(ctx, modalBlock(actor, nil, entity.ModalBlockNoCandidate, pagePath, time.Now()))
	return 0, entity.ErrModalCandidateNotFound
}

// ランキング指標があればscore順、なければ有効RankTargetの新しい順を候補にする。
func (u *ModalUsecase) modalCandidateIDs(ctx context.Context) ([]uint64, error) {
	if u.metrics != nil {
		metrics, err := u.metrics.ListRanking(ctx, nil, modalCandidateFetchLimit, 0)
		if err != nil {
			return nil, err
		}
		ids := make([]uint64, 0, len(metrics))
		for _, metric := range metrics {
			if metric == nil || metric.RankTargetID == 0 {
				continue
			}
			ids = append(ids, metric.RankTargetID)
		}
		if len(ids) > 0 {
			return ids, nil
		}
	}

	beanTargets, err := u.rankTargets.ListActiveByType(ctx, entity.ContentTypeBean)
	if err != nil {
		return nil, err
	}
	articleTargets, err := u.rankTargets.ListActiveByType(ctx, entity.ContentTypeArticle)
	if err != nil {
		return nil, err
	}

	ids := make([]uint64, 0, len(beanTargets)+len(articleTargets))
	for _, target := range append(beanTargets, articleTargets...) {
		if target == nil || target.ID == 0 {
			continue
		}
		ids = append(ids, target.ID)
	}
	return ids, nil
}

// 候補が有効で、保存済み・表示済み・直近クローズ済みではないことを確認する。
func (u *ModalUsecase) validateCandidate(ctx context.Context, actor Actor, rankTargetID uint64, actorKey string) error {
	if err := ensureActiveRankTarget(ctx, u.rankTargets, rankTargetID); err != nil {
		return err
	}

	if actor.UserID != nil {
		saved, err := u.saved.ExistsActive(ctx, *actor.UserID, rankTargetID)
		if err != nil {
			return err
		}
		if saved {
			return errModalAlreadySaved
		}
	}

	shown, err := u.suppression.WasShown(ctx, actorKey, rankTargetID)
	if err != nil {
		return err
	}
	if shown {
		return errModalRecentlyShown
	}

	closed, err := u.suppression.WasClosed(ctx, actorKey, rankTargetID)
	if err != nil {
		return err
	}
	if closed {
		return errModalRecentlyClosed
	}

	return nil
}

var (
	errModalAlreadySaved   = errors.New("modal candidate already saved")
	errModalRecentlyShown  = errors.New("modal candidate recently shown")
	errModalRecentlyClosed = errors.New("modal candidate recently closed")
)

func blockReasonForCandidateError(err error) entity.ModalBlockReason {
	switch {
	case errors.Is(err, errModalAlreadySaved):
		return entity.ModalBlockAlreadySaved
	case errors.Is(err, errModalRecentlyShown):
		return entity.ModalBlockRecentlyShown
	case errors.Is(err, errModalRecentlyClosed):
		return entity.ModalBlockRecentlyClosed
	default:
		return entity.ModalBlockNoCandidate
	}
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
