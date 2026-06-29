package usecase

import (
	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
	"context"
	"time"
)

// Userの保存、保存解除、保存一覧、保存済み確認。
type SavedItemUsecase struct {
	saved       repository.ISavedItemRepository
	rankTargets repository.IRankTargetRepository
	events      repository.IActionEventRepository
}

// UserのGood/Bad評価、再評価、評価削除、評価取得。
type RatingUsecase struct {
	ratings     repository.IRatingRepository
	rankTargets repository.IRankTargetRepository
	events      repository.IActionEventRepository
}

// 保存処理に必要なコンストラクタ。
func NewSavedItemUsecase(saved repository.ISavedItemRepository, rankTargets repository.IRankTargetRepository, events repository.IActionEventRepository) *SavedItemUsecase {
	return &SavedItemUsecase{saved: saved, rankTargets: rankTargets, events: events}
}

// 評価処理に必要なコンストラクタ。
func NewRatingUsecase(ratings repository.IRatingRepository, rankTargets repository.IRankTargetRepository, events repository.IActionEventRepository) *RatingUsecase {
	return &RatingUsecase{ratings: ratings, rankTargets: rankTargets, events: events}
}

// RankTargetが有効か確認し、保存または再保存。
// 保存本体が成功した後、save eventをbest effort（log失敗しても処理は成功）で記録。
func (u *SavedItemUsecase) Save(ctx context.Context, userID uint64, rankTargetID uint64, placement entity.Placement, pagePath string) (*model.SavedItem, error) {
	if pagePath == "" {
		return nil, entity.ErrInvalidInput
	}
	// 保存はログインUserのみ許可。
	// Guestは保存できない。
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	// 保存対象が有効なランキング対象か確認。
	// 存在しない、または無効なRankTargetは保存させない。
	if err := ensureActiveRankTarget(ctx, u.rankTargets, rankTargetID); err != nil {
		return nil, err
	}

	now := time.Now()

	// 未保存なら新規保存、過去に解除済みなら再保存。
	item, err := u.saved.SaveOrRestore(ctx, userID, rankTargetID, now)
	if err != nil {
		return nil, err
	}

	// save eventは補助記録。
	// event作成に失敗しても、保存本体は成功扱いに。
	if u.events != nil {
		_ = u.events.Create(ctx, &model.ActionEvent{
			UserID:       &userID,
			EventType:    entity.EventTypeSave,
			RankTargetID: &rankTargetID,
			Placement:    placement,
			PagePath:     pagePath,
			OccurredAt:   now,
		})
	}

	return item, nil
}

// 指定Userの保存を解除。
func (u *SavedItemUsecase) Remove(ctx context.Context, userID uint64, rankTargetID uint64) error {
	// 保存解除はログインUserのみ許可。
	if err := requireUserID(userID); err != nil {
		return err
	}

	// rankTargetIDが0なら対象を特定できないため不正入力。
	if rankTargetID == 0 {
		return entity.ErrInvalidInput
	}

	// 保存解除は、対象が現在非公開でも実行できてよい。
	return u.saved.Remove(ctx, userID, rankTargetID, time.Now())
}

// 指定Userの有効な保存一覧をページング付きで取得。
func (u *SavedItemUsecase) List(ctx context.Context, userID uint64, page Page) ([]*model.SavedItem, error) {
	// 保存一覧はログインUserのみ取得できる。
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	// limit/offsetを安全な範囲に補正・検証。
	page, err := normalizePage(page, 20, 100, 10000)
	if err != nil {
		return nil, err
	}

	// 削除済みではない保存だけを取得。
	return u.saved.ListActiveByUserID(ctx, userID, page.Limit, page.Offset)
}

// 指定UserがRankTargetを保存済みか確認。
func (u *SavedItemUsecase) Exists(ctx context.Context, userID uint64, rankTargetID uint64) (bool, error) {
	// 保存済み確認はログインUserのみ許可。
	if err := requireUserID(userID); err != nil {
		return false, err
	}

	// rankTargetIDが0なら対象を特定できないため不正入力。
	if rankTargetID == 0 {
		return false, entity.ErrInvalidInput
	}

	return u.saved.ExistsActive(ctx, userID, rankTargetID)
}

// RankTargetが有効か確認し、Good/Bad評価を保存。
// 評価本体が成功した後、rating eventをbest effortで記録。
func (u *RatingUsecase) Rate(ctx context.Context, userID uint64, rankTargetID uint64, score entity.RatingScore, placement entity.Placement, pagePath string) (*model.Rating, error) {
	if pagePath == "" {
		return nil, entity.ErrInvalidInput
	}
	// 評価はログインUserのみ許可。
	// Guestは評価できない。
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	// 評価値はGood(+1)またはBad(-1)のみ許可。
	if score != entity.RatingScoreGood && score != entity.RatingScoreBad {
		return nil, entity.ErrInvalidRatingScore
	}

	// 評価対象が有効なランキング対象か確認。
	// 存在しない、または無効なRankTargetは評価させない。
	if err := ensureActiveRankTarget(ctx, u.rankTargets, rankTargetID); err != nil {
		return nil, err
	}

	now := time.Now()

	// 未評価なら作成、評価済みなら更新。
	rating, err := u.ratings.Upsert(ctx, userID, rankTargetID, score, now)
	if err != nil {
		return nil, err
	}

	// rating eventは補助記録。
	// event作成に失敗しても、評価本体は成功扱いに。
	if u.events != nil {
		_ = u.events.Create(ctx, &model.ActionEvent{
			UserID:       &userID,
			EventType:    entity.EventTypeRating,
			RankTargetID: &rankTargetID,
			Placement:    placement,
			RatingScore:  &score,
			PagePath:     pagePath,
			OccurredAt:   now,
		})
	}

	return rating, nil
}

// 指定Userの評価を削除。
func (u *RatingUsecase) Delete(ctx context.Context, userID uint64, rankTargetID uint64) error {
	// 評価削除はログインUserのみ許可。
	if err := requireUserID(userID); err != nil {
		return err
	}

	// rankTargetIDが0なら対象を特定できないため不正入力。
	if rankTargetID == 0 {
		return entity.ErrInvalidInput
	}

	// 評価削除は、対象が現在非公開でも実行できる。
	return u.ratings.Delete(ctx, userID, rankTargetID)
}

// 指定Userの評価を取得。
func (u *RatingUsecase) Get(ctx context.Context, userID uint64, rankTargetID uint64) (*model.Rating, error) {
	// 評価取得はログインUserのみ許可。
	if err := requireUserID(userID); err != nil {
		return nil, err
	}

	// rankTargetIDが0なら対象を特定できないため不正入力。
	if rankTargetID == 0 {
		return nil, entity.ErrInvalidInput
	}

	return u.ratings.FindByUserAndTarget(ctx, userID, rankTargetID)
}

// RankTarget IDが有効なランキング対象として存在することを確認。
func ensureActiveRankTarget(ctx context.Context, repo repository.IRankTargetRepository, rankTargetID uint64) error {
	// rankTargetIDが0なら対象を特定できないため不正入力。
	if rankTargetID == 0 {
		return entity.ErrInvalidInput
	}

	// 保存・評価対象が有効なRankTargetか確認。
	exists, err := repo.ExistsActiveByID(ctx, rankTargetID)
	if err != nil {
		return err
	}

	// 存在しない、または無効化されたRankTargetは対象にしない。
	if !exists {
		return entity.ErrRankTargetNotFound
	}

	return nil
}
