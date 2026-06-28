package entity

// 認証済みアカウントの権限。
// 未ログイン利用者はusersテーブル行ではなくゲストセッション。
type UserRole string

const (
	// 保存や評価など、通常の認証済みユーザー操作を許可。
	UserRoleUser UserRole = "user"

	// コンテンツ管理や手動バッチ実行などの管理者操作を許可。
	UserRoleAdmin UserRole = "admin"
)

// アカウントがログインやトークン再発行をできる状態か。
// ログインとリフレッシュは有効なユーザーだけに許可する。
type UserStatus string

const (
	// ログインとリフレッシュが可能なアカウント状態。
	UserStatusActive UserStatus = "active"

	// 停止中のためログインとリフレッシュを拒否するアカウント状態。
	UserStatusSuspended UserStatus = "suspended"

	// 退会済みのためログインとリフレッシュを拒否するアカウント状態。
	UserStatusDeleted UserStatus = "deleted"
)

// コーヒー豆と記事を区別しつつ、共通のランキング処理で扱うためのもの。
// ランキング対象はこの種別とコンテンツIDを組み合わせてコンテンツを参照する。
type ContentType string

const (
	// ランキング対象がコーヒー豆を参照する。
	ContentTypeBean ContentType = "bean"

	// ランキング対象が記事を参照する。
	ContentTypeArticle ContentType = "article"
)

// ランキングと推薦に使う行動ログの種類。
// 保存、評価、モーダル系の副作用イベントは、操作の成功後にだけ作成。
type EventType string

const (
	// コーヒー豆または記事の詳細ページを開いた行動。
	EventTypeContentView EventType = "content_view"

	// カードやコンテンツが表示条件を満たした行動。
	EventTypeImpression EventType = "impression"

	// 詳細ページで仕様上有効な滞在時間を記録する行動。
	EventTypeStay EventType = "stay"

	// モーダル内クリックとは別の通常クリック。
	EventTypeClick EventType = "click"

	// 認証済みユーザーの保存成功後にベストエフォートで記録する行動。
	EventTypeSave EventType = "save"

	// 認証済みユーザーの良い/悪い評価成功後にベストエフォートで記録する行動。
	EventTypeRating EventType = "rating"

	// ユーザーが検索条件を変更して再検索した事実。
	EventTypeReSearch EventType = "re_search"
	// 推薦モーダルが表示された行動。
	EventTypeModalImpression EventType = "modal_impression"
	// 推薦モーダル内のクリック。
	EventTypeModalClick EventType = "modal_click"
	// 推薦モーダルを閉じた行動。
	EventTypeModalClose EventType = "modal_close"
)

// 良い/悪いだけの値。
// バリデータ、データベース制約、ユースケースと一致する1と-1だけを定義。
type RatingScore int

const (
	// ユーザーの興味または好みに合うことを示した値。
	RatingScoreGood RatingScore = 1

	// ユーザーの興味の低さまたは不一致を示した値。
	RatingScoreBad RatingScore = -1
)

// 行動イベントが発生した画面上の場所。
// ランキングと推薦では通常クリック、モーダル、関連表示の行動を区別するため。
type Placement string

const (
	// トップページで発生した行動。
	PlacementTop Placement = "top"

	// 検索結果で発生した行動。
	PlacementSearchResult Placement = "search_result"

	// コーヒー豆詳細ページで発生した行動。
	PlacementBeanDetail Placement = "bean_detail"

	// 記事詳細ページで発生した行動。
	PlacementArticleDetail Placement = "article_detail"

	// コーヒー豆詳細に表示された関連記事で発生した行動。
	PlacementRelatedArticle Placement = "related_article"

	// 記事詳細に表示された関連コーヒー豆で発生した行動。
	PlacementRelatedBean Placement = "related_bean"

	// 推薦モーダル内で発生した行動。
	PlacementModal Placement = "modal"

	// 認証済みユーザーの保存一覧で発生した行動。
	PlacementSavedList Placement = "saved_list"
)

