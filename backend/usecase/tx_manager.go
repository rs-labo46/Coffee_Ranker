package usecase

import (
	"context"

	"coffee-ranker/entity"
	"coffee-ranker/repository"

	"gorm.io/gorm"
)

// TxReposは、transaction内で使うRepository群をUsecaseへ渡すための窓口。
// Usecaseはこのinterfaceを通して、transaction用DBに差し替えられたRepositoryを使う。
type ITxRepos interface {
	User() repository.IUserRepository
	RefreshToken() repository.IRefreshTokenRepository
	Bean() repository.IBeanRepository
	Article() repository.IArticleRepository
	RankTarget() repository.IRankTargetRepository
	BeanArticle() repository.IBeanArticleRepository
	ContentMetric() repository.IContentMetricRepository
	BatchRun() repository.IBatchRunRepository
}

// TxManagerは、Usecaseが複数DB更新を1つのtransactionとして実行するためのinterface。
// transactionを使うかどうかはUsecaseが判断する。
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context, tx ITxRepos) error) error
}

type GormTxManager struct {
	db *gorm.DB
}

type gormTxRepos struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *GormTxManager {
	return &GormTxManager{db: db}
}

func (m *GormTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, tx ITxRepos) error) error {
	if m == nil || m.db == nil || fn == nil {
		return entity.ErrTransactionFailed
	}

	var fnErr error

	err := m.db.WithContext(ctx).Transaction(func(txDB *gorm.DB) error {
		fnErr = fn(ctx, &gormTxRepos{db: txDB})
		return fnErr
	})

	if err != nil {
		if fnErr != nil {
			return fnErr
		}
		return entity.ErrTransactionFailed
	}

	return nil
}

func (r *gormTxRepos) User() repository.IUserRepository {
	return repository.NewUserRepository(r.db)
}

func (r *gormTxRepos) RefreshToken() repository.IRefreshTokenRepository {
	return repository.NewRefreshTokenRepository(r.db)
}

func (r *gormTxRepos) Bean() repository.IBeanRepository {
	return repository.NewBeanRepository(r.db)
}

func (r *gormTxRepos) Article() repository.IArticleRepository {
	return repository.NewArticleRepository(r.db)
}

func (r *gormTxRepos) RankTarget() repository.IRankTargetRepository {
	return repository.NewRankTargetRepository(r.db)
}

func (r *gormTxRepos) BeanArticle() repository.IBeanArticleRepository {
	return repository.NewBeanArticleRepository(r.db)
}

func (r *gormTxRepos) ContentMetric() repository.IContentMetricRepository {
	return repository.NewContentMetricRepository(r.db)
}

func (r *gormTxRepos) BatchRun() repository.IBatchRunRepository {
	return repository.NewBatchRunRepository(r.db)
}
