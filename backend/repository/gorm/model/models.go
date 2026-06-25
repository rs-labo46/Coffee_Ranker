package model

import "time"

// 認証済みのユーザーと管理者をusersテーブルへ保存するためのDB
type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	Name         string    `gorm:"not null"`
	Email        string    `gorm:"not null;uniqueIndex"`
	PasswordHash string    `gorm:"not null"`
	Role         string    `gorm:"not null;index"`
	Status       string    `gorm:"not null;index"`
	TokenVersion int       `gorm:"not null;default:0"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"not null;autoUpdateTime"`
}

// リフレッシュトークンと再利用検知に必要なトークン状態を保存
type RefreshToken struct {
	ID                uint64        `gorm:"primaryKey;autoIncrement"`
	UserID            uint64        `gorm:"not null;index"`
	TokenHash         string        `gorm:"not null;uniqueIndex"`
	FamilyID          string        `gorm:"not null;index"`
	UsedAt            *time.Time    `gorm:"index"`
	ReplacedByTokenID *uint64       `gorm:"index"`
	RevokedAt         *time.Time    `gorm:"index"`
	ExpiresAt         time.Time     `gorm:"not null;index"`
	CreatedAt         time.Time     `gorm:"not null;autoCreateTime"`
	User              User          `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	ReplacedByToken   *RefreshToken `gorm:"foreignKey:ReplacedByTokenID;references:ID;constraint:OnUpdate:CASCADE"`
}

// ゲストセッションを保存。ゲストのセッションキーを保存せず、SessionKeyHashだけで照合
type GuestSession struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	SessionKeyHash string    `gorm:"not null;uniqueIndex"`
	FirstSeenAt    time.Time `gorm:"not null"`
	LastSeenAt     time.Time `gorm:"not null;index"`
	ExpiresAt      time.Time `gorm:"not null;index"`
}

// ランキング、推薦、保存、評価の対象になるコーヒー豆を保存
type Bean struct {
	ID          uint64  `gorm:"primaryKey;autoIncrement"`
	Name        string  `gorm:"not null;index"`
	Roaster     *string `gorm:"index"`
	Origin      *string `gorm:"index"`
	Region      *string
	Farm        *string
	Variety     *string
	RoastLevel  string `gorm:"not null;index"`
	Acidity     *int
	Bitterness  *int
	Flavor      *int
	Aroma       *int
	Body        *int
	FlavorNote  *string
	Description *string
	ImageURL    *string
	IsPublished bool      `gorm:"not null;default:false;index"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"not null;autoUpdateTime"`
}

// ランキング対象の記事と、コーヒー豆詳細に紐づく関連記事を保存。
// 記事URL用識別子：重複したslugをDBで防ぐ。
type Article struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`
	Title       string `gorm:"not null;index"`
	Slug        string `gorm:"not null;uniqueIndex"`
	Summary     string `gorm:"not null"`
	Body        *string
	Category    *string `gorm:"index"`
	SourceName  *string
	SourceURL   *string
	ImageURL    *string
	IsPublished bool       `gorm:"not null;default:false;index"`
	PublishedAt *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"not null;autoUpdateTime"`
}

// コーヒー豆詳細へ表示する関連記事の対応関係を保存
// 関連元と関連記事：同じ関連の重複作成を防ぐ。
type BeanArticle struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	BeanID       uint64    `gorm:"not null;index;uniqueIndex:uq_bean_articles_bean_article"`
	ArticleID    uint64    `gorm:"not null;index;uniqueIndex:uq_bean_articles_bean_article"`
	DisplayOrder int       `gorm:"not null;default:0;index"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime"`
	Bean         Bean      `gorm:"foreignKey:BeanID;references:ID;constraint:OnUpdate:CASCADE"`
	Article      Article   `gorm:"foreignKey:ArticleID;references:ID;constraint:OnUpdate:CASCADE"`
}

// コーヒー豆と記事を共通のランキング対象として保存。
// 種別と対象ID：同じ実体コンテンツの重複登録を防ぐ。
type RankTarget struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement"`
	ContentType string    `gorm:"not null;index;uniqueIndex:uq_rank_targets_content"`
	ContentID   uint64    `gorm:"not null;uniqueIndex:uq_rank_targets_content"`
	IsActive    bool      `gorm:"not null;default:true;index"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"not null;autoUpdateTime"`
}

