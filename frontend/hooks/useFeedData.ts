import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ensureGuestSession,
  listArticles,
  listBeans,
  listRecommendations,
  recordEvent,
  searchArticles,
  searchBeans,
  stableSearchHash,
} from "../src/api/client";
import type {
  Article,
  Bean,
  ContentMetric,
  ContentType,
  FeedFilter,
  FeedItem,
  FeedState,
  RankTargetID,
  RecommendationItem,
  RecommendationReason,
  SearchState,
} from "../src/types";

type RankTargetInfo = {
  rankTargetId: RankTargetID;
  score?: number;
  reasons: RecommendationReason[];
  metric?: ContentMetric;
};

type UseFeedDataResult = {
  state: FeedState;
  searching: boolean;
  reload: () => Promise<void>;
  showCatalog: () => void;
  runSearch: (search: SearchState) => Promise<void>;
};

const recommendationPageLimit = 50;

function targetKey(contentType: ContentType, contentId: number): string {
  return `${contentType}:${contentId}`;
}

function buildRankIndex(
  recommendations: RecommendationItem[],
): Map<string, RankTargetInfo> {
  const index = new Map<string, RankTargetInfo>();

  for (const item of recommendations) {
    index.set(targetKey(item.content_type, item.content_id), {
      rankTargetId: item.rank_target_id,
      score: item.score,
      reasons: item.reasons ?? [],
      metric: item.metric,
    });
  }

  return index;
}

async function listCatalogRecommendations(): Promise<RecommendationItem[]> {
  const pages = await Promise.all([
    listRecommendations("bean", recommendationPageLimit, 0).catch(() => []),
    listRecommendations(
      "bean",
      recommendationPageLimit,
      recommendationPageLimit,
    ).catch(() => []),
    listRecommendations("article", recommendationPageLimit, 0).catch(() => []),
    listRecommendations(
      "article",
      recommendationPageLimit,
      recommendationPageLimit,
    ).catch(() => []),
  ]);

  return pages.flat();
}

async function listFeedRecommendations(): Promise<RecommendationItem[]> {
  return listRecommendations("all", recommendationPageLimit, 0).catch(() => []);
}

function rankInfoFor(
  index: Map<string, RankTargetInfo>,
  contentType: ContentType,
  contentId: number,
): RankTargetInfo | undefined {
  return index.get(targetKey(contentType, contentId));
}

function toBeanItem(bean: Bean, info?: RankTargetInfo): FeedItem {
  return {
    key: `bean-${bean.id}`,
    contentType: "bean",
    contentId: bean.id,
    rankTargetId: info?.rankTargetId,
    title: bean.name,
    subtitle:
      [bean.origin, bean.roaster].filter(Boolean).join(" / ") || "Coffee Bean",
    summary:
      bean.description ??
      bean.flavor_note ??
      "味覚スコアと行動データから見るコーヒー豆。",
    body: bean.description,
    imageUrl: bean.image_url,
    badge: roastLabel(bean.roast_level),
    score: info?.score,
    reasons: info?.reasons ?? [],
    metric: info?.metric,
    bean,
  };
}

function toArticleItem(article: Article, info?: RankTargetInfo): FeedItem {
  return {
    key: `article-${article.id}`,
    contentType: "article",
    contentId: article.id,
    rankTargetId: info?.rankTargetId,
    title: article.title,
    subtitle:
      [categoryLabel(article.category), article.source_name]
        .filter(Boolean)
        .join(" / ") || "Article",
    summary: article.summary,
    body: article.body,
    imageUrl: article.image_url,
    badge: categoryLabel(article.category),
    score: info?.score,
    reasons: info?.reasons ?? [],
    metric: info?.metric,
    article,
  };
}

function roastLabel(value: string): string {
  switch (value) {
    case "light":
      return "浅煎り";
    case "medium":
      return "中煎り";
    case "dark":
      return "深煎り";
    default:
      return "Bean";
  }
}

function categoryLabel(value: string | undefined): string {
  switch (value) {
    case "brewing":
      return "抽出";
    case "roast":
      return "焙煎";
    case "beans":
      return "豆知識";
    case "recipe":
      return "レシピ";
    default:
      return "Article";
  }
}

