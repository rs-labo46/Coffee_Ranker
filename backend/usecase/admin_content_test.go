package usecase

import (
	"context"
	"testing"

	"coffee-ranker/entity"
	"coffee-ranker/model"
)

// BeanとArticleの関連作成で、既存関連がある場合は重複作成せずErrConflictを返すことを確認。
func TestAdminRelationUsecaseCreate_RejectsDuplicateRelation(t *testing.T) {
	ctx := context.Background()
	beans := &fakeBeanRepo{exists: true}
	articles := &fakeArticleRepo{exists: true}
	relations := &fakeBeanArticleRepo{exists: true}
	u := NewAdminRelationUsecase(beans, articles, relations, &fakeAuditRepo{}, nil)

	_, err := u.Create(ctx, 1, 2, 0, AdminMeta{AdminUserID: 10})
	assertErrorIs(t, err, entity.ErrConflict)
	if relations.created != nil {
		t.Fatal("relation was created, want duplicate to stop before Create")
	}
}

// BeanとArticleの関連削除で、削除対象の関連が存在しない場合はDeleteを呼ばずErrNotFoundを返すことを確認。
func TestAdminRelationUsecaseDelete_RejectsMissingRelation(t *testing.T) {
	ctx := context.Background()
	relations := &fakeBeanArticleRepo{exists: false}
	u := NewAdminRelationUsecase(&fakeBeanRepo{}, &fakeArticleRepo{}, relations, &fakeAuditRepo{}, nil)

	err := u.Delete(ctx, 1, 2, AdminMeta{AdminUserID: 10})
	assertErrorIs(t, err, entity.ErrNotFound)
	if relations.deleteBeanID != 0 {
		t.Fatal("relation was deleted, want missing relation to stop before Delete")
	}
}

// 関連Articleの一括差し替えで、同じarticleIDが重複している場合はTx開始前にErrInvalidInputで止めることを確認。
func TestAdminRelationUsecaseReplaceByBeanID_RejectsDuplicateArticleIDs(t *testing.T) {
	ctx := context.Background()
	beans := &fakeBeanRepo{exists: true}
	articles := &fakeArticleRepo{exists: true}
	relations := &fakeBeanArticleRepo{}
	tx := &fakeTxManager{repos: fakeTxRepos{relation: relations}}
	u := NewAdminRelationUsecase(beans, articles, relations, &fakeAuditRepo{}, tx)

	err := u.ReplaceByBeanID(ctx, 1, []uint64{2, 2}, AdminMeta{AdminUserID: 10})
	assertErrorIs(t, err, entity.ErrInvalidInput)
	if tx.called {
		t.Fatal("tx called, want duplicate article IDs to stop before transaction")
	}
}

// Bean公開処理で、Beanの公開状態更新とRankTarget有効化が同じTx内で行われ、監査ログも残ることを確認。
func TestAdminBeanUsecasePublish_UpdatesBeanAndRankTargetInTx(t *testing.T) {
	ctx := context.Background()
	beans := &fakeBeanRepo{}
	rankTargets := &fakeRankTargetRepo{findOrCreateTarget: &model.RankTarget{ID: 77, ContentType: entity.ContentTypeBean, ContentID: 5}}
	audits := &fakeAuditRepo{}
	tx := &fakeTxManager{repos: fakeTxRepos{bean: beans, rank: rankTargets}}
	u := NewAdminBeanUsecase(beans, rankTargets, audits, tx)

	err := u.Publish(ctx, 5, AdminMeta{AdminUserID: 10})
	assertNoError(t, err)
	if !tx.called {
		t.Fatal("tx was not called")
	}
	if beans.updatePublishedID != 5 || !beans.updatePublishedValue {
		t.Fatalf("bean publish = id %d active %v, want 5 true", beans.updatePublishedID, beans.updatePublishedValue)
	}
	if rankTargets.updatedID != 77 || !rankTargets.updatedActive {
		t.Fatalf("rank target active = id %d active %v, want 77 true", rankTargets.updatedID, rankTargets.updatedActive)
	}
	if len(audits.created) != 1 || audits.created[0].Action != entity.AuditActionBeanPublish {
		t.Fatalf("audit not written correctly: %+v", audits.created)
	}
}
