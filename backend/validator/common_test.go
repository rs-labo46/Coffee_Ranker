package validator

import (
	"errors"
	"testing"

	"coffee-ranker/entity"
)

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func assertErrorIs(t *testing.T, err error, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("expected error %v, got %v", want, err)
	}
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func ratingScorePtr(value entity.RatingScore) *entity.RatingScore {
	return &value
}

func roastLevelPtr(value entity.RoastLevel) *entity.RoastLevel {
	return &value
}

// IDが0の場合だけ不正入力として止め、正のIDはUsecaseへ渡せることを検証。
func TestValidateID(t *testing.T) {
	assertErrorIs(t, ValidateID(0), entity.ErrInvalidInput)
	assertNoError(t, ValidateID(1))
}

// limit未指定時のデフォルト化と、負数・上限超過の拒否を検証。
func TestNormalizePage(t *testing.T) {
	got, err := NormalizePage(PageQuery{}, 20, 100, 10000)
	assertNoError(t, err)
	if got.Limit != 20 || got.Offset != 0 {
		t.Fatalf("expected default page limit=20 offset=0, got %+v", got)
	}

	got, err = NormalizePage(PageQuery{Limit: 10, Offset: 5}, 20, 100, 10000)
	assertNoError(t, err)
	if got.Limit != 10 || got.Offset != 5 {
		t.Fatalf("expected explicit page limit=10 offset=5, got %+v", got)
	}

	_, err = NormalizePage(PageQuery{Limit: 101}, 20, 100, 10000)
	assertErrorIs(t, err, entity.ErrInvalidPagination)

	_, err = NormalizePage(PageQuery{Limit: 10, Offset: -1}, 20, 100, 10000)
	assertErrorIs(t, err, entity.ErrInvalidPagination)

	_, err = NormalizePage(PageQuery{Limit: 10, Offset: 10001}, 20, 100, 10000)
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}

