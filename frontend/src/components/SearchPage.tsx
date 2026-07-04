import { useEffect, useRef, useState } from "react";
import { ContentVisual } from "./ContentVisual";
import type { FeedFilter, FeedItem, SearchState, SortKey } from "../types";

type SearchPageProps = {
  activeFilter: FeedFilter;
  items: FeedItem[];
  searching: boolean;
  restoreScrollY: number | null;
  restoreRevision: number;
  onScrollPositionChange: (scrollY: number) => void;
  onSearch: (state: SearchState) => Promise<void>;
  onSelect: (item: FeedItem) => void;
};

const defaultSearch: SearchState = {
  q: "",
  sort: "score",
  contentType: "all",
  roastLevel: "",
  category: "",
};

const inputClass =
  "w-full min-h-12 rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-base text-white outline-none transition placeholder:text-stone-500 focus:border-amber-300/50 focus:ring-4 focus:ring-amber-300/10";

const selectClass =
  "w-full min-h-12 appearance-none rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 pr-10 text-base font-semibold text-white outline-none transition focus:border-amber-300/60 focus:ring-4 focus:ring-amber-300/10";

function toFeedFilter(value: string): FeedFilter {
  if (value === "bean" || value === "article") {
    return value;
  }
  return "all";
}

function toSortKey(value: string): SortKey {
  if (value === "newest" || value === "popular") {
    return value;
  }
  return "score";
}

function toRoastLevel(value: string): SearchState["roastLevel"] {
  if (value === "light" || value === "medium" || value === "dark") {
    return value;
  }
  return "";
}

function toArticleCategory(value: string): SearchState["category"] {
  if (
    value === "brewing" ||
    value === "roast" ||
    value === "beans" ||
    value === "recipe"
  ) {
    return value;
  }
  return "";
}

export function SearchPage({
  activeFilter,
  items,
  searching,
  restoreScrollY,
  restoreRevision,
  onScrollPositionChange,
  onSearch,
  onSelect,
}: SearchPageProps) {
  const restoredRevisionRef = useRef<number>(0);
  const [state, setState] = useState<SearchState>({
    ...defaultSearch,
    contentType: activeFilter,
  });

  useEffect(() => {
    if (
      restoreScrollY === null ||
      restoreRevision === 0 ||
      restoredRevisionRef.current === restoreRevision
    ) {
      return;
    }

    restoredRevisionRef.current = restoreRevision;
    window.requestAnimationFrame(() => {
      window.scrollTo({ top: restoreScrollY, behavior: "auto" });
    });
  }, [restoreScrollY, restoreRevision]);

  useEffect(() => {
    const handleScroll = (): void => {
      onScrollPositionChange(window.scrollY);
    };

    window.addEventListener("scroll", handleScroll, { passive: true });
    return () => window.removeEventListener("scroll", handleScroll);
  }, [onScrollPositionChange]);

  function change(next: Partial<SearchState>): void {
    setState((current) => ({ ...current, ...next }));
  }

  return (
    <section className="mx-auto max-w-7xl px-4 py-8 lg:px-8">
      <div className="grid gap-6 lg:grid-cols-[420px_1fr]">
        <aside className="h-fit rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/30 lg:sticky lg:top-28">
          <p className="text-lg font-bold uppercase tracking-[0.28em] text-amber-300">
            Search
          </p>

          <form
            className="mt-6 space-y-3"
            onSubmit={(event) => {
              event.preventDefault();
              void onSearch(state);
            }}
          >
            <input
              className={inputClass}
              value={state.q}
              onChange={(event) => change({ q: event.target.value })}
              placeholder="産地、抽出、味のキーワード"
              maxLength={100}
            />
            <div className="grid grid-cols-2 gap-3">
              <select
                className={selectClass}
                value={state.contentType}
                onChange={(event) =>
                  change({ contentType: toFeedFilter(event.target.value) })
                }
              >
                <option value="all">すべて</option>
                <option value="bean">豆</option>
                <option value="article">記事</option>
              </select>
              <select
                className={selectClass}
                value={state.sort}
                onChange={(event) =>
                  change({ sort: toSortKey(event.target.value) })
                }
              >
                <option value="score">スコア順</option>
                <option value="newest">新着順</option>
                <option value="popular">人気順</option>
              </select>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <select
                className={selectClass}
                value={state.roastLevel}
                onChange={(event) =>
                  change({ roastLevel: toRoastLevel(event.target.value) })
                }
              >
                <option value="">焙煎度</option>
                <option value="light">浅煎り</option>
                <option value="medium">中煎り</option>
                <option value="dark">深煎り</option>
              </select>
              <select
                className={selectClass}
                value={state.category}
                onChange={(event) =>
                  change({ category: toArticleCategory(event.target.value) })
                }
              >
                <option value="">記事カテゴリ</option>
                <option value="brewing">抽出</option>
                <option value="roast">焙煎</option>
                <option value="beans">豆知識</option>
                <option value="recipe">レシピ</option>
              </select>
            </div>
            <button
              type="submit"
              className="w-full rounded-2xl bg-amber-300 px-5 py-3 text-sm font-black text-stone-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:opacity-60"
              disabled={searching}
            >
              {searching ? "検索中" : "検索する"}
            </button>
          </form>
        </aside>

        <div>
          <div className="mb-4 flex items-end justify-between gap-4">
            <div>
              <p className="text-xs font-bold uppercase tracking-[0.28em] text-stone-500">
                Results
              </p>
              <h3 className="mt-1 text-2xl font-black text-white">
                {items.length} 件
              </h3>
            </div>
          </div>
          {items.length === 0 ? (
            <div className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-8 text-stone-400">
              条件を入れて検索してください。
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {items.map((item) => (
                <button
                  key={item.key}
                  data-item-key={item.key}
                  type="button"
                  className="group overflow-hidden rounded-[1.7rem] border border-white/10 bg-white/[0.04] p-3 text-left shadow-xl shadow-black/20 transition hover:-translate-y-1 hover:border-amber-300/40"
                  onClick={() => onSelect(item)}
                >
                  <ContentVisual
                    title={item.title}
                    imageUrl={item.imageUrl}
                    contentType={item.contentType}
                    compact
                  />
                  <div className="p-3">
                    <div className="flex gap-2">
                      <span className="rounded-full bg-white/10 px-2.5 py-1 text-[11px] font-bold text-stone-200">
                        {item.contentType}
                      </span>
                      <span className="rounded-full bg-black/20 px-2.5 py-1 text-[11px] font-bold text-stone-400">
                        {item.badge}
                      </span>
                    </div>
                    <h4 className="mt-3 line-clamp-2 text-base font-black leading-snug text-white">
                      {item.title}
                    </h4>
                    <p className="mt-2 line-clamp-2 text-xs leading-5 text-stone-400">
                      {item.summary}
                    </p>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
