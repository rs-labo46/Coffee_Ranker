package model

import (
	"time"

	"coffee-ranker/entity"
)

// usersテーブルの永続化モデル。
// 生Passwordは持たず、PasswordHashだけを保存する。
type User struct {
	ID           uint64            `json:"id" gorm:"primaryKey;autoIncrement"`
	Name         string            `json:"name" gorm:"not null"`
	Email        string            `json:"email" gorm:"not null;uniqueIndex"`
	PasswordHash string            `json:"-" gorm:"not null"`
	Role         entity.UserRole   `json:"role" gorm:"not null;index"`
	Status       entity.UserStatus `json:"status" gorm:"not null;index"`
	TokenVersion int               `json:"-" gorm:"not null;default:0"`
	CreatedAt    time.Time         `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time         `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

func (User) TableName() string {
	return "users"
}

// refresh_tokensテーブルの永続化モデル。
// 生RefreshTokenは保存せず、TokenHashだけを保存する。
type RefreshToken struct {
	ID                uint64        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID            uint64        `json:"user_id" gorm:"not null;index"`
	TokenHash         string        `json:"-" gorm:"not null;uniqueIndex"`
	FamilyID          string        `json:"-" gorm:"not null;index"`
	UsedAt            *time.Time    `json:"used_at,omitempty" gorm:"index"`
	ReplacedByTokenID *uint64       `json:"replaced_by_token_id,omitempty" gorm:"index"`
	RevokedAt         *time.Time    `json:"revoked_at,omitempty" gorm:"index"`
	ExpiresAt         time.Time     `json:"expires_at" gorm:"not null;index"`
	CreatedAt         time.Time     `json:"created_at" gorm:"not null;autoCreateTime"`
	User              User          `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	ReplacedByToken   *RefreshToken `json:"-" gorm:"foreignKey:ReplacedByTokenID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// guest_sessionsテーブルの永続化モデル。
// 生GuestSessionキーは保存せず、SessionKeyHashだけを保存する。
type GuestSession struct {
	ID             uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionKeyHash string    `json:"-" gorm:"not null;uniqueIndex"`
	FirstSeenAt    time.Time `json:"first_seen_at" gorm:"not null"`
	LastSeenAt     time.Time `json:"last_seen_at" gorm:"not null;index"`
	ExpiresAt      time.Time `json:"expires_at" gorm:"not null;index"`
}

func (GuestSession) TableName() string {
	return "guest_sessions"
}

// beansテーブルの永続化モデル。
type Bean struct {
	ID          uint64            `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string            `json:"name" gorm:"not null;index"`
	Roaster     *string           `json:"roaster,omitempty" gorm:"index"`
	Origin      *string           `json:"origin,omitempty" gorm:"index"`
	Region      *string           `json:"region,omitempty"`
	Farm        *string           `json:"farm,omitempty"`
	Variety     *string           `json:"variety,omitempty"`
	RoastLevel  entity.RoastLevel `json:"roast_level" gorm:"not null;index"`
	Acidity     *int              `json:"acidity,omitempty"`
	Bitterness  *int              `json:"bitterness,omitempty"`
	Flavor      *int              `json:"flavor,omitempty"`
	Aroma       *int              `json:"aroma,omitempty"`
	Body        *int              `json:"body,omitempty"`
	FlavorNote  *string           `json:"flavor_note,omitempty"`
	Description *string           `json:"description,omitempty"`
	ImageURL    *string           `json:"image_url,omitempty"`
	IsPublished bool              `json:"is_published" gorm:"not null;default:false;index"`
	CreatedAt   time.Time         `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

func (Bean) TableName() string {
	return "beans"
}

// articlesテーブルの永続化モデル。
type Article struct {
	ID          uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Title       string     `json:"title" gorm:"not null;index"`
	Slug        string     `json:"slug" gorm:"not null;uniqueIndex"`
	Summary     string     `json:"summary" gorm:"not null"`
	Body        *string    `json:"body,omitempty"`
	Category    *string    `json:"category,omitempty" gorm:"index"`
	SourceName  *string    `json:"source_name,omitempty"`
	SourceURL   *string    `json:"source_url,omitempty"`
	ImageURL    *string    `json:"image_url,omitempty"`
	IsPublished bool       `json:"is_published" gorm:"not null;default:false;index"`
	PublishedAt *time.Time `json:"published_at,omitempty" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

func (Article) TableName() string {
	return "articles"
}

// bean_articlesテーブルの永続化モデル。
type BeanArticle struct {
	ID           uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	BeanID       uint64    `json:"bean_id" gorm:"not null;index;uniqueIndex:uq_bean_articles_bean_article"`
	ArticleID    uint64    `json:"article_id" gorm:"not null;index;uniqueIndex:uq_bean_articles_bean_article"`
	DisplayOrder int       `json:"display_order" gorm:"not null;default:0;index"`
	CreatedAt    time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	Bean         Bean      `json:"-" gorm:"foreignKey:BeanID;references:ID;constraint:OnUpdate:CASCADE"`
	Article      Article   `json:"-" gorm:"foreignKey:ArticleID;references:ID;constraint:OnUpdate:CASCADE"`
}

func (BeanArticle) TableName() string {
	return "bean_articles"
}

// rank_targetsテーブルの永続化モデル。
type RankTarget struct {
	ID          uint64             `json:"id" gorm:"primaryKey;autoIncrement"`
	ContentType entity.ContentType `json:"content_type" gorm:"not null;index;uniqueIndex:uq_rank_targets_content"`
	ContentID   uint64             `json:"content_id" gorm:"not null;uniqueIndex:uq_rank_targets_content"`
	IsActive    bool               `json:"is_active" gorm:"not null;default:true;index"`
	CreatedAt   time.Time          `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time          `json:"updated_at" gorm:"not null;autoUpdateTime"`
}

func (RankTarget) TableName() string {
	return "rank_targets"
}

// action_eventsテーブルの永続化モデル。
// UserIDかGuestSessionIDのどちらか一方だけを入れる。
type ActionEvent struct {
	ID                    uint64              `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                *uint64             `json:"user_id,omitempty" gorm:"index"`
	GuestSessionID        *uint64             `json:"guest_session_id,omitempty" gorm:"index"`
	EventType             entity.EventType    `json:"event_type" gorm:"not null;index"`
	RankTargetID          *uint64             `json:"rank_target_id,omitempty" gorm:"index"`
	Placement             entity.Placement    `json:"placement" gorm:"not null;index"`
	DwellMs               *int64              `json:"dwell_ms,omitempty"`
	RatingScore           *entity.RatingScore `json:"rating_score,omitempty"`
	SearchConditionHash   *string             `json:"search_condition_hash,omitempty" gorm:"index"`
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
	ModalDisplayLogID     *uint64             `json:"modal_display_log_id,omitempty" gorm:"index"`
	PagePath              string              `json:"page_path" gorm:"not null"`
	ReferrerPath          *string             `json:"referrer_path,omitempty"`
	UserAgent             *string             `json:"-"`
	IPAddressHash         *string             `json:"-"`
	RequestID             *string             `json:"request_id,omitempty" gorm:"index"`
	OccurredAt            time.Time           `json:"occurred_at" gorm:"not null;index"`
	User                  *User               `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	GuestSession          *GuestSession       `json:"-" gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	RankTarget            *RankTarget         `json:"-" gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	ModalDisplayLog       *ModalDisplayLog    `json:"-" gorm:"foreignKey:ModalDisplayLogID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (ActionEvent) TableName() string {
	return "action_events"
}

// modal_display_logsテーブルの永続化モデル。
// UserIDかGuestSessionIDのどちらか一方だけを入れる。
type ModalDisplayLog struct {
	ID             uint64              `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         *uint64             `json:"user_id,omitempty" gorm:"index"`
	GuestSessionID *uint64             `json:"guest_session_id,omitempty" gorm:"index"`
	RankTargetID   uint64              `json:"rank_target_id" gorm:"not null;index"`
	Trigger        entity.ModalTrigger `json:"trigger" gorm:"not null;index"`
	PagePath       string              `json:"page_path" gorm:"not null"`
	ShownAt        time.Time           `json:"shown_at" gorm:"not null;index"`
	ClickedAt      *time.Time          `json:"clicked_at,omitempty"`
	ClosedAt       *time.Time          `json:"closed_at,omitempty"`
	CreatedAt      time.Time           `json:"created_at" gorm:"not null;autoCreateTime"`
	User           *User               `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	GuestSession   *GuestSession       `json:"-" gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	RankTarget     RankTarget          `json:"-" gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

func (ModalDisplayLog) TableName() string {
	return "modal_display_logs"
}

// modal_block_logsテーブルの永続化モデル。
// UserIDかGuestSessionIDのどちらか一方だけを入れる。
type ModalBlockLog struct {
	ID                    uint64                  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID                *uint64                 `json:"user_id,omitempty" gorm:"index"`
	GuestSessionID        *uint64                 `json:"guest_session_id,omitempty" gorm:"index"`
	CandidateRankTargetID *uint64                 `json:"candidate_rank_target_id,omitempty" gorm:"index"`
	Reason                entity.ModalBlockReason `json:"reason" gorm:"not null;index"`
	PagePath              string                  `json:"page_path" gorm:"not null"`
	BlockedAt             time.Time               `json:"blocked_at" gorm:"not null;index"`
	User                  *User                   `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	GuestSession          *GuestSession           `json:"-" gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CandidateRankTarget   *RankTarget             `json:"-" gorm:"foreignKey:CandidateRankTargetID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (ModalBlockLog) TableName() string {
	return "modal_block_logs"
}

// saved_itemsテーブルの永続化モデル。
type SavedItem struct {
	ID           uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint64     `json:"user_id" gorm:"not null;index;uniqueIndex:uq_saved_items_user_target"`
	RankTargetID uint64     `json:"rank_target_id" gorm:"not null;index;uniqueIndex:uq_saved_items_user_target"`
	RemovedAt    *time.Time `json:"removed_at,omitempty" gorm:"index"`
	CreatedAt    time.Time  `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime"`
	User         User       `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	RankTarget   RankTarget `json:"-" gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

func (SavedItem) TableName() string {
	return "saved_items"
}

// ratingsテーブルの永続化モデル。
type Rating struct {
	ID           uint64             `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint64             `json:"user_id" gorm:"not null;index;uniqueIndex:uq_ratings_user_target"`
	RankTargetID uint64             `json:"rank_target_id" gorm:"not null;index;uniqueIndex:uq_ratings_user_target"`
	Score        entity.RatingScore `json:"score" gorm:"not null;index"`
	CreatedAt    time.Time          `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time          `json:"updated_at" gorm:"not null;autoUpdateTime"`
	User         User               `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	RankTarget   RankTarget         `json:"-" gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

func (Rating) TableName() string {
	return "ratings"
}

// content_metricsテーブルの永続化モデル。
type ContentMetric struct {
	ID                   uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	RankTargetID         uint64     `json:"rank_target_id" gorm:"not null;uniqueIndex"`
	Score                float64    `json:"score" gorm:"not null;default:0;index"`
	ImpressionCount      int64      `json:"impression_count" gorm:"not null;default:0"`
	ContentViewCount     int64      `json:"content_view_count" gorm:"not null;default:0"`
	ClickCount           int64      `json:"click_count" gorm:"not null;default:0"`
	StayTotalMs          int64      `json:"stay_total_ms" gorm:"not null;default:0"`
	SaveCount            int64      `json:"save_count" gorm:"not null;default:0"`
	RatingCount          int64      `json:"rating_count" gorm:"not null;default:0"`
	GoodCount            int64      `json:"good_count" gorm:"not null;default:0"`
	BadCount             int64      `json:"bad_count" gorm:"not null;default:0"`
	ReSearchCount        int64      `json:"re_search_count" gorm:"not null;default:0"`
	RatingAvg            float64    `json:"rating_avg" gorm:"not null;default:0"`
	GoodRate             float64    `json:"good_rate" gorm:"not null;default:0"`
	BadRate              float64    `json:"bad_rate" gorm:"not null;default:0"`
	ModalImpressionCount int64      `json:"modal_impression_count" gorm:"not null;default:0"`
	ModalClickCount      int64      `json:"modal_click_count" gorm:"not null;default:0"`
	ModalCloseCount      int64      `json:"modal_close_count" gorm:"not null;default:0"`
	ClickRate            float64    `json:"click_rate" gorm:"not null;default:0"`
	SaveRate             float64    `json:"save_rate" gorm:"not null;default:0"`
	ReSearchRate         float64    `json:"re_search_rate" gorm:"not null;default:0"`
	ModalClickRate       float64    `json:"modal_click_rate" gorm:"not null;default:0"`
	ModalCloseRate       float64    `json:"modal_close_rate" gorm:"not null;default:0"`
	PeriodStart          time.Time  `json:"period_start" gorm:"not null;index"`
	PeriodEnd            time.Time  `json:"period_end" gorm:"not null;index"`
	CalculatedAt         time.Time  `json:"calculated_at" gorm:"not null;index"`
	UpdatedAt            time.Time  `json:"updated_at" gorm:"not null;autoUpdateTime"`
	RankTarget           RankTarget `json:"-" gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

func (ContentMetric) TableName() string {
	return "content_metrics"
}

// interest_profilesテーブルの永続化モデル。
// UserIDかGuestSessionIDのどちらか一方だけを入れる。
type InterestProfile struct {
	ID             uint64                   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID         *uint64                  `json:"user_id,omitempty" gorm:"index"`
	GuestSessionID *uint64                  `json:"guest_session_id,omitempty" gorm:"index"`
	Dimension      entity.InterestDimension `json:"dimension" gorm:"not null;index"`
	Value          string                   `json:"value" gorm:"not null;index"`
	Score          float64                  `json:"score" gorm:"not null;default:0;index"`
	LastEventAt    time.Time                `json:"last_event_at" gorm:"not null;index"`
	ExpiresAt      *time.Time               `json:"expires_at,omitempty" gorm:"index"`
	CreatedAt      time.Time                `json:"created_at" gorm:"not null;autoCreateTime"`
	UpdatedAt      time.Time                `json:"updated_at" gorm:"not null;autoUpdateTime"`
	User           *User                    `json:"-" gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	GuestSession   *GuestSession            `json:"-" gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (InterestProfile) TableName() string {
	return "interest_profiles"
}

// batch_runsテーブルの永続化モデル。
type BatchRun struct {
	ID              uint64                `json:"id" gorm:"primaryKey;autoIncrement"`
	JobName         string                `json:"job_name" gorm:"not null;index"`
	Status          entity.BatchStatus    `json:"status" gorm:"not null;index"`
	StartedAt       time.Time             `json:"started_at" gorm:"not null;index"`
	FinishedAt      *time.Time            `json:"finished_at,omitempty"`
	RowsProcessed   int64                 `json:"rows_processed" gorm:"not null;default:0"`
	ErrorMessage    *string               `json:"error_message,omitempty"`
	TriggeredBy     entity.AuditActorType `json:"triggered_by" gorm:"not null;index"`
	TriggeredUserID *uint64               `json:"triggered_user_id,omitempty" gorm:"index"`
	TriggeredUser   *User                 `json:"-" gorm:"foreignKey:TriggeredUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (BatchRun) TableName() string {
	return "batch_runs"
}

// audit_logsテーブルの永続化モデル。
type AuditLog struct {
	ID            uint64                `json:"id" gorm:"primaryKey;autoIncrement"`
	ActorType     entity.AuditActorType `json:"actor_type" gorm:"not null;index"`
	ActorUserID   *uint64               `json:"actor_user_id,omitempty" gorm:"index"`
	Action        entity.AuditAction    `json:"action" gorm:"not null;index"`
	TargetType    *string               `json:"target_type,omitempty" gorm:"index"`
	TargetID      *uint64               `json:"target_id,omitempty" gorm:"index"`
	Detail        *string               `json:"detail,omitempty"`
	IPAddressHash *string               `json:"-"`
	RequestID     *string               `json:"request_id,omitempty" gorm:"index"`
	CreatedAt     time.Time             `json:"created_at" gorm:"not null;autoCreateTime;index"`
	ActorUser     *User                 `json:"-" gorm:"foreignKey:ActorUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
