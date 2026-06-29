package repository

import (
	"context"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BeanとArticleの関連付け、削除、一覧取得、一括差し替え。
type IBeanArticleRepository interface {
	Create(ctx context.Context, relation *model.BeanArticle) error
	Delete(ctx context.Context, beanID uint64, articleID uint64) error
	Exists(ctx context.Context, beanID uint64, articleID uint64) (bool, error)
	ListByBeanID(ctx context.Context, beanID uint64, limit int) ([]*model.BeanArticle, error)
	ListByArticleID(ctx context.Context, articleID uint64, limit int) ([]*model.BeanArticle, error)
	ReplaceByBeanID(ctx context.Context, beanID uint64, articleIDs []uint64) error
}

// Bean/Articleをランキング対象として共通管理するDB操作。
type IRankTargetRepository interface {
	Create(ctx context.Context, target *model.RankTarget) error
	FindByID(ctx context.Context, id uint64) (*model.RankTarget, error)
	FindByContent(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error)
	FindOrCreate(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]*model.RankTarget, error)
	ListActiveByType(ctx context.Context, contentType entity.ContentType) ([]*model.RankTarget, error)
	ExistsActiveByID(ctx context.Context, id uint64) (bool, error)
	UpdateActive(ctx context.Context, id uint64, isActive bool) error
}

type GormBeanRepository struct {
	baseRepo
}

type GormArticleRepository struct {
	baseRepo
}

type GormBeanArticleRepository struct {
	baseRepo
}

type GormRankTargetRepository struct {
	baseRepo
}

// Bean検索で使う絞り込み条件、並び順、ページング条件。
type BeanSearchFilter struct {
	Keyword    *string
	Origin     *string
	RoastLevel *entity.RoastLevel
	Acidity    *int
	Bitterness *int
	Flavor     *int
	Aroma      *int
	Body       *int
	Sort       string
	Limit      int
	Offset     int
}

// Article検索で使う。
type ArticleSearchFilter struct {
	Keyword  *string
	Category *string
	Sort     string
	Limit    int
	Offset   int
}

func NewBeanRepository(db *gorm.DB) IBeanRepository {
	return &GormBeanRepository{baseRepo{db}}
}

func NewArticleRepository(db *gorm.DB) IArticleRepository {
	return &GormArticleRepository{baseRepo{db}}
}

func NewBeanArticleRepository(db *gorm.DB) IBeanArticleRepository {
	return &GormBeanArticleRepository{baseRepo{db}}
}
func NewRankTargetRepository(db *gorm.DB) IRankTargetRepository {
	return &GormRankTargetRepository{baseRepo{db}}
}

// Beanを新規作成。
func (r *GormBeanRepository) Create(ctx context.Context, bean *model.Bean) error {
	return mapDBError(r.db.WithContext(ctx).Create(bean).Error)
}

// Beanの編集内容を保存。
func (r *GormBeanRepository) Update(ctx context.Context, bean *model.Bean) error {
	res := r.db.WithContext(ctx).Where("id = ?", bean.ID).Updates(bean)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 公開状態に関係なくBeanをIDで取得。
func (r *GormBeanRepository) FindByID(ctx context.Context, id uint64) (*model.Bean, error) {
	var bean model.Bean
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&bean).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &bean, nil
}

// 公開中のBeanだけをIDで取得。
func (r *GormBeanRepository) FindPublishedByID(ctx context.Context, id uint64) (*model.Bean, error) {
	var bean model.Bean
	if err := r.db.WithContext(ctx).Where("id = ? AND is_published = true", id).First(&bean).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &bean, nil
}

// 公開中Beanを新しい順に一覧取得。
func (r *GormBeanRepository) ListPublished(ctx context.Context, limit int, offset int) ([]*model.Bean, error) {
	var beans []*model.Bean
	err := r.db.WithContext(ctx).Where("is_published = true").Order("id DESC").Limit(limit).Offset(offset).Find(&beans).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return beans, nil
}

