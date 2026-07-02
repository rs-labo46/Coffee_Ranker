package validator

import (
	"errors"
	"testing"

	"coffee-ranker/entity"
)

// 管理者Bean入力がtrimされ、空の任意項目がnilへ寄せられることを確認する。
// 管理画面から危険な値をUsecaseへ渡さないため、XSS・URL・味覚スコアの入口をまとめて検証する。
func TestAdminBeanValidator_ValidInputNormalizesFields(t *testing.T) {
	v := NewAdminBeanValidator()
	acidity := 5
	bitterness := 1
	flavor := 4
	aroma := 3
	body := 2
	roaster := "  "
	origin := " Ethiopia "
	imageURL := " https://example.com/beans/1.jpg "
	description := " 明るい酸味のBean "

	got, err := v.Bean(AdminBeanRequest{
		Name:        " Ethiopia Yirgacheffe ",
		Roaster:     &roaster,
		Origin:      &origin,
		RoastLevel:  entity.RoastLevelLight,
		Acidity:     &acidity,
		Bitterness:  &bitterness,
		Flavor:      &flavor,
		Aroma:       &aroma,
		Body:        &body,
		Description: &description,
		ImageURL:    &imageURL,
	}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Ethiopia Yirgacheffe" {
		t.Fatalf("Name = %q", got.Name)
	}
	if got.Roaster != nil {
		t.Fatalf("Roaster = %v, want nil", *got.Roaster)
	}
	if got.Origin == nil || *got.Origin != "Ethiopia" {
		t.Fatalf("Origin = %v", got.Origin)
	}
	if got.ImageURL == nil || *got.ImageURL != "https://example.com/beans/1.jpg" {
		t.Fatalf("ImageURL = %v", got.ImageURL)
	}
}

