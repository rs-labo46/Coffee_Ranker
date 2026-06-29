package repository

import (
	"context"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
)

// Beanの作成、更新、公開取得、検索に必要なDB操作。
type IBeanRepository interface {
	Create(ctx context.Context, bean *model.Bean) error
	Update(ctx context.Context, bean *model.Bean) error
	FindByID(ctx context.Context, id uint64) (*model.Bean, error)
	FindPublishedByID(ctx context.Context, id uint64) (*model.Bean, error)
	ListPublished(ctx context.Context, limit int, offset int) ([]*model.Bean, error)
	SearchPublished(ctx context.Context, filter BeanSearchFilter) ([]*model.Bean, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]*model.Bean, error)
	ExistsByID(ctx context.Context, id uint64) (bool, error)
	ExistsPublishedByID(ctx context.Context, id uint64) (bool, error)
	UpdatePublished(ctx context.Context, id uint64, isPublished bool) error
}

// Articleの作成、更新、公開取得、検索に必要なDB操作。
type IArticleRepository interface {
	Create(ctx context.Context, article *model.Article) error
	Update(ctx context.Context, article *model.Article) error
	FindByID(ctx context.Context, id uint64) (*model.Article, error)
	FindPublishedByID(ctx context.Context, id uint64) (*model.Article, error)
	FindBySlug(ctx context.Context, slug string) (*model.Article, error)
	FindPublishedBySlug(ctx context.Context, slug string) (*model.Article, error)
	ListPublished(ctx context.Context, limit int, offset int) ([]*model.Article, error)
	SearchPublished(ctx context.Context, filter ArticleSearchFilter) ([]*model.Article, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]*model.Article, error)
	ExistsByID(ctx context.Context, id uint64) (bool, error)
	ExistsPublishedByID(ctx context.Context, id uint64) (bool, error)
	UpdatePublished(ctx context.Context, id uint64, isPublished bool) error
}

// バッチ実行履歴の作成、成功更新、失敗更新、取得。
type IBatchRunRepository interface {
	Create(ctx context.Context, run *model.BatchRun) error
	MarkSuccess(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64) error
	MarkFailed(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64, errorMessage string) error
	FindLatestByJobName(ctx context.Context, jobName string) (*model.BatchRun, error)
	FindRunningByJobName(ctx context.Context, jobName string) (*model.BatchRun, error)
	List(ctx context.Context, limit int, offset int) ([]*model.BatchRun, error)
}

// 認証、管理者操作、バッチ操作の監査ログを扱うDB操作。
type IAuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
	FindByID(ctx context.Context, id uint64) (*model.AuditLog, error)
	List(ctx context.Context, filter AuditLogFilter) ([]*model.AuditLog, error)
	ListByActor(ctx context.Context, actorType entity.AuditActorType, actorUserID *uint64, limit int) ([]*model.AuditLog, error)
	ListByTarget(ctx context.Context, targetType string, targetID uint64, limit int) ([]*model.AuditLog, error)
	ListByRequestID(ctx context.Context, requestID string) ([]*model.AuditLog, error)
}

// transaction内で使うRepository群をUsecaseへ渡すための窓口。
type ITxRepos interface {
	User() IUserRepository
	RefreshToken() IRefreshTokenRepository
	Bean() IBeanRepository
	Article() IArticleRepository
	RankTarget() IRankTargetRepository
	BeanArticle() IBeanArticleRepository
	ContentMetric() IContentMetricRepository
	BatchRun() IBatchRunRepository
}

// 監査ログ一覧を絞り込むための検索条件。
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

// 複数DB更新を1つのtransactionとして実行するためのinterface。
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx ITxRepos) error) error
}

type GormBatchRunRepository struct {
	baseRepo
}

type GormAuditLogRepository struct {
	baseRepo
}

// GORMのtransaction開始、commit、rollback。
type GormTxManager struct {
	db *gorm.DB
}

// transaction用DBに差し替えたRepositoryをまとめる構造体。
type gormTxRepos struct {
	user          IUserRepository
	refreshToken  IRefreshTokenRepository
	bean          IBeanRepository
	article       IArticleRepository
	rankTarget    IRankTargetRepository
	beanArticle   IBeanArticleRepository
	contentMetric IContentMetricRepository
	batchRun      IBatchRunRepository
}

// バッチ実行履歴Repository
func NewBatchRunRepository(db *gorm.DB) IBatchRunRepository {
	return &GormBatchRunRepository{baseRepo{db}}
}

// 監査ログRepository
func NewIAuditLogRepository(db *gorm.DB) IAuditLogRepository {
	return &GormAuditLogRepository{baseRepo{db}}
}

// GORM transaction管理
func NewTxManager(db *gorm.DB) TxManager {
	return &GormTxManager{db}
}

// バッチ実行履歴を開始状態で作成。
func (r *GormBatchRunRepository) Create(ctx context.Context, run *model.BatchRun) error {
	return mapDBError(r.db.WithContext(ctx).Create(run).Error)
}

