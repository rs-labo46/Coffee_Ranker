package usecase

import (
	"context"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 管理者が監査ログを確認。
// ログイン、ログアウト、管理者操作、バッチ実行などの履歴を確認。
type AdminAuditUsecase struct {
	audits repository.AuditLogRepository
}

// 期限切れデータを削除。
// RefreshToken、GuestSession、一時的なInterestProfileを掃除。
type CleanupUsecase struct {
	refreshTokens repository.RefreshTokenRepository
	guestSessions repository.GuestSessionRepository
	interests     repository.InterestProfileRepository
}

// API利用回数を制限する。
// どのkeyをどの制限値で判定するかはUsecase側で決め、Redis上の残数更新・補充・消費の原子的な処理はRepositoryへ任せる。
type RateLimitUsecase struct {
	rates repository.RateLimitRepository
}

// 管理者がRateLimit状態を操作。
// 管理者によるreset操作は監査ログに残す。
type AdminRateLimitUsecase struct {
	rates  repository.RateLimitRepository
	audits repository.AuditLogRepository
}

// Cleanupで削除した件数を返す結果。
// 管理画面やログで「何件掃除したか」を確認できるように。
type CleanupResult struct {
	RefreshTokensDeleted int64
	GuestSessionsDeleted int64
	InterestDeleted      int64
}

// RateLimit判定に使うルール。
// keyは制限対象、capacityは最大保持数、refillRateは1秒あたりの補充量。
type RateLimitRule struct {
	Key        string
	Capacity   int
	RefillRate float64
}

// AdminAuditUsecaseを組み立てる。
func NewAdminAuditUsecase(audits repository.AuditLogRepository) *AdminAuditUsecase {
	return &AdminAuditUsecase{audits: audits}
}

// CleanupUsecaseを組み立てる。
func NewCleanupUsecase(
	refreshTokens repository.RefreshTokenRepository,
	guestSessions repository.GuestSessionRepository,
	interests repository.InterestProfileRepository,
) *CleanupUsecase {
	return &CleanupUsecase{
		refreshTokens: refreshTokens,
		guestSessions: guestSessions,
		interests:     interests,
	}
}

// RateLimitUsecaseを組み立てる。
func NewRateLimitUsecase(rates repository.RateLimitRepository) *RateLimitUsecase {
	return &RateLimitUsecase{rates: rates}
}

// AdminRateLimitUsecaseを組み立てる。
// 管理者によるRateLimit resetを監査ログに残すため、AuditLogRepositoryも受け取る。
func NewAdminRateLimitUsecase(
	rates repository.RateLimitRepository,
	audits repository.AuditLogRepository,
) *AdminRateLimitUsecase {
	return &AdminRateLimitUsecase{
		rates:  rates,
		audits: audits,
	}
}

// 指定IDの監査ログを1件取得。
// idが0だと対象ログを特定できないため、不正入力として止める。
func (u *AdminAuditUsecase) FindByID(ctx context.Context, id uint64) (*model.AuditLog, error) {
	if id == 0 {
		return nil, entity.ErrInvalidInput
	}

	return u.audits.FindByID(ctx, id)
}

// 監査ログ一覧を条件付きで取得。
// limit/offsetを制限し、大量取得によるDB負荷を防ぐ。
func (u *AdminAuditUsecase) List(ctx context.Context, filter repository.AuditLogFilter) ([]*model.AuditLog, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 || filter.Offset < 0 || filter.Offset > 10000 {
		return nil, entity.ErrInvalidPagination
	}

	return u.audits.List(ctx, filter)
}

// request_idに紐づく監査ログを取得。
// 同じAPIリクエストで発生した操作を追跡するときに使う。
func (u *AdminAuditUsecase) ListByRequestID(ctx context.Context, requestID string) ([]*model.AuditLog, error) {
	if requestID == "" {
		return nil, entity.ErrInvalidInput
	}

	return u.audits.ListByRequestID(ctx, requestID)
}

// 期限切れデータを削除。
// それぞれ独立しているため、Txではまとめない。途中で失敗した場合は、その時点でエラーを返す。
func (u *CleanupUsecase) DeleteExpired(ctx context.Context) (CleanupResult, error) {
	now := time.Now()

	// 期限切れRefreshTokenを削除。
	// 期限切れtokenは再利用できないため、残しておく必要がない。
	refreshDeleted, err := u.refreshTokens.DeleteExpired(ctx, now)
	if err != nil {
		return CleanupResult{}, err
	}

	// 期限切れGuestSessionを削除。
	// 有効期限切れのGuestは、次回アクセス時に新しいsessionを作ればよい。
	guestDeleted, err := u.guestSessions.DeleteExpired(ctx, now)
	if err != nil {
		return CleanupResult{}, err
	}

	// 期限切れInterestProfileを削除。
	// Guestなど一時的な興味情報を残し続けないため。
	interestDeleted, err := u.interests.DeleteExpired(ctx, now)
	if err != nil {
		return CleanupResult{}, err
	}

	return CleanupResult{
		RefreshTokensDeleted: refreshDeleted,
		GuestSessionsDeleted: guestDeleted,
		InterestDeleted:      interestDeleted,
	}, nil
}

// RateLimitを1回分消費。
// 許可された場合は成功、制限超過ならErrRateLimitedを返す。
func (u *RateLimitUsecase) Take(ctx context.Context, rule RateLimitRule, now time.Time) (repository.RateLimitResult, error) {
	if rule.Key == "" || rule.Capacity <= 0 || rule.RefillRate <= 0 {
		return repository.RateLimitResult{}, entity.ErrInvalidInput
	}

	// 呼び出し元が時刻を渡さなかった場合でも動くように。
	if now.IsZero() {
		now = time.Now()
	}

	result, err := u.rates.Take(ctx, rule.Key, rule.Capacity, rule.RefillRate, now)
	if err != nil {
		return repository.RateLimitResult{}, err
	}

	// RepositoryはRedis上の残数更新・補充・消費結果を返す。
	// Usecaseでは、その結果を業務エラーErrRateLimitedに変換。
	if !result.Allowed {
		return result, entity.ErrRateLimited
	}

	return result, nil
}

// 指定keyのRateLimit状態を削除。
// 管理者APIから使う場合はAdminRateLimitUsecase.Resetを使う。
func (u *RateLimitUsecase) Reset(ctx context.Context, key string) error {
	if key == "" {
		return entity.ErrInvalidInput
	}

	return u.rates.Reset(ctx, key)
}

// 管理者操作として指定keyのRateLimit状態を削除。
// 誰がどのRateLimit keyをリセットしたか追跡できるよう、監査ログを残す。
func (u *AdminRateLimitUsecase) Reset(ctx context.Context, key string, meta AdminMeta) error {
	if meta.AdminUserID == 0 {
		return entity.ErrUnauthorized
	}
	if key == "" {
		return entity.ErrInvalidInput
	}

	if err := u.rates.Reset(ctx, key); err != nil {
		return err
	}

	targetType := "rate_limit"
	detail := key

	// RateLimit resetは管理者操作なので監査ログに残す。
	// 監査ログ作成に失敗しても、reset本体は成功扱いにする。
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionRateLimitReset,
		&targetType,
		nil,
		&detail,
		meta.AuditMeta,
	))

	return nil
}
