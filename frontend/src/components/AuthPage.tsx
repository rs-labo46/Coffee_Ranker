import { useState } from "react";
import type { Notice as NoticeType, User } from "../types";
import { Notice } from "./Notice";

type AuthPageProps = {
  user: User | null;
  loading: boolean;
  notice: NoticeType | null;
  onLogin: (email: string, password: string) => Promise<void>;
  onSignup: (name: string, email: string, password: string) => Promise<void>;
  onLogout: () => Promise<void>;
};

type Mode = "login" | "signup";

const inputClass =
  "w-full rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm text-white outline-none transition placeholder:text-stone-500 focus:border-amber-300/50 focus:ring-4 focus:ring-amber-300/10";
const buttonClass =
  "w-full rounded-2xl bg-amber-300 px-5 py-3 text-sm font-black text-stone-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:opacity-60";

export function AuthPage({
  user,
  loading,
  notice,
  onLogin,
  onSignup,
  onLogout,
}: AuthPageProps) {
  const [mode, setMode] = useState<Mode>("login");
  const [name, setName] = useState<string>("");
  const [email, setEmail] = useState<string>("");
  const [password, setPassword] = useState<string>("");

  if (user !== null) {
    return (
      <section className="mx-auto grid max-w-5xl gap-6 px-4 py-8 lg:grid-cols-[1fr_420px] lg:px-8">
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
        <aside className="text-center rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/30">
          <div className="mx-auto grid h-16 w-16 place-items-center rounded-3xl bg-amber-300 text-center text-xl font-black leading-none text-stone-950">
            <span className="block translate-y-[1px]">
              {user.name.slice(0, 2).toUpperCase()}
            </span>
          </div>
          <h3 className="mt-5 text-2xl font-black text-white">{user.name}</h3>
          <p className="mt-1 text-sm text-stone-400">{user.email}</p>
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
      </section>
    );
  }

  return (
    <section
      id="login-panel"
      className="mx-auto grid max-w-5xl gap-6 px-4 py-8 lg:grid-cols-[1fr_420px] lg:px-8"
    >
      <aside className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 shadow-2xl shadow-black/30">
        <p className="text-lg font-bold py-5 uppercase tracking-[0.28em] text-amber-300">
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
        <form
          className="mt-6 space-y-3"
          onSubmit={(event) => {
            event.preventDefault();
            if (mode === "signup") {
              void onSignup(name, email, password);
              return;
            }
            void onLogin(email, password);
          }}
        >
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