// 推薦モーダルを表示候補にした業務上の理由。
// 許可値はバリデータとモーダルユースケースの挙動と常に一致させる。
type ModalTrigger string

const (
	// 初回訪問時のおすすめ表示。
	ModalTriggerFirstVisit ModalTrigger = "first_visit"

	// ユーザーがコンテンツ末尾まで到達した後のおすすめ表示。
	ModalTriggerScrollEnd ModalTrigger = "scroll_end"

	// コーヒー豆詳細で一定時間滞在した後のおすすめ表示。
	ModalTriggerBeanStay ModalTrigger = "bean_stay"

	// 記事詳細で一定時間滞在した後のおすすめ表示。
	ModalTriggerArticleStay ModalTrigger = "article_stay"

	// 同じ産地のコーヒー豆を複数閲覧した後のおすすめ表示。
	ModalTriggerSameOriginViewed ModalTrigger = "same_origin_viewed"

	// 同じ焙煎度のコーヒー豆を複数クリックした後のおすすめ表示。
	ModalTriggerSameRoastClicked ModalTrigger = "same_roast_clicked"

	// 保存済みコンテンツに関連するおすすめ表示。
	ModalTriggerSavedContent ModalTrigger = "saved_content"

	// 好意的に評価したコンテンツに関連するおすすめ表示。
	ModalTriggerGoodRating ModalTrigger = "good_rating"

	// 検索条件を変更した後のおすすめ表示。
	ModalTriggerReSearch ModalTrigger = "re_search"
)

// 推薦モーダルを表示しなかった理由を記録するための値。
// 非表示理由を確認できるようにする。
type ModalBlockReason string

const (
	// ページを開いた直後のモーダル表示を防ぐ理由。
	ModalBlockPageJustOpened ModalBlockReason = "page_just_opened"

	// 同一ページで複数回表示しないための理由。
	ModalBlockPageLimitReached ModalBlockReason = "page_limit_reached"

	// 同一セッションで複数回表示しないための理由。
	ModalBlockSessionLimitReached ModalBlockReason = "session_limit_reached"

	// 同じ候補をクールダウン時間内に再表示しないための理由。
	ModalBlockRecentlyShown ModalBlockReason = "recently_shown"

	// ユーザーが直近で閉じた候補をすぐ再表示しないための理由。
	ModalBlockRecentlyClosed ModalBlockReason = "recently_closed"

	// 保存操作を邪魔しないための理由。
	ModalBlockDuringSave ModalBlockReason = "during_save"

	// 評価操作を邪魔しないための理由。
	ModalBlockDuringRating ModalBlockReason = "during_rating"

	// ログインモーダルと推薦モーダルを重ねないための理由。
	ModalBlockLoginModalOpen ModalBlockReason = "login_modal_open"

	// 保存済みコンテンツを原則として再推薦しない理由。
	ModalBlockAlreadySaved ModalBlockReason = "already_saved"

	// 表示可能な推薦候補が存在しなかった理由。
	ModalBlockNoCandidate ModalBlockReason = "no_candidate"
)

// 認証済みユーザーまたはゲストセッションの興味スコア軸。
// 興味の種類を共通で使うために定義。
type InterestDimension string

const (
	// 産地値への興味。
	InterestDimensionOrigin InterestDimension = "origin"

	// 焙煎度への興味。
	InterestDimensionRoastLevel InterestDimension = "roast_level"

	// 1〜5の酸味値への興味。
	InterestDimensionAcidity InterestDimension = "acidity"

	// 1〜5の苦味値への興味。
	InterestDimensionBitterness InterestDimension = "bitterness"

	// 1〜5の風味値への興味。
	InterestDimensionFlavor InterestDimension = "flavor"

	// 1〜5の香り値への興味。
	InterestDimensionAroma InterestDimension = "aroma"

	// 1〜5のボディ値への興味。
	InterestDimensionBody InterestDimension = "body"

	// 記事カテゴリへの興味。
	InterestDimensionArticleCategory InterestDimension = "article_category"
)

// コーヒー豆と検索条件で使う焙煎度。
// バリデータ、データベース制約、ユースケースと一致させる。
type RoastLevel string

