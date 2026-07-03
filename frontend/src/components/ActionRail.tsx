import type {
  FeedItem,
  Notice as NoticeType,
  RatingScore,
  User,
} from "../types";
import { Notice } from "./Notice";

type ActionRailProps = {
  item: FeedItem | null;
  user: User | null;
  onSave: (item: FeedItem) => Promise<void>;
  onRate: (item: FeedItem, score: RatingScore) => Promise<void>;
  notice: NoticeType | null;
  layout?: "vertical" | "horizontal";
  compact?: boolean;
};

const baseButton =
  "group flex flex-col items-center justify-center border shadow-2xl shadow-black/30 backdrop-blur transition hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-35 disabled:hover:translate-y-0";

const inactiveButton =
  "border-white/10 bg-white/10 text-white hover:bg-white/15 disabled:hover:bg-white/10";

const savedButton =
  "border-amber-300 bg-amber-300 text-stone-950 hover:bg-amber-200";

const goodButton =
  "border-amber-300 bg-amber-300 text-stone-950 hover:bg-amber-200";

const badButton = "border-rose-400 bg-rose-500 text-white hover:bg-rose-400";

export function ActionRail({
  item,
  user,
  onSave,
  onRate,
  notice,
  layout = "vertical",
  compact = false,
}: ActionRailProps) {
  const disabled =
    item === null || item.rankTargetId === undefined || user === null;

  const isSaved = item?.isSaved === true;
  const isGood = item?.ratingScore === 1;
  const isBad = item?.ratingScore === -1;

  const buttonSize = compact
    ? "h-11 min-w-12 rounded-2xl px-2 sm:h-12 sm:min-w-16 sm:px-3"
    : "h-16 w-16 rounded-2xl";

  const wrapperClass =
    layout === "horizontal"
      ? "flex flex-row flex-wrap items-center gap-2"
      : "flex flex-col gap-3";

  return (
    <aside className={wrapperClass} aria-label="コンテンツ操作">
      <button
        type="button"
        className={`${baseButton} ${buttonSize} ${
          isSaved ? savedButton : inactiveButton
        }`}
        disabled={disabled}
        title={isSaved ? "保存を解除する" : "保存する"}
        onClick={() => item !== null && void onSave(item)}
      >
        <span className={compact ? "text-base" : "text-xl"}>★</span>
        <small className="text-[10px] font-semibold leading-none text-current opacity-80 sm:text-[11px]">
          Save
        </small>
      </button>

      <button
        type="button"
        className={`${baseButton} ${buttonSize} ${
          isGood ? goodButton : inactiveButton
        }`}
        disabled={disabled}
        title="Good評価する"
        onClick={() => item !== null && void onRate(item, 1)}
      >
        <span className={compact ? "text-base" : "text-xl"}>👍</span>
        <small className="text-[10px] font-semibold leading-none text-current opacity-80 sm:text-[11px]">
          Good
        </small>
      </button>

      <button
        type="button"
        className={`${baseButton} ${buttonSize} ${
          isBad ? badButton : inactiveButton
        }`}
        disabled={disabled}
        title="Bad評価する"
        onClick={() => item !== null && void onRate(item, -1)}
      >
        <span className={compact ? "text-base" : "text-xl"}>↯</span>
        <small className="text-[10px] font-semibold leading-none text-current opacity-80 sm:text-[11px]">
          Bad
        </small>
      </button>

      {layout === "vertical" ? (
        <>
          <p className="max-w-20 text-center text-[11px] leading-5 text-stone-400">
            {user === null ? "Login required" : "推薦精度に反映"}
          </p>
          <div className="hidden w-64 lg:block">
            <Notice notice={notice} />
          </div>
        </>
      ) : (
        <p className="hidden text-xs font-semibold text-stone-400 sm:block">
          {user === null ? "ログインで操作可能" : "推薦精度に反映"}
        </p>
      )}
    </aside>
  );
}
