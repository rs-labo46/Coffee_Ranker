package migrate

import (
	"strings"
	"testing"
	"time"

	"coffee-ranker/entity"
)

// Seed用Beanが100件生成され、公開・味覚スコア・URLがレビュー可能な状態であることを確認する。
// デモやE2Eで候補不足にならないよう、DB投入前の生成ロジックを単体で固定する。
func TestBuildSeedBeansCreatesReviewableSeedData(t *testing.T) {
	beans := buildSeedBeans()
	if len(beans) != seedBeanCount {
		t.Fatalf("bean count = %d, want %d", len(beans), seedBeanCount)
	}

	names := make(map[string]struct{}, len(beans))
	for _, bean := range beans {
		if bean.Name == "" {
			t.Fatal("bean name is empty")
		}
		if _, ok := names[bean.Name]; ok {
			t.Fatalf("duplicate bean name: %s", bean.Name)
		}
		names[bean.Name] = struct{}{}
		if !bean.IsPublished {
			t.Fatalf("bean %s is not published", bean.Name)
		}
		if !validSeedRoastLevel(bean.RoastLevel) {
			t.Fatalf("invalid roast level: %s", bean.RoastLevel)
		}
		for _, score := range []*int{bean.Acidity, bean.Bitterness, bean.Flavor, bean.Aroma, bean.Body} {
			if score == nil || *score < 1 || *score > 5 {
				t.Fatalf("invalid taste score on %s: %v", bean.Name, score)
			}
		}
		if bean.ImageURL == nil || !strings.HasPrefix(*bean.ImageURL, "https://") {
			t.Fatalf("invalid image url on %s: %v", bean.Name, bean.ImageURL)
		}
	}
}

// Seed用Articleが100件生成され、slug重複なしで公開済みになることを確認する。
// Article詳細・検索・ランキングE2Eでslug重複や非公開混入が起きないようにする。
func TestBuildSeedArticlesCreatesUniquePublishedSlugs(t *testing.T) {
	articles := buildSeedArticles(time.Now())
	if len(articles) != seedArticleCount {
		t.Fatalf("article count = %d, want %d", len(articles), seedArticleCount)
	}

	slugs := make(map[string]struct{}, len(articles))
	for _, article := range articles {
		if article.Title == "" || article.Slug == "" || article.Summary == "" {
			t.Fatalf("article has empty required field: %+v", article)
		}
		if _, ok := slugs[article.Slug]; ok {
			t.Fatalf("duplicate article slug: %s", article.Slug)
		}
		slugs[article.Slug] = struct{}{}
		if !article.IsPublished || article.PublishedAt == nil {
			t.Fatalf("article is not publishable: %+v", article)
		}
		if article.Category == nil || !validSeedArticleCategory(*article.Category) {
			t.Fatalf("invalid category: %v", article.Category)
		}
		if article.Body == nil || *article.Body == "" {
			t.Fatalf("article body is empty: %s", article.Slug)
		}
	}
}

// Seed Beanの焙煎度がEntity enumの範囲内か確認する。
func validSeedRoastLevel(value entity.RoastLevel) bool {
	switch value {
	case entity.RoastLevelLight, entity.RoastLevelMedium, entity.RoastLevelDark:
		return true
	default:
		return false
	}
}

// Seed Articleのカテゴリが検索Validatorの許可値と一致するか確認する。
func validSeedArticleCategory(value string) bool {
	switch value {
	case "brewing", "roast", "beans", "recipe":
		return true
	default:
		return false
	}
}