// 公開中Beanを検索条件と並び順で取得。
func (r *GormBeanRepository) SearchPublished(ctx context.Context, filter BeanSearchFilter) ([]*model.Bean, error) {
	var beans []*model.Bean
	db := r.db.WithContext(ctx).
		Model(&model.Bean{}).
		Select("beans.*").
		Joins("LEFT JOIN rank_targets ON rank_targets.content_type = ? AND rank_targets.content_id = beans.id", entity.ContentTypeBean).
		Joins("LEFT JOIN content_metrics ON content_metrics.rank_target_id = rank_targets.id").
		Where("beans.is_published = true")
	if filter.Keyword != nil {
		like := "%" + *filter.Keyword + "%"
		db = db.Where("beans.name ILIKE ? OR beans.flavor_note ILIKE ? OR beans.description ILIKE ?", like, like, like)
	}
	if filter.Origin != nil {
		db = db.Where("beans.origin = ?", *filter.Origin)
	}
	if filter.RoastLevel != nil {
		db = db.Where("beans.roast_level = ?", *filter.RoastLevel)
	}
	if filter.Acidity != nil {
		db = db.Where("beans.acidity = ?", *filter.Acidity)
	}
	if filter.Bitterness != nil {
		db = db.Where("beans.bitterness = ?", *filter.Bitterness)
	}
	if filter.Flavor != nil {
		db = db.Where("beans.flavor = ?", *filter.Flavor)
	}
	if filter.Aroma != nil {
		db = db.Where("beans.aroma = ?", *filter.Aroma)
	}
	if filter.Body != nil {
		db = db.Where("beans.body = ?", *filter.Body)
	}
	db = applyMetricSort(db, filter.Sort, "beans.id").Limit(filter.Limit).Offset(filter.Offset)
	if err := db.Find(&beans).Error; err != nil {
		return nil, mapDBError(err)
	}
	return beans, nil
}

// 複数IDに一致するBeanを取得。
func (r *GormBeanRepository) FindByIDs(ctx context.Context, ids []uint64) ([]*model.Bean, error) {
	var beans []*model.Bean
	if len(ids) == 0 {
		return beans, nil
	}
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&beans).Error; err != nil {
		return nil, mapDBError(err)
	}
	return beans, nil
}

// Beanが存在するか確認。
func (r *GormBeanRepository) ExistsByID(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Bean{}).Where("id = ?", id).Count(&count).Error
	return existsFromCount(count, err)
}

// 公開中Beanとして存在するか確認。
func (r *GormBeanRepository) ExistsPublishedByID(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Bean{}).Where("id = ? AND is_published = true", id).Count(&count).Error
	return existsFromCount(count, err)
}

// Beanの公開状態を更新。
func (r *GormBeanRepository) UpdatePublished(ctx context.Context, id uint64, isPublished bool) error {
	res := r.db.WithContext(ctx).Model(&model.Bean{}).Where("id = ?", id).Update("is_published", isPublished)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// Articleを新規作成。
func (r *GormArticleRepository) Create(ctx context.Context, article *model.Article) error {
	return mapDBError(r.db.WithContext(ctx).Create(article).Error)
}

// Articleの編集内容を保存。
func (r *GormArticleRepository) Update(ctx context.Context, article *model.Article) error {
	res := r.db.WithContext(ctx).Where("id = ?", article.ID).Updates(article)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 公開状態に関係なくArticleをIDで取得。
func (r *GormArticleRepository) FindByID(ctx context.Context, id uint64) (*model.Article, error) {
	var article model.Article
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&article).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &article, nil
}

// 公開中のArticleだけをIDで取得。
func (r *GormArticleRepository) FindPublishedByID(ctx context.Context, id uint64) (*model.Article, error) {
	var article model.Article
	if err := r.db.WithContext(ctx).Where("id = ? AND is_published = true", id).First(&article).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &article, nil
}

// slugに一致するArticleを取得。
func (r *GormArticleRepository) FindBySlug(ctx context.Context, slug string) (*model.Article, error) {
	var article model.Article
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&article).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &article, nil
}

// 公開中Articleだけをslugで取得。
func (r *GormArticleRepository) FindPublishedBySlug(ctx context.Context, slug string) (*model.Article, error) {
	var article model.Article
	if err := r.db.WithContext(ctx).Where("slug = ? AND is_published = true", slug).First(&article).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &article, nil
}

