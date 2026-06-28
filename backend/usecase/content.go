package usecase

import (
	"context"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 一覧取得のlimitとoffset。
type Page struct {
	Limit  int //何件取得するか
	Offset int //何件目からか
}

// 公開Beanの一覧、詳細、関連記事を取得。
type BeanUsecase struct {
	beans     repository.BeanRepository
	relations repository.BeanArticleRepository
}

// 公開Articleの一覧、詳細、関連Beanを取得。
type ArticleUsecase struct {
	articles  repository.ArticleRepository
	relations repository.BeanArticleRepository
}

// IDが0でないことを確認。共通で使う
func requirePositiveID(id uint64) error {
	if id == 0 {
		return entity.ErrInvalidInput
	}
	return nil
}

// limitとoffsetをデフォルト値・上限値に合わせて検証。
func normalizePage(page Page, defaultLimit int, maxLimit int, maxOffset int) (Page, error) {
	if defaultLimit <= 0 {
		defaultLimit = 20
	}
	if maxLimit <= 0 {
		maxLimit = 100
	}
	if page.Limit == 0 {
		page.Limit = defaultLimit
	}
	if page.Limit < 0 || page.Limit > maxLimit || page.Offset < 0 || page.Offset > maxOffset {
		return Page{}, entity.ErrInvalidPagination
	}
	return page, nil
}

// Bean表示に必要なRepositoryを受け取るコンストラクタ。
func NewBeanUsecase(beans repository.BeanRepository, relations repository.BeanArticleRepository) *BeanUsecase {
	return &BeanUsecase{beans: beans, relations: relations}
}

// Article表示に必要なRepositoryを受け取るコンストラクタ。
func NewArticleUsecase(articles repository.ArticleRepository, relations repository.BeanArticleRepository) *ArticleUsecase {
	return &ArticleUsecase{articles: articles, relations: relations}
}

// 公開コンテンツ一覧をページング付きで取得。
func (u *BeanUsecase) List(ctx context.Context, page Page) ([]*model.Bean, error) {
	page, err := normalizePage(page, 20, 100, 10000)
	if err != nil {
		return nil, err
	}
	return u.beans.ListPublished(ctx, page.Limit, page.Offset)
}

// 指定IDの公開コンテンツ詳細を取得。
func (u *BeanUsecase) GetDetail(ctx context.Context, id uint64) (*model.Bean, error) {
	//IDが０は弾く
	if err := requirePositiveID(id); err != nil {
		return nil, err
	}
	bean, err := u.beans.FindPublishedByID(ctx, id) //公開中のBeanだけを取得
	if err != nil {
		if err == entity.ErrNotFound {
			return nil, entity.ErrBeanNotFound
		}
		return nil, err
	}
	return bean, nil //bean詳細を返す
}

// Beanに紐づく関連記事を取得。
func (u *BeanUsecase) RelatedArticles(ctx context.Context, beanID uint64, limit int) ([]*model.BeanArticle, error) {
	if err := requirePositiveID(beanID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5 //指定なかったら5
	}
	if limit > 20 {
		return nil, entity.ErrInvalidPagination
	}
	return u.relations.ListByBeanID(ctx, beanID, limit) //Beanに紐ずくArticle関連を取得
}

// 公開コンテンツ一覧をページング付きで取得。
func (u *ArticleUsecase) List(ctx context.Context, page Page) ([]*model.Article, error) {
	page, err := normalizePage(page, 20, 100, 10000)
	if err != nil {
		return nil, err
	}
	return u.articles.ListPublished(ctx, page.Limit, page.Offset) //公開中Articleだけ取得だけを取得
}

// Article IDから詳細を取得し、未認証なら取得を制限。
func (u *ArticleUsecase) GetDetailByID(ctx context.Context, id uint64, authenticated bool) (*model.Article, error) {
	//未ログインなら拒否
	if !authenticated {
		return nil, entity.ErrLoginRequired
	}
	if err := requirePositiveID(id); err != nil {
		return nil, err
	}
	//公開中Articleだけ取得
	article, err := u.articles.FindPublishedByID(ctx, id)
	if err != nil {
		if err == entity.ErrNotFound {
			return nil, entity.ErrArticleNotFound
		}
		return nil, err
	}
	return article, nil
}

// Article slugから詳細を取得し、未認証なら詳細取得を制限。
func (u *ArticleUsecase) GetDetailBySlug(ctx context.Context, slug string, authenticated bool) (*model.Article, error) {
	if !authenticated {
		return nil, entity.ErrLoginRequired
	}
	if slug == "" {
		return nil, entity.ErrInvalidInput
	}
	article, err := u.articles.FindPublishedBySlug(ctx, slug)
	if err != nil {
		if err == entity.ErrNotFound {
			return nil, entity.ErrArticleNotFound
		}
		return nil, err
	}
	return article, nil
}

// Articleに紐づく関連Beanを取得。
func (u *ArticleUsecase) RelatedBeans(ctx context.Context, articleID uint64, limit int) ([]*model.BeanArticle, error) {
	if err := requirePositiveID(articleID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		return nil, entity.ErrInvalidPagination
	}
	return u.relations.ListByArticleID(ctx, articleID, limit)
}
