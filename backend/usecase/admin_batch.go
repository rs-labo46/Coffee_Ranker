package usecase

import (
	"context"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 管理者がバッチを手動実行し、実行履歴を確認する。
// 実際の集計処理はRankingBatchUsecase / InterestBatchUsecaseへ任せる。
type AdminBatchUsecase struct {
	runs     repository.IBatchRunRepository
	ranking  *RankingBatchUsecase
	interest *InterestBatchUsecase
}

// AdminBatchUsecase。
// runsは履歴取得に使い、ranking/interestは手動実行の本体処理に使う。
func NewAdminBatchUsecase(
	runs repository.IBatchRunRepository,
	ranking *RankingBatchUsecase,
	interest *InterestBatchUsecase,
) *AdminBatchUsecase {
	return &AdminBatchUsecase{
		runs:     runs,
		ranking:  ranking,
		interest: interest,
	}
}

// 管理者操作としてランキング再計算バッチを実行する。
// 管理者判定そのものはAdminGuardなどのMiddleware/Controller側で済ませる。
func (u *AdminBatchUsecase) RunRanking(ctx context.Context, adminUserID uint64, owner string, meta AuditMeta) (*model.BatchRun, error) {
	// 認証済み管理者IDが取れていない場合は拒否する。
	if adminUserID == 0 {
		return nil, entity.ErrUnauthorized
	}

	// ranking batchが未設定だとpanicするため、明示的に止める。
	if u.ranking == nil {
		return nil, entity.ErrRepositoryFailed
	}

	// AdminBatchUsecaseでは集計処理を書かない。
	// 実際のランキング再計算はRankingBatchUsecaseへ委譲する。
	return u.ranking.Recalculate(ctx, BatchInput{
		JobName:         "ranking",
		Owner:           owner,
		TriggeredBy:     entity.AuditActorAdmin,
		TriggeredUserID: &adminUserID,
		Meta:            meta,
	})
}

// 管理者操作として興味プロフィール再計算バッチを実行。
// UserIDs / GuestSessionIDsで指定された対象だけを再計算する。
func (u *AdminBatchUsecase) RunInterest(ctx context.Context, adminUserID uint64, owner string, userIDs []uint64, guestSessionIDs []uint64, meta AuditMeta) (*model.BatchRun, error) {
	// 認証済み管理者IDが取れていない場合は拒否する。
	if adminUserID == 0 {
		return nil, entity.ErrUnauthorized
	}

	// interest batchが未設定だとpanicするため、明示的に止める。
	if u.interest == nil {
		return nil, entity.ErrRepositoryFailed
	}

	// AdminBatchUsecaseでは興味集計ロジックを書かない。
	// 対象User/Guestの集計はInterestBatchUsecaseへ委譲する。
	return u.interest.Recalculate(ctx, InterestBatchInput{
		BatchInput: BatchInput{
			JobName:         "interest",
			Owner:           owner,
			TriggeredBy:     entity.AuditActorAdmin,
			TriggeredUserID: &adminUserID,
			Meta:            meta,
		},
		UserIDs:         userIDs,
		GuestSessionIDs: guestSessionIDs,
	})
}

// バッチ実行履歴をページング付きで取得する。
// Admin画面で「いつ、どのバッチが、成功/失敗したか」を表示するために使う。
func (u *AdminBatchUsecase) ListRuns(ctx context.Context, page Page) ([]*model.BatchRun, error) {
	// 取得件数が大きすぎるとDB負荷が上がるため、共通ページングルールで制限する。
	page, err := normalizePage(page, 20, 100, 10000)
	if err != nil {
		return nil, err
	}

	return u.runs.List(ctx, page.Limit, page.Offset)
}

// job名に一致する最新のバッチ実行履歴を取得。
func (u *AdminBatchUsecase) Latest(ctx context.Context, jobName string) (*model.BatchRun, error) {
	// jobNameが空だと、どのバッチ履歴を取るのか特定できない。
	if jobName == "" {
		return nil, entity.ErrInvalidInput
	}

	return u.runs.FindLatestByJobName(ctx, jobName)
}