// 公開中Articleを公開日時順に一覧取得。
func (r *GormArticleRepository) ListPublished(ctx context.Context, limit int, offset int) ([]*model.Article, error) {
	var articles []*model.Article
	err := r.db.WithContext(ctx).Where("is_published = true").Order("published_at DESC NULLS LAST").Order("id DESC").Limit(limit).Offset(offset).Find(&articles).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return articles, nil
}

// 公開中Articleを検索条件と並び順で取得。
func (r *GormArticleRepository) SearchPublished(ctx context.Context, filter ArticleSearchFilter) ([]*model.Article, error) {
	var articles []*model.Article
	db := r.db.WithContext(ctx).
		Model(&model.Article{}).
		Select("articles.*").
		Joins("LEFT JOIN rank_targets ON rank_targets.content_type = ? AND rank_targets.content_id = articles.id", entity.ContentTypeArticle).
		Joins("LEFT JOIN content_metrics ON content_metrics.rank_target_id = rank_targets.id").
		Where("articles.is_published = true")
	if filter.Keyword != nil {
		like := "%" + *filter.Keyword + "%"
		db = db.Where("articles.title ILIKE ? OR articles.summary ILIKE ? OR articles.body ILIKE ?", like, like, like)
	}
	if filter.Category != nil {
		db = db.Where("articles.category = ?", *filter.Category)
	}
	if filter.Sort == "newest" {
		db = db.Order("articles.published_at DESC NULLS LAST").Order("articles.id DESC")
	} else {
		db = applyMetricSort(db, filter.Sort, "articles.id")
	}
	if err := db.Limit(filter.Limit).Offset(filter.Offset).Find(&articles).Error; err != nil {
		return nil, mapDBError(err)
	}
	return articles, nil
}

// 複数IDに一致するArticleを取得。
func (r *GormArticleRepository) FindByIDs(ctx context.Context, ids []uint64) ([]*model.Article, error) {
	var articles []*model.Article
	if len(ids) == 0 {
		return articles, nil
	}
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&articles).Error; err != nil {
		return nil, mapDBError(err)
	}
	return articles, nil
}

// Articleが存在するか確認。
func (r *GormArticleRepository) ExistsByID(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Article{}).Where("id = ?", id).Count(&count).Error
	return existsFromCount(count, err)
}

// 公開中Articleとして存在するか確認。
func (r *GormArticleRepository) ExistsPublishedByID(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Article{}).Where("id = ? AND is_published = true", id).Count(&count).Error
	return existsFromCount(count, err)
}

// Articleの公開状態を更新。
func (r *GormArticleRepository) UpdatePublished(ctx context.Context, id uint64, isPublished bool) error {
	res := r.db.WithContext(ctx).Model(&model.Article{}).Where("id = ?", id).Update("is_published", isPublished)
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// BeanとArticleの関連を新規作成。
func (r *GormBeanArticleRepository) Create(ctx context.Context, relation *model.BeanArticle) error {
	return mapDBError(r.db.WithContext(ctx).Create(relation).Error)
}

// 指定BeanとArticleの関連を削除。
func (r *GormBeanArticleRepository) Delete(ctx context.Context, beanID uint64, articleID uint64) error {
	res := r.db.WithContext(ctx).Where("bean_id = ? AND article_id = ?", beanID, articleID).Delete(&model.BeanArticle{})
	return mapRowsAffected(res.RowsAffected, res.Error)
}

// 指定BeanとArticleの関連が存在するか確認。
func (r *GormBeanArticleRepository) Exists(ctx context.Context, beanID uint64, articleID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.BeanArticle{}).Where("bean_id = ? AND article_id = ?", beanID, articleID).Count(&count).Error
	return existsFromCount(count, err)
}

