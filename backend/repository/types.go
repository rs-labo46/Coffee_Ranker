package repository

import (
	"context"
	"time"

	"coffee-ranker/entity"
)

// 検索したBeanの条件をRepositoryへ渡す。
// ValidatorとUsecaseで安全な値に整えた後、RepositoryはDB検索条件としてだけ使う。
type BeanSearchFilter struct {
	Keyword    *string
	Origin     *string
	RoastLevel *entity.RoastLevel
	Acidity    *int
	Bitterness *int
	Flavor     *int
	Aroma      *int
	Body       *int
	Sort       string
	Limit      int
	Offset     int
}

// 検索したArticleの条件をRepositoryへ渡す。
// Repositoryは業務判断をせず、公開済みArticleの絞り込みと並び替えだけ。
type ArticleSearchFilter struct {
	Keyword  *string
	Category *string
	Sort     string
	Limit    int
	Offset   int
}

// ランキング指標の材料になる集計結果。
// Repositoryは集計値だけを返し、score計算はUsecaseで行う。
type ContentMetricAggregate struct {
	RankTargetID         uint64
	ImpressionCount      int64
	ContentViewCount     int64
	ClickCount           int64
	StayTotalMs          int64
	SaveCount            int64
	RatingCount          int64
	GoodCount            int64
	BadCount             int64
	ModalImpressionCount int64
	ModalClickCount      int64
	ModalCloseCount      int64
}

// Good/Bad評価の集計結果。
// 5段階評価を混ぜず、GoodとBadだけの集計に固定する。
type RatingAggregate struct {
	RatingCount int64
	GoodCount   int64
	BadCount    int64
	RatingAvg   float64
	GoodRate    float64
	BadRate     float64
}

// 興味プロフィール：再計算の材料になる集計結果。
// UserまたはGuestSessionの片方だけを持つ前提でUsecaseがスコアへ反映する。
type InterestAggregate struct {
	UserID         *uint64
	GuestSessionID *uint64
	Dimension      entity.InterestDimension
	Value          string
	ScoreDelta     float64
	LastEventAt    time.Time
}

// 監査ログ検索条件をRepositoryへ渡す。
type AuditLogFilter struct {
	ActorType   *entity.AuditActorType
	ActorUserID *uint64
	Action      *entity.AuditAction
	TargetType  *string
	TargetID    *uint64
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}

// TokenBucket方式のRateLimit判定結果。
// Middlewareはこの結果だけを見てHTTPレスポンスへ変換する。
type RateLimitResult struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

// 複数DB更新を1つのtransactionで扱うための境界。
// UsecaseはGORMを直接扱わず、必要なときだけこのinterface経由でTxを開始する。
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx TxRepos) error) error
}

// transaction内で使うRepository群。
// transaction用のDB接続をctxの中に隠すのではなく、transaction用RepositoryセットをTxReposとして明示的に渡す。そうすることで、UsecaseがGORMに依存せず、どの処理がtransaction内なのか分かりやすくする
type TxRepos interface {
	User() UserRepository
	RefreshToken() RefreshTokenRepository
	Bean() BeanRepository
	Article() ArticleRepository
	RankTarget() RankTargetRepository
	BeanArticle() BeanArticleRepository
	ContentMetric() ContentMetricRepository
	BatchRun() BatchRunRepository
}

// RefreshTokenの作成、取得、使用済み化、失効。
// 生RefreshTokenは扱わず、Usecaseでhash化された値だけを保存・検索する。
type RefreshTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	MarkUsed(ctx context.Context, id uint64, usedAt time.Time, replacedByTokenID uint64) error
	Create(ctx context.Context, token *entity.RefreshToken) error
	Revoke(ctx context.Context, id uint64, revokedAt time.Time) error
	RevokeFamily(ctx context.Context, familyID string, revokedAt time.Time) error
	RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// 未ログインユーザーの一時識別情報を保存・取得。
