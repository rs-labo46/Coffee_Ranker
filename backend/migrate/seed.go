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
// 認証動作用Admin、公開済みBean/Article、ランキング対象、初期metricsを冪等に投入する。
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
		user.Name = "Development Admin"
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
		Name:         "Development Admin",
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

// SeedContentは開発環境に必要な公開コンテンツ、ランキング対象、初期metricsを作成する。
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
		farm       string
		variety    string
		roast      entity.RoastLevel
		acidity    int
		bitterness int
		flavor     int
		aroma      int
		body       int
		note       string
		detail     string
		imageURL   string
	}

	patterns := []beanPattern{
		{"Ethiopia Yirgacheffe", "Ethiopia", "Yirgacheffe", "Gedeb Smallholders", "Heirloom", entity.RoastLevelLight, 5, 1, 5, 5, 2, "jasmine, lemon, peach", "ジャスミンのような香りとレモンの明るさが出やすい浅煎り。軽い口当たりで、朝の一杯や作業前のリセットに向く。", "https://images.unsplash.com/photo-1447933601403-0c6688de566e?auto=format&fit=crop&w=1200&q=80"},
		{"Brazil Cerrado", "Brazil", "Cerrado", "Fazenda Primavera", "Mundo Novo", entity.RoastLevelMedium, 2, 3, 3, 3, 4, "nuts, chocolate, caramel", "ナッツ、チョコレート、キャラメルの甘さを感じやすい中煎り。酸味が控えめで、毎日飲む定番として扱いやすい。", "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=1200&q=80"},
		{"Guatemala Antigua", "Guatemala", "Antigua", "Finca La Soledad", "Bourbon", entity.RoastLevelMedium, 4, 3, 4, 4, 3, "orange, cocoa, spice", "オレンジのような酸味とココアの甘さが重なるバランス型。香りとコクの両方があり、食後の一杯にも合わせやすい。", "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?auto=format&fit=crop&w=1200&q=80"},
		{"Indonesia Mandheling", "Indonesia", "Sumatra", "Lintong Estate", "Typica", entity.RoastLevelDark, 1, 5, 3, 4, 5, "earthy, dark chocolate, herbs", "深いコクとハーブ感、ダークチョコの余韻が出やすい深煎り。ミルクと合わせても輪郭が残りやすく、夜の一杯にも向く。", "https://images.unsplash.com/photo-1509042239860-f550ce710b93?auto=format&fit=crop&w=1200&q=80"},
		{"Kenya Nyeri", "Kenya", "Nyeri", "Gichathaini Factory", "SL28", entity.RoastLevelLight, 5, 2, 5, 4, 3, "blackcurrant, grapefruit, syrup", "カシスやグレープフルーツのような輪郭が強い浅煎り。明るい酸味を楽しみたい人に向く。", "https://images.unsplash.com/photo-1459755486867-b55449bb39ff?auto=format&fit=crop&w=1200&q=80"},
		{"Colombia Huila", "Colombia", "Huila", "Finca El Diviso", "Caturra", entity.RoastLevelMedium, 4, 2, 4, 4, 3, "apple, brown sugar, citrus", "りんご、ブラウンシュガー、柑橘の印象を持つ中煎り。甘さと酸味のバランスがあり、軽い余韻を楽しみやすい。", "https://images.unsplash.com/photo-1494314671902-399b18174975?auto=format&fit=crop&w=1200&q=80"},
		{"Costa Rica Tarrazu", "Costa Rica", "Tarrazu", "La Pastora", "Catuai", entity.RoastLevelMedium, 4, 2, 4, 4, 3, "honey, orange, almond", "はちみつ、オレンジ、アーモンドのようなまとまりがある豆。苦味を抑えつつ、甘さのある一杯にしやすい。", "https://images.unsplash.com/photo-1511537190424-bbbab87ac5eb?auto=format&fit=crop&w=1200&q=80"},
		{"Rwanda Huye", "Rwanda", "Huye", "Huye Mountain", "Bourbon", entity.RoastLevelLight, 5, 2, 4, 4, 3, "berry, tea, lemon", "ベリー、紅茶、レモンのような軽さが出やすい浅煎り。香りをゆっくり楽しみたい人に向く。", "https://images.unsplash.com/photo-1461988320302-91bde64fc8e4?auto=format&fit=crop&w=1200&q=80"},
		{"Honduras Marcala", "Honduras", "Marcala", "Las Flores", "Pacas", entity.RoastLevelMedium, 3, 3, 4, 3, 4, "cacao, pear, nuts", "カカオ、洋梨、ナッツの印象がある中煎り。酸味も苦味も強すぎず、毎日の一杯として選びやすい。", "https://images.unsplash.com/photo-1521302080334-4bebac2763a6?auto=format&fit=crop&w=1200&q=80"},
		{"Peru Cajamarca", "Peru", "Cajamarca", "El Diamante", "Typica", entity.RoastLevelMedium, 3, 2, 4, 3, 3, "floral, milk chocolate, citrus", "花のような香り、ミルクチョコ、柑橘の穏やかさがある豆。軽さと甘さのバランスを求める日に合う。", "https://images.unsplash.com/photo-1442512595331-e89e73853f31?auto=format&fit=crop&w=1200&q=80"},
		{"Mexico Chiapas", "Mexico", "Chiapas", "Finca Santa Cruz", "Bourbon", entity.RoastLevelDark, 2, 4, 3, 3, 4, "dark chocolate, walnut, spice", "ダークチョコ、くるみ、スパイスの印象がある深煎り。苦味とコクをしっかり楽しみたい人に向く。", "https://images.unsplash.com/photo-1511920170033-f8396924c348?auto=format&fit=crop&w=1200&q=80"},
		{"Tanzania Kilimanjaro", "Tanzania", "Kilimanjaro", "Machare Estate", "Kent", entity.RoastLevelLight, 5, 2, 4, 4, 3, "citrus, winey, caramel", "柑橘、ワイニー、キャラメルの余韻が出やすい浅煎り。すっきりした後味を好む日に選びやすい。", "https://images.unsplash.com/photo-1461023058943-07fcbe16d735?auto=format&fit=crop&w=1200&q=80"},
	}

	roasters := []string{
		"Harbor Light Roasters",
		"North Hill Coffee",
		"Daily Cup Roastery",
		"Clean Cup Works",
		"Quiet Roast Studio",
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

		description := fmt.Sprintf("%s 産地:%s、焙煎度:%s。", pattern.detail, pattern.origin, pattern.roast)

		items = append(items, &model.Bean{
			Name:        name,
			Roaster:     strPtr(roasters[i%len(roasters)]),
			Origin:      strPtr(pattern.origin),
			Region:      strPtr(pattern.region),
			Farm:        strPtr(pattern.farm),
			Variety:     strPtr(pattern.variety),
			RoastLevel:  pattern.roast,
			Acidity:     intPtr(rotateTaste(pattern.acidity, i)),
			Bitterness:  intPtr(rotateTaste(pattern.bitterness, i/2)),
			Flavor:      intPtr(rotateTaste(pattern.flavor, i/3)),
			Aroma:       intPtr(rotateTaste(pattern.aroma, i/4)),
			Body:        intPtr(rotateTaste(pattern.body, i/5)),
			FlavorNote:  strPtr(pattern.note),
			Description: strPtr(description),
			ImageURL:    strPtr(pattern.imageURL),
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
		imageURL    string
	}

	patterns := []articlePattern{
		{
			titlePrefix: "ハンドドリップの基本手順",
			slugPrefix:  "pour-over-basic",
			category:    "brewing",
			summary:     "粉量、湯温、注ぎ方をそろえて、家庭でも味を再現しやすくするための基本手順。",
			body:        "ハンドドリップは、粉量、湯量、挽き目、湯温、抽出時間をそろえるほど味が安定する。最初は粉15gに湯225g、湯温90〜93度、抽出時間2分30秒前後を基準にすると調整しやすい。蒸らしでは粉全体に湯を行き渡らせ、炭酸ガスを抜いてから本抽出へ入る。注ぐときは中心だけに湯を落とし続けず、粉全体が均一に湿る範囲で小さく円を描く。味が薄ければ挽きを少し細かくし、苦味が重ければ湯温を下げるか抽出時間を短くすると原因を切り分けやすい。",
			imageURL:    "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "焙煎度で変わる味の違い",
			slugPrefix:  "roast-level-guide",
			category:    "roast",
			summary:     "浅煎り、中煎り、深煎りで酸味、苦味、香り、コクがどう変わるかを整理する。",
			body:        "焙煎度は、豆選びで最初に見るべき大きな軸になる。浅煎りは果実感や花の香りが出やすく、軽い口当たりと明るい酸味を楽しみやすい。中煎りは酸味、甘さ、苦味のバランスが取りやすく、初めて買う豆や毎日飲む豆として扱いやすい。深煎りは苦味、コク、チョコレートのような余韻が出やすく、ミルクや氷と合わせても味がぼやけにくい。同じ産地でも焙煎度が変わると印象は大きく変わるため、好みが固まらないうちは焙煎度を変えて飲み比べると選び方が早く身につく。",
			imageURL:    "https://images.unsplash.com/photo-1447933601403-0c6688de566e?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "酸味が好きな人向けの豆選び",
			slugPrefix:  "coffee-acidity-guide",
			category:    "beans",
			summary:     "明るい酸味を楽しみたい人向けに、産地、焙煎度、フレーバーノートの見方をまとめる。",
			body:        "明るい酸味を楽しみたい場合は、浅煎りのエチオピア、ケニア、ルワンダ、コスタリカなどから探すと候補を絞りやすい。フレーバーノートにレモン、ベリー、カシス、ピーチ、紅茶のような表現がある豆は、軽やかな印象になりやすい。酸味は抽出でも変わるため、細かく挽きすぎたり、長く抽出しすぎたりすると尖って感じることがある。まずは中挽き、高めの湯温、短すぎない抽出時間で飲み、薄い場合だけ挽きを細かくする。冷めるにつれて甘さや果実感が出る豆も多いため、熱い時だけで判断しないことも大切になる。",
			imageURL:    "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "自宅で豆を保存する方法",
			slugPrefix:  "home-coffee-storage",
			category:    "recipe",
			summary:     "豆の香りを落としにくくするための保存場所、容器、飲み切る量の考え方。",
			body:        "コーヒー豆は、光、空気、湿気、熱の影響を受けやすい。家庭では密閉できる容器に入れ、直射日光の当たらない涼しい場所に置くのが扱いやすい。袋のまま保存する場合も、口をしっかり閉じて空気に触れる時間を減らすだけで香りの落ち方は変わる。大量に買うと飲み切る前に香りが弱くなりやすいため、まずは2週間から1か月で使い切れる量を選ぶと失敗しにくい。冷凍する場合は小分けにして、取り出した豆に結露がつかないよう常温に戻してから開封すると品質を保ちやすい。",
			imageURL:    "https://images.unsplash.com/photo-1459755486867-b55449bb39ff?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "フレンチプレスの抽出設計",
			slugPrefix:  "french-press-brewing",
			category:    "brewing",
			summary:     "浸漬式で味を安定させるための粒度、湯温、時間、撹拌の考え方。",
			body:        "フレンチプレスは粉と湯を一定時間触れさせる浸漬式の抽出方法で、手順が単純なぶん豆の質感が出やすい。細かく挽きすぎると粉っぽさや渋さが出やすいため、中粗挽きから粗挽きで始めると調整しやすい。抽出時間は4分前後を基準にし、味が薄ければ粉量を増やすか挽きを少し細かくする。撹拌を強くしすぎると雑味が出ることがあるため、湯を注いだ直後に軽くなじませる程度でよい。プレス後は長く置かず、別のサーバーへ移すと過抽出を避けられる。",
			imageURL:    "https://images.unsplash.com/photo-1461988320302-91bde64fc8e4?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "浅煎り豆の扱い方",
			slugPrefix:  "light-roast-brewing",
			category:    "roast",
			summary:     "浅煎りの香りと酸味を出しやすくするための湯温、挽き目、抽出時間の調整。",
			body:        "浅煎り豆は密度が高く、同じ条件でも成分が出にくいことがある。味が薄い場合は、湯温を上げる、挽きを少し細かくする、抽出時間を少し長くするという順番で調整するとよい。酸味が強すぎる場合は、粉量を増やす前に注ぎ方を穏やかにし、湯が粉全体を均一に通っているかを見る。浅煎りは熱い状態よりも少し温度が下がった時に甘さや香りが開くことがある。果実感を楽しみたいなら、深い苦味を出しにいくより、香りと甘さを残す抽出を意識する。",
			imageURL:    "https://images.unsplash.com/photo-1521302080334-4bebac2763a6?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "中煎り豆の選び方",
			slugPrefix:  "medium-roast-selection",
			category:    "beans",
			summary:     "酸味と苦味のバランスを取りたい人向けに、中煎り豆の選び方を整理する。",
			body:        "中煎りは、酸味、甘さ、苦味、コクのバランスを取りやすい焙煎度。初めて買う豆や毎日飲む豆を探す場合は、中煎りから選ぶと失敗しにくい。ナッツ、チョコレート、ブラウンシュガー、オレンジなどの表現がある豆は、飲みやすさと個性の両方を楽しみやすい。ブラックで飲むなら香りと甘さのある豆、ミルクを少し入れるならコクのある豆を選ぶと満足度が上がる。抽出は極端に高温へ寄せず、90度前後から始めると味の輪郭をつかみやすい。",
			imageURL:    "https://images.unsplash.com/photo-1442512595331-e89e73853f31?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "深煎りに合う抽出レシピ",
			slugPrefix:  "dark-roast-recipe",
			category:    "recipe",
			summary:     "深煎りの苦味とコクを活かしつつ、重くなりすぎない抽出レシピを考える。",
			body:        "深煎り豆は苦味とコクが出やすく、抽出を強くしすぎると重たい印象になりやすい。湯温は86〜90度から始め、抽出時間を短めにすると焦げ感や渋さを抑えやすい。アイスコーヒーやカフェオレでは、深煎りの強さが氷やミルクに負けにくく、味の輪郭が残りやすい。甘さを出したい場合は、粉量を増やすよりも抽出後半の注湯を穏やかにすると余韻がきれいにまとまりやすい。苦味だけで選ばず、チョコレート、ナッツ、黒糖のような甘い表現がある豆を選ぶと飲み疲れしにくい。",
			imageURL:    "https://images.unsplash.com/photo-1511920170033-f8396924c348?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "アイスコーヒーを薄くしない考え方",
			slugPrefix:  "iced-coffee-balance",
			category:    "recipe",
			summary:     "氷で薄まる前提で、粉量、湯量、冷却の速度を決めるアイスコーヒーの作り方。",
			body:        "アイスコーヒーは氷で薄まるため、ホットと同じ濃度で抽出すると味がぼやけやすい。急冷式では、抽出に使う湯量を減らし、サーバーに氷を入れてすぐ冷やすと香りを残しやすい。粉量はホットより少し多め、挽き目は中挽きから中細挽きで始めると濃度を作りやすい。深煎りは苦味とコクが残りやすく、浅煎りは果実感のある軽いアイスに仕上げやすい。飲む前に氷が溶ける時間も考え、抽出直後に味が少し濃いと感じるくらいがちょうどよい。",
			imageURL:    "https://images.unsplash.com/photo-1461023058943-07fcbe16d735?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "朝に飲みやすい豆の選び方",
			slugPrefix:  "morning-coffee-selection",
			category:    "beans",
			summary:     "朝の一杯に向く軽さ、甘さ、後味を軸にした豆選び。",
			body:        "朝に飲むコーヒーは、重すぎず、後味が長く残りすぎない豆が扱いやすい。浅煎りなら柑橘や紅茶のような軽さがある豆、中煎りならナッツやブラウンシュガーのような甘さがある豆を選ぶと飲みやすい。寝起きに強い苦味がつらい場合は、深煎りよりも中煎りから始めると負担が少ない。パンやヨーグルトに合わせるなら、酸味が少しある豆の方が食事と馴染みやすい。朝は抽出に時間をかけにくいため、再現しやすいレシピを一つ決めておくと失敗が減る。",
			imageURL:    "https://images.unsplash.com/photo-1511537190424-bbbab87ac5eb?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "ミルクに合うコーヒーの条件",
			slugPrefix:  "coffee-with-milk",
			category:    "beans",
			summary:     "カフェオレやラテで味がぼやけにくい豆の焙煎度、コク、甘さの見方。",
			body:        "ミルクに合わせるコーヒーは、コクと甘さがある豆を選ぶと味がぼやけにくい。深煎りはミルクに負けにくいが、焦げ感が強すぎると後味が重くなることがある。中深煎りでチョコレート、キャラメル、ナッツのような表現がある豆は、ミルクの甘さと合わせやすい。酸味が強い浅煎りでも、ベリーや赤い果実の印象がある豆はミルクと合う場合がある。抽出は少し濃いめにし、ミルクを入れた後でも香りが残る濃度を目指すとよい。",
			imageURL:    "https://images.unsplash.com/photo-1509042239860-f550ce710b93?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "コーヒーの挽き目を調整する基準",
			slugPrefix:  "grind-size-adjustment",
			category:    "brewing",
			summary:     "味が薄い、苦い、渋いと感じた時に挽き目をどう動かすかを整理する。",
			body:        "挽き目は抽出の速さを左右する重要な要素になる。味が薄く、香りも弱い場合は、成分が十分に出ていない可能性があるため、挽きを少し細かくする。苦味や渋さが強い場合は、成分が出すぎている可能性があるため、挽きを粗くするか抽出時間を短くする。細かくしすぎると湯の通りが悪くなり、ドリッパー内で滞留して雑味が出やすい。調整は一度に大きく変えず、同じ豆と同じ粉量で一段階ずつ動かすと原因を見つけやすい。",
			imageURL:    "https://images.unsplash.com/photo-1447933601403-0c6688de566e?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "湯温で味が変わる理由",
			slugPrefix:  "water-temperature-guide",
			category:    "brewing",
			summary:     "湯温を上げ下げした時に、酸味、甘さ、苦味がどう動くかを考える。",
			body:        "湯温が高いほどコーヒーの成分は出やすくなり、低いほど穏やかに抽出される。浅煎りは成分が出にくいため、92〜95度ほどの高めの湯温から始めると香りや甘さを引き出しやすい。深煎りは成分が出やすいため、86〜90度ほどに下げると苦味や渋さを抑えやすい。中煎りは90度前後から始め、酸味が強ければ少し高く、苦味が重ければ少し低くする。湯温だけを変えると味の変化が見えやすいため、挽き目や粉量を同時に変えないことが大切になる。",
			imageURL:    "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "産地ごとの味の傾向",
			slugPrefix:  "origin-flavor-guide",
			category:    "beans",
			summary:     "エチオピア、ブラジル、コロンビアなど、よく見かける産地の味の傾向。",
			body:        "産地は味を予想するための手がかりになるが、同じ国でも地域、品種、精製、焙煎で印象は変わる。エチオピアは花や柑橘、紅茶のような香りを持つ豆が多く、浅煎りで個性が出やすい。ブラジルはナッツ、チョコレート、穏やかな甘さが出やすく、毎日飲む豆として扱いやすい。コロンビアは果実感と甘さのバランスがよく、焙煎度によって軽さもコクも出せる。産地名だけで決めず、焙煎度とフレーバーノートを合わせて見ると失敗が少なくなる。",
			imageURL:    "https://images.unsplash.com/photo-1494314671902-399b18174975?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "精製方法で変わる香りの出方",
			slugPrefix:  "process-flavor-guide",
			category:    "beans",
			summary:     "ウォッシュト、ナチュラル、ハニー製法の違いと、味の選び方。",
			body:        "精製方法は、コーヒーチェリーから種子を取り出す工程で、香りや口当たりに大きく関わる。ウォッシュトはすっきりした酸味とクリーンな後味になりやすく、豆本来の輪郭をつかみやすい。ナチュラルは果肉と一緒に乾燥させるため、ベリーや熟した果実のような香りが出ることがある。ハニー製法は甘さと質感が出やすく、ウォッシュトとナチュラルの中間のような印象になる場合がある。香りの強さを楽しみたい人はナチュラル、透明感を重視する人はウォッシュトから選ぶとよい。",
			imageURL:    "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "ペーパードリップと金属フィルターの違い",
			slugPrefix:  "paper-metal-filter",
			category:    "brewing",
			summary:     "すっきりした味とオイル感のある味を、フィルターの違いから考える。",
			body:        "ペーパーフィルターは微粉やオイルを受け止めるため、すっきりした口当たりになりやすい。香りの輪郭や酸味をきれいに出したい場合は、ペーパーの方が扱いやすいことが多い。金属フィルターはオイルを通しやすく、豆の質感や厚みを感じやすい。深煎りや中深煎りでは、金属フィルターのコクが心地よく出る場合がある。ただし微粉も入りやすいため、挽き目を粗めにして、注ぎ終わった後に長く置かないことが大切になる。",
			imageURL:    "https://images.unsplash.com/photo-1442512595331-e89e73853f31?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "エアロプレスで味を丸くする方法",
			slugPrefix:  "aeropress-soft-cup",
			category:    "brewing",
			summary:     "短時間抽出でも薄くしないための粉量、攪拌、押し込み方。",
			body:        "エアロプレスは短時間で抽出でき、レシピの自由度が高い器具。味を丸くしたい場合は、粉量を少し多めにして、湯温を90度前後に置くとバランスを取りやすい。攪拌を強くしすぎると苦味や渋さが出ることがあるため、数回なじませる程度から始めるとよい。押し込みは一定の力でゆっくり行い、最後に強く押し切らない方が口当たりは穏やかになる。浅煎りなら湯温を少し上げ、中煎りなら抽出時間を短めにすると甘さを残しやすい。",
			imageURL:    "https://images.unsplash.com/photo-1461988320302-91bde64fc8e4?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "コールドブリューの作り方",
			slugPrefix:  "cold-brew-method",
			category:    "recipe",
			summary:     "水出しで甘さとまろやかさを出すための粉量、時間、保存方法。",
			body:        "コールドブリューは低温で時間をかけて抽出するため、酸味や苦味が穏やかに出やすい。粉は中粗挽きから粗挽きにし、水に長く触れても雑味が出にくい状態を作る。粉1に対して水10〜12ほどを基準にし、冷蔵庫で8〜12時間ほど置くと扱いやすい。抽出後は粉を早めに取り除き、清潔な容器で保存すると味が濁りにくい。ミルクで割る場合は少し濃いめに作り、ブラックで飲む場合は氷が溶ける分を考えて濃度を決める。",
			imageURL:    "https://images.unsplash.com/photo-1511920170033-f8396924c348?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "豆の鮮度と飲み頃の考え方",
			slugPrefix:  "coffee-freshness-guide",
			category:    "beans",
			summary:     "焙煎直後から数週間で香りがどう変わるか、飲み頃の目安を整理する。",
			body:        "焙煎直後の豆は香りが強い一方で、ガスが多く抽出が安定しにくいことがある。浅煎りは数日から1週間ほど置くと香りと甘さがまとまりやすく、中煎りや深煎りは比較的早く飲み始めやすい。袋を開けてからは空気に触れるため、香りは少しずつ弱くなる。毎日飲むなら、開封後2〜3週間で飲み切れる量を選ぶと扱いやすい。豆の状態を知るには、同じレシピで数日おきに飲み、香り、泡立ち、後味の変化を見るとよい。",
			imageURL:    "https://images.unsplash.com/photo-1511537190424-bbbab87ac5eb?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "苦味を抑えたい時の調整方法",
			slugPrefix:  "reduce-bitterness",
			category:    "brewing",
			summary:     "苦味や渋さが強い時に、湯温、挽き目、抽出時間をどう直すか。",
			body:        "苦味が強い時は、成分が出すぎているか、焙煎由来の苦味が好みに合っていない可能性がある。まず湯温を少し下げ、抽出時間を短くしてみると変化がわかりやすい。挽き目が細かすぎる場合は湯の通りが悪くなり、後半に渋さが出やすい。深煎り豆で焦げ感が気になるなら、中深煎りや中煎りへ変えるだけで飲みやすくなる。砂糖やミルクで隠す前に、抽出条件を一つずつ動かすと自分の好みを見つけやすい。",
			imageURL:    "https://images.unsplash.com/photo-1509042239860-f550ce710b93?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "甘さを感じるコーヒーの選び方",
			slugPrefix:  "sweet-coffee-selection",
			category:    "beans",
			summary:     "砂糖を入れなくても甘く感じやすい豆の特徴と抽出の考え方。",
			body:        "コーヒーの甘さは砂糖のような直接的な甘味ではなく、香り、酸味、苦味のバランスで感じることが多い。フレーバーノートにキャラメル、ブラウンシュガー、はちみつ、ミルクチョコレートのような表現がある豆は甘さを感じやすい。酸味が強すぎると甘さが隠れ、苦味が重すぎても甘さが沈む。抽出では、薄くしすぎず、苦くしすぎない濃度を探ることが重要になる。中煎りから中深煎りの豆は甘さと飲みやすさのバランスを取りやすい。",
			imageURL:    "https://images.unsplash.com/photo-1494314671902-399b18174975?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "豆を買う前に見るべき表示",
			slugPrefix:  "coffee-label-guide",
			category:    "beans",
			summary:     "産地、焙煎日、焙煎度、精製方法、フレーバーノートの読み方。",
			body:        "豆を買う時は、名前だけでなく、焙煎度、焙煎日、産地、精製方法、フレーバーノートを見ると選びやすくなる。焙煎日は鮮度の目安になり、飲み頃を考える手がかりになる。焙煎度は味の方向性を決めるため、酸味が欲しいなら浅煎り、コクが欲しいなら中深煎りから深煎りを選ぶ。精製方法は香りの出方に関わり、ナチュラルは果実感、ウォッシュトは透明感を期待しやすい。フレーバーノートは正解を当てるものではなく、味の方向性を想像するための言葉として使うとよい。",
			imageURL:    "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "二杯分を安定して淹れる方法",
			slugPrefix:  "two-cup-brewing",
			category:    "recipe",
			summary:     "一杯分から二杯分へ増やす時に、粉量、湯量、抽出時間をどう調整するか。",
			body:        "二杯分を淹れる時は、粉量と湯量を単純に倍にするだけでは味が変わることがある。粉の層が厚くなるため、湯が抜けるまでの時間が長くなり、苦味や渋さが出やすくなる。まずは粉30gに湯450gほどを基準にし、一杯分より挽きを少し粗くして調整するとよい。注湯は一度に入れすぎず、粉全体が均一に湿るように数回に分ける。抽出時間が長くなりすぎる場合は、湯量を急に減らすより、挽き目と注ぎ方を先に見直す。",
			imageURL:    "https://images.unsplash.com/photo-1442512595331-e89e73853f31?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "デザートに合わせるコーヒー",
			slugPrefix:  "coffee-dessert-pairing",
			category:    "recipe",
			summary:     "チョコレート、チーズケーキ、焼き菓子に合わせる豆と焙煎度の考え方。",
			body:        "デザートに合わせるコーヒーは、甘さと香りの方向をそろえるとまとまりやすい。チョコレート系には深煎りや中深煎りの豆が合いやすく、カカオやナッツの余韻が重なりやすい。チーズケーキやフルーツタルトには、柑橘やベリーの印象がある浅煎りから中煎りが合わせやすい。焼き菓子にはブラウンシュガー、キャラメル、ナッツのような甘い香りの豆がよく馴染む。甘いものに合わせる時は、コーヒーを苦くしすぎず、後味が長く残りすぎない濃度にすることも大切になる。",
			imageURL:    "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "ブレンドとシングルオリジンの違い",
			slugPrefix:  "blend-single-origin",
			category:    "beans",
			summary:     "飲みやすさを作るブレンドと、産地の個性を楽しむシングルオリジンの使い分け。",
			body:        "ブレンドは複数の豆を組み合わせ、味の安定感や飲みやすさを作るために設計されることが多い。毎日飲む豆やミルクに合わせる豆を探す場合は、ブレンドの方が扱いやすいことがある。シングルオリジンは一つの産地や農園の個性を楽しみやすく、香りや酸味の違いを知るのに向いている。初めての店では、定番ブレンドで店の味の方向を知り、気に入ったらシングルオリジンへ広げる選び方もよい。どちらが上というより、飲む場面と求める味で使い分けることが重要になる。",
			imageURL:    "https://images.unsplash.com/photo-1461988320302-91bde64fc8e4?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "香りを逃がさない抽出前の準備",
			slugPrefix:  "pre-brew-preparation",
			category:    "brewing",
			summary:     "豆を挽くタイミング、器具の温め、湯の準備で香りを残す方法。",
			body:        "コーヒーの香りは豆を挽いた直後から少しずつ抜けていくため、できるだけ抽出直前に挽くのが理想になる。器具やカップを事前に温めておくと、抽出中や飲む時の温度低下を抑えられる。ペーパーフィルターは湯通しして紙の匂いを流し、ドリッパーとサーバーも温めておくと味が安定しやすい。湯は沸騰直後をそのまま使うより、豆の焙煎度に合わせて少し温度を落とすと調整しやすい。抽出前の小さな準備をそろえるだけでも、同じ豆の香りの出方は変わる。",
			imageURL:    "https://images.unsplash.com/photo-1459755486867-b55449bb39ff?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "週末に試したいゆっくり抽出",
			slugPrefix:  "weekend-slow-brew",
			category:    "recipe",
			summary:     "時間をかけて香りと甘さを引き出す、休日向けの抽出レシピ。",
			body:        "時間に余裕がある日は、いつもより少し丁寧に抽出条件を整えると豆の印象を深く楽しめる。粉量、湯量、湯温を記録し、蒸らしの時間を30〜45秒ほど取ると香りが立ちやすい。注湯は急がず、粉全体が膨らみすぎないように小さな円でゆっくり入れる。浅煎りなら湯温を高めにし、中煎りなら甘さが残る濃度を意識する。普段と同じ豆でも、抽出を落ち着いて行うだけで酸味、甘さ、余韻の見え方が変わる。",
			imageURL:    "https://images.unsplash.com/photo-1521302080334-4bebac2763a6?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "コーヒーを飲み比べる時の順番",
			slugPrefix:  "coffee-tasting-order",
			category:    "beans",
			summary:     "浅煎りから深煎り、軽い味から重い味へ進める飲み比べの基本。",
			body:        "複数のコーヒーを飲み比べる時は、軽い味から重い味へ進めると違いを感じやすい。浅煎りから中煎り、深煎りの順に飲むと、香りや酸味の印象が苦味に隠れにくい。フレーバーノートを当てようとするより、酸味、甘さ、苦味、香り、後味の長さを順番に見ると整理しやすい。温度が下がると印象が変わるため、一口目だけで判断せず、数分おいてからもう一度飲むと違いが見える。記録を残す場合は、難しい言葉より自分の言葉で短く残す方が次の豆選びに使いやすい。",
			imageURL:    "https://images.unsplash.com/photo-1511537190424-bbbab87ac5eb?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "コーヒーの後味を整える抽出",
			slugPrefix:  "clean-aftertaste-brewing",
			category:    "brewing",
			summary:     "飲み終わりの渋さや重さを減らし、きれいな余韻を作る調整方法。",
			body:        "後味が渋い場合は、抽出後半で成分が出すぎている可能性がある。挽き目を少し粗くする、湯温を下げる、最後まで落とし切らないといった調整で余韻が軽くなることがある。粉の層が崩れすぎると湯の流れが偏り、部分的に出すぎる場所ができやすい。注湯は強く当てすぎず、粉全体を均一に湿らせることを意識する。すっきりした後味を狙うなら、濃度を下げるだけでなく、抽出の終わり方を整えることが重要になる。",
			imageURL:    "https://images.unsplash.com/photo-1509042239860-f550ce710b93?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "初めて器具を買う人向けの選び方",
			slugPrefix:  "starter-brewing-tools",
			category:    "brewing",
			summary:     "ドリッパー、ミル、スケール、ケトルの優先順位と、最初に揃えるべき理由。",
			body:        "最初に器具を揃えるなら、味への影響が大きい順に考えると無駄が少ない。ミルは挽き目の安定に関わるため、豆の香りや抽出の再現性に直結する。スケールは粉量と湯量をそろえられるため、毎回の味の違いを減らしやすい。ドリッパーは扱いやすいものを選び、慣れるまでは一つの器具で練習する方が上達しやすい。ケトルは注ぎやすさが重要で、細く安定して湯を落とせるものを選ぶと抽出を組み立てやすい。",
			imageURL:    "https://images.unsplash.com/photo-1494314671902-399b18174975?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "来客時に出しやすいコーヒー",
			slugPrefix:  "guest-friendly-coffee",
			category:    "recipe",
			summary:     "好みが分からない相手にも出しやすい、バランス型の豆と抽出。",
			body:        "来客時は、個性が強すぎる豆よりも、甘さと飲みやすさのある中煎りが扱いやすい。ナッツ、チョコレート、キャラメル、オレンジのような表現がある豆は、幅広い人に受け入れられやすい。二杯以上淹れる場合は、粉量と湯量を増やすだけでなく、挽きを少し粗くして抽出時間が長くなりすぎないようにする。ミルクや砂糖を使う人がいる場合は、少しコクのある豆を選ぶと味が残りやすい。飲む場面を考えて、強い酸味や重い苦味に寄せすぎないことが大切になる。",
			imageURL:    "https://images.unsplash.com/photo-1517701604599-bb29b565090c?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "寝る前に重くなりにくい一杯",
			slugPrefix:  "evening-light-cup",
			category:    "recipe",
			summary:     "夜に飲む時の量、濃度、豆選びを軽めに整える考え方。",
			body:        "夜にコーヒーを飲む場合は、量と濃度を控えめにし、後味が重くない豆を選ぶと飲みやすい。深煎りの強い苦味が残ると重く感じることがあるため、中煎りで甘さのある豆や、軽い浅煎りを薄めに淹れる方法もある。粉量を減らしすぎると味が崩れるため、湯量との比率を保ちながら抽出量を少なくする。ミルクを少し加えると口当たりが穏やかになり、デザートとも合わせやすい。カフェインが気になる場合は、飲む時間と量を決めておくと無理なく楽しめる。",
			imageURL:    "https://images.unsplash.com/photo-1461988320302-91bde64fc8e4?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "豆の個性を感じるメモの取り方",
			slugPrefix:  "coffee-note-taking",
			category:    "beans",
			summary:     "難しい表現に頼らず、次の豆選びに使える記録を残す方法。",
			body:        "コーヒーのメモは、専門的な言葉を正確に使うより、自分が次に選びやすくなる形で残すことが大切。まず酸味、苦味、甘さ、香り、後味を短く書く。果物やナッツの名前が思いつかなければ、軽い、丸い、すっきり、重い、香ばしいといった言葉でも十分役に立つ。抽出条件も粉量、湯量、湯温、抽出時間だけ残しておくと、味が良かった時に再現しやすい。数回分のメモがたまると、自分が浅煎り寄りなのか、中煎りの甘さを好むのかが見えやすくなる。",
			imageURL:    "https://images.unsplash.com/photo-1442512595331-e89e73853f31?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "浅煎りをアイスで楽しむ方法",
			slugPrefix:  "light-roast-iced",
			category:    "recipe",
			summary:     "柑橘やベリーの香りを活かす、浅煎り向けの急冷レシピ。",
			body:        "浅煎りをアイスで淹れる時は、酸味が尖らず香りが残る濃度を作ることが大切。湯温は高めにし、粉量も少し多めにして、短時間で香りと甘さを引き出す。サーバーに氷を入れて急冷すると、香りが閉じにくく、すっきりした後味に仕上がりやすい。抽出が薄い場合は粉量を増やすより、挽きを少し細かくして濃度を出す方が味の輪郭を保ちやすい。柑橘やベリーの印象がある豆は、冷たくしても香りが残りやすく、暑い日に飲みやすい。",
			imageURL:    "https://images.unsplash.com/photo-1461023058943-07fcbe16d735?auto=format&fit=crop&w=1200&q=80",
		},
		{
			titlePrefix: "深煎りをすっきり飲む工夫",
			slugPrefix:  "clean-dark-roast",
			category:    "roast",
			summary:     "深煎りのコクを残しながら、焦げ感や重さを抑える抽出の工夫。",
			body:        "深煎りはコクが出やすい一方で、抽出を強くしすぎると焦げ感や渋さが出やすい。すっきり飲みたい場合は、湯温を少し下げ、抽出時間を短めにする。挽き目は細かくしすぎず、湯が詰まらず流れる状態を保つ。粉量を増やして濃くするより、適切な濃度で止める方が甘さを残しやすい。チョコレートやナッツの印象がある深煎りを選ぶと、苦味だけでなく甘い余韻を楽しめる。",
			imageURL:    "https://images.unsplash.com/photo-1511920170033-f8396924c348?auto=format&fit=crop&w=1200&q=80",
		},
	}

	seriesLabels := []string{"実践編", "応用編", "見直し編", "飲み比べ編"}
	items := make([]*model.Article, 0, seedArticleCount)
	for i := 0; i < seedArticleCount; i++ {
		pattern := patterns[i%len(patterns)]
		number := i + 1
		title := pattern.titlePrefix
		slug := pattern.slugPrefix
		if i >= len(patterns) {
			label := seriesLabels[(i/len(patterns)-1)%len(seriesLabels)]
			title = fmt.Sprintf("%s %s", pattern.titlePrefix, label)
			slug = fmt.Sprintf("%s-%03d", pattern.slugPrefix, number)
		}
		publishedAt := now.Add(-time.Duration(i) * 6 * time.Hour)

		items = append(items, &model.Article{
			Title:       title,
			Slug:        slug,
			Summary:     pattern.summary,
			Body:        strPtr(pattern.body),
			Category:    strPtr(pattern.category),
			SourceName:  strPtr("Coffee Journal"),
			SourceURL:   strPtr(fmt.Sprintf("https://example.com/articles/%s", slug)),
			ImageURL:    strPtr(pattern.imageURL),
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
