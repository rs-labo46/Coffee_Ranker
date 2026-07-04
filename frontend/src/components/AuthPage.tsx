import { useState, type FormEvent } from "react";
import type { FeedItem, Notice as NoticeType, User } from "../types";
import { Notice } from "./Notice";

type AuthPageProps = {
  user: User | null;
  loading: boolean;
  notice: NoticeType | null;
  savedItems: FeedItem[];
  goodItems: FeedItem[];
  onLogin: (email: string, password: string) => Promise<void>;
  onSignup: (name: string, email: string, password: string) => Promise<void>;
  onLogout: () => Promise<void>;
  onSelectItem: (item: FeedItem) => void;
};

type Mode = "login" | "signup";
type AccountSection = "saved" | "good";

type AccountGridProps = {
  title: string;
  description: string;
  items: FeedItem[];
  emptyMessage: string;
  onSelectItem: (item: FeedItem) => void;
};

type AccountSectionButton = {
  value: AccountSection;
  label: string;
  description: string;
  count: number;
};

const inputClass =
  "w-full rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm text-white outline-none transition placeholder:text-stone-500 focus:border-amber-300/50 focus:ring-4 focus:ring-amber-300/10";
const buttonClass =
  "w-full rounded-2xl bg-amber-300 px-5 py-3 text-sm font-black text-stone-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:opacity-60";

