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
  FeedFilter,
  FeedItem,
  FeedState,
  SearchState,
} from "../src/types";

function toBeanItem(
  bean: Bean,
  rankTargetId?: number,
  score?: number,
): FeedItem {
  return {
    key: `bean-${bean.id}`,
    contentType: "bean",
    contentId: bean.id,
    rankTargetId,
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
    score,
    reasons: [],
    bean,
  };
}

function toArticleItem(
  article: Article,
  rankTargetId?: number,
  score?: number,
): FeedItem {
  return {
    key: `article-${article.id}`,
    contentType: "article",
    contentId: article.id,
    rankTargetId,
    title: article.title,
    subtitle:
      [categoryLabel(article.category), article.source_name]
        .filter(Boolean)
        .join(" / ") || "Article",
    summary: article.summary,
    body: article.body,
    imageUrl: article.image_url,
    badge: categoryLabel(article.category),
    score,
    reasons: [],
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

function localSearch(
  beans: Bean[],
  articles: Article[],
  search: SearchState,
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
        .map((bean) => toBeanItem(bean))
        .filter((item) => matchesKeyword(item, search.q))
    : [];

  const articleItems = useArticles
    ? articles
        .filter(
          (article) =>
            search.category === "" || article.category === search.category,
        )
        .map((article) => toArticleItem(article))
        .filter((item) => matchesKeyword(item, search.q))
    : [];

  return uniqueItems([...beanItems, ...articleItems]);
}

export function useFeedData(activeFilter: FeedFilter) {
  const [state, setState] = useState<FeedState>({
    items: [],
    loading: true,
    error: null,
  });
  const [searching, setSearching] = useState<boolean>(false);

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      await ensureGuestSession();
      const [beans, articles, recommendations] = await Promise.all([
        listBeans(100),
        listArticles(100),
        listRecommendations(activeFilter, 50).catch(() => []),
      ]);

      const beansByID = new Map(beans.map((bean) => [bean.id, bean]));
      const articlesByID = new Map(
        articles.map((article) => [article.id, article]),
      );
      const recommended = recommendations.flatMap((item) => {
        if (item.content_type === "bean") {
          const bean = beansByID.get(item.content_id);
          if (bean === undefined) {
            return [];
          }
          return [
            {
              ...toBeanItem(bean, item.rank_target_id, item.score),
              reasons: item.reasons ?? [],
              metric: item.metric,
            },
          ];
        }
        const article = articlesByID.get(item.content_id);
        if (article === undefined) {
          return [];
        }
        return [
          {
            ...toArticleItem(article, item.rank_target_id, item.score),
            reasons: item.reasons ?? [],
            metric: item.metric,
          },
        ];
      });

      const fallback = [
        ...beans.map((bean) => toBeanItem(bean)),
        ...articles.map((article) => toArticleItem(article)),
      ];

      setState({
        items: uniqueItems([...recommended, ...fallback]),
        loading: false,
        error: null,
      });
    } catch (error) {
      setState({
        items: [],
        loading: false,
        error:
          error instanceof Error ? error.message : "読み込みに失敗しました",
      });
    }
  }, [activeFilter]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  const runSearch = useCallback(
    async (search: SearchState) => {
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
        const [beans, articles] = await Promise.all([
          useBeans ? searchBeans(search) : Promise.resolve([]),
          useArticles ? searchArticles(search) : Promise.resolve([]),
        ]);
        let items = uniqueItems([
          ...beans.map((bean) => toBeanItem(bean)),
          ...articles.map((article) => toArticleItem(article)),
        ]);

        if (items.length === 0) {
          const [fallbackBeans, fallbackArticles] = await Promise.all([
            useBeans ? listBeans(100) : Promise.resolve([]),
            useArticles ? listArticles(100) : Promise.resolve([]),
          ]);
          items = localSearch(fallbackBeans, fallbackArticles, search);
        }

        setState({ items, loading: false, error: null });

        await recordEvent({
          event_type: "re_search",
          placement: "search_result",
          search_condition_hash: stableSearchHash(search),
          search_keyword: search.q.trim() || undefined,
          search_roast_level: search.roastLevel || undefined,
          search_category: search.category || undefined,
          page_path: "/",
        }).catch(() => undefined);
      } catch (error) {
        setState({
          items: [],
          loading: false,
          error: error instanceof Error ? error.message : "検索に失敗しました",
        });
      } finally {
        setSearching(false);
      }
    },
    [load],
  );

  const visibleItems = useMemo(
    () => filterItems(state.items, activeFilter),
    [activeFilter, state.items],
  );

  return {
    state: { ...state, items: visibleItems },
    searching,
    reload: load,
    runSearch,
  };
}
