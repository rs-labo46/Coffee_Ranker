import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  getArticle,
  getBean,
  rateItem,
  recordEvent,
  removeSavedItem,
  saveItem,
  showModal,
} from "./api/client";
import { AppNav, BottomNav } from "./components/AppNav";
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
  Notice as NoticeType,
  Placement,
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
  const [detailRestoreItemKey, setDetailRestoreItemKey] = useState<
    string | null
  >(null);
  const [modalItem, setModalItem] = useState<FeedItem | null>(null);
  const [modalOpen, setModalOpen] = useState<boolean>(false);
  const [savedTargetIds, setSavedTargetIds] = useState<Set<number>>(new Set());
  const [ratingScores, setRatingScores] = useState<Map<number, RatingScore>>(
    new Map(),
  );
  const viewedKeys = useRef(new Set<string>());
  const impressedKeys = useRef(new Set<string>());
  const modalShownKeys = useRef(new Set<string>());
  const { state, searching, reload, runSearch } = useFeedData(activeFilter);
  const auth = useAuthState();
  const withActionState = useCallback(
    (item: FeedItem): FeedItem => {
      if (item.rankTargetId === undefined) {
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
    [savedTargetIds, ratingScores],
  );

  const feedItems = useMemo(
    () => state.items.map((item) => withActionState(item)),
    [state.items, withActionState],
  );

  const currentActiveItem =
    activeItem !== null ? withActionState(activeItem) : (feedItems[0] ?? null);

  const currentDetailItem =
    detailItem !== null ? withActionState(detailItem) : null;
  const isAdmin = auth.user?.role === "admin";

  const returnToDetailSource = useCallback(() => {
    setView(detailReturnView);
    setActionNotice(null);
    setDetailItem(null);

    if (detailReturnView === "search") {
      setSearchRestoreRevision((current) => current + 1);
      return;
    }

    setFeedRestoreRevision((current) => current + 1);
  }, [detailReturnView]);

  const selectView = useCallback(
    (nextView: AppView) => {
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
    [view],
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

  const onSelect = useCallback(
    async (item: FeedItem, source: DetailReturnView = "feed") => {
      setActiveItem(item);
      setModalOpen(false);

      if (item.contentType === "article" && auth.user === null) {
        setDetailItem(null);
        setDetailReturnView(source);
        setDetailRestoreItemKey(item.key);
        setActionNotice({
          tone: "info",
          message: "記事の詳細を見るにはログインが必要です。",
        });
        setView("account");
        return;
      }

      setDetailItem(item);
      setDetailReturnView(source);
      setDetailRestoreItemKey(item.key);
      setView("detail");
      setActionNotice(null);

      if (item.rankTargetId !== undefined) {
        void recordEvent({
          event_type: "click",
          rank_target_id: item.rankTargetId,
          placement: source === "search" ? "search_result" : "top",
          page_path: source === "search" ? "/search" : "/",
          dedup_key: `${source}:${item.rankTargetId}:click:${Date.now()}`,
        }).catch(() => undefined);
      }

      try {
        if (item.contentType === "bean") {
          const bean = await getBean(item.contentId);
          setDetailItem({
            ...item,
            bean,
            body: bean.description ?? item.body,
            summary: bean.description ?? item.summary,
          });
          return;
        }

        if (auth.user !== null && item.article !== undefined) {
          const article = await getArticle(item.article.slug);
          setDetailItem({
            ...item,
            article,
            body: article.body ?? item.body,
            summary: article.summary,
          });
        }
      } catch (error) {
        setActionNotice({
          tone: "error",
          message:
            error instanceof Error ? error.message : "詳細取得に失敗しました",
        });
      }
    },
    [auth.user],
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

    const modalKey = `${item.key}:detail_modal`;
    if (modalShownKeys.current.has(modalKey)) {
      return undefined;
    }

    const timer = window.setTimeout(() => {
      const candidate = state.items.find(
        (content) =>
          content.key !== item.key && content.rankTargetId !== undefined,
      );
      if (candidate === undefined || candidate.rankTargetId === undefined) {
        return;
      }

      modalShownKeys.current.add(modalKey);
      void showModal(candidate.rankTargetId, pathFor(item))
        .then((result) => {
          if (result === undefined) {
            return;
          }
          setModalItem(candidate);
          setModalOpen(true);
        })
        .catch(() => undefined);
    }, 9000);

    return () => window.clearTimeout(timer);
  }, [detailItem, state.items, view]);

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
        setActionNotice({
          tone: "error",
          message:
            error instanceof Error ? error.message : "保存に失敗しました",
        });
      }
    },
    [savedTargetIds],
  );

  const onRate = useCallback(async (item: FeedItem, score: RatingScore) => {
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

      setActionNotice({
        tone: "success",
        message: score === 1 ? "Goodを記録しました" : "Badを記録しました",
      });
    } catch (error) {
      setActionNotice({
        tone: "error",
        message: error instanceof Error ? error.message : "評価に失敗しました",
      });
    }
  }, []);

  const openArticleLogin = useCallback(() => {
    setView("account");
  }, []);

  const openSearchResult = useCallback(
    (item: FeedItem) => {
      void onSelect(item, "search");
    },
    [onSelect],
  );

  const openModalItem = useCallback(() => {
    if (modalItem === null) {
      return;
    }
    setModalOpen(false);
    void onSelect(modalItem, detailReturnView);
  }, [detailReturnView, modalItem, onSelect]);

  return (
    <div className="min-h-svh bg-[radial-gradient(circle_at_top_left,rgba(245,158,11,0.16),transparent_30%),linear-gradient(135deg,#0c0a09,#1c1917_42%,#0c0a09)] text-stone-100">
      <div className="flex min-h-svh">
        <AppNav
          view={view}
          activeFilter={activeFilter}
          onViewChange={selectView}
          onFilterChange={selectFeedFilter}
          onRefresh={() => void reload()}
        />
        <div className="min-w-0 flex-1 pb-24 lg:pb-0">
          <TopHeader
            activeFilter={activeFilter}
            onFilterChange={selectFeedFilter}
            user={auth.user}
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
                  user={auth.user}
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
                user={auth.user}
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

          {view === "account" ? (
            <AuthPage
              user={auth.user}
              loading={auth.loading}
              notice={auth.notice}
              onLogin={auth.loginUser}
              onSignup={auth.signupUser}
              onLogout={auth.logoutUser}
            />
          ) : null}
        </div>
      </div>
      <RecommendationModal
        open={modalOpen}
        item={modalItem}
        onClose={() => setModalOpen(false)}
        onOpen={openModalItem}
      />
      <BottomNav view={view} onViewChange={selectView} />
    </div>
  );
}

export default App;
