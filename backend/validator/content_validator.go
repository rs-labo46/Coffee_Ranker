package validator

type ContentValidator struct{}

// NewContentValidatorを生成してDI層やRouterから使えるようにする。
func NewContentValidator() *ContentValidator {
	return &ContentValidator{}
}

// Bean/Article一覧のlimit/offsetを検証。
// 公開中データの有無や取得順はUsecase/Repositoryで判断。
func (v *ContentValidator) List(input PageQuery) (PageQuery, error) {
	return NormalizePage(PageQuery{Limit: input.Limit, Offset: input.Offset}, 20, 100, 10000)
}

// 詳細取得用のIDが0でないかを検証。
// そのBean/Articleが存在するか、公開中かはUsecaseで判断。
func (v *ContentValidator) DetailID(id uint64) error {
	return ValidateID(id)
}

// Article詳細取得用slugの形式を検証。
// slugに対応する記事が存在するかはUsecaseで判断。
func (v *ContentValidator) DetailSlug(slug string) (string, error) {
	return ValidateSlug(slug)
}

// 関連記事・関連Bean取得数のlimitを検証。
// 未指定時はデフォルト値へ寄せ、過大取得を防ぐ。
func (v *ContentValidator) RelatedLimit(limit int) (int, error) {
	if limit == 0 {
		return 5, nil
	}
	// limit/offsetを上限内に正規化し、大量取得によるDB負荷を防ぐ。
	page, err := NormalizePage(PageQuery{Limit: limit}, 5, 20, 0)
	if err != nil {
		return 0, err
	}
	return page.Limit, nil
}
