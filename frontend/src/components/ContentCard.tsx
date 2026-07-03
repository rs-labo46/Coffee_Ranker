import type { MouseEvent } from "react";
import { ContentVisual } from "./ContentVisual";
import type { FeedItem, RatingScore, User } from "../types";

type ContentCardProps = {
  item: FeedItem;
  active: boolean;
  user: User | null;
  showScore?: boolean;
  onSelect: (item: FeedItem) => void;
  onSave: (item: FeedItem) => Promise<void>;
  onRate: (item: FeedItem, score: RatingScore) => Promise<void>;
};

const actionButtonBase =
  "group flex h-12 w-12 flex-col items-center justify-center rounded-2xl border shadow-2xl shadow-black/40 backdrop-blur-md transition hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-35 disabled:hover:translate-y-0 sm:h-14 sm:w-14";

const inactiveButton =
  "border-white/15 bg-black/35 text-white hover:bg-white/20";

const savedButton =
  "border-amber-300 bg-amber-300 text-stone-950 hover:bg-amber-200";

const goodButton =
  "border-amber-300 bg-amber-300 text-stone-950 hover:bg-amber-200";

const badButton = "border-rose-400 bg-rose-500 text-white hover:bg-rose-400";

function ActionButtons({
  item,
  user,
  onSave,
  onRate,
}: Pick<ContentCardProps, "item" | "user" | "onSave" | "onRate">) {
  const disabled = item.rankTargetId === undefined || user === null;
  const disabledLabel =
    user === null
      ? "ログインすると保存・評価できます"
      : "rank_target_idがないため操作できません";

  const isSaved = item.isSaved === true;
  const isGood = item.ratingScore === 1;
  const isBad = item.ratingScore === -1;

  const stop = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
  };

  return (
    <div className="absolute bottom-16 right-3 z-30 flex flex-col items-center gap-2 sm:bottom-18 sm:right-4 lg:bottom-18">
      <button
        type="button"
        className={`${actionButtonBase} ${isSaved ? savedButton : inactiveButton}`}
        disabled={disabled}
        title={
          disabled ? disabledLabel : isSaved ? "保存を解除する" : "保存する"
        }
        onClick={(event) => {
          stop(event);
          void onSave(item);
        }}
      >
        <span className="text-lg leading-none sm:text-xl">★</span>
        <span className="mt-1 text-[10px] font-bold leading-none text-current opacity-80">
          Save
        </span>
      </button>

      <button
        type="button"
        className={`${actionButtonBase} ${isGood ? goodButton : inactiveButton}`}
        disabled={disabled}
        title={disabled ? disabledLabel : "Good評価する"}
        onClick={(event) => {
          stop(event);
          void onRate(item, 1);
        }}
      >
        <span className="text-lg leading-none sm:text-xl">👍</span>
        <span className="mt-1 text-[10px] font-bold leading-none text-current opacity-80">
          Good
        </span>
      </button>

      <button
        type="button"
        className={`${actionButtonBase} ${isBad ? badButton : inactiveButton}`}
        disabled={disabled}
        title={disabled ? disabledLabel : "Bad評価する"}
        onClick={(event) => {
          stop(event);
          void onRate(item, -1);
        }}
      >
        <span className="text-lg leading-none sm:text-xl">↯</span>
        <span className="mt-1 text-[10px] font-bold leading-none text-current opacity-80">
          Bad
        </span>
      </button>
    </div>
  );
}

export function ContentCard({
  item,
  active,
  user,
  showScore = false,
  onSelect,
  onSave,
  onRate,
}: ContentCardProps) {
  const displayScore =
    item.score !== undefined ? Math.min(item.score / 10, 100) : undefined;

  return (
    <article
      className={`relative h-full min-h-0 snap-start overflow-hidden rounded-[1.8rem] border bg-stone-900 shadow-2xl shadow-black/50 transition duration-300 ${
        active
          ? "border-amber-300/60 ring-4 ring-amber-300/10"
          : "border-white/10"
      }`}
      data-item-key={item.key}
    >
      <button
        type="button"
        className="absolute inset-0 z-10 text-left"
        onClick={() => onSelect(item)}
        aria-label={`${item.title} の詳細を表示`}
      />

      <ContentVisual
        title={item.title}
        imageUrl={item.imageUrl}
        contentType={item.contentType}
      />

      <ActionButtons item={item} user={user} onSave={onSave} onRate={onRate} />

      <div className="pointer-events-none absolute inset-x-0 bottom-0 z-20 bg-gradient-to-t from-black via-black/78 to-transparent px-4 pb-5 pt-24 sm:px-6 sm:pb-6 sm:pt-28">
        <div className="max-h-[42svh] overflow-hidden pr-20 sm:pr-24 lg:max-h-[46svh]">
          <div className="mb-2 flex flex-wrap gap-2 sm:mb-3">
            <span className="rounded-full border border-white/15 bg-white/15 px-3 py-1 text-[10px] font-bold text-white backdrop-blur sm:text-xs">
              {item.contentType === "bean" ? "Bean" : "Article"}
            </span>

            <span className="rounded-full border border-white/10 bg-black/25 px-3 py-1 text-[10px] font-semibold text-stone-200 backdrop-blur sm:text-xs">
              {item.badge}
            </span>

            {showScore && displayScore !== undefined ? (
              <span className="hidden rounded-full border border-amber-300/30 bg-amber-300/15 px-3 py-1 text-[10px] font-semibold text-amber-100 backdrop-blur sm:inline-flex sm:text-xs">
                score {displayScore.toFixed(1)}
              </span>
            ) : null}
          </div>

          <h2 className="line-clamp-2 max-w-[34rem] break-words text-lg font-black leading-tight tracking-tight text-white sm:text-2xl lg:text-3xl">
            {item.title}
          </h2>

          <p className="mt-1 line-clamp-1 max-w-[34rem] break-words text-xs font-semibold text-stone-200 sm:mt-2 sm:text-sm">
            {item.subtitle}
          </p>

          <p className="mt-2 line-clamp-2 max-w-[34rem] break-words text-xs leading-5 text-stone-200 sm:mt-3 sm:line-clamp-3 sm:text-sm sm:leading-6">
            {item.summary}
          </p>

          {item.reasons.length > 0 ? (
            <div className="mt-3 hidden flex-wrap gap-2 sm:flex">
              {item.reasons.slice(0, 2).map((reason) => (
                <span
                  key={`${reason.dimension}-${reason.value}`}
                  className="line-clamp-1 max-w-full rounded-full border border-white/10 bg-white/10 px-3 py-1 text-xs text-stone-100 backdrop-blur"
                >
                  {reason.message}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </article>
  );
}