function AccountGrid({
  title,
  description,
  items,
  emptyMessage,
  onSelectItem,
}: AccountGridProps) {
  return (
    <section className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-4 shadow-2xl shadow-black/20 sm:p-5">
      <div className="mb-4 flex items-end justify-between gap-4">
        <div>
          <h3 className="text-base font-black text-white sm:text-lg">
            {title}
          </h3>
          <p className="mt-1 text-xs leading-5 text-stone-400">{description}</p>
        </div>
        <span className="rounded-full border border-white/10 bg-black/20 px-3 py-1 text-xs font-bold text-stone-300">
          {items.length}件
        </span>
      </div>

      {items.length === 0 ? (
        <div className="rounded-3xl border border-white/10 bg-black/20 p-5 text-sm leading-6 text-stone-400">
          {emptyMessage}
        </div>
      ) : (
        <div className="max-h-[34rem] overflow-y-auto pr-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <div className="grid grid-cols-3 gap-2 sm:gap-3">
            {items.map((item) => (
              <button
                key={`${title}-${item.key}`}
                type="button"
                className="group relative aspect-square overflow-hidden rounded-2xl border border-white/10 bg-stone-900 text-left shadow-lg shadow-black/20 transition hover:-translate-y-0.5 hover:border-amber-300/50"
                onClick={() => onSelectItem(item)}
                aria-label={`${item.title} の詳細を表示`}
                title={item.title}
              >
                {item.imageUrl !== undefined && item.imageUrl.trim() !== "" ? (
                  <img
                    src={item.imageUrl}
                    alt=""
                    className="absolute inset-0 h-full w-full object-cover transition duration-300 group-hover:scale-105"
                    loading="lazy"
                  />
                ) : (
                  <div className="absolute inset-0 bg-gradient-to-br from-amber-950 via-stone-900 to-stone-950" />
                )}
                <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent" />
                <span className="absolute bottom-2 left-2 rounded-full bg-black/55 px-2 py-1 text-[10px] font-bold text-white backdrop-blur-sm">
                  {item.contentType === "bean" ? "Bean" : "Article"}
                </span>
              </button>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

export function AuthPage({
  user,
  loading,
  notice,
  savedItems,
  goodItems,
  onLogin,
  onSignup,
  onLogout,
  onSelectItem,
}: AuthPageProps) {
  const [mode, setMode] = useState<Mode>("login");
  const [accountSection, setAccountSection] = useState<AccountSection>("saved");
  const [name, setName] = useState<string>("");
  const [email, setEmail] = useState<string>("");
  const [password, setPassword] = useState<string>("");

  const accountSections: AccountSectionButton[] = [
    {
      value: "saved",
      label: "保存したコンテンツ",
      description: "Saveした豆・記事",
      count: savedItems.length,
    },
    {
      value: "good",
      label: "Goodしたコンテンツ",
      description: "Good評価した豆・記事",
      count: goodItems.length,
    },
  ];

  function submitAuth(event: FormEvent<HTMLFormElement>): void {
    event.preventDefault();
    if (mode === "signup") {
      void onSignup(name, email, password);
      return;
    }
    void onLogin(email, password);
  }

  if (user !== null) {
    const selectedItems = accountSection === "saved" ? savedItems : goodItems;
    const selectedTitle =
      accountSection === "saved" ? "保存したコンテンツ" : "Goodしたコンテンツ";
    const selectedDescription =
      accountSection === "saved"
        ? "Saveした豆・記事を3列グリッドで確認できます。"
        : "Good評価した豆・記事を3列グリッドで確認できます。";
    const emptyMessage =
      accountSection === "saved"
        ? "まだ保存したコンテンツがありません。Feedや詳細画面からSaveしてください。"
        : "まだGood評価したコンテンツがありません。気に入ったカードでGoodを押してください。";

    return (
      <section className="mx-auto grid max-w-6xl gap-6 px-4 py-8 lg:grid-cols-[1fr_420px] lg:px-8">
        <div className="rounded-[2rem] border border-white/10 bg-gradient-to-br from-amber-300/15 via-stone-900 to-stone-950 p-8 shadow-2xl shadow-black/40">
          <p className="text-xs font-bold uppercase tracking-[0.28em] text-amber-300">
            Account
          </p>
          <h2 className="mt-4 text-4xl font-black tracking-tight text-white">
            ログイン済み
          </h2>
          <p className="mt-4 max-w-xl text-sm leading-7 text-stone-300">
            保存、Good/Bad評価、Article詳細本文の閲覧が有効です。推薦精度は行動イベントにより改善されます。
          </p>
        </div>

        <aside className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 text-center shadow-2xl shadow-black/30">
          <div className="mx-auto grid h-16 w-16 place-items-center rounded-3xl bg-amber-300 text-xl font-black leading-none text-stone-950">
            <span className="block translate-y-[1px]">
              {user.name.slice(0, 2).toUpperCase()}
            </span>
          </div>
          <h3 className="mt-5 break-words text-2xl font-black text-white">
            {user.name}
          </h3>
          <p className="mt-1 break-words text-sm text-stone-400">
            {user.email}
          </p>
          <div className="mt-5 rounded-2xl border border-white/10 bg-black/20 p-4 text-sm text-stone-300">
            role: <strong className="text-white">{user.role}</strong>
          </div>
          <button
            type="button"
            className="mt-5 w-full rounded-2xl border border-white/10 bg-white/10 px-5 py-3 text-sm font-bold text-white transition hover:bg-white/15 disabled:opacity-60"
            onClick={() => void onLogout()}
            disabled={loading}
          >
            ログアウト
          </button>
          <div className="mt-4">
            <Notice notice={notice} />
          </div>
        </aside>

        <div className="grid gap-5 lg:col-span-2 lg:grid-cols-[220px_minmax(0,1fr)]">
          <aside className="h-fit rounded-[2rem] border border-white/10 bg-white/[0.04] p-3 shadow-2xl shadow-black/20 lg:sticky lg:top-28">
            <p className="px-3 py-2 text-xs font-bold uppercase tracking-[0.28em] text-amber-300">
              Library
            </p>
            <div className="mt-2 grid gap-2">
              {accountSections.map((section) => (
                <button
                  key={section.value}
                  type="button"
                  className={`rounded-2xl border px-3 py-3 text-left transition ${
                    accountSection === section.value
                      ? "border-amber-300/40 bg-amber-300 text-stone-950"
                      : "border-white/10 bg-black/20 text-stone-300 hover:bg-white/10 hover:text-white"
                  }`}
                  onClick={() => setAccountSection(section.value)}
                >
                  <span className="block text-sm font-black">
                    {section.label}
                  </span>
                  <span className="mt-1 block text-xs opacity-75">
                    {section.description} / {section.count}件
                  </span>
                </button>
              ))}
            </div>
          </aside>

          <AccountGrid
            title={selectedTitle}
            description={selectedDescription}
            items={selectedItems}
            emptyMessage={emptyMessage}
            onSelectItem={onSelectItem}
          />
        </div>
      </section>
    );
  }

  return (
    <section
      id="login-panel"
      className="mx-auto grid max-w-5xl gap-6 px-4 py-8 lg:grid-cols-[1fr_420px] lg:px-8"
    >
      <aside className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/30">
        <p className="py-5 text-lg font-bold uppercase tracking-[0.28em] text-amber-300">
          Account
        </p>
        <div className="grid grid-cols-2 gap-2 rounded-2xl bg-black/20 p-1">
          <button
            type="button"
            className={`rounded-xl px-4 py-3 text-sm font-bold transition ${mode === "signup" ? "bg-white text-stone-950" : "text-stone-400"}`}
            onClick={() => setMode("signup")}
          >
            サインアップ
          </button>
          <button
            type="button"
            className={`rounded-xl px-4 py-3 text-sm font-bold transition ${mode === "login" ? "bg-white text-stone-950" : "text-stone-400"}`}
            onClick={() => setMode("login")}
          >
            ログイン
          </button>
        </div>
        <form className="mt-6 space-y-3" onSubmit={submitAuth}>
          {mode === "signup" ? (
            <input
              className={inputClass}
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="表示名"
              maxLength={50}
            />
          ) : null}
          <input
            className={inputClass}
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="email@example.com"
            type="email"
          />
          <input
            className={inputClass}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder="password"
            type="password"
          />
          <button type="submit" className={buttonClass} disabled={loading}>
            {loading
              ? "処理中"
              : mode === "signup"
                ? "サインアップ"
                : "ログイン"}
          </button>
        </form>

        <div className="mt-4">
          <Notice notice={notice} />
        </div>
      </aside>
    </section>
  );
}