function uniqueItems(items: FeedItem[]): FeedItem[] {
  const seen = new Set<string>();
  const result: FeedItem[] = [];

  for (const item of items) {
    if (seen.has(item.key)) {
      continue;
    }
    seen.add(item.key);
    result.push(item);
  }

  return result;
}

function filterItems(items: FeedItem[], filter: FeedFilter): FeedItem[] {
  if (filter === "all") {
    return items;
  }
  return items.filter((item) => item.contentType === filter);
}

function normalizeSearchText(value: string): string {
  return value
    .toLowerCase()
    .replaceAll("　", " ")
    .replaceAll("ブラジル", "brazil")
    .replaceAll("エチオピア", "ethiopia")
    .replaceAll("コロンビア", "colombia")
    .replaceAll("インドネシア", "indonesia")
    .replaceAll("ホンジュラス", "honduras")
    .replaceAll("ケニア", "kenya")
    .replaceAll("タンザニア", "tanzania")
    .replaceAll("グアテマラ", "guatemala")
    .replaceAll("浅煎り", "light")
    .replaceAll("中煎り", "medium")
    .replaceAll("深煎り", "dark")
    .trim();
}

function matchesKeyword(item: FeedItem, keyword: string): boolean {
  const normalized = normalizeSearchText(keyword);
  if (normalized === "") {
    return true;
  }

  const source = normalizeSearchText(
    [
      item.title,
      item.subtitle,
      item.summary,
      item.badge,
      item.bean?.origin,
      item.bean?.roaster,
      item.bean?.roast_level,
      item.article?.category,
      item.article?.source_name,
    ]
      .filter(Boolean)
      .join(" "),
  );

  return source.includes(normalized);
}

function catalogIndex(items: FeedItem[]): Map<string, FeedItem> {
  return new Map<string, FeedItem>(items.map((item) => [item.key, item]));
}

function mergeCatalogState(
  item: FeedItem,
  catalog: Map<string, FeedItem>,
): FeedItem {
  const source = catalog.get(item.key);
  if (source === undefined) {
    return item;
  }

  return {
    ...item,
    rankTargetId: item.rankTargetId ?? source.rankTargetId,
    score: item.score ?? source.score,
    reasons: item.reasons.length > 0 ? item.reasons : source.reasons,
    metric: item.metric ?? source.metric,
  };
}

function localSearch(
  beans: Bean[],
  articles: Article[],
  search: SearchState,
  rankIndex: Map<string, RankTargetInfo>,
): FeedItem[] {
  const useBeans =
    search.contentType === "all" || search.contentType === "bean";
  const useArticles =
    search.contentType === "all" || search.contentType === "article";

  const beanItems = useBeans
    ? beans
        .filter(
          (bean) =>
            search.roastLevel === "" || bean.roast_level === search.roastLevel,
        )
        .map((bean) =>
          toBeanItem(bean, rankInfoFor(rankIndex, "bean", bean.id)),
        )
        .filter((item) => matchesKeyword(item, search.q))
    : [];

  const articleItems = useArticles
    ? articles
        .filter(
          (article) =>
            search.category === "" || article.category === search.category,
        )
        .map((article) =>
          toArticleItem(article, rankInfoFor(rankIndex, "article", article.id)),
        )
        .filter((item) => matchesKeyword(item, search.q))
    : [];

  return uniqueItems([...beanItems, ...articleItems]);
}

