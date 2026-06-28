package usecase

import (
	"context"
	"strings"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// UserまたはGuestSessionのどちらの行動か。
type Actor struct {
	UserID         *uint64
	GuestSessionID *uint64
}

// フロントから送られる行動イベントを検証して記録する。
type EventUsecase struct {
	events      repository.ActionEventRepository
	rankTargets repository.RankTargetRepository
	dedup       repository.EventDedupRepository
}

// 行動イベント記録に必要なactor、event種別、対象、検索条件をまとめる。
type RecordEventInput struct {
	Actor                 Actor
	EventType             entity.EventType
	RankTargetID          *uint64
	Placement             entity.Placement
	DwellMs               *int64
	SearchConditionHash   *string
	PreviousConditionHash *string
	SearchKeyword         *string
	SearchOrigin          *string
	SearchRoastLevel      *entity.RoastLevel
	SearchAcidity         *int
	SearchBitterness      *int
	SearchAroma           *int
	SearchFlavor          *int
	SearchBody            *int
	SearchCategory        *string
	PagePath              string
	ReferrerPath          *string
	UserAgent             *string
	IPAddressHash         *string
	RequestID             *string
	DedupKey              string
	DedupTTL              time.Duration
}

// 行動イベント記録に必要なRepositoryと重複防止のRepositoryを受け取るコンストラクタ。
func NewEventUsecase(events repository.ActionEventRepository, rankTargets repository.RankTargetRepository, dedup repository.EventDedupRepository) *EventUsecase {
	return &EventUsecase{events: events, rankTargets: rankTargets, dedup: dedup}
}

// クライアントからの行動イベントを検証し、必要なら重複排除して保存。
func (u *EventUsecase) Record(ctx context.Context, input RecordEventInput) (*model.ActionEvent, error) {
	//UserかGuestSessionの片方だけ入っているか確認
	if err := requireActor(input.Actor); err != nil {
		return nil, err //ここを通らないイベントは保存できない
	}

	//POST /eventsで直接受けてよいイベントか確認
	if err := validateDirectEvent(input); err != nil {
		return nil, err
	}

	// RankTargetIDがある場合、その対象が有効なランキング対象か確認
	if input.RankTargetID != nil {
		exists, err := u.rankTargets.ExistsActiveByID(ctx, *input.RankTargetID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, entity.ErrRankTargetNotFound
		}
	}

	//重複防止処理
	if shouldDedup(input.EventType) && input.DedupKey != "" {
		ttl := input.DedupTTL
		if ttl <= 0 {
			ttl = time.Minute
		}
		ok, err := u.dedup.SetIfNotExists(ctx, input.DedupKey, ttl)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil //重複イベントなのでエラーにせず、保存もしないで終了
		}
	}
	//前回の検索条件
	previousHash := input.PreviousConditionHash
	if input.EventType == entity.EventTypeReSearch && previousHash == nil {
		//DBから直近の検索条件hashを取る
		last, err := u.events.FindLastSearchHash(ctx, input.Actor.UserID, input.Actor.GuestSessionID)
		if err != nil {
			return nil, err
		}
		previousHash = last
	}
	event := &model.ActionEvent{
		UserID:                input.Actor.UserID,
		GuestSessionID:        input.Actor.GuestSessionID,
		EventType:             input.EventType,
		RankTargetID:          input.RankTargetID,
		Placement:             input.Placement,
		DwellMs:               input.DwellMs,
		SearchConditionHash:   input.SearchConditionHash, //今回の検索条件
		PreviousConditionHash: previousHash,
		SearchKeyword:         normalizeOptionalText(input.SearchKeyword), //前後空白や空文字を整理する
		SearchOrigin:          normalizeOptionalText(input.SearchOrigin),
		SearchRoastLevel:      input.SearchRoastLevel,
		SearchAcidity:         input.SearchAcidity,
		SearchBitterness:      input.SearchBitterness,
		SearchAroma:           input.SearchAroma,
		SearchFlavor:          input.SearchFlavor,
		SearchBody:            input.SearchBody,
		SearchCategory:        normalizeOptionalText(input.SearchCategory), //集計しやすく
		PagePath:              input.PagePath,
		ReferrerPath:          input.ReferrerPath,
		UserAgent:             input.UserAgent,
		IPAddressHash:         input.IPAddressHash,
		RequestID:             input.RequestID,
		OccurredAt:            time.Now(),
	}

	//DBに保存
	if err := u.events.Create(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

// UserIDとGuestSessionIDの片方だけが入っているか確認。
func validActor(actor Actor) bool {
	if actor.UserID != nil && actor.GuestSessionID == nil {
		return *actor.UserID > 0
	}
	if actor.UserID == nil && actor.GuestSessionID != nil {
		return *actor.GuestSessionID > 0
	}
	return false
}

// actorがUserまたはGuestの片方だけであること。
func requireActor(actor Actor) error {
	//validActor がfalseなら、入力不正として止める
	if !validActor(actor) {
		return entity.ErrInvalidInput
	}
	return nil
}

// POST /eventsで直接記録してよいevent_typeと必須項目を確認。
func validateDirectEvent(input RecordEventInput) error {
	switch input.EventType {
	case entity.EventTypeContentView, entity.EventTypeImpression, entity.EventTypeClick:
		//どのBeanやArticleでどのページで起こったのか
		if input.RankTargetID == nil || *input.RankTargetID == 0 || input.PagePath == "" {
			return entity.ErrInvalidInput
		}
		return nil
	case entity.EventTypeStay:
		//滞在時間がないのはNG。
		if input.RankTargetID == nil || *input.RankTargetID == 0 || input.DwellMs == nil || input.PagePath == "" {
			return entity.ErrInvalidInput
		}
		//短すぎるものや滞在時間が以上なものは記録しない。
		if *input.DwellMs < int64(entity.MinStaySeconds)*1000 || *input.DwellMs > int64(entity.MaxStaySeconds)*1000 {
			return entity.ErrInvalidInput
		}
		return nil
		//検索が変わったことを記録する。
	case entity.EventTypeReSearch:
		//RankTargetIDは不要で,検索は特定のBean/Articleに対する行動ではなく、検索条件に対する行動
		if input.SearchConditionHash == nil || *input.SearchConditionHash == "" || input.PagePath == "" {
			return entity.ErrInvalidInput
		}
		return nil
	default:
		return entity.ErrInvalidEventType
	}
}

// 重複防止が必要なevent_typeかどうかを判定。
func shouldDedup(eventType entity.EventType) bool {
	return eventType == entity.EventTypeImpression || eventType == entity.EventTypeContentView
}

// 任意入力の文字列をtrimし、空文字ならnil。
func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}

	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}

	return &text
}
