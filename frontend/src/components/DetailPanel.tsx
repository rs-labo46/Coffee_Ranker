import { ActionRail } from "./ActionRail";
import { ContentVisual } from "./ContentVisual";
import { MetricPanel } from "./MetricPanel";
import { TasteMeter } from "./TasteMeter";
import type {
  FeedItem,
  Notice as NoticeType,
  RatingScore,
  User,
} from "../types";

type DetailPanelProps = {
  item: FeedItem | null;
  user: User | null;
  showMetrics: boolean;
  notice: NoticeType | null;
  onBack: () => void;
  onSave: (item: FeedItem) => Promise<void>;
  onRate: (item: FeedItem, score: RatingScore) => Promise<void>;
  onOpenArticleLogin: () => void;
};

export function DetailPanel({
  item,
  user,
  showMetrics,
  notice,
  onBack,
  onSave,
  onRate,
  onOpenArticleLogin,
}: DetailPanelProps) {
  if (item === null) {
    return (
      <aside className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/30">
        <p className="text-xs font-bold uppercase tracking-[0.28em] text-amber-300">
          Detail
        </p>
        <h2 className="mt-3 text-2xl font-black tracking-tight text-white">
          カードを選ぶと詳細がここに出ます
        </h2>
        <p className="mt-3 text-sm leading-6 text-stone-400">
          枠組みは固定し、中央のカードと右側の内容だけを切り替える構成です。
        </p>
      </aside>
    );
  }

  const actions = (
    <ActionRail
      item={item}
      user={user}
      onSave={onSave}
      onRate={onRate}
      notice={notice}
      layout="horizontal"
      compact
    />
  );

  return (
    <aside className="w-full max-h-[calc(100svh_-_14.5rem_-_env(safe-area-inset-bottom))] overflow-y-auto rounded-[2rem] border border-white/10 bg-white/[0.04] p-4 shadow-2xl shadow-black/30 [scrollbar-width:none] lg:max-h-[calc(100svh_-_8rem)] [&::-webkit-scrollbar]:hidden">
      <div className="relative">
        <ContentVisual
          title={item.title}
          imageUrl={item.imageUrl}
          contentType={item.contentType}
          compact
        />
        <button
          type="button"
          className="absolute left-3 top-3 z-30 inline-flex h-10 w-10 items-center justify-center rounded-full bg-black/45 text-2xl font-black leading-none text-white shadow-lg shadow-black/30 backdrop-blur-sm transition hover:bg-black/65 focus:outline-none focus:ring-2 focus:ring-amber-300/70"
          onClick={onBack}
          aria-label="Feedへ戻る"
        >
          ←
        </button>
      </div>

      <div className="p-2 pt-5">
        <div className="flex flex-wrap gap-2">
          <span className="rounded-full bg-white/10 px-3 py-1 text-xs font-bold text-white">
            {item.contentType === "bean" ? "Bean Detail" : "Article Detail"}
          </span>
          <span className="rounded-full bg-black/20 px-3 py-1 text-xs font-semibold text-stone-300">
            {item.badge}
          </span>
        </div>
        <h2 className="mt-4 text-2xl font-black leading-tight tracking-tight text-white">
          {item.title}
        </h2>
        <p className="mt-2 text-sm font-semibold text-stone-300">
          {item.subtitle}
        </p>
        <p className="mt-5 text-sm leading-7 text-stone-300">
          {item.body ?? item.summary}
        </p>

        {item.contentType === "article" &&
        item.article?.body === undefined &&
        user === null ? (
          <div className="mt-5 rounded-3xl border border-amber-300/20 bg-amber-300/10 p-4">
            <p className="text-sm font-bold text-amber-100">
              記事本文を読むにはログインが必要です。
            </p>
            <p className="mt-2 text-xs leading-5 text-amber-50/80">
              ログイン後、詳細本文と保存・評価が使えるようになります。
            </p>
            <button
              type="button"
              className="mx-auto mt-4 block rounded-full bg-amber-300 px-4 py-2 text-sm font-black text-stone-950 transition hover:bg-amber-200"
              onClick={onOpenArticleLogin}
            >
              ログインする
            </button>
          </div>
        ) : null}

        {item.bean !== undefined ? (
          <div className="mt-5 grid grid-cols-2 gap-3">
            <TasteMeter label="酸味" value={item.bean.acidity} />
            <TasteMeter label="苦味" value={item.bean.bitterness} />
            <TasteMeter label="風味" value={item.bean.flavor} />
            <TasteMeter label="香り" value={item.bean.aroma} />
            <TasteMeter label="ボディ" value={item.bean.body} />

            <div className="col-start-2 justify-self-end rounded-2xl border border-white/10 bg-black/20 p-2">
              <p className="mb-2 px-1 text-right text-xs font-semibold text-stone-300">
                保存・評価
              </p>
              <div className="flex justify-end">{actions}</div>
            </div>
          </div>
        ) : (
          <div className="mt-5 flex justify-end">
            <div className="rounded-2xl border border-white/10 bg-black/20 p-3">
              <div className="flex justify-end">{actions}</div>
            </div>
          </div>
        )}

        {item.reasons.length > 0 ? (
          <section className="mt-5 rounded-3xl border border-white/10 bg-black/20 p-4">
            <h3 className="text-sm font-bold text-white">推薦理由</h3>
            <div className="mt-3 space-y-2">
              {item.reasons.slice(0, 4).map((reason) => (
                <p
                  key={`${reason.dimension}-${reason.value}`}
                  className="text-xs leading-5 text-stone-300"
                >
                  {reason.message}
                </p>
              ))}
            </div>
          </section>
        ) : null}

        {showMetrics ? (
          <div className="mt-5">
            <MetricPanel metric={item.metric} />
          </div>
        ) : null}
      </div>
    </aside>
  );
}
