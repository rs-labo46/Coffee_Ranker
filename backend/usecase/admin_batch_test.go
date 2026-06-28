package usecase

import (
	"context"
	"testing"

	"coffee-ranker/entity"
)

// バッチ実行履歴一覧で、ページング未指定時にlimit=20/offset=0へ補正されることを確認。
func TestAdminBatchUsecaseListRuns_DefaultsPagination(t *testing.T) {
	ctx := context.Background()
	runs := &fakeBatchRunRepo{}
	u := NewAdminBatchUsecase(runs, nil, nil)

	_, err := u.ListRuns(ctx, Page{})
	assertNoError(t, err)

	if runs.listLimit != 20 || runs.listOffset != 0 {
		t.Fatalf("pagination = limit %d offset %d, want 20/0", runs.listLimit, runs.listOffset)
	}
}

// RankingBatchUsecaseのDI漏れがある場合、panicではなくErrRepositoryFailedで止めることを確認。
func TestAdminBatchUsecaseRunRanking_RejectsMissingBatchDependency(t *testing.T) {
	ctx := context.Background()
	u := NewAdminBatchUsecase(&fakeBatchRunRepo{}, nil, nil)

	_, err := u.RunRanking(ctx, 1, "owner", AuditMeta{})
	assertErrorIs(t, err, entity.ErrRepositoryFailed)
}
