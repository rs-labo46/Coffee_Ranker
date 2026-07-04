package usecase

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 計算済みContentMetricから表示用ランキングを作る。
type RankingUsecase struct {
	metrics  repository.IContentMetricRepository
	beans    repository.IBeanRepository
	articles repository.IArticleRepository
}

// ランキング候補をもとに、User/Guest向けの推薦候補を取得する。
// InterestProfileと候補コンテンツの属性を照合し、興味に近い候補を上位に寄せる。
// Userの場合は保存済みコンテンツと閲覧済みコンテンツを候補から除外する。
type RecommendationUsecase struct {
	metrics   repository.IContentMetricRepository
	interests repository.IInterestProfileRepository
	saved     repository.ISavedItemRepository
	events    repository.IActionEventRepository
	beans     repository.IBeanRepository
	articles  repository.IArticleRepository
}

// ランキング指標と、それに対応するBean/Article本体をまとめる。
type RankingResult struct {
	Metrics  []*model.ContentMetric `json:"metrics"`
	Targets  []*model.RankTarget    `json:"targets"`
	Beans    []*model.Bean          `json:"beans"`
	Articles []*model.Article       `json:"articles"`
}

// 推薦取得に必要なactor、content_type、ページング条件をまとめる。
type RecommendationInput struct {
	Actor       Actor
	ContentType *entity.ContentType
	Page        Page
}

// 推薦結果1件分の表示DTO。
// Metricをそのまま返すだけでは「なぜ推薦されたか」が分からないため、理由も一緒に返す。
type RecommendationItem struct {
	RankTargetID  uint64                 `json:"rank_target_id"`
	ContentType   entity.ContentType     `json:"content_type"`
	ContentID     uint64                 `json:"content_id"`
	Score         float64                `json:"score"`
	BaseScore     float64                `json:"base_score"`
	InterestScore float64                `json:"interest_score"`
	Reasons       []RecommendationReason `json:"reasons"`
	Metric        *model.ContentMetric   `json:"metric"`
}

// 推薦理由。
// フロントで「産地 Ethiopia への興味に近い」などを表示するためのDTO。
type RecommendationReason struct {
	Dimension entity.InterestDimension `json:"dimension"`
	Value     string                   `json:"value"`
	Score     float64                  `json:"score"`
	Message   string                   `json:"message"`
}

// ランキング取得に必要なRepositoryを受け取るコンストラクタ。
func NewRankingUsecase(metrics repository.IContentMetricRepository, beans repository.IBeanRepository, articles repository.IArticleRepository) *RankingUsecase {
	return &RankingUsecase{metrics: metrics, beans: beans, articles: articles}
}

// 推薦候補取得に必要なRepositoryを受け取るコンストラクタ。
// InterestProfileを使って、計算済みランキング候補をUser/Guestの興味に近い順へ補正する。
func NewRecommendationUsecase(
	metrics repository.IContentMetricRepository,
	interests repository.IInterestProfileRepository,
	saved repository.ISavedItemRepository,
	events repository.IActionEventRepository,
	beans repository.IBeanRepository,
	articles repository.IArticleRepository,
) *RecommendationUsecase {
	return &RecommendationUsecase{
		metrics:   metrics,
		interests: interests,
		saved:     saved,
		events:    events,
		beans:     beans,
		articles:  articles,
	}
}

// ContentMetricとRankTargetをもとにランキング表示用データを取得。
func (u *RankingUsecase) List(ctx context.Context, contentType *entity.ContentType, page Page) (RankingResult, error) {
	// ランキング一覧のlimit/offsetを安全な範囲に補正・検証する。
	page, err := normalizePage(page, 20, 100, 10000)
	if err != nil {
		return RankingResult{}, err
	}

	// 計算済みContentMetricをスコア順で取得する。
	metrics, err := u.metrics.ListRanking(ctx, contentType, page.Limit, page.Offset)
	if err != nil {
		return RankingResult{}, err
	}

	// RankTargetのcontent_typeを見て、BeanIDとArticleIDへ分ける。
	targets, err := rankTargetsFromMetrics(metrics)
	if err != nil {
		return RankingResult{}, err
	}
	beanIDs, articleIDs, err := splitContentIDsFromMetrics(metrics)
	if err != nil {
		return RankingResult{}, err
	}

	// Bean本体をまとめて取得。
	beans, err := u.beans.FindByIDs(ctx, beanIDs)
	if err != nil {
		return RankingResult{}, err
	}

	// Article本体をまとめて取得する。
	articles, err := u.articles.FindByIDs(ctx, articleIDs)
	if err != nil {
		return RankingResult{}, err
	}

	return RankingResult{
		Metrics:  metrics,
		Targets:  targets,
		Beans:    beans,
		Articles: articles,
	}, nil
}