export function useFeedData(activeFilter: FeedFilter): UseFeedDataResult {
  const [state, setState] = useState<FeedState>({
    items: [],
    catalogItems: [],
    loading: true,
    error: null,
  });
  const [searching, setSearching] = useState<boolean>(false);

  const load = useCallback(async (): Promise<void> => {
    setState((current) => ({ ...current, loading: true, error: null }));

    try {
      await ensureGuestSession();
      const [beans, articles, recommendations, catalogRecommendations] =
        await Promise.all([
          listBeans(100),
          listArticles(100),
          listFeedRecommendations(),
          listCatalogRecommendations(),
        ]);

      const beansByID = new Map<number, Bean>(
        beans.map((bean) => [bean.id, bean]),
      );
      const articlesByID = new Map<number, Article>(
        articles.map((article) => [article.id, article]),
      );
      const rankIndex = buildRankIndex([
        ...catalogRecommendations,
        ...recommendations,
      ]);

      const recommendedItems = recommendations.flatMap((item) => {
        const info = rankInfoFor(rankIndex, item.content_type, item.content_id);

        if (item.content_type === "bean") {
          const bean = beansByID.get(item.content_id);
          return bean === undefined ? [] : [toBeanItem(bean, info)];
        }

        const article = articlesByID.get(item.content_id);
        return article === undefined ? [] : [toArticleItem(article, info)];
      });

      const fallbackItems = [
        ...beans.map((bean) =>
          toBeanItem(bean, rankInfoFor(rankIndex, "bean", bean.id)),
        ),
        ...articles.map((article) =>
          toArticleItem(article, rankInfoFor(rankIndex, "article", article.id)),
        ),
      ];

      const catalogItems = uniqueItems([...recommendedItems, ...fallbackItems]);

      setState({
        items: catalogItems,
        catalogItems,
        loading: false,
        error: null,
      });
    } catch (error) {
      setState({
        items: [],
        catalogItems: [],
        loading: false,
        error:
          error instanceof Error ? error.message : "読み込みに失敗しました",
      });
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  const showCatalog = useCallback((): void => {
    setState((current) => ({
      ...current,
      items: current.catalogItems,
      loading: false,
      error: null,
    }));
  }, []);

  const runSearch = useCallback(
    async (search: SearchState): Promise<void> => {
      const hasQuery =
        search.q.trim() !== "" ||
        search.roastLevel !== "" ||
        search.category !== "" ||
        search.contentType !== "all";

      if (!hasQuery) {
        await load();
        return;
      }

      setSearching(true);
      setState((current) => ({ ...current, error: null }));

      try {
        const useBeans =
          search.contentType === "all" || search.contentType === "bean";
        const useArticles =
          search.contentType === "all" || search.contentType === "article";
        const catalog = catalogIndex(state.catalogItems);
        const rankIndex = new Map<string, RankTargetInfo>();

        for (const item of state.catalogItems) {
          if (item.rankTargetId !== undefined) {
            rankIndex.set(targetKey(item.contentType, item.contentId), {
              rankTargetId: item.rankTargetId,
              score: item.score,
              reasons: item.reasons,
              metric: item.metric,
            });
          }
        }

        const [beans, articles] = await Promise.all([
          useBeans ? searchBeans(search) : Promise.resolve([]),
          useArticles ? searchArticles(search) : Promise.resolve([]),
        ]);

        let items = uniqueItems([
          ...beans.map((bean) =>
            mergeCatalogState(
              toBeanItem(bean, rankInfoFor(rankIndex, "bean", bean.id)),
              catalog,
            ),
          ),
          ...articles.map((article) =>
            mergeCatalogState(
              toArticleItem(
                article,
                rankInfoFor(rankIndex, "article", article.id),
              ),
              catalog,
            ),
          ),
        ]);

        if (items.length === 0) {
          const [fallbackBeans, fallbackArticles] = await Promise.all([
            useBeans ? listBeans(100) : Promise.resolve([]),
            useArticles ? listArticles(100) : Promise.resolve([]),
          ]);
          items = localSearch(
            fallbackBeans,
            fallbackArticles,
            search,
            rankIndex,
          );
        }

        setState((current) => ({
          ...current,
          items,
          loading: false,
          error: null,
        }));

        await recordEvent({
          event_type: "re_search",
          placement: "search_result",
          search_condition_hash: stableSearchHash(search),
          search_keyword: search.q.trim() || undefined,
          search_roast_level: search.roastLevel || undefined,
          search_category: search.category || undefined,
          page_path: "/search",
        }).catch(() => undefined);
      } catch (error) {
        setState((current) => ({
          ...current,
          items: [],
          loading: false,
          error: error instanceof Error ? error.message : "検索に失敗しました",
        }));
      } finally {
        setSearching(false);
      }
    },
    [load, state.catalogItems],
  );

  const visibleItems = useMemo(
    () => filterItems(state.items, activeFilter),
    [activeFilter, state.items],
  );

  return {
    state: { ...state, items: visibleItems },
    searching,
    reload: load,
    showCatalog,
    runSearch,
  };
}
