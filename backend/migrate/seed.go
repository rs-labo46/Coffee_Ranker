package migrate

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	seedBeanCount    = 100
	seedArticleCount = 100
)

// Seedは開発・Postman確認に必要な初期データを作成する。
// 認証確認用Admin、公開済みBean/Article、ランキング対象、初期metricsを冪等に投入する。
func Seed(db *gorm.DB) error {
	admin, err := SeedAdmin(db)
	if err != nil {
		return err
	}

	if err := SeedContent(db, admin); err != nil {
		return err
	}

	return nil
}

// SeedAdminはSEED_ADMIN_EMAILとSEED_ADMIN_PASSWORDがある場合だけ管理者を作成する。
// 既存emailがある場合はrole/status/passwordをseed値へ揃える。
func SeedAdmin(db *gorm.DB) (*model.User, error) {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("SEED_ADMIN_EMAIL")))
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return nil, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var user model.User
	err = db.Where("email = ?", email).First(&user).Error
	if err == nil {
		user.Name = "Seed Admin"
		user.PasswordHash = string(hash)
		user.Role = entity.UserRoleAdmin
		user.Status = entity.UserStatusActive
		if err := db.Save(&user).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user = model.User{
		Name:         "Seed Admin",
		Email:        email,
		PasswordHash: string(hash),
		Role:         entity.UserRoleAdmin,
		Status:       entity.UserStatusActive,
		TokenVersion: 0,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// SeedContentは画面確認・検索・ランキング・推薦確認に必要な公開コンテンツを作成する。
func SeedContent(db *gorm.DB, admin *model.User) error {
	now := time.Now()

	beans, err := seedBeans(db)
	if err != nil {
		return err
	}

	articles, err := seedArticles(db, now)
	if err != nil {
		return err
	}

	if err := seedBeanArticles(db, beans, articles); err != nil {
		return err
	}

	targets, err := seedRankTargets(db, beans, articles)
	if err != nil {
		return err
	}

	if err := seedContentMetrics(db, targets, now); err != nil {
		return err
	}

	if admin != nil {
		if err := seedAdminInterestProfiles(db, admin.ID, now); err != nil {
			return err
		}
	}

	return nil
}

func seedBeans(db *gorm.DB) ([]*model.Bean, error) {
	items := buildSeedBeans()
	for _, item := range items {
		if err := upsertBean(db, item); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func buildSeedBeans() []*model.Bean {
	type beanPattern struct {
		namePrefix string
		origin     string
		region     string
		variety    string
		roast      entity.RoastLevel
		acidity    int
		bitterness int
		flavor     int
		aroma      int
		body       int
		note       string
	}

	patterns := []beanPattern{
		{"Ethiopia Yirgacheffe", "Ethiopia", "Yirgacheffe", "Heirloom", entity.RoastLevelLight, 5, 1, 5, 5, 2, "jasmine, lemon, peach"},
		{"Brazil Cerrado", "Brazil", "Cerrado", "Mundo Novo", entity.RoastLevelMedium, 2, 3, 3, 3, 4, "nuts, chocolate, caramel"},
		{"Guatemala Antigua", "Guatemala", "Antigua", "Bourbon", entity.RoastLevelMedium, 4, 3, 4, 4, 3, "orange, cocoa, spice"},
		{"Indonesia Mandheling", "Indonesia", "Sumatra", "Typica", entity.RoastLevelDark, 1, 5, 3, 4, 5, "earthy, dark chocolate, herbs"},
		{"Kenya Nyeri", "Kenya", "Nyeri", "SL28", entity.RoastLevelLight, 5, 2, 5, 4, 3, "blackcurrant, grapefruit, syrup"},
		{"Colombia Huila", "Colombia", "Huila", "Caturra", entity.RoastLevelMedium, 4, 2, 4, 4, 3, "apple, brown sugar, citrus"},
		{"Costa Rica Tarrazu", "Costa Rica", "Tarrazu", "Catuai", entity.RoastLevelMedium, 4, 2, 4, 4, 3, "honey, orange, almond"},
		{"Rwanda Huye", "Rwanda", "Huye", "Bourbon", entity.RoastLevelLight, 5, 2, 4, 4, 3, "berry, tea, lemon"},
		{"Honduras Marcala", "Honduras", "Marcala", "Pacas", entity.RoastLevelMedium, 3, 3, 4, 3, 4, "cacao, pear, nuts"},
		{"Peru Cajamarca", "Peru", "Cajamarca", "Typica", entity.RoastLevelMedium, 3, 2, 4, 3, 3, "floral, milk chocolate, citrus"},
		{"Mexico Chiapas", "Mexico", "Chiapas", "Bourbon", entity.RoastLevelDark, 2, 4, 3, 3, 4, "dark chocolate, walnut, spice"},
		{"Tanzania Kilimanjaro", "Tanzania", "Kilimanjaro", "Kent", entity.RoastLevelLight, 5, 2, 4, 4, 3, "citrus, winey, caramel"},
	}

	roasters := []string{
		"Coffee Ranker Roasters",
		"Seed Lab Coffee",
		"Ranker Daily Roast",
		"Clean Cup Works",
	}

	items := make([]*model.Bean, 0, seedBeanCount)
	for i := 0; i < seedBeanCount; i++ {
		pattern := patterns[i%len(patterns)]
		lot := i + 1
		name := fmt.Sprintf("%s Lot %03d", pattern.namePrefix, lot)
		if i == 0 {
			name = "Ethiopia Yirgacheffe Light Roast"
		}
		if i == 1 {
			name = "Brazil Cerrado Medium Roast"
		}
		if i == 2 {
			name = "Guatemala Antigua Medium Roast"
		}
		if i == 3 {
			name = "Indonesia Mandheling Dark Roast"
		}

		items = append(items, &model.Bean{
			Name:        name,
			Roaster:     strPtr(roasters[i%len(roasters)]),
			Origin:      strPtr(pattern.origin),
			Region:      strPtr(pattern.region),
			Farm:        strPtr(fmt.Sprintf("Seed Farm %03d", lot)),
			Variety:     strPtr(pattern.variety),
			RoastLevel:  pattern.roast,
			Acidity:     intPtr(rotateTaste(pattern.acidity, i)),
			Bitterness:  intPtr(rotateTaste(pattern.bitterness, i/2)),
			Flavor:      intPtr(rotateTaste(pattern.flavor, i/3)),
			Aroma:       intPtr(rotateTaste(pattern.aroma, i/4)),
			Body:        intPtr(rotateTaste(pattern.body, i/5)),
			FlavorNote:  strPtr(pattern.note),
			Description: strPtr(fmt.Sprintf("検索、ランキング、推薦確認に使うSeed用Bean。産地=%s、焙煎度=%s、ロット=%03d。", pattern.origin, pattern.roast, lot)),
			ImageURL:    strPtr(fmt.Sprintf("https://example.com/images/beans/seed-bean-%03d.jpg", lot)),
			IsPublished: true,
		})
	}
	return items
}

func seedArticles(db *gorm.DB, now time.Time) ([]*model.Article, error) {
	items := buildSeedArticles(now)
	for _, item := range items {
		if err := upsertArticle(db, item); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func buildSeedArticles(now time.Time) []*model.Article {
	type articlePattern struct {
		titlePrefix string
		slugPrefix  string
		category    string
		summary     string
		body        string
	}

	patterns := []articlePattern{
		{"ハンドドリップの基本手順", "pour-over-basic", "brewing", "抽出レシピ確認用のSeed記事。湯温、粉量、注ぎ方を扱う。", "ハンドドリップは粉量、湯温、注湯回数を揃えると再現性が上がる。"},
		{"焙煎度で変わる味の違い", "roast-level-guide", "roast", "浅煎り、中煎り、深煎りの違いを確認するSeed記事。", "浅煎りは酸味や香り、中煎りはバランス、深煎りは苦味とコクが出やすい。"},
		{"酸味が好きな人向けの選び方", "coffee-acidity-guide", "beans", "酸味軸の推薦理由確認に使うSeed記事。", "酸味を楽しみたい場合は、エチオピアやケニアなどの浅煎りを候補にしやすい。"},
		{"自宅で豆を保存する方法", "home-coffee-storage", "recipe", "保存・記事一覧確認に使うSeed記事。", "豆は密閉し、光と湿気を避けて保存する。短期間で飲み切る量を買うのが安全。"},
		{"フレンチプレスの抽出設計", "french-press-brewing", "brewing", "浸漬式の抽出確認に使うSeed記事。", "フレンチプレスは粒度を粗めにし、抽出時間を固定すると味の再現性が上がる。"},
		{"浅煎り豆の扱い方", "light-roast-brewing", "roast", "浅煎り検索と推薦確認に使うSeed記事。", "浅煎りは湯温を高めにし、粉全体に均一に湯を行き渡らせると成分を出しやすい。"},
		{"中煎り豆の選び方", "medium-roast-selection", "beans", "中煎り検索確認に使うSeed記事。", "中煎りは酸味と甘さのバランスが取りやすく、日常用の候補にしやすい。"},
		{"深煎りに合うレシピ", "dark-roast-recipe", "recipe", "深煎りと苦味軸確認に使うSeed記事。", "深煎りは短めの抽出やミルク合わせで、苦味とコクのバランスを取りやすい。"},
	}

	items := make([]*model.Article, 0, seedArticleCount)
	for i := 0; i < seedArticleCount; i++ {
		pattern := patterns[i%len(patterns)]
		number := i + 1
		title := fmt.Sprintf("%s %03d", pattern.titlePrefix, number)
		slug := fmt.Sprintf("%s-%03d", pattern.slugPrefix, number)
		if i < len(patterns) {
			title = pattern.titlePrefix
			slug = pattern.slugPrefix
		}
		publishedAt := now.Add(-time.Duration(i) * 6 * time.Hour)

		items = append(items, &model.Article{
			Title:       title,
			Slug:        slug,
			Summary:     fmt.Sprintf("%s No.%03d", pattern.summary, number),
			Body:        strPtr(fmt.Sprintf("%s Seed記事番号%03d。検索、ランキング、推薦、関連記事確認に使う。", pattern.body, number)),
			Category:    strPtr(pattern.category),
			SourceName:  strPtr("Coffee Ranker"),
			SourceURL:   strPtr(fmt.Sprintf("https://example.com/articles/%s", slug)),
			ImageURL:    strPtr(fmt.Sprintf("https://example.com/images/articles/seed-article-%03d.jpg", number)),
			IsPublished: true,
			PublishedAt: &publishedAt,
		})
	}
	return items
}

func seedBeanArticles(db *gorm.DB, beans []*model.Bean, articles []*model.Article) error {
	if len(beans) == 0 || len(articles) == 0 {
		return nil
	}

	for index, bean := range beans {
		articleIndexes := []int{
			index % len(articles),
			(index + len(articles)/3) % len(articles),
			(index + len(articles)*2/3) % len(articles),
		}
		for displayOrder, articleIndex := range articleIndexes {
			relation := &model.BeanArticle{
				BeanID:       bean.ID,
				ArticleID:    articles[articleIndex].ID,
				DisplayOrder: displayOrder + 1,
			}
			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "bean_id"}, {Name: "article_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"display_order"}),
			}).Create(relation).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedRankTargets(db *gorm.DB, beans []*model.Bean, articles []*model.Article) ([]*model.RankTarget, error) {
	targets := make([]*model.RankTarget, 0, len(beans)+len(articles))

	for _, bean := range beans {
		target, err := upsertRankTarget(db, entity.ContentTypeBean, bean.ID)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	for _, article := range articles {
		target, err := upsertRankTarget(db, entity.ContentTypeArticle, article.ID)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}

	return targets, nil
}

func seedContentMetrics(db *gorm.DB, targets []*model.RankTarget, now time.Time) error {
	periodEnd := now
	periodStart := now.Add(-24 * time.Hour)

	for index, target := range targets {
		impressionCount := int64(120 + index*7)
		contentViewCount := int64(40 + index*3)
		clickCount := int64(12 + index%30)
		saveCount := int64(4 + index%18)
		ratingCount := int64(8 + index%22)
		goodCount := int64(5 + index%18)
		if goodCount > ratingCount {
			goodCount = ratingCount
		}
		badCount := ratingCount - goodCount
		modalImpressionCount := int64(6 + index%20)
		modalClickCount := int64(1 + index%6)
		if modalClickCount > modalImpressionCount {
			modalClickCount = modalImpressionCount
		}
		modalCloseCount := modalImpressionCount - modalClickCount

		metric := &model.ContentMetric{
			RankTargetID:         target.ID,
			Score:                1000 - float64(index*3) + float64(index%11),
			ImpressionCount:      impressionCount,
			ContentViewCount:     contentViewCount,
			ClickCount:           clickCount,
			StayTotalMs:          180000 + int64(index%40)*15000,
			SaveCount:            saveCount,
			RatingCount:          ratingCount,
			GoodCount:            goodCount,
			BadCount:             badCount,
			ReSearchCount:        int64(index % 9),
			RatingAvg:            safeRate(goodCount-badCount, ratingCount),
			GoodRate:             safeRate(goodCount, ratingCount),
			BadRate:              safeRate(badCount, ratingCount),
			ModalImpressionCount: modalImpressionCount,
			ModalClickCount:      modalClickCount,
			ModalCloseCount:      modalCloseCount,
			ClickRate:            safeRate(clickCount, impressionCount),
			SaveRate:             safeRate(saveCount, contentViewCount),
			ReSearchRate:         safeRate(int64(index%9), contentViewCount),
			ModalClickRate:       safeRate(modalClickCount, modalImpressionCount),
			ModalCloseRate:       safeRate(modalCloseCount, modalImpressionCount),
			PeriodStart:          periodStart,
			PeriodEnd:            periodEnd,
			CalculatedAt:         now,
		}
		if err := upsertContentMetric(db, metric); err != nil {
			return err
		}
	}
	return nil
}

func seedAdminInterestProfiles(db *gorm.DB, userID uint64, now time.Time) error {
	expiresAt := now.Add(30 * 24 * time.Hour)
	profiles := []*model.InterestProfile{
		{UserID: &userID, Dimension: entity.InterestDimensionOrigin, Value: "Ethiopia", Score: 10, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionOrigin, Value: "Kenya", Score: 8, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionOrigin, Value: "Colombia", Score: 7, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionRoastLevel, Value: string(entity.RoastLevelLight), Score: 8, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionRoastLevel, Value: string(entity.RoastLevelMedium), Score: 6, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionAcidity, Value: "5", Score: 6, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionFlavor, Value: "5", Score: 5, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionArticleCategory, Value: "brewing", Score: 5, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionArticleCategory, Value: "roast", Score: 4, LastEventAt: now, ExpiresAt: &expiresAt},
		{UserID: &userID, Dimension: entity.InterestDimensionArticleCategory, Value: "beans", Score: 4, LastEventAt: now, ExpiresAt: &expiresAt},
	}

	for _, profile := range profiles {
		if err := upsertUserInterestProfile(db, profile); err != nil {
			return err
		}
	}
	return nil
}

func upsertBean(db *gorm.DB, bean *model.Bean) error {
	var existing model.Bean
	err := db.Where("name = ?", bean.Name).First(&existing).Error
	if err == nil {
		bean.ID = existing.ID
		existing.Roaster = bean.Roaster
		existing.Origin = bean.Origin
		existing.Region = bean.Region
		existing.Farm = bean.Farm
		existing.Variety = bean.Variety
		existing.RoastLevel = bean.RoastLevel
		existing.Acidity = bean.Acidity
		existing.Bitterness = bean.Bitterness
		existing.Flavor = bean.Flavor
		existing.Aroma = bean.Aroma
		existing.Body = bean.Body
		existing.FlavorNote = bean.FlavorNote
		existing.Description = bean.Description
		existing.ImageURL = bean.ImageURL
		existing.IsPublished = bean.IsPublished
		return db.Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(bean).Error
}

func upsertArticle(db *gorm.DB, article *model.Article) error {
	var existing model.Article
	err := db.Where("slug = ?", article.Slug).First(&existing).Error
	if err == nil {
		article.ID = existing.ID
		existing.Title = article.Title
		existing.Summary = article.Summary
		existing.Body = article.Body
		existing.Category = article.Category
		existing.SourceName = article.SourceName
		existing.SourceURL = article.SourceURL
		existing.ImageURL = article.ImageURL
		existing.IsPublished = article.IsPublished
		existing.PublishedAt = article.PublishedAt
		return db.Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(article).Error
}

func upsertRankTarget(db *gorm.DB, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error) {
	target := &model.RankTarget{ContentType: contentType, ContentID: contentID, IsActive: true}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "content_type"}, {Name: "content_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"is_active"}),
	}).Create(target).Error; err != nil {
		return nil, err
	}

	var saved model.RankTarget
	if err := db.Where("content_type = ? AND content_id = ?", contentType, contentID).First(&saved).Error; err != nil {
		return nil, err
	}
	return &saved, nil
}

func upsertContentMetric(db *gorm.DB, metric *model.ContentMetric) error {
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "rank_target_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"score",
			"impression_count",
			"content_view_count",
			"click_count",
			"stay_total_ms",
			"save_count",
			"rating_count",
			"good_count",
			"bad_count",
			"re_search_count",
			"rating_avg",
			"good_rate",
			"bad_rate",
			"modal_impression_count",
			"modal_click_count",
			"modal_close_count",
			"click_rate",
			"save_rate",
			"re_search_rate",
			"modal_click_rate",
			"modal_close_rate",
			"period_start",
			"period_end",
			"calculated_at",
		}),
	}).Create(metric).Error
}

func upsertUserInterestProfile(db *gorm.DB, profile *model.InterestProfile) error {
	if profile.UserID == nil {
		return nil
	}

	var existing model.InterestProfile
	err := db.Where("user_id = ? AND guest_session_id IS NULL AND dimension = ? AND value = ?", *profile.UserID, profile.Dimension, profile.Value).First(&existing).Error
	if err == nil {
		profile.ID = existing.ID
		existing.Score = profile.Score
		existing.LastEventAt = profile.LastEventAt
		existing.ExpiresAt = profile.ExpiresAt
		return db.Save(&existing).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(profile).Error
}

func rotateTaste(base int, offset int) int {
	value := ((base + offset - 1) % 5) + 1
	if value < 1 {
		return 1
	}
	if value > 5 {
		return 5
	}
	return value
}

func safeRate(numerator int64, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}
