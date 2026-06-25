package entity

import "errors"

var (
	// ユースケース層へ安全に渡せない入力。
	ErrInvalidInput = errors.New("invalid input")

	// 一覧取得のlimitやoffsetが許可範囲外。
	ErrInvalidPagination = errors.New("invalid pagination")

	// content_typeがbeanまたはarticle以外。
	ErrInvalidContentType = errors.New("invalid content type")

	// event_typeが定義済み値ではない、またはPOST /eventsで直接受け付けない値。
	ErrInvalidEventType = errors.New("invalid event type")

	// rating_scoreがGood(+1)またはBad(-1)以外。
	ErrInvalidRatingScore = errors.New("invalid rating score")

	// 検索条件が文字数、範囲、許可値、危険入力のいずれかに違反した状態。
	ErrInvalidSearchCondition = errors.New("invalid search condition")

	// トークンの詳細を漏らさず、認証情報の欠落または信頼不能な状態。
	ErrUnauthorized = errors.New("unauthorized")

	// 認証済み行動主体にその操作権限がない状態。
	ErrForbidden = errors.New("forbidden")

	// ログインが必要な操作を未ログインで実行した状態。
	ErrLoginRequired = errors.New("login required")

	// メールアドレスの存在有無を漏らさず、ログイン失敗。
	ErrInvalidCredentials = errors.New("invalid credentials")

	// Signup時に同じemailが既に存在する状態。
	ErrEmailAlreadyExists = errors.New("email already exists")

	// suspended状態のUserがログイン、refresh、保護APIを実行した状態。
	ErrUserSuspended = errors.New("user suspended")

	// deleted状態のUserがログイン、refresh、保護APIを実行した状態。
	ErrUserDeleted = errors.New("user deleted")

	// 機密性の高い失敗理由を外へ出さず、トークン不正。
	ErrInvalidToken = errors.New("invalid token")

	// 保存済みトークン情報を漏らさず、リフレッシュトークンの期限切れ。
	ErrRefreshTokenExpired = errors.New("refresh token expired")

	// 失効済みRefreshTokenが使われた状態。
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")

	// トークンファミリー失効につなげるためのリフレッシュトークン再利用検知。
	ErrRefreshTokenReuseDetected = errors.New("refresh token reuse detected")

	// DB transactionの開始、commit、rollback境界で失敗した状態。
	ErrTransactionFailed = errors.New("transaction failed")

	// データ作成に失敗した状態。
	ErrCreateFailed = errors.New("create failed")

	// データ更新に失敗した状態。
	ErrUpdateFailed = errors.New("update failed")

	// データ削除または失効処理に失敗した状態。
	ErrDeleteFailed = errors.New("delete failed")

	// RefreshTokenや認証状態の失効処理に失敗した状態。
	ErrRevokeFailed = errors.New("revoke failed")

	// RepositoryまたはDB処理に失敗した状態。
	ErrRepositoryFailed = errors.New("repository failed")

	// ユースケースが必要とするコーヒー豆が存在しない状態。
	ErrBeanNotFound = errors.New("bean not found")

	// ユースケースが必要とする記事が存在しない状態。
	ErrArticleNotFound = errors.New("article not found")

	// 保存、評価、イベント、ランキング処理に必要なランキング対象が存在しない状態。
	ErrRankTargetNotFound = errors.New("rank target not found")

	// ゲストセッションが存在しない、期限切れ、または検証不能な状態。
	ErrGuestSessionNotFound = errors.New("guest session not found")

	// 保存対象が既に保存済みで、再保存ではなく重複として扱う状態。
	ErrSavedItemAlreadyExists = errors.New("saved item already exists")

	// 保存解除対象が保存中ではない状態。
	ErrSavedItemNotFound = errors.New("saved item not found")

	// 自分の評価が存在しない状態。
	ErrRatingNotFound = errors.New("rating not found")

	// 推薦モーダルの表示候補がない状態。
	ErrModalCandidateNotFound = errors.New("modal candidate not found")

	// 推薦モーダル表示ログが存在しない、またはactor条件に一致しない状態。
	ErrModalDisplayLogNotFound = errors.New("modal display log not found")

	// レート制限ルールにより操作が拒否された状態。
	ErrRateLimited = errors.New("rate limited")

	// バッチが既に実行中で二重実行できない状態。
	ErrBatchAlreadyRunning = errors.New("batch already running")

	// バッチロックの取得に失敗した状態。
	ErrBatchLockFailed = errors.New("batch lock failed")
)
