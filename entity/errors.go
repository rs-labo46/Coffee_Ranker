package entity

import "errors"

var (
	// ユースケース層へ安全に渡せない入力。
	ErrInvalidInput = errors.New("invalid input")

	// トークンの詳細を漏らさず、認証情報の欠落または信頼不能な状態。
	ErrUnauthorized = errors.New("unauthorized")

	// 認証済み行動主体にその操作権限がない状態。
	ErrForbidden = errors.New("forbidden")

	// メールアドレスの存在有無を漏らさず、ログイン失敗。
	ErrInvalidCredentials = errors.New("invalid credentials")

	// 機密性の高い失敗理由を外へ出さず、トークン不正。
	ErrInvalidToken = errors.New("invalid token")

	// 保存済みトークン情報を漏らさず、リフレッシュトークンの期限切れ。
	ErrRefreshTokenExpired = errors.New("refresh token expired")

	// トークンファミリー失効につなげるためのリフレッシュトークン再利用検知。
	ErrRefreshTokenReuseDetected = errors.New("refresh token reuse detected")

	// ユースケースが必要とするコーヒー豆が存在しない状態。
	ErrBeanNotFound = errors.New("bean not found")

	// ユースケースが必要とする記事が存在しない状態。
	ErrArticleNotFound = errors.New("article not found")

	// 保存、評価、イベント、ランキング処理に必要なランキング対象が存在しない状態。
	ErrRankTargetNotFound = errors.New("rank target not found")

	// レート制限ルールにより操作が拒否された状態。
	ErrRateLimited = errors.New("rate limited")
)