// score上位のContentMetricだけを取得する。
func (u *RankingUsecase) Top(ctx context.Context, limit int) ([]*model.ContentMetric, error) {
	// 未指定なら10件。
	if limit <= 0 {
		limit = 10
	}

	// 取りすぎを防ぐ。
	if limit > 100 {
		return nil, entity.ErrInvalidPagination
	}

	return u.metrics.ListTopByScore(ctx, limit)
}

const (
	recommendationInterestLimit    = 20
	recommendationMaxFetchLimit    = 100
	recommendationInterestWeight   = 1.0
	recommendationRecentEventLimit = 200
)

// Actorに応じた推薦候補を取得する。
// 計算済みランキングをベースにし、User/Guestの興味プロフィールと候補属性が一致した場合はscoreを加算する。
// Userの場合は保存済みコンテンツを、User/Guest共通で直近閲覧済みコンテンツを候補から除外する。
func (u *RecommendationUsecase) List(ctx context.Context, input RecommendationInput) ([]RecommendationItem, error) {
	// UserまたはGuestSessionの片方だけであることを確認する。
	if err := requireActor(input.Actor); err != nil {
		return nil, err
	}

	// 推薦候補は最大50件までに制限する。
	page, err := normalizePage(input.Page, 20, 50, 500)
	if err != nil {
		return nil, err
	}

	// User/Guestの興味プロフィールを取得する。
	profiles, err := u.loadInterestProfiles(ctx, input.Actor)
	if err != nil {
		return nil, err
	}

	// 保存済み除外・閲覧済み除外・興味スコア反映で候補が減るため、内部取得は多めにする。
	fetchLimit := recommendationFetchLimit(page.Limit, input.Actor.UserID != nil, len(profiles) > 0)

	// ベース候補は計算済みランキングから取得する。
	metrics, err := u.metrics.ListRanking(ctx, input.ContentType, fetchLimit, page.Offset)
	if err != nil {
		return nil, err
	}

	// Userは保存済みコンテンツを推薦候補から除外する。
	if input.Actor.UserID != nil {
		metrics, err = u.filterSavedMetrics(ctx, *input.Actor.UserID, metrics)
		if err != nil {
			return nil, err
		}
	}

	// 直近閲覧済みのRankTargetは推薦候補から除外する。
	metrics, err = u.filterViewedMetrics(ctx, input.Actor, metrics)
	if err != nil {
		return nil, err
	}

	// 興味プロフィールと候補コンテンツの属性を照合し、scoreと推薦理由を作る。
	items, err := u.buildRecommendationItems(ctx, metrics, profiles)
	if err != nil {
		return nil, err
	}

	return trimRecommendationItems(items, page.Limit), nil
}

// Userが保存済みのRankTargetを推薦候補から除外する。
// 保存済みRankTargetIDを一括取得し、候補ごとのExistsActive呼び出しによるN+1を避ける。
func (u *RecommendationUsecase) filterSavedMetrics(ctx context.Context, userID uint64, metrics []*model.ContentMetric) ([]*model.ContentMetric, error) {
	rankTargetIDs, err := rankTargetIDsFromMetrics(metrics)
	if err != nil {
		return nil, err
	}

	savedIDs, err := u.saved.ListActiveRankTargetIDsByUser(ctx, userID, rankTargetIDs)
	if err != nil {
		return nil, err
	}

	filtered := make([]*model.ContentMetric, 0, len(metrics))
	for _, metric := range metrics {
		if metric == nil {
			continue
		}
		if savedIDs[metric.RankTargetID] {
			continue
		}
		filtered = append(filtered, metric)
	}
	return filtered, nil
}

// User/Guestの直近閲覧済みRankTargetを推薦候補から除外する。
func (u *RecommendationUsecase) filterViewedMetrics(ctx context.Context, actor Actor, metrics []*model.ContentMetric) ([]*model.ContentMetric, error) {
	events, err := u.events.FindRecentByActor(ctx, actor.UserID, actor.GuestSessionID, recommendationRecentEventLimit)
	if err != nil {
		return nil, err
	}

	viewed := viewedRankTargetSet(events)
	if len(viewed) == 0 {
		return metrics, nil
	}

	filtered := make([]*model.ContentMetric, 0, len(metrics))
	for _, metric := range metrics {
		if metric == nil {
			continue
		}
		if viewed[metric.RankTargetID] {
			continue
		}
		filtered = append(filtered, metric)
	}
	return filtered, nil
}

