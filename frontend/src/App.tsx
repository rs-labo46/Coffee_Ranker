import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  clickModal,
  closeModal,
  getArticle,
  getBean,
  isAuthError,
  listRatings,
  listSavedItems,
  rateItem,
  recordEvent,
  removeSavedItem,
  saveItem,
  showModal,
} from "./api/client";
import { AppNav, BottomNav } from "./components/AppNav";
import { AdminPage } from "./components/AdminPage";
import { AuthPage } from "./components/AuthPage";
import { DetailPanel } from "./components/DetailPanel";
import { LoadingState } from "./components/LoadingState";
import { Notice } from "./components/Notice";
import { RecommendationModal } from "./components/RecommendationModal";
import { ReelsFeed } from "./components/ReelsFeed";
import { SearchPage } from "./components/SearchPage";
import { TopHeader } from "./components/TopHeader";
import { useAuthState } from "../hooks/useAuthState";
import { useFeedData } from "../hooks/useFeedData";
import type {
  AppView,
  DetailReturnView,
  FeedFilter,
  FeedItem,
  FeedItemKey,
  ModalShowResponse,
  Notice as NoticeType,
  Placement,
  RankTargetID,
  RatingScore,
} from "./types";

function placementFor(item: FeedItem): Placement {
  return item.contentType === "bean" ? "bean_detail" : "article_detail";
}

