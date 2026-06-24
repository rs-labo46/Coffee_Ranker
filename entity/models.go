package entity

import "time"

// 保存や評価、管理操作を行えるアカウント
type User struct {
	ID           uint64
	Name         string
	Email        string
	PasswordHash string
	Role         UserRole
	Status       UserStatus
	TokenVersion int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// アクセストークン再発行とリフレッシュローテーションに使う保存状態。
type RefreshToken struct {
	ID                uint64
	UserID            uint64
	TokenHash         string
	FamilyID          string
	UsedAt            *time.Time
	ReplacedByTokenID *uint64
	RevokedAt         *time.Time
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

// 未ログインの行動を一時的に識別。
type GuestSession struct {
	ID             uint64
	SessionKeyHash string
	FirstSeenAt    time.Time
	LastSeenAt     time.Time
	ExpiresAt      time.Time
}

// ランキング、推薦、保存、評価の対象になるコーヒー豆。
// 公開状態によって、公開一覧、ランキング、モーダル候補へ出せるかを制御。
type Bean struct {
	ID          uint64
	Name        string
	Roaster     *string
	Origin      *string
	Region      *string
	Farm        *string
	Variety     *string
	RoastLevel  RoastLevel
	Acidity     *int
	Bitterness  *int
	Flavor      *int
	Aroma       *int
	Body        *int
	FlavorNote  *string
	Description *string
	ImageURL    *string
	IsPublished bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ランキング、推薦、保存、評価の対象になるコーヒー関連記事。
type Article struct {
	ID          uint64
	Title       string
	Slug        string
	Summary     string
	Body        *string
	Category    *string
	SourceName  *string
	SourceURL   *string
	ImageURL    *string
	IsPublished bool
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// コーヒー豆と記事の関連付け。
type BeanArticle struct {
	ID           uint64
	BeanID       uint64
	ArticleID    uint64
	DisplayOrder int
	CreatedAt    time.Time
}

// BeanやArticleをランキング対象として共通で扱うための情報。
type RankTarget struct {
	ID          uint64
	ContentType ContentType
	ContentID   uint64
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// 表示、クリック、保存、評価などのユーザー行動を記録する情報。
// UserIDかGuestSessionIDのどちらか一方だけを入れる。
type ActionEvent struct {
	ID                    uint64
	UserID                *uint64
	GuestSessionID        *uint64
	EventType             EventType
	RankTargetID          *uint64
	Placement             Placement
	DwellMs               *int64
	RatingScore           *RatingScore
	SearchConditionHash   *string
	PreviousConditionHash *string
	SearchKeyword         *string
	SearchOrigin          *string
	SearchRoastLevel      *RoastLevel
	SearchAcidity         *int
	SearchBitterness      *int
	SearchAroma           *int
	SearchFlavor          *int
	SearchBody            *int
	SearchCategory        *string
	ModalDisplayLogID     *uint64
	PagePath              string
	ReferrerPath          *string
	UserAgent             *string
	IPAddressHash         *string
	RequestID             *string
	OccurredAt            time.Time
}

// 推薦モーダルを表示した記録。
// クリックした時刻や閉じた時刻もここに残す。
type ModalDisplayLog struct {
	ID             uint64
	UserID         *uint64
	GuestSessionID *uint64
	RankTargetID   uint64
	Trigger        ModalTrigger
	PagePath       string
	ShownAt        time.Time
	ClickedAt      *time.Time
	ClosedAt       *time.Time
	CreatedAt      time.Time
}

// 推薦モーダルを表示しなかった記録。
// なぜ表示しなかったかをReasonに残す。
type ModalBlockLog struct {
	ID                    uint64
	UserID                *uint64
	GuestSessionID        *uint64
	CandidateRankTargetID *uint64
	Reason                ModalBlockReason
	PagePath              string
	BlockedAt             time.Time
}

// ログインユーザーがBeanやArticleを保存した状態。
// RemovedAtがnilなら保存中、値があれば保存解除済み。
type SavedItem struct {
	ID           uint64
	UserID       uint64
	RankTargetID uint64
	RemovedAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ログインユーザーがBeanやArticleに付けたGood/Bad評価。
// ScoreはGoodなら+1、Badなら-1を入れる。
type Rating struct {
	ID           uint64
	UserID       uint64
	RankTargetID uint64
	Score        RatingScore
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ランキング対象ごとの集計結果。
// 表示数、クリック数、保存数、評価数などをまとめる。
type ContentMetric struct {
	ID                   uint64
	RankTargetID         uint64
	Score                float64
	ImpressionCount      int64
	ContentViewCount     int64
	ClickCount           int64
	StayTotalMs          int64
	SaveCount            int64
	RatingCount          int64
	GoodCount            int64
	BadCount             int64
	RatingAvg            float64
	GoodRate             float64
	BadRate              float64
	ModalImpressionCount int64
	ModalClickCount      int64
	ModalCloseCount      int64
	ClickRate            float64
	SaveRate             float64
	ModalClickRate       float64
	ModalCloseRate       float64
	PeriodStart          time.Time
	PeriodEnd            time.Time
	CalculatedAt         time.Time
	UpdatedAt            time.Time
}

// ユーザーやゲストが何に興味を持っているかを表す情報。
// 産地、焙煎度、味の傾向などをスコア。
type InterestProfile struct {
	ID             uint64
	UserID         *uint64
	GuestSessionID *uint64
	Dimension      InterestDimension
	Value          string
	Score          float64
	LastEventAt    time.Time
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// バッチ処理の実行結果。
// 実行中、成功、失敗などの状態を記録。
type BatchRun struct {
	ID              uint64
	JobName         string
	Status          BatchStatus
	StartedAt       time.Time
	FinishedAt      *time.Time
	RowsProcessed   int64
	ErrorMessage    *string
	TriggeredBy     AuditActorType
	TriggeredUserID *uint64
}

// ログイン、管理画面操作、バッチ実行などの重要な操作履歴。
// 後から誰が何をしたか確認できるように残す。
type AuditLog struct {
	ID            uint64
	ActorType     AuditActorType
	ActorUserID   *uint64
	Action        AuditAction
	TargetType    *string
	TargetID      *uint64
	Detail        *string
	IPAddressHash *string
	RequestID     *string
	CreatedAt     time.Time
}