// User/Guestに応じて興味プロフィール上位を取得する。
func (u *RecommendationUsecase) loadInterestProfiles(ctx context.Context, actor Actor) ([]*model.InterestProfile, error) {
	if actor.UserID != nil {
		return u.interests.ListTopByUser(ctx, *actor.UserID, recommendationInterestLimit)
	}
	if actor.GuestSessionID != nil {
		return u.interests.ListTopByGuest(ctx, *actor.GuestSessionID, recommendationInterestLimit)
	}
	return nil, entity.ErrInvalidInput
}

// 保存済み除外や興味スコア反映で候補が減るため、内部取得件数を増やす。
func recommendationFetchLimit(pageLimit int, hasUser bool, hasProfiles bool) int {
	fetchLimit := pageLimit

	if hasUser {
		fetchLimit = pageLimit * 2
	}
	if hasProfiles {
		fetchLimit = pageLimit * 4
	}

	if fetchLimit < pageLimit {
		return pageLimit
	}
	if fetchLimit > recommendationMaxFetchLimit {
		return recommendationMaxFetchLimit
	}
	return fetchLimit
}

// ContentMetricに紐づくBean/Articleを取得し、InterestProfileと一致する属性があればscoreと推薦理由を作る。
func (u *RecommendationUsecase) buildRecommendationItems(ctx context.Context, metrics []*model.ContentMetric, profiles []*model.InterestProfile) ([]RecommendationItem, error) {
	if len(profiles) == 0 {
		return metricsToRecommendationItems(metrics)
	}

	beanIDs, articleIDs, err := splitContentIDsFromMetrics(metrics)
	if err != nil {
		return nil, err
	}

	beans, err := u.beans.FindByIDs(ctx, beanIDs)
	if err != nil {
		return nil, err
	}

	articles, err := u.articles.FindByIDs(ctx, articleIDs)
	if err != nil {
		return nil, err
	}

	beanByID := beanMapByID(beans)
	articleByID := articleMapByID(articles)

	items := make([]RecommendationItem, 0, len(metrics))
	for _, metric := range metrics {
		if metric == nil {
			continue
		}

		reasons := recommendationReasonsForMetric(metric, profiles, beanByID, articleByID)
		interestScore := totalReasonScore(reasons)

		copied := *metric
		copied.Score = metric.Score + interestScore

		items = append(items, RecommendationItem{
			RankTargetID:  copied.RankTargetID,
			ContentType:   copied.RankTarget.ContentType,
			ContentID:     copied.RankTarget.ContentID,
			Score:         copied.Score,
			BaseScore:     metric.Score,
			InterestScore: interestScore,
			Reasons:       reasons,
			Metric:        &copied,
		})
	}

	sort.SliceStable(items, func(i int, j int) bool {
		if items[i].Score == items[j].Score {
			return items[i].RankTargetID > items[j].RankTargetID
		}
		return items[i].Score > items[j].Score
	})

	return items, nil
}

// InterestProfileがない場合は、ランキング順の候補を推薦DTOへ変換する。
func metricsToRecommendationItems(metrics []*model.ContentMetric) ([]RecommendationItem, error) {
	items := make([]RecommendationItem, 0, len(metrics))
	for _, metric := range metrics {
		if metric == nil {
			continue
		}
		if metric.RankTargetID == 0 {
			return nil, entity.ErrRankTargetNotFound
		}

		copied := *metric
		items = append(items, RecommendationItem{
			RankTargetID: copied.RankTargetID,
			ContentType:  copied.RankTarget.ContentType,
			ContentID:    copied.RankTarget.ContentID,
			Score:        copied.Score,
			BaseScore:    metric.Score,
			Reasons:      nil,
			Metric:       &copied,
		})
	}
	return items, nil
}

// Bean IDをkeyにしたmapを作る。
func beanMapByID(beans []*model.Bean) map[uint64]*model.Bean {
	items := make(map[uint64]*model.Bean, len(beans))
	for _, bean := range beans {
		if bean == nil || bean.ID == 0 {
			continue
		}
		items[bean.ID] = bean
	}
	return items
}

// Article IDをkeyにしたmapを作る。
func articleMapByID(articles []*model.Article) map[uint64]*model.Article {
	items := make(map[uint64]*model.Article, len(articles))
	for _, article := range articles {
		if article == nil || article.ID == 0 {
			continue
		}
		items[article.ID] = article
	}
	return items
}