// Bean説明文・画像URL・味覚スコア・焙煎度の不正値を拒否することを確認する。
// Admin入力は公開画面に出るため、scriptやjavascript URLをUsecaseへ渡さないことが必須。
func TestAdminBeanValidator_RejectsUnsafeInput(t *testing.T) {
	v := NewAdminBeanValidator()
	unsafeDescription := "<script>alert(1)</script>"
	javascriptURL := "javascript:alert(1)"
	badScore := 6

	tests := []struct {
		name  string
		input AdminBeanRequest
	}{
		{
			name: "description script",
			input: validAdminBeanRequest(func(input *AdminBeanRequest) {
				input.Description = &unsafeDescription
			}),
		},
		{
			name: "javascript image url",
			input: validAdminBeanRequest(func(input *AdminBeanRequest) {
				input.ImageURL = &javascriptURL
			}),
		},
		{
			name: "taste score out of range",
			input: validAdminBeanRequest(func(input *AdminBeanRequest) {
				input.Acidity = &badScore
			}),
		},
		{
			name: "invalid roast level",
			input: validAdminBeanRequest(func(input *AdminBeanRequest) {
				input.RoastLevel = entity.RoastLevel("city")
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Bean(tt.input, false)
			if !errors.Is(err, entity.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

// Articleのslug、本文、カテゴリ、外部URLの不正値を拒否することを確認する。
// 記事本文や外部リンクはXSSと不正遷移の入口になりやすいため、Validatorで止める。
func TestAdminArticleValidator_RejectsUnsafeInput(t *testing.T) {
	v := NewAdminArticleValidator()
	body := "<h1>unsafe</h1>"
	badCategory := "coffee"
	dataURL := "data:text/html;base64,PHNjcmlwdD4="

	tests := []struct {
		name  string
		input AdminArticleRequest
	}{
		{
			name: "invalid slug",
			input: validAdminArticleRequest(func(input *AdminArticleRequest) {
				input.Slug = "Bad Slug"
			}),
		},
		{
			name: "html body",
			input: validAdminArticleRequest(func(input *AdminArticleRequest) {
				input.Body = &body
			}),
		},
		{
			name: "unknown category",
			input: validAdminArticleRequest(func(input *AdminArticleRequest) {
				input.Category = &badCategory
			}),
		},
		{
			name: "data source url",
			input: validAdminArticleRequest(func(input *AdminArticleRequest) {
				input.SourceURL = &dataURL
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.Article(tt.input, false)
			if !errors.Is(err, entity.ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

// 一括関連更新で0 ID・重複ID・過大件数を拒否することを確認する。
// 関連の一括差し替えは既存関連を壊しやすいため、DB存在確認前にID形状と重複を止める。
func TestAdminRelationValidator_RejectsInvalidReplacement(t *testing.T) {
	v := NewAdminRelationValidator()

	if _, err := v.Replace(ReplaceRelationsRequest{ArticleIDs: []uint64{1, 2, 1}}); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("duplicate error = %v, want ErrInvalidInput", err)
	}
	if _, err := v.Replace(ReplaceRelationsRequest{ArticleIDs: []uint64{1, 0}}); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("zero id error = %v, want ErrInvalidInput", err)
	}

	tooMany := make([]uint64, 101)
	for i := range tooMany {
		tooMany[i] = uint64(i + 1)
	}
	if _, err := v.Replace(ReplaceRelationsRequest{ArticleIDs: tooMany}); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("too many error = %v, want ErrInvalidInput", err)
	}
}

// 手動バッチとRateLimit resetの管理者入力を検証する。
// ownerやkeyを安全な文字列に正規化し、0 IDやscript混入をUsecaseへ渡さないことを確認する。
func TestAdminBatchAndRateLimitValidator(t *testing.T) {
	batch := NewAdminBatchValidator()
	got, err := batch.Batch(BatchRunRequest{Owner: " admin-job ", UserIDs: []uint64{1}, GuestSessionIDs: []uint64{2}})
	if err != nil {
		t.Fatalf("unexpected batch error: %v", err)
	}
	if got.Owner != "admin-job" {
		t.Fatalf("Owner = %q", got.Owner)
	}
	if _, err := batch.Batch(BatchRunRequest{Owner: "job", UserIDs: []uint64{0}}); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("zero user id error = %v, want ErrInvalidInput", err)
	}

	rateLimit := NewAdminRateLimitValidator()
	reset, err := rateLimit.Reset(RateLimitResetRequest{Key: " rate:login:ip:abc "})
	if err != nil {
		t.Fatalf("unexpected rate limit error: %v", err)
	}
	if reset.Key != "rate:login:ip:abc" {
		t.Fatalf("Key = %q", reset.Key)
	}
	if _, err := rateLimit.Reset(RateLimitResetRequest{Key: "<script>"}); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("script key error = %v, want ErrInvalidInput", err)
	}
}

// 監査ログ検索条件をRepository filterへ変換できることを確認する。
// HTTP Queryの文字列をそのままRepositoryへ流さず、actor種別・ID・ページングを安全な形にする。
func TestAdminAuditValidator_ListBuildsSafeFilter(t *testing.T) {
	v := NewAdminAuditValidator()
	filter, err := v.List(AuditQuery{ActorType: "admin", ActorUserID: 10, Action: "login", TargetType: " user ", TargetID: 20, Limit: 0, Offset: 5})
	if err != nil {
		t.Fatalf("unexpected audit error: %v", err)
	}
	if filter.ActorType == nil || *filter.ActorType != entity.AuditActorAdmin {
		t.Fatalf("ActorType = %v", filter.ActorType)
	}
	if filter.ActorUserID == nil || *filter.ActorUserID != 10 {
		t.Fatalf("ActorUserID = %v", filter.ActorUserID)
	}
	if filter.TargetType == nil || *filter.TargetType != "user" {
		t.Fatalf("TargetType = %v", filter.TargetType)
	}
	if filter.TargetID == nil || *filter.TargetID != 20 {
		t.Fatalf("TargetID = %v", filter.TargetID)
	}
	if filter.Limit != 20 || filter.Offset != 5 {
		t.Fatalf("page = limit:%d offset:%d", filter.Limit, filter.Offset)
	}

	if _, err := v.List(AuditQuery{ActorType: "guest"}); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("invalid actor error = %v, want ErrInvalidInput", err)
	}
}

// 各ケースで一部だけ壊すための正常なBean管理入力を返す。
func validAdminBeanRequest(change func(*AdminBeanRequest)) AdminBeanRequest {
	one := 1
	request := AdminBeanRequest{
		Name:       "Valid Bean",
		RoastLevel: entity.RoastLevelMedium,
		Acidity:    &one,
	}
	if change != nil {
		change(&request)
	}
	return request
}

// 各ケースで一部だけ壊すための正常なArticle管理入力を返す。
func validAdminArticleRequest(change func(*AdminArticleRequest)) AdminArticleRequest {
	category := "brewing"
	request := AdminArticleRequest{
		Title:    "Valid Article",
		Slug:     "valid-article",
		Summary:  "Valid summary",
		Category: &category,
	}
	if change != nil {
		change(&request)
	}
	return request
}
