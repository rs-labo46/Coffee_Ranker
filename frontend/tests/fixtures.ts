import type {
  Article,
  AuditLog,
  BatchRun,
  Bean,
  ContentMetric,
  FeedItem,
  RankTarget,
  RankingResult,
  RecommendationItem,
  User,
} from "../src/types";

export const user: User = {
  id: 1,
  name: "Rin",
  email: "rin@example.com",
  role: "user",
  status: "active",
};

export const adminUser: User = {
  id: 2,
  name: "Admin",
  email: "admin@example.com",
  role: "admin",
  status: "active",
};

export function makeBean(overrides: Partial<Bean> = {}): Bean {
  return {
    id: 1,
    name: "Ethiopia Test Bean",
    roaster: "Test Roaster",
    origin: "Ethiopia",
    region: "Yirgacheffe",
    farm: "Test Farm",
    variety: "Heirloom",
    roast_level: "light",
    acidity: 5,
    bitterness: 2,
    flavor: 5,
    aroma: 4,
    body: 3,
    flavor_note: "citrus",
    description: "明るい酸味のテスト豆",
    image_url: "",
    is_published: true,
    created_at: "2026-07-06T00:00:00Z",
    updated_at: "2026-07-06T00:00:00Z",
    ...overrides,
  };
}

export function makeArticle(overrides: Partial<Article> = {}): Article {
  return {
    id: 10,
    title: "初心者向け抽出ガイド",
    slug: "brewing-guide",
    summary: "抽出の基礎をまとめた記事",
    body: "抽出本文",
    category: "brewing",
    source_name: "Coffee Ranker Editorial",
    source_url: "",
    image_url: "",
    is_published: true,
    published_at: "2026-07-06T00:00:00Z",
    created_at: "2026-07-06T00:00:00Z",
    updated_at: "2026-07-06T00:00:00Z",
    ...overrides,
  };
}

export function makeMetric(
  rankTargetId: number,
  overrides: Partial<ContentMetric> = {},
): ContentMetric {
  return {
    id: rankTargetId,
    rank_target_id: rankTargetId,
    score: 82,
    impression_count: 100,
    content_view_count: 40,
    click_count: 20,
    stay_total_ms: 120000,
    save_count: 8,
    rating_count: 6,
    good_count: 5,
    bad_count: 1,
    re_search_count: 3,
    rating_avg: 0.66,
    good_rate: 0.83,
    bad_rate: 0.17,
    modal_impression_count: 10,
    modal_click_count: 4,
    modal_close_count: 6,
    click_rate: 0.2,
    save_rate: 0.08,
    re_search_rate: 0.03,
    modal_click_rate: 0.4,
    modal_close_rate: 0.6,
    period_start: "2026-07-05T00:00:00Z",
    period_end: "2026-07-06T00:00:00Z",
    calculated_at: "2026-07-06T02:00:00Z",
    updated_at: "2026-07-06T02:00:00Z",
    ...overrides,
  };
}

export function makeRankTarget(
  overrides: Partial<RankTarget> = {},
): RankTarget {
  return {
    id: 100,
    content_type: "bean",
    content_id: 1,
    is_active: true,
    created_at: "2026-07-06T00:00:00Z",
    updated_at: "2026-07-06T00:00:00Z",
    ...overrides,
  };
}

export function makeFeedItem(overrides: Partial<FeedItem> = {}): FeedItem {
  const bean = makeBean();
  return {
    key: `bean-${bean.id}`,
    contentType: "bean",
    contentId: bean.id,
    rankTargetId: 100,
    title: bean.name,
    subtitle: "Ethiopia / Test Roaster",
    summary: bean.description ?? "テスト用コンテンツ",
    body: bean.description,
    imageUrl: "",
    badge: "浅煎り",
    score: 82,
    reasons: [],
    metric: makeMetric(100),
    bean,
    isSaved: false,
    ratingScore: null,
    ...overrides,
  };
}

export function makeRecommendation(
  overrides: Partial<RecommendationItem> = {},
): RecommendationItem {
  return {
    rank_target_id: 100,
    content_type: "bean",
    content_id: 1,
    score: 82,
    base_score: 80,
    interest_score: 2,
    reasons: [
      {
        dimension: "origin",
        value: "Ethiopia",
        score: 2,
        message: "エチオピア系の閲覧傾向があります。",
      },
    ],
    metric: makeMetric(100),
    ...overrides,
  };
}

export function makeRankingResult(
  overrides: Partial<RankingResult> = {},
): RankingResult {
  return {
    metrics: [makeMetric(100)],
    targets: [makeRankTarget()],
    beans: [makeBean()],
    articles: [],
    ...overrides,
  };
}

export function makeBatchRun(overrides: Partial<BatchRun> = {}): BatchRun {
  return {
    id: 1,
    job_name: "ranking",
    status: "success",
    started_at: "2026-07-06T02:00:00Z",
    finished_at: "2026-07-06T02:01:00Z",
    rows_processed: 10,
    triggered_by: "system",
    ...overrides,
  };
}

export function makeAuditLog(overrides: Partial<AuditLog> = {}): AuditLog {
  return {
    id: 1,
    actor_type: "admin",
    actor_user_id: 2,
    action: "admin_login",
    target_type: "user",
    target_id: 2,
    detail: "{}",
    request_id: "req-test",
    created_at: "2026-07-06T02:00:00Z",
    ...overrides,
  };
}