const (
	// 浅煎り。
	RoastLevelLight RoastLevel = "light"
	// 中煎り。
	RoastLevelMedium RoastLevel = "medium"
	// 深煎り。
	RoastLevelDark RoastLevel = "dark"
)

// バッチ処理の状態。
// バッチ成功状態と指標更新は後のトランザクション層で整合。
type BatchStatus string

const (
	// 開始済みでまだ終了していないバッチ。
	BatchStatusRunning BatchStatus = "running"
	// 正常終了したバッチ。
	BatchStatusSuccess BatchStatus = "success"
	// エラーで終了したバッチ。
	BatchStatusFailed BatchStatus = "failed"
)

// 操作した人や仕組みの種類。
// 未ログイン利用者の通常行動は監査ログではなく行動イベントに記録する。
type AuditActorType string

const (
	// 認証済み一般ユーザーによる操作。
	AuditActorUser AuditActorType = "user"
	// 管理者による操作。
	AuditActorAdmin AuditActorType = "admin"
	// 自動処理などシステムによる操作。
	AuditActorSystem AuditActorType = "system"
)

// 後から追跡すべき認証・管理・バッチ操作。
// 通常のゲスト行動は行動イベントの責務であるため、この列挙には含めない。
type AuditAction string

const (
	// ログイン操作。
	AuditActionLogin AuditAction = "login"

	// ログアウト操作。
	AuditActionLogout AuditAction = "logout"

	// リフレッシュトークンの再利用検知。
	AuditActionRefreshReuseDetected AuditAction = "refresh_reuse_detected"

	// 管理者によるコーヒー豆作成。
	AuditActionBeanCreate AuditAction = "bean_create"

	// 管理者によるコーヒー豆更新。
	AuditActionBeanUpdate AuditAction = "bean_update"

	// 管理者によるコーヒー豆公開。
	AuditActionBeanPublish AuditAction = "bean_publish"

	// 管理者によるコーヒー豆非公開。
	AuditActionBeanUnpublish AuditAction = "bean_unpublish"

	// 管理者による記事作成。
	AuditActionArticleCreate AuditAction = "article_create"

	// 管理者による記事更新。
	AuditActionArticleUpdate AuditAction = "article_update"

	// 管理者による記事公開。
	AuditActionArticlePublish AuditAction = "article_publish"

	// 管理者による記事非公開。
	AuditActionArticleUnpublish AuditAction = "article_unpublish"

	// 自動ランキングバッチ実行。
	AuditActionRankingBatchRun AuditAction = "ranking_batch_run"

	// 管理者による手動バッチ実行。
	AuditActionManualBatchRun AuditAction = "manual_batch_run"

	// 管理者によるRateLimit状態のリセット。
	AuditActionRateLimitReset AuditAction = "rate_limit_reset"
)

const (
	// stayイベントとして受け付ける最小滞在時間。
	MinStaySeconds = 3
	// コーヒー豆詳細でモーダル候補になり得る滞在時間。
	BeanModalStaySeconds = 15

	// 記事詳細でモーダル候補になり得る滞在時間。
	ArticleModalStaySeconds = 30

	// 強い興味として扱う滞在時間。
	StrongStaySeconds = 120

	// 放置されたタブで興味スコアが過剰に増えないよう、1回のstayイベントを制限する。
	MaxStaySeconds = 1800

	// impressionとして扱う最小表示比率。
	ImpressionVisibleRatio = 0.5

	// impressionとして扱う最小表示時間。
	ImpressionVisibleSeconds = 1

	// 同一ページで推薦モーダルを繰り返し表示しないための上限。
	MaxModalPerPage = 1

	// 同一セッションで推薦モーダルを繰り返し表示しないための上限。
	MaxModalPerSession = 1

	// 同じモーダル候補の再表示を抑制する時間。
	ModalCooldownHours = 24

	// ランキング集計の対象期間を日数で表す。
	RankingWindowDays = 30

	// 日次ランキングバッチを実行する時刻。
	RankingBatchHour = 2
)