// 生GuestSessionキーは扱わず、hash化済みの値だけを保存する。
type GuestSessionRepository interface {
	Create(ctx context.Context, session *entity.GuestSession) error
	FindByID(ctx context.Context, id uint64) (*entity.GuestSession, error)
	FindBySessionKeyHash(ctx context.Context, hash string) (*entity.GuestSession, error)
	Touch(ctx context.Context, id uint64, lastSeenAt time.Time, expiresAt time.Time) error
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// Beanの作成、更新、公開一覧、検索、詳細取得。
// 公開可否の業務判断はUsecaseで行い、Repositoryは指定条件でDBを操作する。
type BeanRepository interface {
	Create(ctx context.Context, bean *entity.Bean) error
	Update(ctx context.Context, bean *entity.Bean) error
	FindByID(ctx context.Context, id uint64) (*entity.Bean, error)
	FindPublishedByID(ctx context.Context, id uint64) (*entity.Bean, error)
	ListPublished(ctx context.Context, limit int, offset int) ([]*entity.Bean, error)
	SearchPublished(ctx context.Context, filter BeanSearchFilter) ([]*entity.Bean, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]*entity.Bean, error)
	ExistsByID(ctx context.Context, id uint64) (bool, error)
	ExistsPublishedByID(ctx context.Context, id uint64) (bool, error)
	UpdatePublished(ctx context.Context, id uint64, isPublished bool) error
}

// Articleの作成、更新、公開一覧、検索、詳細取得。
// slugの形式検証やGuest閲覧制御はRepositoryではなくValidator・Usecaseで行う。
type ArticleRepository interface {
	Create(ctx context.Context, article *entity.Article) error
	Update(ctx context.Context, article *entity.Article) error
	FindByID(ctx context.Context, id uint64) (*entity.Article, error)
	FindPublishedByID(ctx context.Context, id uint64) (*entity.Article, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Article, error)
	FindPublishedBySlug(ctx context.Context, slug string) (*entity.Article, error)
	ListPublished(ctx context.Context, limit int, offset int) ([]*entity.Article, error)
	SearchPublished(ctx context.Context, filter ArticleSearchFilter) ([]*entity.Article, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]*entity.Article, error)
	ExistsByID(ctx context.Context, id uint64) (bool, error)
	ExistsPublishedByID(ctx context.Context, id uint64) (bool, error)
	UpdatePublished(ctx context.Context, id uint64, isPublished bool) error
}

// BeanとArticleの関連付け。
// 一括差し替えのtransaction境界はUsecaseとTxManagerが決める。
type BeanArticleRepository interface {
	Create(ctx context.Context, relation *entity.BeanArticle) error
	Delete(ctx context.Context, beanID uint64, articleID uint64) error
	Exists(ctx context.Context, beanID uint64, articleID uint64) (bool, error)
	ListByBeanID(ctx context.Context, beanID uint64, limit int) ([]*entity.BeanArticle, error)
	ListByArticleID(ctx context.Context, articleID uint64, limit int) ([]*entity.BeanArticle, error)
	ReplaceByBeanID(ctx context.Context, beanID uint64, articleIDs []uint64) error
}

// BeanとArticleを共通ランキング対象として取得・更新する。
// 実体コンテンツの存在確認はUsecaseが各Repositoryを使って判断する。
type RankTargetRepository interface {
	Create(ctx context.Context, target *entity.RankTarget) error
	FindByID(ctx context.Context, id uint64) (*entity.RankTarget, error)
	FindByContent(ctx context.Context, contentType entity.ContentType, contentID uint64) (*entity.RankTarget, error)
	FindOrCreate(ctx context.Context, contentType entity.ContentType, contentID uint64) (*entity.RankTarget, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]*entity.RankTarget, error)
	ListActiveByType(ctx context.Context, contentType entity.ContentType) ([]*entity.RankTarget, error)
	ExistsActiveByID(ctx context.Context, id uint64) (bool, error)
	UpdateActive(ctx context.Context, id uint64, isActive bool) error
}

