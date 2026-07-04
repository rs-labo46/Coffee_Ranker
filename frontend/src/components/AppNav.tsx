import type { ReactNode } from "react";
import type { AppView, FeedFilter, User } from "../types";

type AppNavProps = {
  view: AppView;
  activeFilter: FeedFilter;
  onViewChange: (view: AppView) => void;
  onFilterChange: (filter: FeedFilter) => void;
  user: User | null;
  onRefresh: () => void;
};

type NavItem = {
  view: AppView;
  label: string;
  icon: ReactNode;
};

type FilterItem = {
  value: FeedFilter;
  label: string;
};

function HomeIcon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M3 11.5 12 4l9 7.5" />
      <path d="M5.5 10.5V20h13v-9.5" />
      <path d="M9.5 20v-5h5v5" />
    </svg>
  );
}

function SearchIcon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </svg>
  );
}

function AccountIcon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle cx="12" cy="8" r="4" />
      <path d="M4 21c1.6-4 4.4-6 8-6s6.4 2 8 6" />
    </svg>
  );
}

function AdminIcon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      className="h-5 w-5"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M12 3 4 6v5c0 5 3.4 8.5 8 10 4.6-1.5 8-5 8-10V6l-8-3Z" />
      <path d="M9 12h6" />
      <path d="M12 9v6" />
    </svg>
  );
}

const navItems: NavItem[] = [
  { view: "feed", label: "Feed", icon: <HomeIcon /> },
  { view: "search", label: "Search", icon: <SearchIcon /> },
  { view: "account", label: "Account", icon: <AccountIcon /> },
];

const adminNavItem: NavItem = {
  view: "admin",
  label: "Admin",
  icon: <AdminIcon />,
};

const filters: FilterItem[] = [
  { value: "all", label: "All" },
  { value: "bean", label: "Beans" },
  { value: "article", label: "Articles" },
];

export function AppNav({
  view,
  activeFilter,
  user,
  onViewChange,
  onFilterChange,
  onRefresh,
}: AppNavProps) {
  const visibleNavItems: NavItem[] =
    user?.role === "admin" ? [...navItems, adminNavItem] : navItems;

  return (
    <aside className="sticky top-0 hidden h-svh w-24 shrink-0 flex-col items-center gap-3 border-r border-white/10 bg-stone-950/82 px-3 py-3 backdrop-blur lg:flex xl:w-28">
      <button
        className="grid h-12 w-12 place-items-center rounded-2xl bg-amber-300 text-base font-black leading-none text-stone-950 shadow-lg shadow-amber-950/40 xl:h-14 xl:w-14 xl:text-lg"
        type="button"
        onClick={onRefresh}
        aria-label="再読み込み"
        title="再読み込み"
      >
        CR
      </button>

      <nav className="flex flex-col gap-2" aria-label="主要ナビゲーション">
        {visibleNavItems.map((item) => (
          <button
            key={item.view}
            type="button"
            className={`flex h-14 w-14 flex-col items-center justify-center rounded-2xl border text-xs transition xl:h-16 xl:w-16 ${
              view === item.view
                ? "border-amber-300/50 bg-amber-300/15 text-amber-100"
                : "border-white/10 bg-white/[0.03] text-stone-400 hover:bg-white/10 hover:text-white"
            }`}
            onClick={() => onViewChange(item.view)}
            aria-label={item.label}
            title={item.label}
          >
            <span className="grid h-5 w-5 place-items-center">{item.icon}</span>
            <small className="mt-1 text-[9px] font-bold xl:text-[10px]">
              {item.label}
            </small>
          </button>
        ))}
      </nav>

      <div className="mt-auto flex w-full flex-col gap-2 rounded-3xl border border-white/10 bg-white/[0.03] p-2">
        {filters.map((filter) => (
          <button
            key={filter.value}
            type="button"
            className={`w-full rounded-2xl px-2 py-2.5 text-center text-[11px] font-bold leading-none transition xl:text-xs ${
              activeFilter === filter.value
                ? "bg-white text-stone-950 shadow-sm shadow-black/30"
                : "text-stone-400 hover:bg-white/10 hover:text-white"
            }`}
            onClick={() => onFilterChange(filter.value)}
          >
            <span className="block truncate">{filter.label}</span>
          </button>
        ))}
      </div>
    </aside>
  );
}

export function BottomNav({
  view,
  onViewChange,
}: Pick<AppNavProps, "view" | "onViewChange">) {
  return (
    <nav className="fixed inset-x-2 bottom-2 z-50 rounded-[1.5rem] border border-white/10 bg-black/80 p-2 shadow-2xl shadow-black/60 backdrop-blur-xl lg:hidden">
      <div className="grid grid-cols-3 gap-2">
        {navItems.map((item) => (
          <button
            key={item.view}
            type="button"
            className={`flex h-12 items-center justify-center rounded-2xl transition ${
              view === item.view
                ? "bg-amber-300 text-stone-950"
                : "text-stone-400 hover:bg-white/10 hover:text-white"
            }`}
            onClick={() => onViewChange(item.view)}
            aria-label={item.label}
            title={item.label}
          >
            {item.icon}
          </button>
        ))}
      </div>
    </nav>
  );
}
