package validator

import (
	"strings"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/repository"
)

type AdminBeanValidator struct{}
type AdminArticleValidator struct{}
type AdminRelationValidator struct{}
type AdminBatchValidator struct{}
type AdminAuditValidator struct{}
type AdminRateLimitValidator struct{}

type AdminBeanRequest struct {
	ID          uint64            `json:"id,omitempty"`
	Name        string            `json:"name"`
	Roaster     *string           `json:"roaster,omitempty"`
	Origin      *string           `json:"origin,omitempty"`
	Region      *string           `json:"region,omitempty"`
	Farm        *string           `json:"farm,omitempty"`
	Variety     *string           `json:"variety,omitempty"`
	RoastLevel  entity.RoastLevel `json:"roast_level"`
	Acidity     *int              `json:"acidity,omitempty"`
	Bitterness  *int              `json:"bitterness,omitempty"`
	Flavor      *int              `json:"flavor,omitempty"`
	Aroma       *int              `json:"aroma,omitempty"`
	Body        *int              `json:"body,omitempty"`
	FlavorNote  *string           `json:"flavor_note,omitempty"`
	Description *string           `json:"description,omitempty"`
	ImageURL    *string           `json:"image_url,omitempty"`
	IsPublished bool              `json:"is_published"`
}

type AdminArticleRequest struct {
	ID          uint64     `json:"id,omitempty"`
	Title       string     `json:"title"`
	Slug        string     `json:"slug"`
	Summary     string     `json:"summary"`
	Body        *string    `json:"body,omitempty"`
	Category    *string    `json:"category,omitempty"`
	SourceName  *string    `json:"source_name,omitempty"`
	SourceURL   *string    `json:"source_url,omitempty"`
	ImageURL    *string    `json:"image_url,omitempty"`
	IsPublished bool       `json:"is_published"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type RelationRequest struct {
	BeanID       uint64 `json:"bean_id"`
	ArticleID    uint64 `json:"article_id"`
	DisplayOrder int    `json:"display_order"`
}

type ReplaceRelationsRequest struct {
	ArticleIDs []uint64 `json:"article_ids"`
}

type BatchRunRequest struct {
	Owner           string   `json:"owner"`
	UserIDs         []uint64 `json:"user_ids,omitempty"`
	GuestSessionIDs []uint64 `json:"guest_session_ids,omitempty"`
}

type AuditQuery struct {
	ActorType   string `query:"actor_type"`
	ActorUserID int    `query:"actor_user_id"`
	Action      string `query:"action"`
	TargetType  string `query:"target_type"`
	TargetID    int    `query:"target_id"`
	Limit       int    `query:"limit"`
	Offset      int    `query:"offset"`
}

type RateLimitResetRequest struct {
	Key string `json:"key"`
}

// NewAdminBeanValidatorを生成してDI層やRouterから使う。
func NewAdminBeanValidator() *AdminBeanValidator { return &AdminBeanValidator{} }

// NewAdminArticleValidatorを生成してDI層やRouterから使う。
func NewAdminArticleValidator() *AdminArticleValidator { return &AdminArticleValidator{} }

// NewAdminRelationValidatorを生成してDI層やRouterから使う。
func NewAdminRelationValidator() *AdminRelationValidator { return &AdminRelationValidator{} }

// NewAdminBatchValidatorを生成してDI層やRouterから使う。
func NewAdminBatchValidator() *AdminBatchValidator { return &AdminBatchValidator{} }

// NewAdminAuditValidatorを生成してDI層やRouterから使う。
func NewAdminAuditValidator() *AdminAuditValidator { return &AdminAuditValidator{} }

// NewAdminRateLimitValidatorを生成してDI層やRouterから使う。
func NewAdminRateLimitValidator() *AdminRateLimitValidator { return &AdminRateLimitValidator{} }

// 管理画面のBean作成・更新Requestを検証。
// requireID=trueの場合はID必須とし、名前、焙煎度、任意項目、URL、味覚スコア1〜5を確認。
func (v *AdminBeanValidator) Bean(input AdminBeanRequest, requireID bool) (AdminBeanRequest, error) {
	if requireID {
		// IDはDB存在確認ではなく、0でない正の値かだけ確認。
		if err := ValidateID(input.ID); err != nil {
			return AdminBeanRequest{}, err
		}
	}
	name, err := NormalizeText(input.Name, 1, 100)
	if err != nil {
		return AdminBeanRequest{}, err
	}
	input.Name = name
	if err := ValidateRequiredRoastLevel(input.RoastLevel); err != nil {
		return AdminBeanRequest{}, err
	}
	var normalizeErr error
	input.Roaster, normalizeErr = NormalizeOptionalText(input.Roaster, 100)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	input.Origin, normalizeErr = NormalizeOptionalText(input.Origin, 50)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	input.Region, normalizeErr = NormalizeOptionalText(input.Region, 50)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	input.Farm, normalizeErr = NormalizeOptionalText(input.Farm, 100)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	input.Variety, normalizeErr = NormalizeOptionalText(input.Variety, 100)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	input.FlavorNote, normalizeErr = NormalizeOptionalText(input.FlavorNote, 500)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	input.Description, normalizeErr = NormalizeOptionalText(input.Description, 2000)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	input.ImageURL, normalizeErr = ValidateURL(input.ImageURL)
	if normalizeErr != nil {
		return AdminBeanRequest{}, normalizeErr
	}
	for _, score := range []*int{input.Acidity, input.Bitterness, input.Flavor, input.Aroma, input.Body} {
		// 味覚スコアは1〜5だけ許可。
		if err := ValidateTasteScore(score); err != nil {
			return AdminBeanRequest{}, err
		}
	}
	return input, nil
}

// 管理画面でBeanを特定するIDが0でないかを検証。
// Beanが存在するかどうかはUsecaseで判断。
func (v *AdminBeanValidator) ID(id uint64) error { return ValidateID(id) }

// 管理画面のArticle作成・更新Requestを検証。
// requireID=trueの場合はID必須とし、title、slug、summary、本文、カテゴリ、URLを確認。
func (v *AdminArticleValidator) Article(input AdminArticleRequest, requireID bool) (AdminArticleRequest, error) {
	if requireID {
		// IDはDB存在確認ではなく、0でない正の値かだけ確認。
		if err := ValidateID(input.ID); err != nil {
			return AdminArticleRequest{}, err
		}
	}
	title, err := NormalizeText(input.Title, 1, 120)
	if err != nil {
		return AdminArticleRequest{}, err
	}
	slug, err := ValidateSlug(input.Slug)
	if err != nil {
		return AdminArticleRequest{}, err
	}
	summary, err := NormalizeText(input.Summary, 1, 300)
	if err != nil {
		return AdminArticleRequest{}, err
	}
	input.Title = title
	input.Slug = slug
	input.Summary = summary
	var normalizeErr error
	input.Body, normalizeErr = NormalizeOptionalText(input.Body, 10000)
	if normalizeErr != nil {
		return AdminArticleRequest{}, normalizeErr
	}
	input.Category, normalizeErr = ValidateCategory(input.Category)
	if normalizeErr != nil {
		return AdminArticleRequest{}, normalizeErr
	}
	input.SourceName, normalizeErr = NormalizeOptionalText(input.SourceName, 100)
	if normalizeErr != nil {
		return AdminArticleRequest{}, normalizeErr
	}
	input.SourceURL, normalizeErr = ValidateURL(input.SourceURL)
	if normalizeErr != nil {
		return AdminArticleRequest{}, normalizeErr
	}
	input.ImageURL, normalizeErr = ValidateURL(input.ImageURL)
	if normalizeErr != nil {
		return AdminArticleRequest{}, normalizeErr
	}
	return input, nil
}

// 管理画面でArticleを特定するIDが0でないかを検証。
// Articleが存在するかどうかはUsecaseで判断。
func (v *AdminArticleValidator) ID(id uint64) error { return ValidateID(id) }

// BeanとArticleの関連付けRequestを検証。
// bean_id/article_id/display_orderの形だけを確認し、DB存在確認はUsecaseで行う。
func (v *AdminRelationValidator) Relation(input RelationRequest) (RelationRequest, error) {
	// IDはDB存在確認ではなく、0でない正の値かだけ確認。
	if err := ValidateID(input.BeanID); err != nil {
		return RelationRequest{}, err
	}
	// IDはDB存在確認ではなく、0でない正の値かだけ確認。
	if err := ValidateID(input.ArticleID); err != nil {
		return RelationRequest{}, err
	}
	if input.DisplayOrder < 0 || input.DisplayOrder > 10000 {
		return RelationRequest{}, entity.ErrInvalidInput
	}
	return input, nil
}

// 関連一覧・一括更新で使うbean_idが0でないかを検証。
// Beanが存在するかどうかはUsecaseで判断。
func (v *AdminRelationValidator) BeanID(id uint64) error { return ValidateID(id) }

// 関連Articleの一括差し替えRequestを検証。
// article_idsが0でないこと、重複しないこと、件数上限内であることを確認。
func (v *AdminRelationValidator) Replace(input ReplaceRelationsRequest) (ReplaceRelationsRequest, error) {
	seen := make(map[uint64]struct{}, len(input.ArticleIDs))
	for _, id := range input.ArticleIDs {
		if id == 0 {
			return ReplaceRelationsRequest{}, entity.ErrInvalidInput
		}
		if _, ok := seen[id]; ok {
			return ReplaceRelationsRequest{}, entity.ErrInvalidInput
		}
		seen[id] = struct{}{}
	}
	if len(input.ArticleIDs) > 100 {
		return ReplaceRelationsRequest{}, entity.ErrInvalidInput
	}
	return input, nil
}

// 手動バッチ実行Requestを検証。
// owner文字列と、対象user_id/guest_session_idが0でないことだけを確認。
func (v *AdminBatchValidator) Batch(input BatchRunRequest) (BatchRunRequest, error) {
	owner, err := NormalizeText(input.Owner, 1, 100)
	if err != nil {
		return BatchRunRequest{}, err
	}
	input.Owner = owner
	for _, id := range input.UserIDs {
		if id == 0 {
			return BatchRunRequest{}, entity.ErrInvalidInput
		}
	}
	for _, id := range input.GuestSessionIDs {
		if id == 0 {
			return BatchRunRequest{}, entity.ErrInvalidInput
		}
	}
	return input, nil
}

// バッチ実行履歴一覧のページングを検証。
// 実行履歴の存在や検索結果はUsecaseで判断。
func (v *AdminBatchValidator) List(input PageQuery) (PageQuery, error) {
	return NormalizePage(input, 20, 100, 10000)
}

// 取得対象のbatch job名が空でなく安全な文字列かを検証。
// jobが存在するかどうかはUsecaseで判断。
func (v *AdminBatchValidator) JobName(jobName string) (string, error) {
	name, err := NormalizeText(jobName, 1, 100)
	if err != nil {
		return "", err
	}
	return name, nil
}

// 監査ログ一覧queryをRepository filterへ変換できる形に検証。
// actor_type、actor_user_id、action、target_type、target_id、ページングを確認。
func (v *AdminAuditValidator) List(input AuditQuery) (repository.AuditLogFilter, error) {
	// limit/offsetを上限内に正規化し、大量取得によるDB負荷を防ぎます。
	page, err := NormalizePage(PageQuery{Limit: input.Limit, Offset: input.Offset}, 20, 100, 10000)
	if err != nil {
		return repository.AuditLogFilter{}, err
	}
	filter := repository.AuditLogFilter{Limit: page.Limit, Offset: page.Offset}
	if strings.TrimSpace(input.ActorType) != "" {
		actor := entity.AuditActorType(strings.TrimSpace(input.ActorType))
		switch actor {
		case entity.AuditActorUser, entity.AuditActorAdmin, entity.AuditActorSystem:
			filter.ActorType = &actor
		default:
			return repository.AuditLogFilter{}, entity.ErrInvalidInput
		}
	}
	if input.ActorUserID < 0 {
		return repository.AuditLogFilter{}, entity.ErrInvalidInput
	}
	if input.ActorUserID > 0 {
		actorUserID := uint64(input.ActorUserID)
		filter.ActorUserID = &actorUserID
	}
	if strings.TrimSpace(input.Action) != "" {
		action := entity.AuditAction(strings.TrimSpace(input.Action))
		filter.Action = &action
	}
	if strings.TrimSpace(input.TargetType) != "" {
		targetType, err := NormalizeText(input.TargetType, 1, 100)
		if err != nil {
			return repository.AuditLogFilter{}, err
		}
		filter.TargetType = &targetType
	}
	if input.TargetID < 0 {
		return repository.AuditLogFilter{}, entity.ErrInvalidInput
	}
	if input.TargetID > 0 {
		targetID := uint64(input.TargetID)
		filter.TargetID = &targetID
	}
	return filter, nil
}

// 監査ログ詳細取得用IDが0でないかを検証。
// 監査ログが存在するかどうかはUsecaseで判断。
func (v *AdminAuditValidator) ID(id uint64) error { return ValidateID(id) }

// request_id検索用文字列が空でなく安全な長さかを検証。
// 対応する監査ログがあるかはUsecaseで判断。
func (v *AdminAuditValidator) RequestID(requestID string) (string, error) {
	return NormalizeText(requestID, 1, 128)
}

// RateLimit reset対象keyが空でなく安全な文字列かを検証。
// keyの実在やreset処理はUsecase/Repositoryで行う。
func (v *AdminRateLimitValidator) Reset(input RateLimitResetRequest) (RateLimitResetRequest, error) {
	key, err := NormalizeText(input.Key, 1, 255)
	if err != nil {
		return RateLimitResetRequest{}, err
	}
	input.Key = key
	return input, nil
}
