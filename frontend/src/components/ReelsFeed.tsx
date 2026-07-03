import { useEffect, useRef } from "react";
import { ContentCard } from "./ContentCard";
import type { FeedItem, RatingScore, User } from "../types";

type ReelsFeedProps = {
  items: FeedItem[];
  activeItem: FeedItem | null;
  restoreItemKey: string | null;
  restoreRevision: number;
  user: User | null;
  showScore?: boolean;
  onActiveChange: (item: FeedItem) => void;
  onSelect: (item: FeedItem) => void;
  onSave: (item: FeedItem) => Promise<void>;
  onRate: (item: FeedItem, score: RatingScore) => Promise<void>;
};

const feedHeight =
  "h-[calc(100svh_-_13.75rem_-_env(safe-area-inset-bottom))] min-h-[420px] lg:h-[calc(100svh_-_10.5rem)] lg:min-h-[560px] lg:max-h-[760px]";

const feedOffset = "translate-y-6 lg:translate-y-0";

function escapeSelector(value: string): string {
  if (typeof CSS !== "undefined" && CSS.escape !== undefined) {
    return CSS.escape(value);
  }

  return value.replace(/["\\]/g, "\\$&");
}

export function ReelsFeed({
  items,
  activeItem,
  restoreItemKey,
  restoreRevision,
  user,
  showScore = false,
  onActiveChange,
  onSelect,
  onSave,
  onRate,
}: ReelsFeedProps) {
  const feedRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const root = feedRef.current;
    if (root === null || restoreItemKey === null || restoreRevision === 0) {
      return;
    }

    window.requestAnimationFrame(() => {
      const target = root.querySelector<HTMLElement>(
        `[data-item-key="${escapeSelector(restoreItemKey)}"]`,
      );

      target?.scrollIntoView({ block: "start", behavior: "auto" });
    });
  }, [items, restoreItemKey, restoreRevision]);

  useEffect(() => {
    const root = feedRef.current;
    if (root === null) {
      return undefined;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];

        const key = visible?.target.getAttribute("data-item-key");
        if (key === null || key === undefined) {
          return;
        }

        const item = items.find((candidate) => candidate.key === key);
        if (item !== undefined) {
          onActiveChange(item);
        }
      },
      { root, threshold: [0.65, 0.8, 0.95] },
    );

    const cards = root.querySelectorAll<HTMLElement>("[data-item-key]");
    cards.forEach((card) => observer.observe(card));

    return () => observer.disconnect();
  }, [items, onActiveChange]);

  if (items.length === 0) {
    return (
      <div
        className={`flex ${feedHeight} ${feedOffset} flex-col justify-end rounded-[1.8rem] border border-white/10 bg-stone-900 p-6 shadow-2xl shadow-black/40 lg:p-8`}
      >
        <p className="text-xs font-bold uppercase tracking-[0.28em] text-amber-300">
          No contents
        </p>
        <h2 className="mt-3 text-2xl font-black tracking-tight text-white lg:text-3xl">
          表示できる豆・記事がありません
        </h2>
        <p className="mt-3 max-w-md text-sm leading-6 text-stone-400">
          API起動、Seed投入、公開状態を確認してください。
        </p>
      </div>
    );
  }

  return (
    <main
      className={`${feedHeight} ${feedOffset} snap-y snap-mandatory space-y-4 overflow-y-auto scroll-smooth rounded-[1.8rem] pr-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden`}
      ref={feedRef}
      aria-label="コーヒーフィード"
    >
      {items.map((item) => (
        <ContentCard
          key={item.key}
          item={item}
          active={activeItem?.key === item.key}
          user={user}
          showScore={showScore}
          onSelect={onSelect}
          onSave={onSave}
          onRate={onRate}
        />
      ))}
    </main>
  );
}
