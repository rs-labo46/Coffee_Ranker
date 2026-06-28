package validator

import "coffee-ranker/entity"

type RankingValidator struct{}
type RecommendationValidator struct{}

type RankingQuery struct {
	ContentType string `query:"content_type"`
	Limit       int    `query:"limit"`
	Offset      int    `query:"offset"`
}

type RecommendationQuery struct {
	ContentType string `query:"content_type"`
	Limit       int    `query:"limit"`
	Offset      int    `query:"offset"`
}

type ValidRankingQuery struct {
	ContentType *entity.ContentType
	Page        PageQuery
}

type ValidRecommendationQuery struct {
	ContentType *entity.ContentType
	Page        PageQuery
}

// NewRankingValidatorを生成してDI層やRouterから使えるようにする。
func NewRankingValidator() *RankingValidator {
	return &RankingValidator{}
}

// NewRecommendationValidatorを生成してDI層やRouterから使えるようにする。
func NewRecommendationValidator() *RecommendationValidator {
	return &RecommendationValidator{}
}

// ランキング一覧queryのcontent_typeとページングを検証。
// ランキングスコアの取得や並び順はRankingUsecaseで判断。
func (v *RankingValidator) List(input RankingQuery) (ValidRankingQuery, error) {
	contentType, err := ValidateContentType(input.ContentType)
	if err != nil {
		return ValidRankingQuery{}, err
	}
	// limit/offsetを上限内に正規化し、大量取得によるDB負荷を防ぐ。
	page, err := NormalizePage(PageQuery{Limit: input.Limit, Offset: input.Offset}, 20, 100, 10000)
	if err != nil {
		return ValidRankingQuery{}, err
	}
	return ValidRankingQuery{ContentType: contentType, Page: page}, nil
}

// TOP表示ランキングのlimitを検証。
// 未指定時はデフォルト値へ寄せ、過大取得を防ぐ。
func (v *RankingValidator) Top(limit int) (int, error) {
	if limit == 0 {
		return 10, nil
	}
	if limit < 0 || limit > 100 {
		return 0, entity.ErrInvalidPagination
	}
	return limit, nil
}

// 推薦一覧queryのcontent_type、limit、contextを検証。
// 保存済み除外や興味スコアによる候補選定はUsecaseで判断。
func (v *RecommendationValidator) List(input RecommendationQuery) (ValidRecommendationQuery, error) {
	contentType, err := ValidateContentType(input.ContentType)
	if err != nil {
		return ValidRecommendationQuery{}, err
	}
	// limit/offsetを上限内に正規化し、大量取得によるDB負荷を防ぐ。
	page, err := NormalizePage(PageQuery{Limit: input.Limit, Offset: input.Offset}, 20, 50, 500)
	if err != nil {
		return ValidRecommendationQuery{}, err
	}
	return ValidRecommendationQuery{ContentType: contentType, Page: page}, nil
}
