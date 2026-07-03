import { ContentVisual } from "./ContentVisual";
import type { FeedItem } from "../types";

type RecommendationModalProps = {
  open: boolean;
  item: FeedItem | null;
  onClose: () => void;
  onOpen: () => void;
};

export function RecommendationModal({
  open,
  item,
  onClose,
  onOpen,
}: RecommendationModalProps) {
  if (!open || item === null) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 py-4 backdrop-blur-sm [padding-bottom:max(1rem,env(safe-area-inset-bottom))] [padding-top:max(1rem,env(safe-area-inset-top))]">
      <section className="flex max-h-[calc(100svh-2rem-env(safe-area-inset-top)-env(safe-area-inset-bottom))] w-full max-w-lg flex-col overflow-hidden rounded-[2rem] border border-amber-300/25 bg-stone-950 shadow-2xl shadow-black/60 sm:max-h-[min(760px,calc(100svh-2rem))]">
        <div className="shrink-0 p-4 pb-2 sm:p-5 sm:pb-3">
          <p className="text-xs font-black uppercase tracking-[0.32em] text-amber-300">
            Recommendation
          </p>
          <h2 className="mt-2 text-2xl font-black tracking-tight text-white sm:text-3xl">
            次にこれも見てみる
          </h2>
          <p className="mt-2 text-sm leading-6 text-stone-400">
            今見ている内容に近い候補です。開くと詳細画面へ移動します。
          </p>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-4 sm:px-5">
          <button
            type="button"
            className="block w-full overflow-hidden rounded-[1.5rem] border border-white/10 text-left transition hover:border-amber-300/50"
            onClick={onOpen}
          >
            <ContentVisual
              title={item.title}
              imageUrl={item.imageUrl}
              contentType={item.contentType}
              compact
            />
            <div className="p-4">
              <div className="flex flex-wrap gap-2">
                <span className="rounded-full border border-white/10 bg-white/10 px-3 py-1 text-xs font-bold text-white">
                  {item.contentType === "bean" ? "Bean" : "Article"}
                </span>
                <span className="rounded-full border border-white/10 bg-black/25 px-3 py-1 text-xs font-semibold text-stone-300">
                  {item.badge}
                </span>
              </div>
              <h3 className="mt-3 line-clamp-2 text-xl font-black leading-tight text-white">
                {item.title}
              </h3>
              <p className="mt-2 line-clamp-2 text-sm leading-6 text-stone-300">
                {item.summary}
              </p>
            </div>
          </button>
        </div>

        <div className="shrink-0 border-t border-white/10 bg-stone-950/95 p-4 [padding-bottom:max(1rem,env(safe-area-inset-bottom))]">
          <div className="flex gap-2">
            <button
              type="button"
              className="flex-1 rounded-full border border-white/10 bg-white/10 px-4 py-3 text-sm font-bold text-white transition hover:bg-white/15"
              onClick={onClose}
            >
              閉じる
            </button>
            <button
              type="button"
              className="flex-1 rounded-full bg-amber-300 px-4 py-3 text-sm font-black text-stone-950 transition hover:bg-amber-200"
              onClick={onOpen}
            >
              詳細を見る
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}
