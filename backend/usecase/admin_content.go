package usecase

import (
	"context"
	"strconv"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 管理者がBeanを作成・更新・公開・非公開にする。
// 公開/非公開では、Beanの公開状態とRankTargetの有効状態を同じTxでそろえる。
type AdminBeanUsecase struct {
	beans  repository.IBeanRepository
	audits repository.IAuditLogRepository
	tx     repository.TxManager
}

// 管理者がArticleを作成・更新・公開・非公開にするUsecase。
// 公開/非公開では、Articleの公開状態とRankTargetの有効状態を同じTxでそろえる。
type AdminArticleUsecase struct {
	articles repository.IArticleRepository
	audits   repository.IAuditLogRepository
	tx       repository.TxManager
}

// 管理者がBeanとArticleの関連を管理する。
// Bean詳細に表示する関連記事の作成・削除・一括差し替えをする。
type AdminRelationUsecase struct {
	beans     repository.IBeanRepository
	articles  repository.IArticleRepository
	relations repository.IBeanArticleRepository
	audits    repository.IAuditLogRepository
	tx        repository.TxManager
}

// 管理者操作の監査ログに使う情報。
// roleがadminかどうかはAdminGuardで確認済み。
type AdminMeta struct {
	AdminUserID uint64
	AuditMeta   AuditMeta
}

// AdminBeanUsecase。
// RankTarget更新はPublish/Unpublish内でTxRepos経由で行う。
func NewAdminBeanUsecase(
	beans repository.IBeanRepository,
	rankTargets repository.IRankTargetRepository,
	audits repository.IAuditLogRepository,
	tx repository.TxManager,
) *AdminBeanUsecase {
	return &AdminBeanUsecase{
		beans:  beans,
		audits: audits,
		tx:     tx,
	}
}

// AdminArticleUsecase。
// RankTarget更新はPublish/Unpublish内でTxRepos経由で行う。
func NewAdminArticleUsecase(
	articles repository.IArticleRepository,
	rankTargets repository.IRankTargetRepository,
	audits repository.IAuditLogRepository,
	tx repository.TxManager,
) *AdminArticleUsecase {
	return &AdminArticleUsecase{
		articles: articles,
		audits:   audits,
		tx:       tx,
	}
}

// AdminRelationUsecase。
func NewAdminRelationUsecase(
	beans repository.IBeanRepository,
	articles repository.IArticleRepository,
	relations repository.IBeanArticleRepository,
	audits repository.IAuditLogRepository,
	tx repository.TxManager,
) *AdminRelationUsecase {
	return &AdminRelationUsecase{
		beans:     beans,
		articles:  articles,
		relations: relations,
		audits:    audits,
		tx:        tx,
	}
}

// 管理者IDが取れているか確認する。
// 本当にadminかどうかはAdminGuardで確認。
func requireAdminMeta(meta AdminMeta) error {
	if meta.AdminUserID == 0 {
		return entity.ErrUnauthorized
	}
	return nil
}

// Beanを新規作成。
// 作成時点ではランキング対象にしない。公開時にRankTargetを作る。
func (u *AdminBeanUsecase) Create(ctx context.Context, bean *model.Bean, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if bean == nil {
		return entity.ErrInvalidInput
	}

	if err := u.beans.Create(ctx, bean); err != nil {
		return err
	}

	// 監査ログは補助記録。
	// 失敗してもBean作成は成功扱いにする。
	targetType := "bean"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionBeanCreate,
		&targetType,
		&bean.ID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// Beanを更新。
// 存在しないBeanを更新しないよう、先にFindByIDで確認する。
func (u *AdminBeanUsecase) Update(ctx context.Context, bean *model.Bean, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if bean == nil || bean.ID == 0 {
		return entity.ErrInvalidInput
	}

	if _, err := u.beans.FindByID(ctx, bean.ID); err != nil {
		return err
	}

	if err := u.beans.Update(ctx, bean); err != nil {
		return err
	}

	targetType := "bean"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionBeanUpdate,
		&targetType,
		&bean.ID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// Beanを公開し、ランキング対象としても有効化する。
// Beanだけ公開されてRankTargetが無効、というズレを防ぐためTxでまとめる。
func (u *AdminBeanUsecase) Publish(ctx context.Context, beanID uint64, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if beanID == 0 {
		return entity.ErrInvalidInput
	}
	if u.tx == nil {
		return entity.ErrRepositoryFailed
	}

	if err := u.tx.WithinTx(ctx, func(ctx context.Context, tx repository.ITxRepos) error {
		if err := tx.Bean().UpdatePublished(ctx, beanID, true); err != nil {
			return err
		}

		target, err := tx.RankTarget().FindOrCreate(ctx, entity.ContentTypeBean, beanID)
		if err != nil {
			return err
		}

		return tx.RankTarget().UpdateActive(ctx, target.ID, true)
	}); err != nil {
		return err
	}

	targetType := "bean"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionBeanPublish,
		&targetType,
		&beanID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// Beanを非公開にし、ランキング対象としても無効化。
// 非公開Beanがランキングや推薦に出ないようにする。
func (u *AdminBeanUsecase) Unpublish(ctx context.Context, beanID uint64, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if beanID == 0 {
		return entity.ErrInvalidInput
	}
	if u.tx == nil {
		return entity.ErrRepositoryFailed
	}

	if err := u.tx.WithinTx(ctx, func(ctx context.Context, tx repository.ITxRepos) error {
		if err := tx.Bean().UpdatePublished(ctx, beanID, false); err != nil {
			return err
		}

		target, err := tx.RankTarget().FindByContent(ctx, entity.ContentTypeBean, beanID)
		if err != nil {
			return err
		}

		return tx.RankTarget().UpdateActive(ctx, target.ID, false)
	}); err != nil {
		return err
	}

	targetType := "bean"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionBeanUnpublish,
		&targetType,
		&beanID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// Articleを新規作成。
// 作成時点ではランキング対象にしない。公開時にRankTargetを作る。
func (u *AdminArticleUsecase) Create(ctx context.Context, article *model.Article, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if article == nil {
		return entity.ErrInvalidInput
	}

	if err := u.articles.Create(ctx, article); err != nil {
		return err
	}

	targetType := "article"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionArticleCreate,
		&targetType,
		&article.ID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// Articleを更新。
// 存在しないArticleを更新しないよう、先にFindByIDで確認。
func (u *AdminArticleUsecase) Update(ctx context.Context, article *model.Article, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if article == nil || article.ID == 0 {
		return entity.ErrInvalidInput
	}

	if _, err := u.articles.FindByID(ctx, article.ID); err != nil {
		return err
	}

	if err := u.articles.Update(ctx, article); err != nil {
		return err
	}

	targetType := "article"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionArticleUpdate,
		&targetType,
		&article.ID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// Articleを公開し、ランキング対象としても有効化。
// Articleの公開状態とRankTargetの有効状態を同じTxでそろえる。
func (u *AdminArticleUsecase) Publish(ctx context.Context, articleID uint64, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if articleID == 0 {
		return entity.ErrInvalidInput
	}
	if u.tx == nil {
		return entity.ErrRepositoryFailed
	}

	if err := u.tx.WithinTx(ctx, func(ctx context.Context, tx repository.ITxRepos) error {
		if err := tx.Article().UpdatePublished(ctx, articleID, true); err != nil {
			return err
		}

		target, err := tx.RankTarget().FindOrCreate(ctx, entity.ContentTypeArticle, articleID)
		if err != nil {
			return err
		}

		return tx.RankTarget().UpdateActive(ctx, target.ID, true)
	}); err != nil {
		return err
	}

	targetType := "article"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionArticlePublish,
		&targetType,
		&articleID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// Articleを非公開にし、ランキング対象としても無効化。
// 非公開Articleがランキングや推薦に出ないようにする。
func (u *AdminArticleUsecase) Unpublish(ctx context.Context, articleID uint64, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if articleID == 0 {
		return entity.ErrInvalidInput
	}
	if u.tx == nil {
		return entity.ErrRepositoryFailed
	}

	if err := u.tx.WithinTx(ctx, func(ctx context.Context, tx repository.ITxRepos) error {
		if err := tx.Article().UpdatePublished(ctx, articleID, false); err != nil {
			return err
		}

		target, err := tx.RankTarget().FindByContent(ctx, entity.ContentTypeArticle, articleID)
		if err != nil {
			return err
		}

		return tx.RankTarget().UpdateActive(ctx, target.ID, false)
	}); err != nil {
		return err
	}

	targetType := "article"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionArticleUnpublish,
		&targetType,
		&articleID,
		nil,
		meta.AuditMeta,
	))

	return nil
}

// BeanとArticleの関連を1件作成する。
// 同じBeanとArticleの関連がすでにある場合は作成しない。
func (u *AdminRelationUsecase) Create(ctx context.Context, beanID uint64, articleID uint64, displayOrder int, meta AdminMeta) (*model.BeanArticle, error) {
	if err := requireAdminMeta(meta); err != nil {
		return nil, err
	}
	if beanID == 0 || articleID == 0 {
		return nil, entity.ErrInvalidInput
	}

	if err := ensureBeanExists(ctx, u.beans, beanID); err != nil {
		return nil, err
	}
	if err := ensureArticleExists(ctx, u.articles, articleID); err != nil {
		return nil, err
	}

	// DBのunique制約に当たる前に、Usecaseで重複を業務エラーとして止める。
	exists, err := u.relations.Exists(ctx, beanID, articleID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, entity.ErrConflict
	}

	relation := &model.BeanArticle{
		BeanID:       beanID,
		ArticleID:    articleID,
		DisplayOrder: displayOrder,
	}
	if err := u.relations.Create(ctx, relation); err != nil {
		return nil, err
	}

	targetType := "bean_article"
	detail := "create"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionBeanUpdate,
		&targetType,
		&relation.ID,
		&detail,
		meta.AuditMeta,
	))

	return relation, nil
}

// BeanとArticleの関連を1件削除。
// 存在しない関連を削除したことにしないため、先にExistsで確認。
func (u *AdminRelationUsecase) Delete(ctx context.Context, beanID uint64, articleID uint64, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if beanID == 0 || articleID == 0 {
		return entity.ErrInvalidInput
	}

	exists, err := u.relations.Exists(ctx, beanID, articleID)
	if err != nil {
		return err
	}
	if !exists {
		return entity.ErrNotFound
	}

	if err := u.relations.Delete(ctx, beanID, articleID); err != nil {
		return err
	}

	targetType := "bean_article"
	targetID := beanID
	detail := "delete article_id=" + strconv.FormatUint(articleID, 10)
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionBeanUpdate,
		&targetType,
		&targetID,
		&detail,
		meta.AuditMeta,
	))

	return nil
}

// Beanに紐づく関連記事を一括で差し替える。
// 既存関連の削除と新規関連の作成をまとめて行うためTxで処理する。
func (u *AdminRelationUsecase) ReplaceByBeanID(ctx context.Context, beanID uint64, articleIDs []uint64, meta AdminMeta) error {
	if err := requireAdminMeta(meta); err != nil {
		return err
	}
	if beanID == 0 {
		return entity.ErrInvalidInput
	}
	if u.tx == nil {
		return entity.ErrRepositoryFailed
	}

	if err := ensureBeanExists(ctx, u.beans, beanID); err != nil {
		return err
	}

	// 同じArticleIDが含まれると、ReplaceByBeanID内のCreateでunique制約に当たる。
	if hasDuplicateUint64(articleIDs) {
		return entity.ErrInvalidInput
	}

	// 空配列：関連Articleをすべて外す
	// 空でない場合だけ、Articleの存在確認を行う。
	for _, articleID := range articleIDs {
		if err := ensureArticleExists(ctx, u.articles, articleID); err != nil {
			return err
		}
	}

	if err := u.tx.WithinTx(ctx, func(ctx context.Context, tx repository.ITxRepos) error {
		return tx.BeanArticle().ReplaceByBeanID(ctx, beanID, articleIDs)
	}); err != nil {
		return err
	}

	targetType := "bean_article"
	targetID := beanID
	detail := "replace"
	safeAudit(ctx, u.audits, auditLog(
		entity.AuditActorAdmin,
		&meta.AdminUserID,
		entity.AuditActionBeanUpdate,
		&targetType,
		&targetID,
		&detail,
		meta.AuditMeta,
	))

	return nil
}

// Beanに紐づく関連記事の表示順を更新。
// 実体はReplaceByBeanIDと同じで、articleIDsの順番がdisplay_orderになる。
func (u *AdminRelationUsecase) UpdateDisplayOrder(ctx context.Context, beanID uint64, articleIDs []uint64, meta AdminMeta) error {
	return u.ReplaceByBeanID(ctx, beanID, articleIDs, meta)
}

// Beanが存在することを確認。
// 存在しないBeanにArticleを関連付けないため。
func ensureBeanExists(ctx context.Context, beans repository.IBeanRepository, beanID uint64) error {
	if beanID == 0 {
		return entity.ErrInvalidInput
	}

	exists, err := beans.ExistsByID(ctx, beanID)
	if err != nil {
		return err
	}
	if !exists {
		return entity.ErrBeanNotFound
	}

	return nil
}

// Articleが存在することを確認する。
// 存在しないArticleをBeanの関連記事として表示しないため。
func ensureArticleExists(ctx context.Context, articles repository.IArticleRepository, articleID uint64) error {
	if articleID == 0 {
		return entity.ErrInvalidInput
	}

	exists, err := articles.ExistsByID(ctx, articleID)
	if err != nil {
		return err
	}
	if !exists {
		return entity.ErrArticleNotFound
	}

	return nil
}

// uint64の重複を確認する。
// 同じArticleを同じBeanに2回関連付けるとDB制約に当たるため、事前に止める。
func hasDuplicateUint64(values []uint64) bool {
	seen := make(map[uint64]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
