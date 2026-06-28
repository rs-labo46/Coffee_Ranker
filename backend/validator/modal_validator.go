package validator

import "coffee-ranker/entity"

type ModalValidator struct{}

type ShowModalRequest struct {
	RankTargetID uint64              `json:"rank_target_id"`
	Trigger      entity.ModalTrigger `json:"trigger"`
	PagePath     string              `json:"page_path"`
}

type ModalActionRequest struct {
	ModalDisplayLogID uint64 `json:"modal_display_log_id"`
	PagePath          string `json:"page_path"`
}

// NewModalValidatorを生成してDI層やRouterから使う。
func NewModalValidator() *ModalValidator {
	return &ModalValidator{}
}

// モーダル表示Requestのrank_target_id、trigger、page_pathを検証。
// 表示可否、表示回数上限、保存済み除外はModalUsecaseで判断。
func (v *ModalValidator) Show(input ShowModalRequest) (ShowModalRequest, error) {
	// IDはDB存在確認ではなく、0でない正の値かだけ確認。
	if err := ValidateID(input.RankTargetID); err != nil {
		return ShowModalRequest{}, err
	}
	if err := validateModalTrigger(input.Trigger); err != nil {
		return ShowModalRequest{}, err
	}
	// page_pathは内部パスだけ許可し、外部URLやjavascript:を拒否。
	pagePath, err := ValidatePagePath(input.PagePath)
	if err != nil {
		return ShowModalRequest{}, err
	}
	input.PagePath = pagePath
	return input, nil
}

// モーダルクリック/クローズRequestのmodal_display_log_idとpage_pathを検証。
// 本人の表示ログかどうかはactor条件付き更新でUsecase/Repositoryが確認。
func (v *ModalValidator) Action(input ModalActionRequest) (ModalActionRequest, error) {
	// IDはDB存在確認ではなく、0でない正の値かだけ確認。
	if err := ValidateID(input.ModalDisplayLogID); err != nil {
		return ModalActionRequest{}, err
	}
	// page_pathは内部パスだけ許可し、外部URLやjavascript:を拒否。
	pagePath, err := ValidatePagePath(input.PagePath)
	if err != nil {
		return ModalActionRequest{}, err
	}
	input.PagePath = pagePath
	return input, nil
}

// モーダル表示のきっかけtriggerが許可enumかを検証。
// triggerの妥当性だけを見て、候補選定自体はUsecaseに任せる。
func validateModalTrigger(trigger entity.ModalTrigger) error {
	switch trigger {
	case entity.ModalTriggerFirstVisit, entity.ModalTriggerScrollEnd, entity.ModalTriggerBeanStay, entity.ModalTriggerArticleStay, entity.ModalTriggerSameOriginViewed, entity.ModalTriggerSameRoastClicked, entity.ModalTriggerSavedContent, entity.ModalTriggerGoodRating, entity.ModalTriggerReSearch:
		return nil
	default:
		return entity.ErrInvalidInput
	}
}