// 候補のContentTypeに応じて、興味プロフィールとの一致理由を返す。
func recommendationReasonsForMetric(
	metric *model.ContentMetric,
	profiles []*model.InterestProfile,
	beans map[uint64]*model.Bean,
	articles map[uint64]*model.Article,
) []RecommendationReason {
	if metric == nil || metric.RankTarget.ContentID == 0 {
		return nil
	}

	switch metric.RankTarget.ContentType {
	case entity.ContentTypeBean:
		bean := beans[metric.RankTarget.ContentID]
		if bean == nil {
			return nil
		}
		return recommendationReasonsForBean(bean, profiles)

	case entity.ContentTypeArticle:
		article := articles[metric.RankTarget.ContentID]
		if article == nil {
			return nil
		}
		return recommendationReasonsForArticle(article, profiles)

	default:
		return nil
	}
}

// Bean属性とInterestProfileが一致した分だけ推薦理由を作る。
func recommendationReasonsForBean(bean *model.Bean, profiles []*model.InterestProfile) []RecommendationReason {
	if bean == nil {
		return nil
	}

	reasons := make([]RecommendationReason, 0)
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		if profileMatchesBean(profile, bean) {
			reasons = append(reasons, recommendationReasonFromProfile(profile))
		}
	}
	return reasons
}

// Article属性とInterestProfileが一致した分だけ推薦理由を作る。
func recommendationReasonsForArticle(article *model.Article, profiles []*model.InterestProfile) []RecommendationReason {
	if article == nil {
		return nil
	}

	reasons := make([]RecommendationReason, 0)
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		if profileMatchesArticle(profile, article) {
			reasons = append(reasons, recommendationReasonFromProfile(profile))
		}
	}
	return reasons
}

// InterestProfileを表示用の推薦理由に変換する。
func recommendationReasonFromProfile(profile *model.InterestProfile) RecommendationReason {
	score := profile.Score * recommendationInterestWeight
	return RecommendationReason{
		Dimension: profile.Dimension,
		Value:     profile.Value,
		Score:     score,
		Message:   recommendationReasonMessage(profile.Dimension, profile.Value),
	}
}

// 推薦理由の短い説明文を作る。
func recommendationReasonMessage(dimension entity.InterestDimension, value string) string {
	label := interestDimensionLabel(dimension)
	if label == "" {
		return "興味傾向に近い候補"
	}
	return label + " " + value + " への興味に近い候補"
}

// InterestDimensionを表示向けの短い日本語に変換する。
func interestDimensionLabel(dimension entity.InterestDimension) string {
	switch dimension {
	case entity.InterestDimensionOrigin:
		return "産地"
	case entity.InterestDimensionRoastLevel:
		return "焙煎度"
	case entity.InterestDimensionAcidity:
		return "酸味"
	case entity.InterestDimensionBitterness:
		return "苦味"
	case entity.InterestDimensionFlavor:
		return "風味"
	case entity.InterestDimensionAroma:
		return "香り"
	case entity.InterestDimensionBody:
		return "ボディ"
	case entity.InterestDimensionArticleCategory:
		return "記事カテゴリ"
	default:
		return ""
	}
}

// 推薦理由の合計スコアを返す。
func totalReasonScore(reasons []RecommendationReason) float64 {
	score := 0.0
	for _, reason := range reasons {
		score += reason.Score
	}
	return score
}

// InterestProfileがBeanの属性と一致するか確認する。
func profileMatchesBean(profile *model.InterestProfile, bean *model.Bean) bool {
	switch profile.Dimension {
	case entity.InterestDimensionOrigin:
		return bean.Origin != nil && sameInterestValue(*bean.Origin, profile.Value)

	case entity.InterestDimensionRoastLevel:
		return sameInterestValue(string(bean.RoastLevel), profile.Value)

	case entity.InterestDimensionAcidity:
		return bean.Acidity != nil && sameInterestValue(strconv.Itoa(*bean.Acidity), profile.Value)

	case entity.InterestDimensionBitterness:
		return bean.Bitterness != nil && sameInterestValue(strconv.Itoa(*bean.Bitterness), profile.Value)

	case entity.InterestDimensionFlavor:
		return bean.Flavor != nil && sameInterestValue(strconv.Itoa(*bean.Flavor), profile.Value)

	case entity.InterestDimensionAroma:
		return bean.Aroma != nil && sameInterestValue(strconv.Itoa(*bean.Aroma), profile.Value)

	case entity.InterestDimensionBody:
		return bean.Body != nil && sameInterestValue(strconv.Itoa(*bean.Body), profile.Value)

	default:
		return false
	}
}