// 行動イベント記録、ランキング集計、興味集計。
// event_typeの許可判断はUsecase/Validatorが行い、Repositoryは保存と集計だけを扱う。
type ActionEventRepository interface {
	Create(ctx context.Context, event *entity.ActionEvent) error
	BulkCreate(ctx context.Context, events []*entity.ActionEvent) error
	FindRecentByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*entity.ActionEvent, error)
	FindLastSearchHash(ctx context.Context, userID *uint64, guestSessionID *uint64) (*string, error)
	AggregateContentMetrics(ctx context.Context, periodStart time.Time, periodEnd time.Time) ([]ContentMetricAggregate, error)
	AggregateUserInterest(ctx context.Context, userID uint64, periodStart time.Time, periodEnd time.Time) ([]InterestAggregate, error)
	AggregateGuestInterest(ctx context.Context, guestSessionID uint64, periodStart time.Time, periodEnd time.Time) ([]InterestAggregate, error)
	CountByActorAndTypeSince(ctx context.Context, userID *uint64, guestSessionID *uint64, eventType entity.EventType, since time.Time) (int64, error)
	CountByTargetAndTypeSince(ctx context.Context, rankTargetID uint64, eventType entity.EventType, since time.Time) (int64, error)
}

// モーダル表示履歴の作成と本人条件付き更新。
// モーダル表示ログのIDだけを信じて更新すると、他人のモーダル履歴まで操作できてしまうため、必ず誰のログかも一緒に確認してから更新する
type ModalDisplayLogRepository interface {
	Create(ctx context.Context, log *entity.ModalDisplayLog) error
	FindByIDForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64) (*entity.ModalDisplayLog, error)
	MarkClickedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, clickedAt time.Time) error
	MarkClosedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, closedAt time.Time) error
	CountShownInSession(ctx context.Context, userID *uint64, guestSessionID *uint64, since time.Time) (int64, error)
	CountShownOnPage(ctx context.Context, userID *uint64, guestSessionID *uint64, pagePath string, since time.Time) (int64, error)
	WasTargetShownRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error)
	WasTargetClosedRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error)
}

// モーダル非表示理由の保存と分析用取得。
// 表示するかしないかの判断自体はModalUsecaseで行う。
type ModalBlockLogRepository interface {
	Create(ctx context.Context, log *entity.ModalBlockLog) error
	ListByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*entity.ModalBlockLog, error)
	CountByReasonSince(ctx context.Context, reason entity.ModalBlockReason, since time.Time) (int64, error)
	ListByCandidate(ctx context.Context, rankTargetID uint64, limit int) ([]*entity.ModalBlockLog, error)
}

// 保存、保存解除、再保存、保存一覧。
// Guest不可などの業務判断はUsecaseで行う。
type SavedItemRepository interface {
	Create(ctx context.Context, item *entity.SavedItem) error
	FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*entity.SavedItem, error)
	SaveOrRestore(ctx context.Context, userID uint64, rankTargetID uint64, now time.Time) (*entity.SavedItem, error)
	Remove(ctx context.Context, userID uint64, rankTargetID uint64, removedAt time.Time) error
	ListActiveByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*entity.SavedItem, error)
	ExistsActive(ctx context.Context, userID uint64, rankTargetID uint64) (bool, error)
	CountActiveByTarget(ctx context.Context, rankTargetID uint64) (int64, error)
}

// Good/Bad評価、再評価、評価集計。
// 評価値はentity.RatingScoreで受け取り、1と-1以外を混ぜない。
type RatingRepository interface {
	Create(ctx context.Context, rating *entity.Rating) error
	FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*entity.Rating, error)
	Upsert(ctx context.Context, userID uint64, rankTargetID uint64, score entity.RatingScore, now time.Time) (*entity.Rating, error)
	Delete(ctx context.Context, userID uint64, rankTargetID uint64) error
	CountByTarget(ctx context.Context, rankTargetID uint64) (int64, error)
	AggregateByTarget(ctx context.Context, rankTargetID uint64) (RatingAggregate, error)
}