// Beanに紐づくArticle関連を表示順で取得。
func (r *GormBeanArticleRepository) ListByBeanID(ctx context.Context, beanID uint64, limit int) ([]*model.BeanArticle, error) {
	var relations []*model.BeanArticle
	err := r.db.WithContext(ctx).Preload("Article").Where("bean_id = ?", beanID).Order("display_order ASC").Order("id ASC").Limit(limit).Find(&relations).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return relations, nil
}

// Articleに紐づくBean関連を表示順で取得。
func (r *GormBeanArticleRepository) ListByArticleID(ctx context.Context, articleID uint64, limit int) ([]*model.BeanArticle, error) {
	var relations []*model.BeanArticle
	err := r.db.WithContext(ctx).Preload("Bean").Where("article_id = ?", articleID).Order("display_order ASC").Order("id ASC").Limit(limit).Find(&relations).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return relations, nil
}

// 指定Beanの関連Articleを一括で差し替える。
func (r *GormBeanArticleRepository) ReplaceByBeanID(ctx context.Context, beanID uint64, articleIDs []uint64) error {
	if err := r.db.WithContext(ctx).Where("bean_id = ?", beanID).Delete(&model.BeanArticle{}).Error; err != nil {
		return mapDBError(err)
	}
	if len(articleIDs) == 0 {
		return nil
	}
	relations := make([]*model.BeanArticle, 0, len(articleIDs))
	for i, articleID := range articleIDs {
		relations = append(relations, &model.BeanArticle{BeanID: beanID, ArticleID: articleID, DisplayOrder: i})
	}
	return mapDBError(r.db.WithContext(ctx).Create(&relations).Error)
}

// ランキング対象を新規作成。
func (r *GormRankTargetRepository) Create(ctx context.Context, target *model.RankTarget) error {
	return mapDBError(r.db.WithContext(ctx).Create(target).Error)
}

// ランキング対象をIDで取得。
func (r *GormRankTargetRepository) FindByID(ctx context.Context, id uint64) (*model.RankTarget, error) {
	var target model.RankTarget
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&target).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &target, nil
}

// content_typeとcontent_idに一致するランキング対象を取得。
func (r *GormRankTargetRepository) FindByContent(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error) {
	var target model.RankTarget
	if err := r.db.WithContext(ctx).Where("content_type = ? AND content_id = ?", contentType, contentID).First(&target).Error; err != nil {
		return nil, mapDBError(err)
	}
	return &target, nil
}

// ランキング対象があれば取得し、なければ作成。
func (r *GormRankTargetRepository) FindOrCreate(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error) {
	target := model.RankTarget{ContentType: contentType, ContentID: contentID, IsActive: true}
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "content_type"}, {Name: "content_id"}}, DoNothing: true}).
		Create(&target).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return r.FindByContent(ctx, contentType, contentID)
}

// 複数IDに一致するランキング対象を取得。
func (r *GormRankTargetRepository) FindByIDs(ctx context.Context, ids []uint64) ([]*model.RankTarget, error) {
	var targets []*model.RankTarget
	if len(ids) == 0 {
		return targets, nil
	}
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&targets).Error; err != nil {
		return nil, mapDBError(err)
	}
	return targets, nil
}

// 指定コンテンツ種別の有効なランキング対象を取得。
func (r *GormRankTargetRepository) ListActiveByType(ctx context.Context, contentType entity.ContentType) ([]*model.RankTarget, error) {
	var targets []*model.RankTarget
	err := r.db.WithContext(ctx).Where("content_type = ? AND is_active = true", contentType).Order("id DESC").Find(&targets).Error
	if err != nil {
		return nil, mapDBError(err)
	}
	return targets, nil
}

// 有効なランキング対象として存在するか確認。
func (r *GormRankTargetRepository) ExistsActiveByID(ctx context.Context, id uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RankTarget{}).Where("id = ? AND is_active = true", id).Count(&count).Error
	return existsFromCount(count, err)
}

// ランキング対象の有効状態を更新。
func (r *GormRankTargetRepository) UpdateActive(ctx context.Context, id uint64, isActive bool) error {
	res := r.db.WithContext(ctx).Model(&model.RankTarget{}).Where("id = ?", id).Update("is_active", isActive)
	return mapRowsAffected(res.RowsAffected, res.Error)
}
