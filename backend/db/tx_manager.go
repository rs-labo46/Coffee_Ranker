package db

import (
	"context"

	"coffee-ranker/repository"
	"coffee-ranker/usecase"

	"gorm.io/gorm"
)

// GORMのtransaction開始、commit、rollbackを担当する。
type GormTxManager struct {
	db *gorm.DB
}

// transaction用DBに差し替えたRepository群。
type gormTxRepos struct {
	user          repository.IUserRepository
	refreshToken  repository.IRefreshTokenRepository
	bean          repository.IBeanRepository
	article       repository.IArticleRepository
	rankTarget    repository.IRankTargetRepository
	beanArticle   repository.IBeanArticleRepository
	contentMetric repository.IContentMetricRepository
	batchRun      repository.IBatchRunRepository
}

// TxManagerを作成する。
func NewTxManager(database *gorm.DB) usecase.TxManager {
	return &GormTxManager{db: database}
}

// Usecaseから渡された処理をGORM transaction内で実行する。
// fnがerrorを返した場合はrollbackし、nilを返した場合のみcommitする。
func (m *GormTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, tx usecase.ITxRepos) error) error {
	return m.db.WithContext(ctx).Transaction(func(txDB *gorm.DB) error {
		txRepos := &gormTxRepos{
			user:          repository.NewUserRepository(txDB),
			refreshToken:  repository.NewRefreshTokenRepository(txDB),
			bean:          repository.NewBeanRepository(txDB),
			article:       repository.NewArticleRepository(txDB),
			rankTarget:    repository.NewRankTargetRepository(txDB),
			beanArticle:   repository.NewBeanArticleRepository(txDB),
			contentMetric: repository.NewContentMetricRepository(txDB),
			batchRun:      repository.NewBatchRunRepository(txDB),
		}

		return fn(ctx, txRepos)
	})
}

func (r *gormTxRepos) User() repository.IUserRepository {
	return r.user
}

func (r *gormTxRepos) RefreshToken() repository.IRefreshTokenRepository {
	return r.refreshToken
}

func (r *gormTxRepos) Bean() repository.IBeanRepository {
	return r.bean
}

func (r *gormTxRepos) Article() repository.IArticleRepository {
	return r.article
}

func (r *gormTxRepos) RankTarget() repository.IRankTargetRepository {
	return r.rankTarget
}

func (r *gormTxRepos) BeanArticle() repository.IBeanArticleRepository {
	return r.beanArticle
}

func (r *gormTxRepos) ContentMetric() repository.IContentMetricRepository {
	return r.contentMetric
}

func (r *gormTxRepos) BatchRun() repository.IBatchRunRepository {
	return r.batchRun
}
