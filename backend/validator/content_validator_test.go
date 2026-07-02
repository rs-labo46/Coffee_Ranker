package validator

import (
	"testing"

	"coffee-ranker/entity"
)

// Bean/Article一覧のlimit/offsetを安全な範囲へ正規化することを検証。
func TestContentValidatorList(t *testing.T) {
	v := NewContentValidator()

	got, err := v.List(PageQuery{})
	assertNoError(t, err)
	if got.Limit != 20 || got.Offset != 0 {
		t.Fatalf("expected default list page, got %+v", got)
	}

	_, err = v.List(PageQuery{Limit: 101})
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}

// 詳細取得用IDが0ではないことだけを検証。
func TestContentValidatorDetailID(t *testing.T) {
	v := NewContentValidator()

	assertNoError(t, v.DetailID(1))
	assertErrorIs(t, v.DetailID(0), entity.ErrInvalidInput)
}

// Article詳細slugがURL安全形式かを検証。
func TestContentValidatorDetailSlug(t *testing.T) {
	v := NewContentValidator()

	got, err := v.DetailSlug("light-roast-guide")
	assertNoError(t, err)
	if got != "light-roast-guide" {
		t.Fatalf("expected slug, got %q", got)
	}

	_, err = v.DetailSlug("Light Roast")
	assertErrorIs(t, err, entity.ErrInvalidInput)
}

// 関連コンテンツ取得数のデフォルト化と上限超過拒否を検証。
func TestContentValidatorRelatedLimit(t *testing.T) {
	v := NewContentValidator()

	got, err := v.RelatedLimit(0)
	assertNoError(t, err)
	if got != 5 {
		t.Fatalf("expected default related limit 5, got %d", got)
	}

	got, err = v.RelatedLimit(20)
	assertNoError(t, err)
	if got != 20 {
		t.Fatalf("expected related limit 20, got %d", got)
	}

	_, err = v.RelatedLimit(21)
	assertErrorIs(t, err, entity.ErrInvalidPagination)
}
