package usecase

import (
	"context"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 計算済みContentMetricから表示用ランキングを作る。
type RankingUsecase struct {
	metrics  repository.ContentMetricRepository
	beans    repository.BeanRepository
	articles repository.ArticleRepository
}

// ランキング候補をもとに、User/Guest向けの推薦候補を取得。
// Userの場合だけ、保存済みコンテンツを候補から除外する。
type RecommendationUsecase struct {
	metrics repository.ContentMetricRepository
	saved   repository.SavedItemRepository
}

// ランキング指標と、それに対応するBean/Article本体をまとめる。
type RankingResult struct {
	Metrics  []*model.ContentMetric
	Beans    []*model.Bean
	Articles []*model.Article
}

// 推薦取得に必要なactor、content_type、ページング条件をまとめる。
type RecommendationInput struct {
	Actor       Actor
	ContentType *entity.ContentType
	Page        Page
}

// ランキング取得に必要なRepositoryを受け取るコンストラクタ。
// rankTargetsは現状このUsecase内では直接使わない。
func NewRankingUsecase(metrics repository.ContentMetricRepository, rankTargets repository.RankTargetRepository, beans repository.BeanRepository, articles repository.ArticleRepository) *RankingUsecase {
	return &RankingUsecase{metrics: metrics, beans: beans, articles: articles}
}

// 推薦候補取得に必要なRepositoryを受け取るコンストラクタ。
// interestsはModalUsecase側で使うため、このUsecaseでは保持しない。
func NewRecommendationUsecase(metrics repository.ContentMetricRepository, interests repository.InterestProfileRepository, saved repository.SavedItemRepository) *RecommendationUsecase {
	return &RecommendationUsecase{metrics: metrics, saved: saved}
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

// Actorに応じた推薦候補を取得する。
// 現時点では、個人の興味プロフィールによる再スコアリングは未実装。
// ベース候補はランキングから取得し、Userの場合は保存済みを除外する。
func (u *RecommendationUsecase) List(ctx context.Context, input RecommendationInput) ([]*model.ContentMetric, error) {
	// UserまたはGuestSessionの片方だけであることを確認する。
	if err := requireActor(input.Actor); err != nil {
		return nil, err
	}

	// 推薦候補は最大50件までに制限する。
	page, err := normalizePage(input.Page, 20, 50, 500)
	if err != nil {
		return nil, err
	}

	// Userの場合は保存済み除外で件数が減るため、多めに候補を取得する。
	// N+1の根本対策ではないが、MVPでは安全寄りの暫定対応。
	fetchLimit := page.Limit
	if input.Actor.UserID != nil {
		fetchLimit = page.Limit * 2
	}

	// ベース候補は計算済みランキングから取得する。
	metrics, err := u.metrics.ListRanking(ctx, input.ContentType, fetchLimit, page.Offset)
	if err != nil {
		return nil, err
	}

	// Guestは保存機能がないため、そのまま件数を丸めて返す。
	if input.Actor.UserID == nil {
		return trimMetrics(metrics, page.Limit), nil
	}

	// Userは保存済みコンテンツを推薦候補から除外する。
	filtered, err := u.filterSavedMetrics(ctx, *input.Actor.UserID, metrics)
	if err != nil {
		return nil, err
	}

	return trimMetrics(filtered, page.Limit), nil
}

// Userが保存済みのRankTargetを推薦候補から除外する。
// 現状はExistsActiveを1件ずつ呼ぶため、件数が増えるとN+1になりやすい。
// 今はMVPとして許容し、後で一括取得Repositoryに置き換える。
func (u *RecommendationUsecase) filterSavedMetrics(ctx context.Context, userID uint64, metrics []*model.ContentMetric) ([]*model.ContentMetric, error) {
	filtered := make([]*model.ContentMetric, 0, len(metrics))

	for _, metric := range metrics {
		if metric == nil {
			continue
		}

		// RankTargetIDがないMetricは不正データなので止める。
		if metric.RankTargetID == 0 {
			return nil, entity.ErrRankTargetNotFound
		}

		// すでに保存済みなら推薦候補から外す。
		saved, err := u.saved.ExistsActive(ctx, userID, metric.RankTargetID)
		if err != nil {
			return nil, err
		}
		if saved {
			continue
		}

		filtered = append(filtered, metric)
	}

	return filtered, nil
}

// 指定件数を超えたmetricsを切り詰める。
func trimMetrics(metrics []*model.ContentMetric, limit int) []*model.ContentMetric {
	if limit <= 0 || len(metrics) <= limit {
		return metrics
	}
	return metrics[:limit]
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
		case entity.ContentTypeBean: //Beanランキング対象:beanIDsに入れる
			beanIDs = append(beanIDs, metric.RankTarget.ContentID)
		case entity.ContentTypeArticle: //Articleランキング対象:articleIDsに入れる
			articleIDs = append(articleIDs, metric.RankTarget.ContentID)
		default:
			//想定外のcontent_typeなので不正データとして止める
			return nil, nil, entity.ErrInvalidInput
		}
	}

	return beanIDs, articleIDs, nil
}
