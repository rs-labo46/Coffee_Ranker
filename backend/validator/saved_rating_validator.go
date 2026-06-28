package validator

import (
	"coffee-ranker/entity"
)

type SavedItemValidator struct{}
type RatingValidator struct{}

type SaveRequest struct {
	RankTargetID uint64           `json:"rank_target_id"`
	Placement    entity.Placement `json:"placement"`
	PagePath     string           `json:"page_path"`
}

type RatingRequest struct {
	RankTargetID uint64             `json:"rank_target_id"`
	Score        entity.RatingScore `json:"score"`
	Placement    entity.Placement   `json:"placement"`
	PagePath     string             `json:"page_path"`
}

// NewSavedItemValidatorを生成してDI層やRouterから使えるようにする。
func NewSavedItemValidator() *SavedItemValidator {
	return &SavedItemValidator{}
}

// NewRatingValidatorを生成してDI層やRouterから使えるようにする。
func NewRatingValidator() *RatingValidator {
	return &RatingValidator{}
}

// 保存Requestのrank_target_id、placement、page_pathを検証。
// Userが保存可能か、対象RankTargetが有効かはUsecaseで判断。
func (v *SavedItemValidator) Save(input SaveRequest) (SaveRequest, error) {
	// IDはDB存在確認ではなく、0でない正の値かだけ確認。
	if err := ValidateID(input.RankTargetID); err != nil {
		return SaveRequest{}, err
	}
	if err := ValidatePlacement(input.Placement, false); err != nil {
		return SaveRequest{}, err
	}
	// page_pathは内部パスだけ許可し、外部URLやjavascript:を拒否。
	pagePath, err := ValidatePagePath(input.PagePath)
	if err != nil {
		return SaveRequest{}, err
	}
	input.PagePath = pagePath
	return input, nil
}

// 保存解除や確認に使うrank_target_idが0でないかを検証。
// 保存済みかどうかはUsecaseで判断。
func (v *SavedItemValidator) RankTargetID(id uint64) error {
	return ValidateID(id)
}

// 保存一覧のページング条件を検証。
// 認証済みUserの保存データ取得はUsecaseで行う。
func (v *SavedItemValidator) List(input PageQuery) (PageQuery, error) {
	return NormalizePage(input, 20, 100, 10000)
}

// 評価Requestのrank_target_id、rating_score、placement、page_pathを検証。
// Guest不可、対象有効性、再評価処理はUsecaseで判断。
func (v *RatingValidator) Rate(input RatingRequest) (RatingRequest, error) {
	// IDはDB存在確認ではなく、0でない正の値かだけ確認。
	if err := ValidateID(input.RankTargetID); err != nil {
		return RatingRequest{}, err
	}
	// rating_scoreはGood(+1)またはBad(-1)だけ許可。
	if err := ValidateRatingScore(input.Score); err != nil {
		return RatingRequest{}, err
	}
	if err := ValidatePlacement(input.Placement, false); err != nil {
		return RatingRequest{}, err
	}
	// page_pathは内部パスだけ許可し、外部URLやjavascript:を拒否。
	pagePath, err := ValidatePagePath(input.PagePath)
	if err != nil {
		return RatingRequest{}, err
	}
	input.PagePath = pagePath
	return input, nil
}

// 評価取得・削除に使うrank_target_idが0でないかを検証。
// 評価が存在するかどうかはUsecaseで判断。
func (v *RatingValidator) RankTargetID(id uint64) error {
	return ValidateID(id)
}
