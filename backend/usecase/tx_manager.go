package usecase

import (
	"context"

	"coffee-ranker/repository"
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