// 必須文字列のtrim、空文字拒否、最大長拒否、危険HTML拒否を検証。
func TestNormalizeText(t *testing.T) {
	got, err := NormalizeText("  Coffee  ", 1, 20)
	assertNoError(t, err)
	if got != "Coffee" {
		t.Fatalf("expected trimmed text Coffee, got %q", got)
	}

	_, err = NormalizeText("   ", 1, 20)
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = NormalizeText("abcdef", 1, 5)
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = NormalizeText("<script>alert(1)</script>", 1, 100)
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// 任意文字列がnil/空ならnilへ寄り、値がある時だけ安全性検証することを確認。
func TestNormalizeOptionalText(t *testing.T) {
	got, err := NormalizeOptionalText(nil, 20)
	assertNoError(t, err)
	if got != nil {
		t.Fatalf("expected nil, got %q", *got)
	}

	got, err = NormalizeOptionalText(stringPtr("   "), 20)
	assertNoError(t, err)
	if got != nil {
		t.Fatalf("expected blank optional text to become nil, got %q", *got)
	}

	got, err = NormalizeOptionalText(stringPtr("  Ethiopia  "), 20)
	assertNoError(t, err)
	if got == nil || *got != "Ethiopia" {
		t.Fatalf("expected trimmed optional text Ethiopia, got %v", got)
	}

	_, err = NormalizeOptionalText(stringPtr("<b>bad</b>"), 20)
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// http/httpsだけを許可し、javascript/data/ftp/制御文字を拒否することを検証。
func TestValidateURL(t *testing.T) {
	got, err := ValidateURL(stringPtr(" https://example.com/image.png "))
	assertNoError(t, err)
	if got == nil || *got != "https://example.com/image.png" {
		t.Fatalf("expected trimmed https URL, got %v", got)
	}

	got, err = ValidateURL(stringPtr(""))
	assertNoError(t, err)
	if got != nil {
		t.Fatalf("expected empty URL to become nil, got %q", *got)
	}

	invalidValues := []string{"javascript:alert(1)", "data:text/html,xxx", "ftp://example.com/file", "https://example.com/\x00bad"}
	for _, value := range invalidValues {
		_, err := ValidateURL(stringPtr(value))
		assertErrorIs(t, err, entity.ErrInvalidInput)
	}
}

// URL用slugとして使える小文字英数字とハイフンだけを許可することを検証。
func TestValidateSlug(t *testing.T) {
	got, err := ValidateSlug(" light-roast-guide ")
	assertNoError(t, err)
	if got != "light-roast-guide" {
		t.Fatalf("expected normalized slug, got %q", got)
	}

	invalidValues := []string{"ab", "Light-Roast", "light--roast", "-light", "light-", "日本語"}
	for _, value := range invalidValues {
		_, err := ValidateSlug(value)
		assertErrorIs(t, err, entity.ErrInvalidInput)
	}
}

// emailのtrim・小文字化・形式確認を行い、HTML混入を拒否することを検証。
func TestValidateEmail(t *testing.T) {
	got, err := ValidateEmail(" USER@EXAMPLE.COM ")
	assertNoError(t, err)
	if got != "user@example.com" {
		t.Fatalf("expected lower-cased email, got %q", got)
	}

	invalidValues := []string{"", "not-email", "user@example.com<script>", "a@"}
	for _, value := range invalidValues {
		_, err := ValidateEmail(value)
		assertErrorIs(t, err, entity.ErrInvalidInput)
	}
}

// Signup用passwordが8〜72bytesかつ空白のみでないことを検証。
func TestValidatePasswordForSignup(t *testing.T) {
	assertNoError(t, ValidatePasswordForSignup("password123"))
	assertErrorIs(t, ValidatePasswordForSignup("short"), entity.ErrInvalidInput)
	assertErrorIs(t, ValidatePasswordForSignup("        "), entity.ErrInvalidInput)
	assertErrorIs(t, ValidatePasswordForSignup("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"+"aaaa"), entity.ErrInvalidInput)
}

// Loginでは空文字と72bytes超だけを止めることを検証。
func TestValidatePasswordForLogin(t *testing.T) {
	assertNoError(t, ValidatePasswordForLogin("a"))
	assertErrorIs(t, ValidatePasswordForLogin(""), entity.ErrInvalidInput)
	assertErrorIs(t, ValidatePasswordForLogin("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"+"aaaa"), entity.ErrInvalidInput)
}

// 任意入力のroast_levelで空文字をnil扱いにし、許可enumだけを通すことを検証。
func TestValidateRoastLevel(t *testing.T) {
	got, err := ValidateRoastLevel("")
	assertNoError(t, err)
	if got != nil {
		t.Fatalf("expected empty roast_level to become nil, got %v", got)
	}

	got, err = ValidateRoastLevel("medium")
	assertNoError(t, err)
	if got == nil || *got != entity.RoastLevelMedium {
		t.Fatalf("expected medium roast level, got %v", got)
	}

	_, err = ValidateRoastLevel("city")
	assertErrorIs(t, err, entity.ErrInvalidSearchCondition)
}

// 作成・更新時のroast_level必須入力が許可enumだけかを検証。
func TestValidateRequiredRoastLevel(t *testing.T) {
	assertNoError(t, ValidateRequiredRoastLevel(entity.RoastLevelLight))
	assertErrorIs(t, ValidateRequiredRoastLevel(entity.RoastLevel("city")), entity.ErrInvalidInput)
}

// content_typeがbean/articleのみ許可され、空文字は未指定nilになることを検証。
func TestValidateContentType(t *testing.T) {
	got, err := ValidateContentType("")
	assertNoError(t, err)
	if got != nil {
		t.Fatalf("expected empty content_type to become nil, got %v", got)
	}

	got, err = ValidateContentType("bean")
	assertNoError(t, err)
	if got == nil || *got != entity.ContentTypeBean {
		t.Fatalf("expected bean content type, got %v", got)
	}

	_, err = ValidateContentType("coffee")
	assertErrorIs(t, err, entity.ErrInvalidContentType)
}

// modal placementを通常APIでは拒否し、許可フラグ付きなら通すことを検証。
func TestValidatePlacement(t *testing.T) {
	assertNoError(t, ValidatePlacement(entity.PlacementTop, false))
	assertErrorIs(t, ValidatePlacement(entity.PlacementModal, false), entity.ErrInvalidInput)
	assertNoError(t, ValidatePlacement(entity.PlacementModal, true))
	assertErrorIs(t, ValidatePlacement(entity.Placement("unknown"), false), entity.ErrInvalidInput)
}

// Good(+1)とBad(-1)だけを評価値として許可することを検証。
func TestValidateRatingScore(t *testing.T) {
	assertNoError(t, ValidateRatingScore(entity.RatingScoreGood))
	assertNoError(t, ValidateRatingScore(entity.RatingScoreBad))
	assertErrorIs(t, ValidateRatingScore(entity.RatingScore(0)), entity.ErrInvalidRatingScore)
	assertErrorIs(t, ValidateRatingScore(entity.RatingScore(5)), entity.ErrInvalidRatingScore)
}

// 味覚値nilを未指定として許可し、1〜5以外を拒否することを検証。
func TestValidateTasteScore(t *testing.T) {
	assertNoError(t, ValidateTasteScore(nil))
	assertNoError(t, ValidateTasteScore(intPtr(3)))
	assertErrorIs(t, ValidateTasteScore(intPtr(0)), entity.ErrInvalidInput)
	assertErrorIs(t, ValidateTasteScore(intPtr(6)), entity.ErrInvalidInput)
}

// アプリ内パスだけを許可し、外部URLやjavascript/dataを拒否することを検証。
func TestValidatePagePath(t *testing.T) {
	got, err := ValidatePagePath(" /beans/1 ")
	assertNoError(t, err)
	if got != "/beans/1" {
		t.Fatalf("expected normalized page path, got %q", got)
	}

	invalidValues := []string{"", "https://example.com", "//example.com", "/path<script>", "javascript:alert(1)"}
	for _, value := range invalidValues {
		_, err := ValidatePagePath(value)
		assertErrorIs(t, err, entity.ErrInvalidInput)
	}
}

// 任意hash文字列のtrimと空文字・危険HTML・長すぎる値の拒否を検証。
func TestValidateHash(t *testing.T) {
	got, err := ValidateHash(nil)
	assertNoError(t, err)
	if got != nil {
		t.Fatalf("expected nil hash, got %v", got)
	}

	got, err = ValidateHash(stringPtr(" hash-123 "))
	assertNoError(t, err)
	if got == nil || *got != "hash-123" {
		t.Fatalf("expected normalized hash, got %v", got)
	}

	_, err = ValidateHash(stringPtr(""))
	assertErrorIs(t, err, entity.ErrInvalidInput)

	_, err = ValidateHash(stringPtr("<script>bad</script>"))
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// 記事カテゴリの許可値だけを通し、空文字は未指定nilにすることを検証。
func TestValidateCategory(t *testing.T) {
	got, err := ValidateCategory(nil)
	assertNoError(t, err)
	if got != nil {
		t.Fatalf("expected nil category, got %v", got)
	}

	got, err = ValidateCategory(stringPtr(" brewing "))
	assertNoError(t, err)
	if got == nil || *got != "brewing" {
		t.Fatalf("expected brewing category, got %v", got)
	}

	_, err = ValidateCategory(stringPtr("security"))
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// sort未指定をscoreへ寄せ、score/newest/popular以外を拒否することを検証。
func TestNormalizeSort(t *testing.T) {
	got, err := NormalizeSort("")
	assertNoError(t, err)
	if got != "score" {
		t.Fatalf("expected default sort score, got %q", got)
	}

	got, err = NormalizeSort("popular")
	assertNoError(t, err)
	if got != "popular" {
		t.Fatalf("expected popular sort, got %q", got)
	}

	_, err = NormalizeSort("random")
	assertErrorIs(t, err, entity.ErrInvalidSearchCondition)
}