function pathFor(item: FeedItem): string {
  if (item.contentType === "article" && item.article !== undefined) {
    return `/articles/${item.article.slug}`;
  }
  return `/beans/${item.contentId}`;
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

function modalResponseToItem(response: ModalShowResponse): FeedItem | null {
  const target = response.target;
  if (target === undefined) {
    return null;
  }

  if (target.content_type === "bean" && response.bean !== undefined) {
    const bean = response.bean;
    return {
      key: `bean-${bean.id}`,
      contentType: "bean",
      contentId: bean.id,
      rankTargetId: target.id,
      title: bean.name,
      subtitle:
        [bean.origin, bean.roaster].filter(Boolean).join(" / ") ||
        "Coffee Bean",
      summary:
        bean.description ??
        bean.flavor_note ??
        "行動データから推薦されたコーヒー豆です。",
      body: bean.description,
      imageUrl: bean.image_url,
      badge: roastLabel(bean.roast_level),
      reasons: [],
      bean,
    };
  }

  if (target.content_type === "article" && response.article !== undefined) {
    const article = response.article;
    return {
      key: `article-${article.id}`,
      contentType: "article",
      contentId: article.id,
      rankTargetId: target.id,
      title: article.title,
      subtitle:
        [categoryLabel(article.category), article.source_name]
          .filter(Boolean)
          .join(" / ") || "Article",
      summary: article.summary,
      body: article.body,
      imageUrl: article.image_url,
      badge: categoryLabel(article.category),
      reasons: [],
      article,
    };
  }

  return null;
}

function App() {
  const [view, setView] = useState<AppView>("feed");
  const [activeFilter, setActiveFilter] = useState<FeedFilter>("all");
  const [activeItem, setActiveItem] = useState<FeedItem | null>(null);
  const [detailItem, setDetailItem] = useState<FeedItem | null>(null);
  const [actionNotice, setActionNotice] = useState<NoticeType | null>(null);
  const [feedRestoreRevision, setFeedRestoreRevision] = useState<number>(0);
  const [searchRestoreRevision, setSearchRestoreRevision] = useState<number>(0);
  const [detailReturnView, setDetailReturnView] =
    useState<DetailReturnView>("feed");
  const [detailRestoreItemKey, setDetailRestoreItemKey] =
    useState<FeedItemKey | null>(null);
  const [modalItem, setModalItem] = useState<FeedItem | null>(null);
  const [modalOpen, setModalOpen] = useState<boolean>(false);
  const [modalDisplayLogId, setModalDisplayLogId] = useState<number | null>(
    null,
  );
  const [modalPagePath, setModalPagePath] = useState<string>("/");
  const [savedTargetIds, setSavedTargetIds] = useState<Set<RankTargetID>>(
    () => new Set<RankTargetID>(),
  );
  const [ratingScores, setRatingScores] = useState<
    Map<RankTargetID, RatingScore>
  >(() => new Map<RankTargetID, RatingScore>());
  const viewedKeys = useRef<Set<FeedItemKey>>(new Set<FeedItemKey>());
  const impressedKeys = useRef<Set<FeedItemKey>>(new Set<FeedItemKey>());
  const modalShownKeys = useRef<Set<FeedItemKey>>(new Set<FeedItemKey>());
  const { state, searching, reload, runSearch } = useFeedData(activeFilter);
  const auth = useAuthState();
  const authUser = auth.user;
  const markSessionExpired = auth.markSessionExpired;

  const withActionState = useCallback(
    (item: FeedItem): FeedItem => {
      if (item.rankTargetId === undefined || authUser === null) {
        return {
          ...item,
          isSaved: false,
          ratingScore: null,
        };
      }

      return {
        ...item,
        isSaved: savedTargetIds.has(item.rankTargetId),
        ratingScore: ratingScores.get(item.rankTargetId) ?? null,
      };
    },
    [authUser, savedTargetIds, ratingScores],
  );

  const feedItems = useMemo(
    () => state.items.map((item) => withActionState(item)),
    [state.items, withActionState],
  );

  const catalogItems = useMemo(
    () => state.catalogItems.map((item) => withActionState(item)),
    [state.catalogItems, withActionState],
  );

  const currentActiveItem =
    activeItem !== null ? withActionState(activeItem) : (feedItems[0] ?? null);

  const currentDetailItem =
    detailItem !== null ? withActionState(detailItem) : null;
  const isAdmin = authUser?.role === "admin";

  const savedItems = useMemo(
    () =>
      catalogItems.filter(
        (item) =>
          item.rankTargetId !== undefined &&
          savedTargetIds.has(item.rankTargetId),
      ),
    [catalogItems, savedTargetIds],
  );

  const goodItems = useMemo(
    () =>
      catalogItems.filter(
        (item) =>
          item.rankTargetId !== undefined &&
          ratingScores.get(item.rankTargetId) === 1,
      ),
    [catalogItems, ratingScores],
  );

  const returnToDetailSource = useCallback(() => {
    setView(detailReturnView);
    setActionNotice(null);
    setDetailItem(null);

    if (detailReturnView === "search") {
      setSearchRestoreRevision((current) => current + 1);
      return;
    }

    if (detailReturnView === "feed") {
      setFeedRestoreRevision((current) => current + 1);
    }
  }, [detailReturnView]);

  const selectView = useCallback(
    (nextView: AppView) => {
      if (nextView === "admin" && authUser?.role !== "admin") {
        setActionNotice({
          tone: "error",
          message: "管理画面はAdminだけが利用できます。",
        });
        setView("account");
        return;
      }

      if (view === "detail") {
        if (nextView === "feed") {
          setFeedRestoreRevision((current) => current + 1);
        }
        if (nextView === "search") {
          setSearchRestoreRevision((current) => current + 1);
        }
      }

      if (nextView !== "detail") {
        setDetailItem(null);
      }

      setView(nextView);
      setActionNotice(null);
    },
    [authUser, view],
  );

  const selectFeedFilter = useCallback((filter: FeedFilter) => {
    setActiveFilter(filter);
    setView("feed");
    setActionNotice(null);
    setActiveItem(null);
    setDetailItem(null);
  }, []);

  const recordImpression = useCallback((item: FeedItem) => {
    if (
      item.rankTargetId === undefined ||
      impressedKeys.current.has(item.key)
    ) {
      return;
    }
    impressedKeys.current.add(item.key);
    void recordEvent({
      event_type: "impression",
      rank_target_id: item.rankTargetId,
      placement: "top",
      page_path: "/",
      dedup_key: `feed:${item.rankTargetId}:impression`,
      dedup_ttl_seconds: 3600,
    }).catch(() => undefined);
  }, []);

  const onActiveChange = useCallback(
    (item: FeedItem) => {
      setActiveItem(item);
      recordImpression(item);
    },
    [recordImpression],
  );

  useEffect(() => {
    if (authUser === null) {
      queueMicrotask(() => {
        setSavedTargetIds(new Set<RankTargetID>());
        setRatingScores(new Map<RankTargetID, RatingScore>());
      });
      return undefined;
    }

    let cancelled = false;

    async function loadUserActions(): Promise<void> {
      try {
        const [saved, ratings] = await Promise.all([
          listSavedItems(100, 0),
          listRatings(100, 0),
        ]);

        if (cancelled) {
          return;
        }

        setSavedTargetIds(
          new Set<RankTargetID>(
            saved.map((item) => item.rank_target_id as RankTargetID),
          ),
        );
        setRatingScores(
          new Map<RankTargetID, RatingScore>(
            ratings.map((rating) => [
              rating.rank_target_id as RankTargetID,
              rating.score,
            ]),
          ),
        );
      } catch (error) {
        if (!cancelled && isAuthError(error)) {
          markSessionExpired();
        }
      }
    }

    void loadUserActions();

    return () => {
      cancelled = true;
    };
  }, [authUser, markSessionExpired]);

  const onSelect = useCallback(
    async (item: FeedItem, source: DetailReturnView = "feed") => {
      setActiveItem(item);
      setModalOpen(false);
      setDetailReturnView(source);
      setDetailRestoreItemKey(item.key);

      if (item.rankTargetId !== undefined) {
        void recordEvent({
          event_type: "click",
          rank_target_id: item.rankTargetId,
          placement: source === "search" ? "search_result" : "top",
          page_path:
            source === "search"
              ? "/search"
              : source === "account"
                ? "/account"
                : "/",
          dedup_key: `${source}:${item.rankTargetId}:click:${Date.now()}`,
        }).catch(() => undefined);
      }

      if (item.contentType === "article") {
        if (authUser === null) {
          setDetailItem(null);
          setActionNotice({
            tone: "info",
            message: "記事の詳細を見るにはログインが必要です。",
          });
          setView("account");
          return;
        }

        if (item.article === undefined) {
          setDetailItem(null);
          setActionNotice({
            tone: "error",
            message: "記事情報が不足しているため詳細を開けません。",
          });
          return;
        }

        try {
          const article = await getArticle(item.article.slug);
          setDetailItem({
            ...item,
            article,
            body: article.body ?? item.body,
            summary: article.summary,
          });
          setView("detail");
          setActionNotice(null);
        } catch (error) {
          setDetailItem(null);
          if (isAuthError(error)) {
            markSessionExpired(
              "セッションの有効期限が切れました。記事詳細を見るにはログインし直してください。",
            );
            setActionNotice({
              tone: "info",
              message: "記事詳細を見るにはログインし直してください。",
            });
            setView("account");
            return;
          }

          setActionNotice({
            tone: "error",
            message:
              error instanceof Error ? error.message : "詳細取得に失敗しました",
          });
        }
        return;
      }

      setDetailItem(item);
      setView("detail");
      setActionNotice(null);

      try {
        const bean = await getBean(item.contentId);
        setDetailItem({
          ...item,
          bean,
          body: bean.description ?? item.body,
          summary: bean.description ?? item.summary,
        });
      } catch (error) {
        setActionNotice({
          tone: "error",
          message:
            error instanceof Error ? error.message : "詳細取得に失敗しました",
        });
      }
    },
    [authUser, markSessionExpired],
  );

  useEffect(() => {
    const item = detailItem;
    if (view !== "detail" || item === null || item.rankTargetId === undefined) {
      return undefined;
    }

    const viewKey = `${item.key}:content_view`;
    if (!viewedKeys.current.has(viewKey)) {
      viewedKeys.current.add(viewKey);
      void recordEvent({
        event_type: "content_view",
        rank_target_id: item.rankTargetId,
        placement: placementFor(item),
        page_path: pathFor(item),
        dedup_key: viewKey,
        dedup_ttl_seconds: 3600,
      }).catch(() => undefined);
    }

    const startedAt = Date.now();
    const timer = window.setTimeout(() => {
      const dwellMs = Date.now() - startedAt;
      void recordEvent({
        event_type: "stay",
        rank_target_id: item.rankTargetId,
        placement: placementFor(item),
        dwell_ms: dwellMs,
        page_path: pathFor(item),
      }).catch(() => undefined);
    }, 4000);

    return () => window.clearTimeout(timer);
  }, [detailItem, view]);

  useEffect(() => {
    const item = detailItem;
    if (view !== "detail" || item === null || item.rankTargetId === undefined) {
      return undefined;
    }

    const sourceRankTargetId = item.rankTargetId;
    const modalKey = `${item.key}:detail_modal`;
    if (modalShownKeys.current.has(modalKey)) {
      return undefined;
    }

    const timer = window.setTimeout(
      () => {
        const sourcePagePath = pathFor(item);
        const trigger =
          item.contentType === "bean" ? "bean_stay" : "article_stay";
        modalShownKeys.current.add(modalKey);
        void showModal(sourceRankTargetId, sourcePagePath, trigger)
          .then((log) => {
            const existing = [...catalogItems, ...feedItems].find(
              (content) =>
                content.key !== item.key &&
                content.rankTargetId === log.rank_target_id,
            );
            const candidate = existing ?? modalResponseToItem(log);
            if (candidate === null) {
              return;
            }
            setModalDisplayLogId(log.id);
            setModalPagePath(sourcePagePath);
            setModalItem(withActionState(candidate));
            setModalOpen(true);
          })
          .catch(() => undefined);
      },
      item.contentType === "bean" ? 15000 : 30000,
    );

    return () => window.clearTimeout(timer);
  }, [catalogItems, detailItem, feedItems, view, withActionState]);

  const onSave = useCallback(
    async (item: FeedItem) => {
      if (item.rankTargetId === undefined) {
        setActionNotice({
          tone: "error",
          message:
            "rank_target_idがないため保存できません。推薦API経由のカードで確認してください。",
        });
        return;
      }

      const rankTargetId = item.rankTargetId;
      const alreadySaved = savedTargetIds.has(rankTargetId);

      try {
        if (alreadySaved) {
          await removeSavedItem(rankTargetId);
          setSavedTargetIds((current) => {
            const next = new Set(current);
            next.delete(rankTargetId);
            return next;
          });
          setActionNotice({ tone: "success", message: "保存を解除しました" });
          return;
        }

        await saveItem(rankTargetId, placementFor(item), pathFor(item));
        setSavedTargetIds((current) => {
          const next = new Set(current);
          next.add(rankTargetId);
          return next;
        });
        setActionNotice({ tone: "success", message: "保存しました" });
      } catch (error) {
        if (isAuthError(error)) {
          markSessionExpired();
          setView("account");
          return;
        }

        setActionNotice({
          tone: "error",
          message:
            error instanceof Error ? error.message : "保存に失敗しました",
        });
      }
    },
    [markSessionExpired, savedTargetIds],
  );

  const onRate = useCallback(
    async (item: FeedItem, score: RatingScore) => {
      if (item.rankTargetId === undefined) {
        setActionNotice({
          tone: "error",
          message:
            "rank_target_idがないため評価できません。推薦API経由のカードで確認してください。",
        });
        return;
      }

      const rankTargetId = item.rankTargetId;

      try {
        await rateItem(rankTargetId, score, placementFor(item), pathFor(item));

        setRatingScores((current) => {
          const next = new Map(current);
          next.set(rankTargetId, score);
          return next;
        });

        setActionNotice(null);
      } catch (error) {
        if (isAuthError(error)) {
          markSessionExpired();
          setView("account");
          return;
        }

        setActionNotice({
          tone: "error",
          message:
            error instanceof Error ? error.message : "評価に失敗しました",
        });
      }
    },
    [markSessionExpired],
  );

  const openArticleLogin = useCallback(() => {
    setView("account");
  }, []);

  const openAdmin = useCallback(() => {
    if (authUser?.role !== "admin") {
      setActionNotice({
        tone: "error",
        message: "管理画面はAdminだけが利用できます。",
      });
      setView("account");
      return;
    }
    setActionNotice(null);
    setDetailItem(null);
    setView("admin");
  }, [authUser]);

  const openSearchResult = useCallback(
    (item: FeedItem) => {
      void onSelect(item, "search");
    },
    [onSelect],
  );

  const openAccountItem = useCallback(
    (item: FeedItem) => {
      void onSelect(item, "account");
    },
    [onSelect],
  );

  const closeRecommendationModal = useCallback(() => {
    const logID = modalDisplayLogId;
    const pagePath = modalPagePath;
    setModalOpen(false);
    setModalDisplayLogId(null);

    if (logID !== null) {
      void closeModal(logID, pagePath).catch(() => undefined);
    }
  }, [modalDisplayLogId, modalPagePath]);

  const openModalItem = useCallback(() => {
    if (modalItem === null) {
      return;
    }

    const item = modalItem;
    const logID = modalDisplayLogId;
    const pagePath = modalPagePath;
    setModalOpen(false);
    setModalDisplayLogId(null);

    if (logID !== null) {
      void clickModal(logID, pagePath).catch(() => undefined);
    }

    void onSelect(item, detailReturnView);
  }, [detailReturnView, modalDisplayLogId, modalItem, modalPagePath, onSelect]);

  return (
    <div className="min-h-svh bg-[radial-gradient(circle_at_top_left,rgba(245,158,11,0.16),transparent_30%),linear-gradient(135deg,#0c0a09,#1c1917_42%,#0c0a09)] text-stone-100">
      <div className="flex min-h-svh">
        <AppNav
          view={view}
          activeFilter={activeFilter}
          user={authUser}
          onViewChange={selectView}
          onFilterChange={selectFeedFilter}
          onRefresh={() => void reload()}
        />
        <div className="min-w-0 flex-1 pb-24 lg:pb-0">
          <TopHeader
            activeFilter={activeFilter}
            onFilterChange={selectFeedFilter}
            user={authUser}
          />

          {view === "feed" ? (
            <main className="mx-auto flex min-h-[calc(100svh_-_11.5rem_-_env(safe-area-inset-bottom))] max-w-[680px] flex-col justify-center px-3 py-4 sm:px-4 lg:min-h-[calc(100svh_-_6.5rem)] lg:px-6">
              <Notice notice={actionNotice} />
              {state.loading || state.error !== null ? (
                <LoadingState
                  error={state.error}
                  onRetry={() => void reload()}
                />
              ) : (
                <ReelsFeed
                  items={feedItems}
                  activeItem={currentActiveItem}
                  restoreItemKey={detailRestoreItemKey}
                  restoreRevision={feedRestoreRevision}
                  user={authUser}
                  showScore={activeFilter === "all"}
                  onActiveChange={onActiveChange}
                  onSelect={(item) => void onSelect(item, "feed")}
                  onSave={onSave}
                  onRate={onRate}
                />
              )}
            </main>
          ) : null}

          {view === "detail" ? (
            <main className="mx-auto flex min-h-[calc(100svh_-_13.5rem_-_env(safe-area-inset-bottom))] max-w-4xl flex-col justify-center px-4 py-4 pb-24 lg:min-h-[calc(100svh_-_6.5rem)] lg:px-8 lg:pb-4">
              <Notice notice={actionNotice} />
              <DetailPanel
                item={currentDetailItem}
                user={authUser}
                showMetrics={isAdmin}
                notice={actionNotice}
                onBack={returnToDetailSource}
                onSave={onSave}
                onRate={onRate}
                onOpenArticleLogin={openArticleLogin}
              />
            </main>
          ) : null}

          {view === "search" ? (
            <SearchPage
              activeFilter={activeFilter}
              items={feedItems}
              searching={searching}
              restoreItemKey={detailRestoreItemKey}
              restoreRevision={searchRestoreRevision}
              onSearch={runSearch}
              onSelect={openSearchResult}
            />
          ) : null}

          {view === "admin" && authUser?.role === "admin" ? (
            <AdminPage
              user={authUser}
              notice={actionNotice}
              onSessionExpired={(message?: string) => {
                markSessionExpired(message);
                setView("account");
              }}
            />
          ) : null}

          {view === "account" ? (
            <AuthPage
              user={authUser}
              loading={auth.loading}
              notice={auth.notice}
              savedItems={savedItems}
              goodItems={goodItems}
              onLogin={auth.loginUser}
              onSignup={auth.signupUser}
              onLogout={auth.logoutUser}
              onSelectItem={openAccountItem}
              onOpenAdmin={openAdmin}
            />
          ) : null}
        </div>
      </div>
      <RecommendationModal
        open={modalOpen}
        item={modalItem}
        onClose={closeRecommendationModal}
        onOpen={openModalItem}
      />
      <BottomNav view={view} onViewChange={selectView} />
    </div>
  );
}

export default App;
