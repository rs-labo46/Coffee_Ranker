package validator

import (
	"time"

	"coffee-ranker/entity"
)

type EventValidator struct{}

type EventRequest struct {
	EventType             entity.EventType    `json:"event_type"`
	RankTargetID          *uint64             `json:"rank_target_id,omitempty"`
	Placement             entity.Placement    `json:"placement"`
	DwellMs               *int64              `json:"dwell_ms,omitempty"`
	SearchConditionHash   *string             `json:"search_condition_hash,omitempty"`
	PreviousConditionHash *string             `json:"previous_condition_hash,omitempty"`
	SearchKeyword         *string             `json:"search_keyword,omitempty"`
	SearchOrigin          *string             `json:"search_origin,omitempty"`
	SearchRoastLevel      *entity.RoastLevel  `json:"search_roast_level,omitempty"`
	SearchAcidity         *int                `json:"search_acidity,omitempty"`
	SearchBitterness      *int                `json:"search_bitterness,omitempty"`
	SearchAroma           *int                `json:"search_aroma,omitempty"`
	SearchFlavor          *int                `json:"search_flavor,omitempty"`
	SearchBody            *int                `json:"search_body,omitempty"`
	SearchCategory        *string             `json:"search_category,omitempty"`
	PagePath              string              `json:"page_path"`
	ReferrerPath          *string             `json:"referrer_path,omitempty"`
	DedupKey              string              `json:"dedup_key,omitempty"`
	DedupTTLSeconds       int                 `json:"dedup_ttl_seconds,omitempty"`
	RatingScore           *entity.RatingScore `json:"rating_score,omitempty"`
}

type ValidEvent struct {
	EventRequest
	DedupTTL time.Duration
}

// NewEventValidatorを生成してDI層やRouterから使えるようにする。
func NewEventValidator() *EventValidator {
	return &EventValidator{}
}

// POST /eventsで直接受け付ける行動ログRequestを検証。
// event_type別に必須項目、placement、page_path、滞在時間、検索条件を確認。
func (v *EventValidator) Record(input EventRequest) (ValidEvent, error) {
	// page_pathは内部パスだけ許可し、外部URLやjavascript:を拒否。
	pagePath, err := ValidatePagePath(input.PagePath)
	if err != nil {
		return ValidEvent{}, err
	}
	input.PagePath = pagePath
	if input.RatingScore != nil {
		return ValidEvent{}, entity.ErrInvalidInput
	}
	if err := validateDirectEventType(input); err != nil {
		return ValidEvent{}, err
	}
	if err := ValidatePlacement(input.Placement, false); err != nil && input.EventType != entity.EventTypeReSearch {
		return ValidEvent{}, err
	}
	if input.EventType == entity.EventTypeReSearch {
		input.Placement = entity.PlacementSearchResult
	}
	if input.RankTargetID != nil && *input.RankTargetID == 0 {
		return ValidEvent{}, entity.ErrInvalidInput
	}
	if input.DwellMs != nil {
		min := int64(entity.MinStaySeconds) * 1000
		max := int64(entity.MaxStaySeconds) * 1000
		if *input.DwellMs < min || *input.DwellMs > max {
			return ValidEvent{}, entity.ErrInvalidInput
		}
	}
	if err := validateEventSearchFields(&input); err != nil {
		return ValidEvent{}, err
	}
	ttl := time.Duration(input.DedupTTLSeconds) * time.Second
	if ttl < 0 {
		return ValidEvent{}, entity.ErrInvalidInput
	}
	return ValidEvent{EventRequest: input, DedupTTL: ttl}, nil
}

// POST /eventsで直接記録してよいevent_typeだけを許可。
// save/rating/modal系は専用Usecase後に作るため、ここでは拒否。
func validateDirectEventType(input EventRequest) error {
	switch input.EventType {
	case entity.EventTypeContentView, entity.EventTypeImpression, entity.EventTypeClick:
		if input.RankTargetID == nil || *input.RankTargetID == 0 {
			return entity.ErrInvalidInput
		}
		return nil
	case entity.EventTypeStay:
		if input.RankTargetID == nil || *input.RankTargetID == 0 || input.DwellMs == nil {
			return entity.ErrInvalidInput
		}
		return nil
	case entity.EventTypeReSearch:
		if input.SearchConditionHash == nil || *input.SearchConditionHash == "" {
			return entity.ErrInvalidInput
		}
		return nil
	case entity.EventTypeSave, entity.EventTypeRating, entity.EventTypeModalImpression, entity.EventTypeModalClick, entity.EventTypeModalClose:
		return entity.ErrInvalidEventType
	default:
		return entity.ErrInvalidEventType
	}
}

// re_searchイベントに含まれる検索条件を検証。
// Bean/Article検索条件の形式だけを確認し、検索結果の有無は判断しない。
func validateEventSearchFields(input *EventRequest) error {
	var err error
	input.SearchConditionHash, err = ValidateHash(input.SearchConditionHash)
	if err != nil {
		return err
	}
	input.PreviousConditionHash, err = ValidateHash(input.PreviousConditionHash)
	if err != nil {
		return err
	}
	input.SearchKeyword, err = NormalizeOptionalText(input.SearchKeyword, 100)
	if err != nil {
		return err
	}
	input.SearchOrigin, err = NormalizeOptionalText(input.SearchOrigin, 50)
	if err != nil {
		return err
	}
	input.SearchCategory, err = ValidateCategory(input.SearchCategory)
	if err != nil {
		return err
	}
	for _, score := range []*int{input.SearchAcidity, input.SearchBitterness, input.SearchAroma, input.SearchFlavor, input.SearchBody} {
		// 味覚スコアは1〜5だけ許可。
		if err := ValidateTasteScore(score); err != nil {
			return err
		}
	}
	if input.SearchRoastLevel != nil {
		if err := ValidateRequiredRoastLevel(*input.SearchRoastLevel); err != nil {
			return err
		}
	}
	return nil
}