// InterestProfileがArticleの属性と一致するか確認する。
func profileMatchesArticle(profile *model.InterestProfile, article *model.Article) bool {
	switch profile.Dimension {
	case entity.InterestDimensionArticleCategory:
		return article.Category != nil && sameInterestValue(*article.Category, profile.Value)

	default:
		return false
	}
}

// 余計な空白や大文字小文字の差で一致漏れしないようにする。
func sameInterestValue(actual string, expected string) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	if actual == "" || expected == "" {
		return false
	}
	return strings.EqualFold(actual, expected)
}

// 指定件数を超えた推薦DTOを切り詰める。
func trimRecommendationItems(items []RecommendationItem, limit int) []RecommendationItem {
	if limit <= 0 || len(items) <= limit {
		return items
	}
	return items[:limit]
}

// ContentMetricのRankTargetIDだけを取り出す。
func rankTargetIDsFromMetrics(metrics []*model.ContentMetric) ([]uint64, error) {
	ids := make([]uint64, 0, len(metrics))
	for _, metric := range metrics {
		if metric == nil {
			continue
		}
		if metric.RankTargetID == 0 {
			return nil, entity.ErrRankTargetNotFound
		}
		ids = append(ids, metric.RankTargetID)
	}
	return ids, nil
}

// 閲覧済みとして除外するRankTargetIDのsetを作る。
func viewedRankTargetSet(events []*model.ActionEvent) map[uint64]bool {
	viewed := make(map[uint64]bool)
	for _, event := range events {
		if event == nil || event.RankTargetID == nil {
			continue
		}
		if isViewedRecommendationEvent(event.EventType) {
			viewed[*event.RankTargetID] = true
		}
	}
	return viewed
}

// 推薦候補から除外する閲覧済み系イベントか確認する。
func isViewedRecommendationEvent(eventType entity.EventType) bool {
	switch eventType {
	case entity.EventTypeContentView, entity.EventTypeClick, entity.EventTypeModalClick:
		return true
	default:
		return false
	}
}

// ContentMetric内にPreloadされたRankTargetをレスポンス用に取り出す。
// FrontendがBean/Article一覧とrank_target_idを安定して紐付けるために使う。
func rankTargetsFromMetrics(metrics []*model.ContentMetric) ([]*model.RankTarget, error) {
	targets := make([]*model.RankTarget, 0, len(metrics))

	for _, metric := range metrics {
		if metric == nil {
			continue
		}
		if metric.RankTarget.ID == 0 || metric.RankTarget.ContentID == 0 {
			return nil, entity.ErrRankTargetNotFound
		}

		target := metric.RankTarget
		targets = append(targets, &target)
	}

	return targets, nil
}

// ContentMetric内のRankTargetからBean IDとArticle IDを分ける。
// ListRanking側でRankTargetがPreloadされていない場合は、正しく分けられないためエラー。
func splitContentIDsFromMetrics(metrics []*model.ContentMetric) ([]uint64, []uint64, error) {
	beanIDs := make([]uint64, 0)
	articleIDs := make([]uint64, 0)

	for _, metric := range metrics {
		if metric == nil {
			continue
		}

		// RankTargetIDがないMetricは不正データ。
		if metric.RankTargetID == 0 {
			return nil, nil, entity.ErrRankTargetNotFound
		}

		// RankTarget本体が読み込まれていない、または不正なRankTargetなら処理を止める。
		// Repository側でPreload("RankTarget")していない場合、RankTarget.IDやContentIDは0になる。
		if metric.RankTarget.ID == 0 || metric.RankTarget.ContentID == 0 {
			return nil, nil, entity.ErrRankTargetNotFound
		}
		// RankTargetの種類を見て、Bean IDとArticle IDに振り分ける。
		// それぞれBeanRepository / ArticleRepositoryで本体をまとめて取得
		switch metric.RankTarget.ContentType {
		case entity.ContentTypeBean:
			beanIDs = append(beanIDs, metric.RankTarget.ContentID)
		case entity.ContentTypeArticle:
			articleIDs = append(articleIDs, metric.RankTarget.ContentID)
		default:
			return nil, nil, entity.ErrInvalidInput
		}
	}

	return beanIDs, articleIDs, nil
}
