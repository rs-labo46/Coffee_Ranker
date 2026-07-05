package validator

import "coffee-ranker/entity"

type ModalValidator struct{}

type ShowModalRequest struct {
	RankTargetID       uint64              `json:"rank_target_id,omitempty"`
	SourceRankTargetID uint64              `json:"source_rank_target_id,omitempty"`
	Trigger            entity.ModalTrigger `json:"trigger"`
	PagePath           string              `json:"page_path"`
}

type ModalActionRequest struct {
	ModalDisplayLogID uint64 `json:"modal_display_log_id"`
	PagePath          string `json:"page_path"`
}

// NewModalValidatorを生成してDI層やRouterから使う。
func NewModalValidator() *ModalValidator {
	return &ModalValidator{}
}

// モーダル表示Requestの候補ID、source ID、trigger、page_pathを検証。
// rank_target_id未指定時の候補選定、表示可否、保存済み除外はModalUsecaseで判断。
func (v *ModalValidator) Show(input ShowModalRequest) (ShowModalRequest, error) {
	// rank_target_idは互換用の明示候補。未指定ならBackend側で候補を選ぶ。
	if input.RankTargetID != 0 {
		if err := ValidateID(input.RankTargetID); err != nil {
			return ShowModalRequest{}, err
		}
	}
	// source_rank_target_idは現在見ている対象。候補から除外するために使う。
	if input.SourceRankTargetID != 0 {
		if err := ValidateID(input.SourceRankTargetID); err != nil {
			return ShowModalRequest{}, err
		}
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