// ランキング指標の保存とランキング取得。
// score計算はUsecaseで行い、Repositoryは保存済み指標の読み書きだけを扱う。
type ContentMetricRepository interface {
	Upsert(ctx context.Context, metric *entity.ContentMetric) error
	BulkUpsert(ctx context.Context, metrics []*entity.ContentMetric) error
	FindByRankTargetID(ctx context.Context, rankTargetID uint64) (*entity.ContentMetric, error)
	FindByRankTargetIDs(ctx context.Context, rankTargetIDs []uint64) ([]*entity.ContentMetric, error)
	ListRanking(ctx context.Context, contentType *entity.ContentType, limit int, offset int) ([]*entity.ContentMetric, error)
	ListTopByScore(ctx context.Context, limit int) ([]*entity.ContentMetric, error)
	LatestCalculatedAt(ctx context.Context) (*time.Time, error)
}

// 興味スコアの保存、取得、推薦用取得。
// 興味スコアは再計算可能な派生データであり、業務判断はUsecaseで行う。
type InterestProfileRepository interface {
	Upsert(ctx context.Context, profile *entity.InterestProfile) error
	BulkUpsert(ctx context.Context, profiles []*entity.InterestProfile) error
	FindByUser(ctx context.Context, userID uint64) ([]*entity.InterestProfile, error)
	FindByGuestSession(ctx context.Context, guestSessionID uint64) ([]*entity.InterestProfile, error)
	ListTopByUser(ctx context.Context, userID uint64, limit int) ([]*entity.InterestProfile, error)
	ListTopByGuest(ctx context.Context, guestSessionID uint64, limit int) ([]*entity.InterestProfile, error)
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

// バッチ開始、成功、失敗、履歴取得。
// 成功更新をContentMetric更新とTxで合わせるかはUsecaseが決める。
type BatchRunRepository interface {
	Create(ctx context.Context, run *entity.BatchRun) error
	MarkSuccess(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64) error
	MarkFailed(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64, errorMessage string) error
	FindLatestByJobName(ctx context.Context, jobName string) (*entity.BatchRun, error)
	FindRunningByJobName(ctx context.Context, jobName string) (*entity.BatchRun, error)
	List(ctx context.Context, limit int, offset int) ([]*entity.BatchRun, error)
}

// 監査ログ作成、検索、追跡。
// AuditLogは原則失敗しても本体処理は成功扱いにする。本体処理のrollback条件にしない。
type AuditLogRepository interface {
	Create(ctx context.Context, log *entity.AuditLog) error
	FindByID(ctx context.Context, id uint64) (*entity.AuditLog, error)
	List(ctx context.Context, filter AuditLogFilter) ([]*entity.AuditLog, error)
	ListByActor(ctx context.Context, actorType entity.AuditActorType, actorUserID *uint64, limit int) ([]*entity.AuditLog, error)
	ListByTarget(ctx context.Context, targetType string, targetID uint64, limit int) ([]*entity.AuditLog, error)
	ListByRequestID(ctx context.Context, requestID string) ([]*entity.AuditLog, error)
}

// API RateLimit状態をRedisで管理する。
// TokenBucketの補充と消費はRedis側で一体的に行い、競合を防ぐ。
type RateLimitRepository interface {
	Take(ctx context.Context, key string, capacity int, refillRate float64, now time.Time) (RateLimitResult, error)
	Reset(ctx context.Context, key string) error
}

// 行動イベントの重複防止キーをRedisで管理する。
// impressionやcontent_viewの二重送信をDB保存前に抑制する。
type EventDedupRepository interface {
	SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
}

// 推薦モーダルを何回出したか、同じ候補をしばらく出さないかをRedisで一時的に管理する
type ModalSuppressionRepository interface {
	SetShown(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error
	WasShown(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error)
	IncrementPageCount(ctx context.Context, actorKey string, pagePath string, ttl time.Duration) (int64, error)
	IncrementSessionCount(ctx context.Context, actorKey string, ttl time.Duration) (int64, error)
	SetClosed(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error
	WasClosed(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error)
}

// 同じバッチが同時に動かないように、Redisで実行中の印を管理する。
// 自分が取ったlockだけ解除・延長し、別の処理のlockは触らない。
type BatchLockRepository interface {
	Acquire(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string, owner string) error
	Extend(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error)
}