// バッチ実行履歴を成功状態に更新。
func (r *GormBatchRunRepository) MarkSuccess(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64) error {
	updates := model.BatchRun{Status: entity.BatchStatusSuccess, FinishedAt: &finishedAt, RowsProcessed: rowsProcessed}
	res := r.db.WithContext(ctx).Model(&model.BatchRun{}).Where("id = ?", id).Select("status", "finished_at", "rows_processed", "error_message").Updates(&updates)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// バッチ実行履歴を失敗状態に更新し、エラー内容を保存。
func (r *GormBatchRunRepository) MarkFailed(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64, errorMessage string) error {
	updates := model.BatchRun{Status: entity.BatchStatusFailed, FinishedAt: &finishedAt, RowsProcessed: rowsProcessed, ErrorMessage: &errorMessage}
	res := r.db.WithContext(ctx).Model(&model.BatchRun{}).Where("id = ?", id).Select("status", "finished_at", "rows_processed", "error_message").Updates(&updates)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 最新バッチ実行履歴を取得。
func (r *GormBatchRunRepository) FindLatestByJobName(ctx context.Context, jobName string) (*model.BatchRun, error) {
	var run model.BatchRun
	if err := r.db.WithContext(ctx).Where("job_name = ?", jobName).Order("started_at DESC").First(&run).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &run, nil
}

// 実行中のバッチ履歴を取得。
func (r *GormBatchRunRepository) FindRunningByJobName(ctx context.Context, jobName string) (*model.BatchRun, error) {
	var run model.BatchRun
	if err := r.db.WithContext(ctx).Where("job_name = ? AND status = ?", jobName, entity.BatchStatusRunning).Order("started_at DESC").First(&run).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &run, nil
}

// バッチ実行履歴を新しい順に一覧で取得。
func (r *GormBatchRunRepository) List(ctx context.Context, limit int, offset int) ([]*model.BatchRun, error) {
	var runs []*model.BatchRun
	err := r.db.WithContext(ctx).Order("started_at DESC").Limit(limit).Offset(offset).Find(&runs).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return runs, nil
}

// 監査ログを1件保存。
func (r *GormAuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return mapDBError(r.db.WithContext(ctx).Create(log).Error)
}

// 監査ログをIDで取得。
func (r *GormAuditLogRepository) FindByID(ctx context.Context, id uint64) (*model.AuditLog, error) {
	var log model.AuditLog
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&log).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &log, nil
}

// 条件に合う監査ログを新しい順に取得。
func (r *GormAuditLogRepository) List(ctx context.Context, filter AuditLogFilter) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	db := r.db.WithContext(ctx).Model(&model.AuditLog{})
	if filter.ActorType != nil {
		db = db.Where("actor_type = ?", *filter.ActorType)
	}
	if filter.ActorUserID != nil {
		db = db.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.Action != nil {
		db = db.Where("action = ?", *filter.Action)
	}
	if filter.TargetType != nil {
		db = db.Where("target_type = ?", *filter.TargetType)
	}
	if filter.TargetID != nil {
		db = db.Where("target_id = ?", *filter.TargetID)
	}
	if filter.From != nil {
		db = db.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		db = db.Where("created_at < ?", *filter.To)
	}
	if err := db.Order("created_at DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&logs).Error; err != nil {
		return nil, mapDBError(err)
	}
	return logs, nil
}

// 操作者種別と操作者IDで監査ログを取得。
func (r *GormAuditLogRepository) ListByActor(ctx context.Context, actorType entity.AuditActorType, actorUserID *uint64, limit int) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	db := r.db.WithContext(ctx).Where("actor_type = ?", actorType)
	if actorUserID != nil {
		db = db.Where("actor_user_id = ?", *actorUserID)
	} else {
		db = db.Where("actor_user_id IS NULL")
	}
	if err := db.Order("created_at DESC").Limit(limit).Find(&logs).Error; err != nil {
		return nil, mapDBError(err)
	}
	return logs, nil
}

// 操作対象の種類とIDで監査ログを取得。
func (r *GormAuditLogRepository) ListByTarget(ctx context.Context, targetType string, targetID uint64, limit int) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	err := r.db.WithContext(ctx).Where("target_type = ? AND target_id = ?", targetType, targetID).Order("created_at DESC").Limit(limit).Find(&logs).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return logs, nil
}

// 同じrequest_idに紐づく監査ログを取得。
func (r *GormAuditLogRepository) ListByRequestID(ctx context.Context, requestID string) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	err := r.db.WithContext(ctx).Where("request_id = ?", requestID).Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return logs, nil
}

// Usecaseから渡された処理をGORM transaction内で実行。
func (m *GormTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, tx ITxRepos) error) error {
	return m.db.WithContext(ctx).Transaction(func(txDB *gorm.DB) error {
		txRepos := &gormTxRepos{
			user:          NewUserRepository(txDB),
			refreshToken:  NewRefreshTokenRepository(txDB),
			bean:          NewBeanRepository(txDB),
			article:       NewArticleRepository(txDB),
			rankTarget:    NewRankTargetRepository(txDB),
			beanArticle:   NewBeanArticleRepository(txDB),
			contentMetric: NewContentMetricRepository(txDB),
			batchRun:      NewBatchRunRepository(txDB),
		}
		return fn(ctx, txRepos)
	})
}

func (r *gormTxRepos) User() IUserRepository {
	return r.user
}

func (r *gormTxRepos) RefreshToken() IRefreshTokenRepository {
	return r.refreshToken
}

func (r *gormTxRepos) Bean() IBeanRepository {
	return r.bean
}

func (r *gormTxRepos) Article() IArticleRepository {
	return r.article
}

func (r *gormTxRepos) RankTarget() IRankTargetRepository {
	return r.rankTarget
}

func (r *gormTxRepos) BeanArticle() IBeanArticleRepository {
	return r.beanArticle
}
func (r *gormTxRepos) ContentMetric() IContentMetricRepository {
	return r.contentMetric
}

func (r *gormTxRepos) BatchRun() IBatchRunRepository {
	return r.batchRun
}