// ランキングと推薦に使う行動イベントを保存。
// 行動主体IDの片方だけを許可する：ユーザー行動とゲスト行動の混在を防ぐ。
type ActionEvent struct {
	ID                    uint64  `gorm:"primaryKey;autoIncrement"`
	UserID                *uint64 `gorm:"index"`
	GuestSessionID        *uint64 `gorm:"index"`
	EventType             string  `gorm:"not null;index"`
	RankTargetID          *uint64 `gorm:"index"`
	Placement             string  `gorm:"not null;index"`
	DwellMs               *int64
	RatingScore           *int
	SearchConditionHash   *string `gorm:"index"`
	PreviousConditionHash *string
	SearchKeyword         *string
	SearchOrigin          *string
	SearchRoastLevel      *string
	SearchAcidity         *int
	SearchBitterness      *int
	SearchAroma           *int
	SearchFlavor          *int
	SearchBody            *int
	SearchCategory        *string
	ModalDisplayLogID     *uint64 `gorm:"index"`
	PagePath              string  `gorm:"not null"`
	ReferrerPath          *string
	UserAgent             *string
	IPAddressHash         *string
	RequestID             *string          `gorm:"index"`
	OccurredAt            time.Time        `gorm:"not null;index"`
	User                  *User            `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	GuestSession          *GuestSession    `gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE"`
	RankTarget            *RankTarget      `gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
	ModalDisplayLog       *ModalDisplayLog `gorm:"foreignKey:ModalDisplayLogID;references:ID;constraint:OnUpdate:CASCADE"`
}

// 推薦モーダルを実際に表示した履歴を保存。
// 行動主体を片方だけに制限し、クリックやクローズ更新でactor条件を使える状態を保つ。
type ModalDisplayLog struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	UserID         *uint64   `gorm:"index"`
	GuestSessionID *uint64   `gorm:"index"`
	RankTargetID   uint64    `gorm:"not null;index"`
	Trigger        string    `gorm:"not null;index"`
	PagePath       string    `gorm:"not null"`
	ShownAt        time.Time `gorm:"not null;index"`
	ClickedAt      *time.Time
	ClosedAt       *time.Time
	CreatedAt      time.Time     `gorm:"not null;autoCreateTime"`
	User           *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	GuestSession   *GuestSession `gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE"`
	RankTarget     RankTarget    `gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

// 推薦モーダルを表示しなかった理由を保存。
// ユーザーとゲストの非表示理由が混ざらないようにする。
type ModalBlockLog struct {
	ID                    uint64        `gorm:"primaryKey;autoIncrement"`
	UserID                *uint64       `gorm:"index"`
	GuestSessionID        *uint64       `gorm:"index"`
	CandidateRankTargetID *uint64       `gorm:"index"`
	Reason                string        `gorm:"not null;index"`
	PagePath              string        `gorm:"not null"`
	BlockedAt             time.Time     `gorm:"not null;index"`
	User                  *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	GuestSession          *GuestSession `gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE"`
	CandidateRankTarget   *RankTarget   `gorm:"foreignKey:CandidateRankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

// 認証済みユーザーがランキング対象を保存した状態を保存
// ユーザーとランキング対象の複合一意制約で、同じ対象の二重保存を防ぐ。
type SavedItem struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement"`
	UserID       uint64     `gorm:"not null;index;uniqueIndex:uq_saved_items_user_target"`
	RankTargetID uint64     `gorm:"not null;index;uniqueIndex:uq_saved_items_user_target"`
	RemovedAt    *time.Time `gorm:"index"`
	CreatedAt    time.Time  `gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"not null;autoUpdateTime"`
	User         User       `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	RankTarget   RankTarget `gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

// 認証済みユーザーがランキング対象へ付けたGood/Bad評価を保存。
// 評価値：1と-1だけを許可し、5段階評価の混入を防ぐ。
type Rating struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement"`
	UserID       uint64     `gorm:"not null;index;uniqueIndex:uq_ratings_user_target"`
	RankTargetID uint64     `gorm:"not null;index;uniqueIndex:uq_ratings_user_target"`
	Score        int        `gorm:"not null;index"`
	CreatedAt    time.Time  `gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time  `gorm:"not null;autoUpdateTime"`
	User         User       `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	RankTarget   RankTarget `gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

// 1つのランキング対象に対する最新の集計指標を保存。
// ランキング対象IDを一意：APIアクセスごとに再集計しないために最新値だけを保持。
type ContentMetric struct {
	ID                   uint64     `gorm:"primaryKey;autoIncrement"`
	RankTargetID         uint64     `gorm:"not null;uniqueIndex"`
	Score                float64    `gorm:"not null;default:0;index"`
	ImpressionCount      int64      `gorm:"not null;default:0"`
	ContentViewCount     int64      `gorm:"not null;default:0"`
	ClickCount           int64      `gorm:"not null;default:0"`
	StayTotalMs          int64      `gorm:"not null;default:0"`
	SaveCount            int64      `gorm:"not null;default:0"`
	RatingCount          int64      `gorm:"not null;default:0"`
	GoodCount            int64      `gorm:"not null;default:0"`
	BadCount             int64      `gorm:"not null;default:0"`
	RatingAvg            float64    `gorm:"not null;default:0"`
	GoodRate             float64    `gorm:"not null;default:0"`
	BadRate              float64    `gorm:"not null;default:0"`
	ModalImpressionCount int64      `gorm:"not null;default:0"`
	ModalClickCount      int64      `gorm:"not null;default:0"`
	ModalCloseCount      int64      `gorm:"not null;default:0"`
	ClickRate            float64    `gorm:"not null;default:0"`
	SaveRate             float64    `gorm:"not null;default:0"`
	ModalClickRate       float64    `gorm:"not null;default:0"`
	ModalCloseRate       float64    `gorm:"not null;default:0"`
	PeriodStart          time.Time  `gorm:"not null;index"`
	PeriodEnd            time.Time  `gorm:"not null;index"`
	CalculatedAt         time.Time  `gorm:"not null;index"`
	UpdatedAt            time.Time  `gorm:"not null;autoUpdateTime"`
	RankTarget           RankTarget `gorm:"foreignKey:RankTargetID;references:ID;constraint:OnUpdate:CASCADE"`
}

// 認証済みユーザーまたはゲストセッションごとの興味スコアを保存。
// 同じ主体の同じ興味軸を重複作成しない。
type InterestProfile struct {
	ID             uint64        `gorm:"primaryKey;autoIncrement"`
	UserID         *uint64       `gorm:"index"`
	GuestSessionID *uint64       `gorm:"index"`
	Dimension      string        `gorm:"not null;index"`
	Value          string        `gorm:"not null;index"`
	Score          float64       `gorm:"not null;default:0;index"`
	LastEventAt    time.Time     `gorm:"not null;index"`
	ExpiresAt      *time.Time    `gorm:"index"`
	CreatedAt      time.Time     `gorm:"not null;autoCreateTime"`
	UpdatedAt      time.Time     `gorm:"not null;autoUpdateTime"`
	User           *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE"`
	GuestSession   *GuestSession `gorm:"foreignKey:GuestSessionID;references:ID;constraint:OnUpdate:CASCADE"`
}

