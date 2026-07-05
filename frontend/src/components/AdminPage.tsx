import { useCallback, useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import {
  adminCreateArticle,
  adminCreateBean,
  adminCreateRelation,
  adminDeleteExpired,
  adminListAuditLogs,
  adminListBatchRuns,
  adminPublishArticle,
  adminPublishBean,
  adminReplaceBeanArticles,
  adminRunInterestBatch,
  adminRunRankingBatch,
  adminUnpublishArticle,
  adminUnpublishBean,
  isAuthError,
  listArticles,
  listBeans,
} from "../api/client";
import type {
  AdminArticleInput,
  AdminBeanInput,
  AdminPanel,
  Article,
  AuditLog,
  BatchRun,
  Bean,
  Notice as NoticeType,
  User,
} from "../types";
import { Notice } from "./Notice";

type AdminPageProps = {
  user: User;
  notice: NoticeType | null;
  onSessionExpired: (message?: string) => void;
};

type AdminMenuItem = {
  value: AdminPanel;
  title: string;
  description: string;
};

type AdminAction = {
  label: string;
  description: string;
  onClick: () => Promise<void>;
};

const adminMenu: AdminMenuItem[] = [
  {
    value: "dashboard",
    title: "Dashboard",
    description: "管理機能の全体確認",
  },
  {
    value: "beans",
    title: "豆管理",
    description: "Bean作成・公開切替",
  },
  {
    value: "articles",
    title: "記事管理",
    description: "Article作成・公開切替",
  },
  {
    value: "relations",
    title: "関連付け管理",
    description: "BeanとArticleを紐付け",
  },
  {
    value: "batches",
    title: "バッチ手動実行",
    description: "ランキング・興味再計算",
  },
  {
    value: "audit",
    title: "監査ログ確認",
    description: "管理操作と実行履歴",
  },
];

const inputClass =
  "w-full rounded-2xl border border-white/10 bg-black/25 px-4 py-3 text-sm text-white outline-none transition placeholder:text-stone-500 focus:border-amber-300/50 focus:ring-4 focus:ring-amber-300/10";
const selectClass =
  "min-h-12 w-full rounded-2xl border border-white/10 bg-black/25 px-4 py-3 text-base font-semibold text-white outline-none transition focus:border-amber-300/50 focus:ring-4 focus:ring-amber-300/10";
const primaryButton =
  "rounded-2xl bg-amber-300 px-5 py-3 text-sm font-black text-stone-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:opacity-60";
const ghostButton =
  "rounded-2xl border border-white/10 bg-white/10 px-4 py-2.5 text-sm font-bold text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-50";

const defaultBeanInput: AdminBeanInput = {
  name: "",
  roaster: "",
  origin: "",
  roast_level: "medium",
  acidity: 3,
  bitterness: 3,
  flavor: 3,
  aroma: 3,
  body: 3,
  flavor_note: "",
  description: "",
  image_url: "",
  is_published: false,
};

const defaultArticleInput: AdminArticleInput = {
  title: "",
  slug: "",
  summary: "",
  body: "",
  category: "brewing",
  source_name: "Coffee Ranker Editorial",
  source_url: "",
  image_url: "",
  is_published: false,
};

function toOptionalText(value: string | undefined): string | undefined {
  const trimmed = value?.trim() ?? "";
  return trimmed === "" ? undefined : trimmed;
}

function toScore(value: number | undefined): number | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (Number.isNaN(value)) {
    return undefined;
  }
  return Math.min(Math.max(value, 1), 5);
}

function dateText(value: string | undefined): string {
  if (value === undefined || value === "") {
    return "-";
  }
  return new Date(value).toLocaleString("ja-JP");
}

function statusClass(status: string): string {
  switch (status) {
    case "success":
      return "border-emerald-400/30 bg-emerald-400/10 text-emerald-200";
    case "failed":
      return "border-rose-400/30 bg-rose-400/10 text-rose-200";
    case "running":
      return "border-amber-300/30 bg-amber-300/10 text-amber-100";
    default:
      return "border-white/10 bg-white/10 text-stone-300";
  }
}

const slugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

function makeSlug(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80);
}

function timestampSlug(): string {
  const now = new Date();
  const stamp = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, "0"),
    String(now.getDate()).padStart(2, "0"),
    String(now.getHours()).padStart(2, "0"),
    String(now.getMinutes()).padStart(2, "0"),
    String(now.getSeconds()).padStart(2, "0"),
  ].join("");
  return `article-${stamp}`;
}

function resolveArticleSlug(inputSlug: string, title: string): string {
  const slug = makeSlug(inputSlug);
  if (slugPattern.test(slug) && slug.length >= 3 && slug.length <= 120) {
    return slug;
  }

  const titleSlug = makeSlug(title);
  if (
    slugPattern.test(titleSlug) &&
    titleSlug.length >= 3 &&
    titleSlug.length <= 120
  ) {
    return titleSlug;
  }

  return timestampSlug();
}

function AdminCard({
  title,
  children,
  action,
}: {
  title: string;
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <section className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-5 shadow-2xl shadow-black/25 sm:p-6">
      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-lg font-black text-white">{title}</h3>
        {action}
      </div>
      {children}
    </section>
  );
}

function BatchTable({ runs }: { runs: BatchRun[] }) {
  if (runs.length === 0) {
    return (
      <div className="rounded-3xl border border-white/10 bg-black/20 p-5 text-sm text-stone-400">
        まだバッチ実行履歴がありません。
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-3xl border border-white/10">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-white/[0.06] text-xs uppercase tracking-[0.2em] text-stone-400">
          <tr>
            <th className="px-4 py-3">Job</th>
            <th className="px-4 py-3">Status</th>
            <th className="px-4 py-3">Rows</th>
            <th className="px-4 py-3">Started</th>
            <th className="px-4 py-3">Finished</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/10">
          {runs.map((run) => (
            <tr key={run.id} className="text-stone-300">
              <td className="px-4 py-3 font-bold text-white">{run.job_name}</td>
              <td className="px-4 py-3">
                <span className={`rounded-full border px-3 py-1 text-xs font-bold ${statusClass(run.status)}`}>
                  {run.status}
                </span>
              </td>
              <td className="px-4 py-3">{run.rows_processed}</td>
              <td className="px-4 py-3">{dateText(run.started_at)}</td>
              <td className="px-4 py-3">{dateText(run.finished_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AuditList({ logs }: { logs: AuditLog[] }) {
  if (logs.length === 0) {
    return (
      <div className="rounded-3xl border border-white/10 bg-black/20 p-5 text-sm text-stone-400">
        監査ログがありません。
      </div>
    );
  }

  return (
    <div className="grid gap-3">
      {logs.map((log) => (
        <article
          key={log.id}
          className="rounded-3xl border border-white/10 bg-black/20 p-4"
        >
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-full bg-amber-300 px-3 py-1 text-xs font-black text-stone-950">
              {log.actor_type}
            </span>
            <span className="text-sm font-black text-white">{log.action}</span>
            <span className="ml-auto text-xs text-stone-500">
              {dateText(log.created_at)}
            </span>
          </div>
          <p className="mt-2 text-xs leading-5 text-stone-400">
            target: {log.target_type ?? "-"} / {log.target_id ?? "-"}
          </p>
          {log.request_id !== undefined ? (
            <p className="mt-1 break-all text-xs text-stone-500">
              request: {log.request_id}
            </p>
          ) : null}
        </article>
      ))}
    </div>
  );
}

export function AdminPage({ user, notice, onSessionExpired }: AdminPageProps) {
  const [panel, setPanel] = useState<AdminPanel>("dashboard");
  const [beans, setBeans] = useState<Bean[]>([]);
  const [articles, setArticles] = useState<Article[]>([]);
  const [batchRuns, setBatchRuns] = useState<BatchRun[]>([]);
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [beanInput, setBeanInput] = useState<AdminBeanInput>(defaultBeanInput);
  const [articleInput, setArticleInput] =
    useState<AdminArticleInput>(defaultArticleInput);
  const [relationBeanID, setRelationBeanID] = useState<string>("");
  const [relationArticleID, setRelationArticleID] = useState<string>("");
  const [relationOrder, setRelationOrder] = useState<string>("0");
  const [relationArticleIDs, setRelationArticleIDs] = useState<string>("");
  const [noticeState, setNoticeState] = useState<NoticeType | null>(notice);
  const [loading, setLoading] = useState<boolean>(false);

  const publishedBeans = useMemo(
    () => beans.filter((bean) => bean.is_published).length,
    [beans],
  );
  const publishedArticles = useMemo(
    () => articles.filter((article) => article.is_published).length,
    [articles],
  );

  const handleError = useCallback(
    (error: unknown, fallback: string): void => {
      if (isAuthError(error)) {
        onSessionExpired("管理者セッションが切れました。ログインし直してください。");
        return;
      }
      setNoticeState({
        tone: "error",
        message: error instanceof Error ? error.message : fallback,
      });
    },
    [onSessionExpired],
  );

  const reloadAdminData = useCallback(async (): Promise<void> => {
    setLoading(true);
    try {
      const [nextBeans, nextArticles, nextRuns, nextLogs] = await Promise.all([
        listBeans(100, 0),
        listArticles(100, 0),
        adminListBatchRuns(20, 0).catch(() => []),
        adminListAuditLogs(20, 0).catch(() => []),
      ]);
      setBeans(nextBeans);
      setArticles(nextArticles);
      setBatchRuns(nextRuns);
      setAuditLogs(nextLogs);
      setNoticeState(null);
    } catch (error) {
      handleError(error, "管理データの取得に失敗しました");
    } finally {
      setLoading(false);
    }
  }, [handleError]);

  useEffect(() => {
    queueMicrotask(() => {
      void reloadAdminData();
    });
  }, [reloadAdminData]);

  const actions: AdminAction[] = [
    {
      label: "ランキング再計算",
      description: "content_metricsを再集計します。",
      onClick: async () => {
        const run = await adminRunRankingBatch();
        setBatchRuns((current) => [run, ...current]);
        setNoticeState({ tone: "success", message: "ランキング再計算を開始しました" });
      },
    },
    {
      label: "興味スコア再計算",
      description: "User/Guestのinterest_profilesを再集計します。",
      onClick: async () => {
        const run = await adminRunInterestBatch();
        setBatchRuns((current) => [run, ...current]);
        setNoticeState({ tone: "success", message: "興味スコア再計算を開始しました" });
      },
    },
    {
      label: "期限切れ削除",
      description: "期限切れRefreshToken / GuestSession等を掃除します。",
      onClick: async () => {
        await adminDeleteExpired();
        setNoticeState({ tone: "success", message: "期限切れデータ削除を実行しました" });
      },
    },
  ];

  async function runAdminAction(action: AdminAction): Promise<void> {
    setLoading(true);
    try {
      await action.onClick();
      await reloadAdminData();
    } catch (error) {
      handleError(error, `${action.label}に失敗しました`);
    } finally {
      setLoading(false);
    }
  }

  function setBeanField<K extends keyof AdminBeanInput>(
    key: K,
    value: AdminBeanInput[K],
  ): void {
    setBeanInput((current) => ({ ...current, [key]: value }));
  }

  function setArticleField<K extends keyof AdminArticleInput>(
    key: K,
    value: AdminArticleInput[K],
  ): void {
    setArticleInput((current) => ({ ...current, [key]: value }));
  }

  async function submitBean(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setLoading(true);
    try {
      const created = await adminCreateBean({
        ...beanInput,
        name: beanInput.name.trim(),
        roaster: toOptionalText(beanInput.roaster),
        origin: toOptionalText(beanInput.origin),
        region: toOptionalText(beanInput.region),
        farm: toOptionalText(beanInput.farm),
        variety: toOptionalText(beanInput.variety),
        acidity: toScore(beanInput.acidity),
        bitterness: toScore(beanInput.bitterness),
        flavor: toScore(beanInput.flavor),
        aroma: toScore(beanInput.aroma),
        body: toScore(beanInput.body),
        flavor_note: toOptionalText(beanInput.flavor_note),
        description: toOptionalText(beanInput.description),
        image_url: toOptionalText(beanInput.image_url),
      });
      setBeans((current) => [created, ...current]);
      setBeanInput(defaultBeanInput);
      setNoticeState({ tone: "success", message: "Beanを作成しました" });
    } catch (error) {
      handleError(error, "Bean作成に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function submitArticle(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    const title = articleInput.title.trim();
    const summary = articleInput.summary.trim();
    const slug = resolveArticleSlug(articleInput.slug, title);

    if (title === "") {
      setNoticeState({ tone: "error", message: "記事タイトルを入力してください" });
      return;
    }

    if (summary === "") {
      setNoticeState({ tone: "error", message: "記事概要を入力してください" });
      return;
    }

    setArticleInput((current) => ({ ...current, title, summary, slug }));
    setLoading(true);
    try {
      const created = await adminCreateArticle({
        ...articleInput,
        title,
        slug,
        summary,
        body: toOptionalText(articleInput.body),
        category: articleInput.category,
        source_name: toOptionalText(articleInput.source_name),
        source_url: toOptionalText(articleInput.source_url),
        image_url: toOptionalText(articleInput.image_url),
      });
      setArticles((current) => [created, ...current]);
      setArticleInput(defaultArticleInput);
      setNoticeState({ tone: "success", message: "Articleを作成しました" });
    } catch (error) {
      handleError(error, "Article作成に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function toggleBean(bean: Bean): Promise<void> {
    setLoading(true);
    try {
      if (bean.is_published) {
        await adminUnpublishBean(bean.id);
      } else {
        await adminPublishBean(bean.id);
      }
      setBeans((current) =>
        current.map((item) =>
          item.id === bean.id
            ? { ...item, is_published: !item.is_published }
            : item,
        ),
      );
    } catch (error) {
      handleError(error, "Bean公開状態の変更に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function toggleArticle(article: Article): Promise<void> {
    setLoading(true);
    try {
      if (article.is_published) {
        await adminUnpublishArticle(article.id);
      } else {
        await adminPublishArticle(article.id);
      }
      setArticles((current) =>
        current.map((item) =>
          item.id === article.id
            ? { ...item, is_published: !item.is_published }
            : item,
        ),
      );
    } catch (error) {
      handleError(error, "Article公開状態の変更に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function submitRelation(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const beanID = Number(relationBeanID);
    const articleID = Number(relationArticleID);
    const order = Number(relationOrder);
    if (!Number.isInteger(beanID) || !Number.isInteger(articleID) || beanID <= 0 || articleID <= 0) {
      setNoticeState({ tone: "error", message: "bean_idとarticle_idは1以上の整数で入力してください" });
      return;
    }
    setLoading(true);
    try {
      await adminCreateRelation(beanID, articleID, Number.isInteger(order) ? order : 0);
      setNoticeState({ tone: "success", message: "関連付けを作成しました" });
    } catch (error) {
      handleError(error, "関連付け作成に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function submitRelationReplace(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    const beanID = Number(relationBeanID);
    const articleIDs = relationArticleIDs
      .split(",")
      .map((value) => Number(value.trim()))
      .filter((value) => Number.isInteger(value) && value > 0);
    if (!Number.isInteger(beanID) || beanID <= 0 || articleIDs.length === 0) {
      setNoticeState({ tone: "error", message: "bean_idとarticle_idsを正しく入力してください" });
      return;
    }
    setLoading(true);
    try {
      await adminReplaceBeanArticles(beanID, articleIDs);
      setNoticeState({ tone: "success", message: "関連Articleを一括更新しました" });
    } catch (error) {
      handleError(error, "関連Article一括更新に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="mx-auto max-w-7xl px-4 py-6 lg:px-8">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-bold uppercase tracking-[0.32em] text-amber-300">
            Admin
          </p>
          <h2 className="mt-3 text-3xl font-black tracking-tight text-white sm:text-4xl">
            管理画面
          </h2>
          <p className="mt-2 text-sm leading-6 text-stone-400">
            {user.name} / {user.email}
          </p>
        </div>
        <button
          type="button"
          className={ghostButton}
          onClick={() => void reloadAdminData()}
          disabled={loading}
        >
          再読み込み
        </button>
      </div>

      <Notice notice={noticeState} />

      <div className="grid gap-5 lg:grid-cols-[240px_minmax(0,1fr)]">
        <aside className="h-fit rounded-[2rem] border border-white/10 bg-white/[0.04] p-3 shadow-2xl shadow-black/20 lg:sticky lg:top-28">
          <div className="grid gap-2">
            {adminMenu.map((item) => (
              <button
                key={item.value}
                type="button"
                className={`rounded-2xl border px-4 py-3 text-left transition ${
                  panel === item.value
                    ? "border-amber-300/40 bg-amber-300 text-stone-950"
                    : "border-white/10 bg-black/20 text-stone-300 hover:bg-white/10 hover:text-white"
                }`}
                onClick={() => setPanel(item.value)}
              >
                <span className="block text-sm font-black">{item.title}</span>
                <span className="mt-1 block text-xs opacity-75">
                  {item.description}
                </span>
              </button>
            ))}
          </div>
        </aside>

        <div className="grid gap-5">
          {panel === "dashboard" ? (
            <>
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                <AdminCard title="公開Bean">
                  <p className="text-4xl font-black text-white">{publishedBeans}</p>
                  <p className="mt-2 text-sm text-stone-400">取得済みBean {beans.length}件</p>
                </AdminCard>
                <AdminCard title="公開Article">
                  <p className="text-4xl font-black text-white">{publishedArticles}</p>
                  <p className="mt-2 text-sm text-stone-400">取得済みArticle {articles.length}件</p>
                </AdminCard>
                <AdminCard title="Batch履歴">
                  <p className="text-4xl font-black text-white">{batchRuns.length}</p>
                  <p className="mt-2 text-sm text-stone-400">直近20件</p>
                </AdminCard>
                <AdminCard title="Audit履歴">
                  <p className="text-4xl font-black text-white">{auditLogs.length}</p>
                  <p className="mt-2 text-sm text-stone-400">直近20件</p>
                </AdminCard>
              </div>
              <AdminCard title="管理アクション">
                <div className="grid gap-3 lg:grid-cols-3">
                  {actions.map((action) => (
                    <button
                      key={action.label}
                      type="button"
                      className="rounded-3xl border border-white/10 bg-black/20 p-4 text-left transition hover:border-amber-300/40 hover:bg-white/10 disabled:opacity-50"
                      onClick={() => void runAdminAction(action)}
                      disabled={loading}
                    >
                      <span className="block text-base font-black text-white">
                        {action.label}
                      </span>
                      <span className="mt-2 block text-xs leading-5 text-stone-400">
                        {action.description}
                      </span>
                    </button>
                  ))}
                </div>
              </AdminCard>
              <AdminCard title="直近Batch">
                <BatchTable runs={batchRuns.slice(0, 5)} />
              </AdminCard>
            </>
          ) : null}

          {panel === "beans" ? (
            <>
              <AdminCard title="Bean作成">
                <form className="grid gap-3" onSubmit={(event) => void submitBean(event)}>
                  <input className={inputClass} value={beanInput.name} onChange={(event) => setBeanField("name", event.target.value)} placeholder="Bean名" maxLength={100} />
                  <div className="grid gap-3 sm:grid-cols-2">
                    <input className={inputClass} value={beanInput.roaster ?? ""} onChange={(event) => setBeanField("roaster", event.target.value)} placeholder="Roaster" maxLength={100} />
                    <input className={inputClass} value={beanInput.origin ?? ""} onChange={(event) => setBeanField("origin", event.target.value)} placeholder="Origin" maxLength={50} />
                  </div>
                  <select className={selectClass} value={beanInput.roast_level} onChange={(event) => setBeanField("roast_level", event.target.value as AdminBeanInput["roast_level"])}>
                    <option value="light">浅煎り</option>
                    <option value="medium">中煎り</option>
                    <option value="dark">深煎り</option>
                  </select>
                  <div className="grid gap-3 sm:grid-cols-5">
                    {(["acidity", "bitterness", "flavor", "aroma", "body"] as const).map((key) => (
                      <label key={key} className="text-xs font-bold text-stone-400">
                        {key}
                        <input className={`${inputClass} mt-1`} type="number" min={1} max={5} value={beanInput[key] ?? 3} onChange={(event) => setBeanField(key, Number(event.target.value))} />
                      </label>
                    ))}
                  </div>
                  <textarea className={inputClass} value={beanInput.description ?? ""} onChange={(event) => setBeanField("description", event.target.value)} placeholder="説明文" rows={4} maxLength={2000} />
                  <input className={inputClass} value={beanInput.image_url ?? ""} onChange={(event) => setBeanField("image_url", event.target.value)} placeholder="画像URL" />
                  <button type="submit" className={primaryButton} disabled={loading}>Beanを作成</button>
                </form>
              </AdminCard>
              <AdminCard title="Bean一覧">
                <div className="grid gap-3">
                  {beans.map((bean) => (
                    <article key={bean.id} className="rounded-3xl border border-white/10 bg-black/20 p-4">
                      <div className="flex flex-wrap items-center gap-3">
                        <div>
                          <h4 className="font-black text-white">{bean.name}</h4>
                          <p className="mt-1 text-xs text-stone-400">ID {bean.id} / {bean.origin ?? "originなし"} / {bean.roast_level}</p>
                        </div>
                        <button type="button" className={`${ghostButton} ml-auto`} onClick={() => void toggleBean(bean)} disabled={loading}>
                          {bean.is_published ? "非公開にする" : "公開する"}
                        </button>
                      </div>
                    </article>
                  ))}
                </div>
              </AdminCard>
            </>
          ) : null}

          {panel === "articles" ? (
            <>
              <AdminCard title="Article作成">
                <form className="grid gap-3" onSubmit={(event) => void submitArticle(event)}>
                  <input className={inputClass} value={articleInput.title} onChange={(event) => {
                    const title = event.target.value;
                    setArticleInput((current) => ({
                      ...current,
                      title,
                      slug: current.slug === "" ? makeSlug(title) : current.slug,
                    }));
                  }} placeholder="記事タイトル" maxLength={120} />
                  <input className={inputClass} value={articleInput.slug} onChange={(event) => setArticleField("slug", makeSlug(event.target.value))} placeholder="slug 未入力時は自動生成" maxLength={120} />
                  <p className="-mt-1 text-xs leading-5 text-stone-500">
                    slugは英小文字・数字・ハイフンのみ。日本語タイトルの場合は作成時にarticle-年月日時分秒形式へ自動補正します。
                  </p>
                  <select className={selectClass} value={articleInput.category ?? "brewing"} onChange={(event) => setArticleField("category", event.target.value as AdminArticleInput["category"])}>
                    <option value="brewing">抽出</option>
                    <option value="roast">焙煎</option>
                    <option value="beans">豆知識</option>
                    <option value="recipe">レシピ</option>
                  </select>
                  <textarea className={inputClass} value={articleInput.summary} onChange={(event) => setArticleField("summary", event.target.value)} placeholder="概要" rows={3} maxLength={300} />
                  <textarea className={inputClass} value={articleInput.body ?? ""} onChange={(event) => setArticleField("body", event.target.value)} placeholder="本文" rows={6} maxLength={5000} />
                  <input className={inputClass} value={articleInput.image_url ?? ""} onChange={(event) => setArticleField("image_url", event.target.value)} placeholder="画像URL" />
                  <button type="submit" className={primaryButton} disabled={loading}>Articleを作成</button>
                </form>
              </AdminCard>
              <AdminCard title="Article一覧">
                <div className="grid gap-3">
                  {articles.map((article) => (
                    <article key={article.id} className="rounded-3xl border border-white/10 bg-black/20 p-4">
                      <div className="flex flex-wrap items-center gap-3">
                        <div>
                          <h4 className="font-black text-white">{article.title}</h4>
                          <p className="mt-1 text-xs text-stone-400">ID {article.id} / {article.slug} / {article.category ?? "categoryなし"}</p>
                        </div>
                        <button type="button" className={`${ghostButton} ml-auto`} onClick={() => void toggleArticle(article)} disabled={loading}>
                          {article.is_published ? "非公開にする" : "公開する"}
                        </button>
                      </div>
                    </article>
                  ))}
                </div>
              </AdminCard>
            </>
          ) : null}

          {panel === "relations" ? (
            <AdminCard title="関連付け管理">
              <div className="grid gap-5 lg:grid-cols-2">
                <form className="grid gap-3" onSubmit={(event) => void submitRelation(event)}>
                  <p className="text-sm font-bold text-white">1件追加</p>
                  <input className={inputClass} value={relationBeanID} onChange={(event) => setRelationBeanID(event.target.value)} placeholder="bean_id" inputMode="numeric" />
                  <input className={inputClass} value={relationArticleID} onChange={(event) => setRelationArticleID(event.target.value)} placeholder="article_id" inputMode="numeric" />
                  <input className={inputClass} value={relationOrder} onChange={(event) => setRelationOrder(event.target.value)} placeholder="display_order" inputMode="numeric" />
                  <button type="submit" className={primaryButton} disabled={loading}>関連を追加</button>
                </form>
                <form className="grid gap-3" onSubmit={(event) => void submitRelationReplace(event)}>
                  <p className="text-sm font-bold text-white">Beanごと一括差し替え</p>
                  <input className={inputClass} value={relationBeanID} onChange={(event) => setRelationBeanID(event.target.value)} placeholder="bean_id" inputMode="numeric" />
                  <input className={inputClass} value={relationArticleIDs} onChange={(event) => setRelationArticleIDs(event.target.value)} placeholder="article_ids 例: 1,2,3" />
                  <button type="submit" className={primaryButton} disabled={loading}>一括更新</button>
                </form>
              </div>
            </AdminCard>
          ) : null}

          {panel === "batches" ? (
            <>
              <AdminCard title="バッチ手動実行">
                <div className="grid gap-3 lg:grid-cols-3">
                  {actions.map((action) => (
                    <button key={action.label} type="button" className="rounded-3xl border border-white/10 bg-black/20 p-4 text-left transition hover:border-amber-300/40 hover:bg-white/10" onClick={() => void runAdminAction(action)} disabled={loading}>
                      <span className="block font-black text-white">{action.label}</span>
                      <span className="mt-2 block text-xs leading-5 text-stone-400">{action.description}</span>
                    </button>
                  ))}
                </div>
              </AdminCard>
              <AdminCard title="バッチ実行履歴"><BatchTable runs={batchRuns} /></AdminCard>
            </>
          ) : null}

          {panel === "audit" ? (
            <AdminCard title="監査ログ"><AuditList logs={auditLogs} /></AdminCard>
          ) : null}
        </div>
      </div>
    </section>
  );
}
