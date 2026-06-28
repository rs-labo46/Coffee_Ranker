package validator

import (
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"coffee-ranker/entity"
)

type PageQuery struct {
	Limit  int `json:"limit" query:"limit"`
	Offset int `json:"offset" query:"offset"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var htmlTagPattern = regexp.MustCompile(`(?i)<\s*/?\s*[a-z][^>]*>`)
var scriptPattern = regexp.MustCompile(`(?i)<\s*script\b|<\s*iframe\b|\bon[a-z]+\s*=|javascript\s*:|data\s*:`)

// URL pathやRequest内のIDが0ではないかを検証。
// DBに存在するかはUsecaseで確認するため、ここではIDの形だけを見る。
func ValidateID(id uint64) error {
	if id == 0 {
		return entity.ErrInvalidInput
	}
	return nil
}

// limit/offsetを安全なページング条件へ正規化。
// 大量取得を防ぐため、上限値と負数を検証。
func NormalizePage(page PageQuery, defaultLimit int, maxLimit int, maxOffset int) (PageQuery, error) {
	if defaultLimit <= 0 {
		defaultLimit = 20
	}
	if maxLimit <= 0 {
		maxLimit = 100
	}
	if maxOffset <= 0 {
		maxOffset = 10000
	}
	if page.Limit == 0 {
		page.Limit = defaultLimit
	}
	if page.Limit < 0 || page.Limit > maxLimit || page.Offset < 0 || page.Offset > maxOffset {
		return PageQuery{}, entity.ErrInvalidPagination
	}
	return page, nil
}

// 必須文字列をtrimし、文字数と危険文字を検証。
// scriptタグ、javascript:、data:、制御文字などをUsecaseへ渡さないための入口。
func NormalizeText(value string, minLength int, maxLength int) (string, error) {
	text := strings.TrimSpace(value)
	if minLength > 0 && utf8.RuneCountInString(text) < minLength {
		return "", entity.ErrInvalidInput
	}
	if maxLength > 0 && utf8.RuneCountInString(text) > maxLength {
		return "", entity.ErrInvalidInput
	}
	if hasUnsafeText(text) {
		return "", entity.ErrInvalidInput
	}
	return text, nil
}

// 任意文字列をtrimし、空ならnilへ寄せる。
// 値がある場合だけ文字数と危険文字を検証。
func NormalizeOptionalText(value *string, maxLength int) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil, nil
	}
	if maxLength > 0 && utf8.RuneCountInString(text) > maxLength {
		return nil, entity.ErrInvalidInput
	}
	if hasUnsafeText(text) {
		return nil, entity.ErrInvalidInput
	}
	return &text, nil
}

// 画像URLや外部リンクURLが安全な形式かを検証。
// http/httpsだけを許可し、javascript:やdata:によるXSSを防ぐ。
func ValidateURL(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil, nil
	}
	if hasControl(text) || scriptPattern.MatchString(text) {
		return nil, entity.ErrInvalidInput
	}
	parsed, err := url.ParseRequestURI(text)
	if err != nil {
		return nil, entity.ErrInvalidInput
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, entity.ErrInvalidInput
	}
	return &text, nil
}

// ArticleのslugがURLに使える安全な形式かを検証。
// 英小文字・数字・ハイフンだけを許可し、先頭末尾ハイフンや連続ハイフンを拒否。
func ValidateSlug(slug string) (string, error) {
	text := strings.TrimSpace(slug)
	length := utf8.RuneCountInString(text)
	if length < 3 || length > 120 || !slugPattern.MatchString(text) {
		return "", entity.ErrInvalidInput
	}
	return text, nil
}

// emailをtrimして小文字化し、メール形式と最大長を検証。
// email重複やログイン可否はUsecaseで判断。
func ValidateEmail(email string) (string, error) {
	text := strings.ToLower(strings.TrimSpace(email))
	if text == "" || len(text) > 254 || hasControl(text) || htmlTagPattern.MatchString(text) {
		return "", entity.ErrInvalidInput
	}
	addr, err := mail.ParseAddress(text)
	if err != nil || addr.Address != text {
		return "", entity.ErrInvalidInput
	}
	return text, nil
}

// Signup時の生passwordがbcryptで扱える長さかを検証。
// password一致やhash化処理そのものはUsecaseで行う。
func ValidatePasswordForSignup(password string) error {
	length := len([]byte(password))
	if length < 8 || length > 72 || strings.TrimSpace(password) == "" {
		return entity.ErrInvalidInput
	}
	return nil
}

// Login時のpassword入力が空でなく処理可能な長さかを検証。
// passwordが正しいかどうかはUsecaseで照合。
func ValidatePasswordForLogin(password string) error {
	length := len([]byte(password))
	if length == 0 || length > 72 {
		return entity.ErrInvalidInput
	}
	return nil
}

// queryなど任意入力のroast_levelが許可enumかを検証。
// 空文字は未指定としてnilにし、light/medium/dark以外を拒否。
func ValidateRoastLevel(value string) (*entity.RoastLevel, error) {
	if value == "" {
		return nil, nil
	}
	level := entity.RoastLevel(value)
	switch level {
	case entity.RoastLevelLight, entity.RoastLevelMedium, entity.RoastLevelDark:
		return &level, nil
	default:
		return nil, entity.ErrInvalidSearchCondition
	}
}

// 作成・更新時に必須のroast_levelが許可enumかを検証。
// 空文字や未定義値を拒否し、DBに壊れたenumが入ることを防ぐ。
func ValidateRequiredRoastLevel(value entity.RoastLevel) error {
	switch value {
	case entity.RoastLevelLight, entity.RoastLevelMedium, entity.RoastLevelDark:
		return nil
	default:
		return entity.ErrInvalidInput
	}
}

// content_typeがbean/articleのどちらかかを検証。
// content_typeの実体がDBに存在するかはUsecaseで判断。
func ValidateContentType(value string) (*entity.ContentType, error) {
	if value == "" {
		return nil, nil
	}
	contentType := entity.ContentType(value)
	switch contentType {
	case entity.ContentTypeBean, entity.ContentTypeArticle:
		return &contentType, nil
	default:
		return nil, entity.ErrInvalidContentType
	}
}

// 行動ログや保存・評価のplacementが許可値かを検証。
// modalを許可するかどうかは呼び出し元のAPI用途に応じて切り替える。
func ValidatePlacement(value entity.Placement, allowModal bool) error {
	switch value {
	case entity.PlacementTop, entity.PlacementSearchResult, entity.PlacementBeanDetail, entity.PlacementArticleDetail, entity.PlacementRelatedArticle, entity.PlacementRelatedBean, entity.PlacementSavedList:
		return nil
	case entity.PlacementModal:
		if allowModal {
			return nil
		}
	}
	return entity.ErrInvalidInput
}

// Good/Bad評価が+1または-1だけかを検証。
// 5段階評価や0が混入し、興味スコア集計が壊れることを防ぐ。
func ValidateRatingScore(score entity.RatingScore) error {
	if score != entity.RatingScoreGood && score != entity.RatingScoreBad {
		return entity.ErrInvalidRatingScore
	}
	return nil
}

// 酸味・苦味・風味・香り・ボディの値が1〜5かを検証。
// nilは未指定として許可し、範囲外だけを拒否。
func ValidateTasteScore(value *int) error {
	if value == nil {
		return nil
	}
	if *value < 1 || *value > 5 {
		return entity.ErrInvalidInput
	}
	return nil
}

// page_pathがアプリ内パスとして安全かを検証。
// 外部URL、javascript:、data:、制御文字を拒否。
func ValidatePagePath(pagePath string) (string, error) {
	text := strings.TrimSpace(pagePath)
	if text == "" || !strings.HasPrefix(text, "/") || utf8.RuneCountInString(text) > 255 {
		return "", entity.ErrInvalidInput
	}
	if strings.HasPrefix(strings.ToLower(text), "//") || hasControl(text) || scriptPattern.MatchString(text) {
		return "", entity.ErrInvalidInput
	}
	return text, nil
}

// 検索条件hashなど任意hash文字列の長さと危険文字を検証。
// hashの意味やDB上の存在確認はUsecaseで判断。
func ValidateHash(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.TrimSpace(*value)
	if text == "" || utf8.RuneCountInString(text) > 128 || hasControl(text) || htmlTagPattern.MatchString(text) {
		return nil, entity.ErrInvalidInput
	}
	return &text, nil
}

// 記事カテゴリが許可カテゴリかを検証。
// 空は未指定としてnilにし、未定義カテゴリや危険文字を拒否。
func ValidateCategory(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil, nil
	}
	switch text {
	case "brewing", "roast", "beans", "recipe":
		return &text, nil
	default:
		return nil, entity.ErrInvalidInput
	}
}

// sortがscore/newest/popularのどれかかを検証。
// 空文字はデフォルトのscoreへ正規化。
func NormalizeSort(sort string) (string, error) {
	text := strings.TrimSpace(sort)
	if text == "" {
		return "score", nil
	}
	switch text {
	case "score", "newest", "popular":
		return text, nil
	default:
		return "", entity.ErrInvalidSearchCondition
	}
}

// 文字列にXSSにつながる危険な断片が含まれるかを判定。
// script/iframe/on属性/javascript:/data:を検出する共通ヘルパー。
func hasUnsafeText(value string) bool {
	if hasControl(value) {
		return true
	}
	return scriptPattern.MatchString(value) || htmlTagPattern.MatchString(value)
}

// 改行・タブ以外の制御文字が含まれるかを判定。
// ログ汚染や表示崩れにつながる入力を止めるために使う。
func hasControl(value string) bool {
	for _, r := range value {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
}
