package validator

import (
	"strings"

	"coffee-ranker/entity"
)

type SearchValidator struct{}

type BeanSearchQuery struct {
	Q          string `query:"q"`
	Origin     string `query:"origin"`
	RoastLevel string `query:"roast_level"`
	Acidity    int    `query:"acidity"`
	Bitterness int    `query:"bitterness"`
	Flavor     int    `query:"flavor"`
	Aroma      int    `query:"aroma"`
	Body       int    `query:"body"`
	Sort       string `query:"sort"`
	Limit      int    `query:"limit"`
	Offset     int    `query:"offset"`
}

type ArticleSearchQuery struct {
	Q        string `query:"q"`
	Category string `query:"category"`
	Sort     string `query:"sort"`
	Limit    int    `query:"limit"`
	Offset   int    `query:"offset"`
}

type ValidBeanSearch struct {
	Keyword    *string
	Origin     *string
	RoastLevel *entity.RoastLevel
	Acidity    *int
	Bitterness *int
	Flavor     *int
	Aroma      *int
	Body       *int
	Sort       string
	Page       PageQuery
}

type ValidArticleSearch struct {
	Keyword  *string
	Category *string
	Sort     string
	Page     PageQuery
}

// NewSearchValidatorを生成してDI層やRouterから使えるようにする。
func NewSearchValidator() *SearchValidator {
	return &SearchValidator{}
}

// Bean検索queryをUsecaseへ渡せる形に検証・正規化。
// 検索語、産地、roast_level、味覚値、sort、limit/offsetを確認。
func (v *SearchValidator) BeanSearch(input BeanSearchQuery) (ValidBeanSearch, error) {
	// limit/offsetを上限内に正規化し、大量取得によるDB負荷を防ぐ。
	page, err := NormalizePage(PageQuery{Limit: input.Limit, Offset: input.Offset}, 20, 50, 500)
	if err != nil {
		return ValidBeanSearch{}, err
	}
	sort, err := NormalizeSort(input.Sort)
	if err != nil {
		return ValidBeanSearch{}, err
	}
	keyword, err := NormalizeOptionalText(optionalString(input.Q), 100)
	if err != nil {
		return ValidBeanSearch{}, entity.ErrInvalidSearchCondition
	}
	origin, err := NormalizeOptionalText(optionalString(input.Origin), 50)
	if err != nil {
		return ValidBeanSearch{}, entity.ErrInvalidSearchCondition
	}
	roastLevel, err := ValidateRoastLevel(strings.TrimSpace(input.RoastLevel))
	if err != nil {
		return ValidBeanSearch{}, err
	}
	acidity, err := optionalTaste(input.Acidity)
	if err != nil {
		return ValidBeanSearch{}, entity.ErrInvalidSearchCondition
	}
	bitterness, err := optionalTaste(input.Bitterness)
	if err != nil {
		return ValidBeanSearch{}, entity.ErrInvalidSearchCondition
	}
	flavor, err := optionalTaste(input.Flavor)
	if err != nil {
		return ValidBeanSearch{}, entity.ErrInvalidSearchCondition
	}
	aroma, err := optionalTaste(input.Aroma)
	if err != nil {
		return ValidBeanSearch{}, entity.ErrInvalidSearchCondition
	}
	body, err := optionalTaste(input.Body)
	if err != nil {
		return ValidBeanSearch{}, entity.ErrInvalidSearchCondition
	}
	return ValidBeanSearch{Keyword: keyword, Origin: origin, RoastLevel: roastLevel, Acidity: acidity, Bitterness: bitterness, Flavor: flavor, Aroma: aroma, Body: body, Sort: sort, Page: page}, nil
}

// Article検索queryをUsecaseへ渡せる形に検証・正規化。
// 検索語、カテゴリ、sort、limit/offsetを確認。
func (v *SearchValidator) ArticleSearch(input ArticleSearchQuery) (ValidArticleSearch, error) {
	// limit/offsetを上限内に正規化し、大量取得によるDB負荷を防ぎます。
	page, err := NormalizePage(PageQuery{Limit: input.Limit, Offset: input.Offset}, 20, 50, 500)
	if err != nil {
		return ValidArticleSearch{}, err
	}
	sort, err := NormalizeSort(input.Sort)
	if err != nil {
		return ValidArticleSearch{}, err
	}
	keyword, err := NormalizeOptionalText(optionalString(input.Q), 100)
	if err != nil {
		return ValidArticleSearch{}, entity.ErrInvalidSearchCondition
	}
	category, err := ValidateCategory(optionalString(input.Category))
	if err != nil {
		return ValidArticleSearch{}, entity.ErrInvalidSearchCondition
	}
	return ValidArticleSearch{Keyword: keyword, Category: category, Sort: sort, Page: page}, nil
}

// 空文字をnilに変換し、任意条件をUsecase inputへ渡しやすく。
// 文字列の安全性検証は呼び出し元で済ませる。
func optionalString(value string) *string {
	text := strings.TrimSpace(value)
	if text == "" {
		return nil
	}
	return &text
}

// 味覚条件の0を未指定nil、1〜5を有効値として扱う。
// 0以外の範囲外値は検索条件不正として拒否。
func optionalTaste(value int) (*int, error) {
	if value == 0 {
		return nil, nil
	}
	if value < 1 || value > 5 {
		return nil, entity.ErrInvalidInput
	}
	return &value, nil
}
