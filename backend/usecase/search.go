package usecase

import (
	"context"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 公開Bean/Articleの検索条件を整える。
type SearchUsecase struct {
	beans    repository.BeanRepository
	articles repository.ArticleRepository
}

// Bean検索で受け取る条件:味覚条件、産地、焙煎度、並び順、ページングをまとめる。
type BeanSearchInput struct {
	Keyword    *string
	Origin     *string
	RoastLevel *entity.RoastLevel
	Acidity    *int
	Bitterness *int
	Flavor     *int
	Aroma      *int
	Body       *int
	Sort       string
	Page       Page
}

// Article検索で受け取る条件:キーワード、カテゴリ、並び順、ページングをまとめる。
type ArticleSearchInput struct {
	Keyword  *string
	Category *string
	Sort     string
	Page     Page
}

// 検索に必要なBean/Article Repositoryを受け取るコンストラクタ。
func NewSearchUsecase(beans repository.BeanRepository, articles repository.ArticleRepository) *SearchUsecase {
	return &SearchUsecase{beans: beans, articles: articles}
}

// Bean検索条件を正規化し、公開Beanだけを検索。
func (u *SearchUsecase) SearchBeans(ctx context.Context, input BeanSearchInput) ([]*model.Bean, error) {
	// 検索系は一覧より上限を小さくし、DB負荷を抑える。
	page, err := normalizePage(input.Page, 20, 50, 500)
	if err != nil {
		return nil, err
	}

	// sortはscore/newest/popularだけ許可。
	sort, err := normalizeSort(input.Sort)
	if err != nil {
		return nil, err
	}

	// Repositoryへ渡す検索条件を組み立て
	filter := repository.BeanSearchFilter{
		Keyword:    normalizeOptionalText(input.Keyword),
		Origin:     normalizeOptionalText(input.Origin),
		RoastLevel: input.RoastLevel,
		Acidity:    input.Acidity,
		Bitterness: input.Bitterness,
		Flavor:     input.Flavor,
		Aroma:      input.Aroma,
		Body:       input.Body,
		Sort:       sort,
		Limit:      page.Limit,
		Offset:     page.Offset,
	}

	// 非公開Beanを検索結果に出さない
	return u.beans.SearchPublished(ctx, filter)
}

// Article検索条件を正規化し、公開Articleだけを検索。
func (u *SearchUsecase) SearchArticles(ctx context.Context, input ArticleSearchInput) ([]*model.Article, error) {
	// 検索系は一覧より上限を小さくし、DB負荷を抑える。
	page, err := normalizePage(input.Page, 20, 50, 500)
	if err != nil {
		return nil, err
	}

	// sortはscore/newest/popularだけ許可。
	sort, err := normalizeSort(input.Sort)
	if err != nil {
		return nil, err
	}

	// Repositoryへ渡す検索条件を組み立て、文字列系は空白をtrimし、空文字ならnilに。
	filter := repository.ArticleSearchFilter{
		Keyword:  normalizeOptionalText(input.Keyword),
		Category: normalizeOptionalText(input.Category),
		Sort:     sort,
		Limit:    page.Limit,
		Offset:   page.Offset,
	}

	// 非公開Articleを検索結果に出さない
	return u.articles.SearchPublished(ctx, filter)
}

// sort指定をscore/newest/popularのどれかに正規化。
func normalizeSort(sort string) (string, error) {
	// 未指定ならランキングスコア順をデフォルトに。
	if sort == "" {
		return "score", nil
	}

	// 許可していないsortをRepositoryへ渡さない。
	switch sort {
	case "score", "newest", "popular":
		return sort, nil
	default:
		return "", entity.ErrInvalidSearchCondition
	}
}