// ランキングや興味プロフィール再計算バッチの実行結果を保存。
// 実行主体を保存し、自動実行と管理者の手動実行を後から追跡できるようにする。
type BatchRun struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement"`
	JobName         string    `gorm:"not null;index"`
	Status          string    `gorm:"not null;index"`
	StartedAt       time.Time `gorm:"not null;index"`
	FinishedAt      *time.Time
	RowsProcessed   int64 `gorm:"not null;default:0"`
	ErrorMessage    *string
	TriggeredBy     string  `gorm:"not null;index"`
	TriggeredUserID *uint64 `gorm:"index"`
	TriggeredUser   *User   `gorm:"foreignKey:TriggeredUserID;references:ID;constraint:OnUpdate:CASCADE"`
}

// 認証、管理、バッチなど後から追跡すべき操作を保存。
// IPアドレスやtokenなどの危険な情報は、そのままDBに保存しない。
// 後から操作を確認できるように、IPのhash値と操作対象のIDだけを保存。
type AuditLog struct {
	ID            uint64  `gorm:"primaryKey;autoIncrement"`
	ActorType     string  `gorm:"not null;index"`
	ActorUserID   *uint64 `gorm:"index"`
	Action        string  `gorm:"not null;index"`
	TargetType    *string `gorm:"index"`
	TargetID      *uint64 `gorm:"index"`
	Detail        *string
	IPAddressHash *string
	RequestID     *string   `gorm:"index"`
	CreatedAt     time.Time `gorm:"not null;autoCreateTime;index"`
	ActorUser     *User     `gorm:"foreignKey:ActorUserID;references:ID;constraint:OnUpdate:CASCADE"`
}
