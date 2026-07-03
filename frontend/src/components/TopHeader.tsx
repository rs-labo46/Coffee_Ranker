import type { FeedFilter, User } from "../types";

type TopHeaderProps = {
  activeFilter: FeedFilter;
  onFilterChange: (filter: FeedFilter) => void;
  user: User | null;
};

const filters: Array<{ value: FeedFilter; label: string }> = [
  { value: "all", label: "For You" },
  { value: "bean", label: "Beans" },
  { value: "article", label: "Articles" },
];

export function TopHeader({
  activeFilter,
  onFilterChange,
  user,
}: TopHeaderProps) {
  return (
    <header className="sticky top-0 z-30 border-b border-white/10 bg-stone-950/88 px-4 py-2.5 backdrop-blur-xl sm:px-6 lg:px-8 lg:py-3">
      <div className="mx-auto grid max-w-6xl grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-2 lg:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)]">
        <div className="min-w-0 justify-self-start text-left">
          <p className="text-[15px] font-bold uppercase tracking-[0.32em] text-amber-300 sm:text-xs">
            Coffee Ranker
          </p>
        </div>

        <nav
          className="order-3 col-span-2 flex justify-self-center rounded-full border border-white/10 bg-white/[0.04] p-1 lg:order-none lg:col-span-1 lg:col-start-2 lg:row-start-1"
          aria-label="表示カテゴリ"
        >
          {filters.map((filter) => (
            <button
              key={filter.value}
              type="button"
              className={`rounded-full px-4 py-2 text-sm font-bold transition sm:px-5 lg:px-6 ${
                filter.value === activeFilter
                  ? "bg-white text-stone-950"
                  : "text-stone-400 hover:text-white"
              }`}
              onClick={() => onFilterChange(filter.value)}
            >
              {filter.label}
            </button>
          ))}
        </nav>

        <div className="justify-self-end rounded-full border border-white/10 bg-white/[0.04] px-3 py-2 text-sm font-semibold text-stone-200 lg:col-start-3 lg:row-start-1">
          <span className="inline-flex items-center gap-2 align-middle">
            <span
              className="h-2 w-2 rounded-full bg-emerald-400"
              aria-hidden="true"
            />
            <span className="max-w-28 truncate">
              {user === null ? "Guest" : user.name}
            </span>
          </span>
        </div>
      </div>
    </header>
  );
}
