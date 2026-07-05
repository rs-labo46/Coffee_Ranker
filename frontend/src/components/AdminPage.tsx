import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import {
  adminCreateArticle,
  adminCreateBean,
  adminCreateRelation,
  adminDeleteExpired,
  adminDeleteRelation,
  adminFindAuditLogsByRequestID,
  adminLatestBatchRun,
  adminListArticles,
  adminListAuditLogs,
  adminListBatchRuns,
  adminListBeans,
  adminPublishArticle,
  adminPublishBean,
  adminReplaceBeanArticles,
  adminResetRateLimit,
  adminRunInterestBatch,
  adminRunRankingBatch,
  adminUnpublishArticle,
  adminUnpublishBean,
  adminUpdateArticle,
  adminUpdateBean,
  isAuthError,
  listRankings,
} from "../api/client";
import type {
  AdminArticleInput,
  AdminBeanInput,
  AdminPanel,
  Article,
  AuditLog,
  BatchRun,
  Bean,
  ContentMetric,
  Notice as NoticeType,
  RankingResult,
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
  { value: "dashboard", title: "Dashboard", description: "全体指標と状態" },
  { value: "beans", title: "豆管理", description: "作成・編集・公開" },
  { value: "articles", title: "記事管理", description: "作成・編集・公開" },
  { value: "relations", title: "関連付け", description: "BeanとArticle" },
  { value: "batches", title: "バッチ", description: "手動実行・履歴" },
  { value: "audit", title: "監査ログ", description: "操作履歴確認" },
  { value: "rate_limits", title: "RateLimit", description: "制限状態の解除" },
];

const inputClass =
  "min-w-0 w-full rounded-2xl border border-white/10 bg-black/25 px-4 py-3 text-sm text-white outline-none transition placeholder:text-stone-500 focus:border-amber-300/50 focus:ring-4 focus:ring-amber-300/10";
const selectClass =
  "min-h-12 min-w-0 w-full rounded-2xl border border-white/10 bg-black/25 px-4 py-3 text-base font-semibold text-white outline-none transition focus:border-amber-300/50 focus:ring-4 focus:ring-amber-300/10";
const primaryButton =
  "inline-flex w-full items-center justify-center rounded-2xl bg-amber-300 px-5 py-3 text-center text-sm font-black text-stone-950 transition hover:bg-amber-200 disabled:cursor-not-allowed disabled:opacity-60 sm:w-auto";
const ghostButton =
  "inline-flex w-full items-center justify-center rounded-2xl border border-white/10 bg-white/10 px-4 py-2.5 text-center text-sm font-bold text-white transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto";
const dangerButton =
  "inline-flex w-full items-center justify-center rounded-2xl border border-rose-300/25 bg-rose-400/10 px-4 py-2.5 text-center text-sm font-bold text-rose-100 transition hover:bg-rose-400/15 disabled:cursor-not-allowed disabled:opacity-50 sm:w-auto";

const defaultBeanInput: AdminBeanInput = {
  name: "",
  roaster: "",
  origin: "",
  region: "",
  farm: "",
  variety: "",
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

const slugPattern = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;

function toOptionalText(value: string | undefined): string | undefined {
  const trimmed = value?.trim() ?? "";
  return trimmed === "" ? undefined : trimmed;
}

function toScore(value: number | undefined): number | undefined {
  if (value === undefined || Number.isNaN(value)) {
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

function makeSlug(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 96);
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
    String(now.getMilliseconds()).padStart(3, "0"),
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

function beanToInput(bean: Bean): AdminBeanInput {
  return {
    name: bean.name,
    roaster: bean.roaster ?? "",
    origin: bean.origin ?? "",
    region: bean.region ?? "",
    farm: bean.farm ?? "",
    variety: bean.variety ?? "",
    roast_level: bean.roast_level,
    acidity: bean.acidity ?? 3,
    bitterness: bean.bitterness ?? 3,
    flavor: bean.flavor ?? 3,
    aroma: bean.aroma ?? 3,
    body: bean.body ?? 3,
    flavor_note: bean.flavor_note ?? "",
    description: bean.description ?? "",
    image_url: bean.image_url ?? "",
    is_published: bean.is_published,
  };
}

function articleToInput(article: Article): AdminArticleInput {
  return {
    title: article.title,
    slug: article.slug,
    summary: article.summary,
    body: article.body ?? "",
    category: article.category as AdminArticleInput["category"],
    source_name: article.source_name ?? "",
    source_url: article.source_url ?? "",
    image_url: article.image_url ?? "",
    is_published: article.is_published,
  };
}

function buildBeanPayload(input: AdminBeanInput): AdminBeanInput {
  return {
    ...input,
    name: input.name.trim(),
    roaster: toOptionalText(input.roaster),
    origin: toOptionalText(input.origin),
    region: toOptionalText(input.region),
    farm: toOptionalText(input.farm),
    variety: toOptionalText(input.variety),
    acidity: toScore(input.acidity),
    bitterness: toScore(input.bitterness),
    flavor: toScore(input.flavor),
    aroma: toScore(input.aroma),
    body: toScore(input.body),
    flavor_note: toOptionalText(input.flavor_note),
    description: toOptionalText(input.description),
    image_url: toOptionalText(input.image_url),
  };
}

function buildArticlePayload(input: AdminArticleInput): AdminArticleInput {
  const title = input.title.trim();
  return {
    ...input,
    title,
    slug: resolveArticleSlug(input.slug, title),
    summary: input.summary.trim(),
    body: toOptionalText(input.body),
    category: input.category,
    source_name: toOptionalText(input.source_name),
    source_url: toOptionalText(input.source_url),
    image_url: toOptionalText(input.image_url),
  };
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
    <section className="min-w-0 overflow-hidden rounded-[1.5rem] border border-white/10 bg-white/[0.04] p-4 shadow-2xl shadow-black/25 sm:rounded-[2rem] sm:p-6">
      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h3 className="break-words text-lg font-black text-white">{title}</h3>
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
    <div className="max-w-full overflow-x-auto rounded-3xl border border-white/10">
      <table className="min-w-[720px] text-left text-sm">
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
                <span
                  className={`rounded-full border px-3 py-1 text-xs font-bold ${statusClass(run.status)}`}
                >
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
          className="min-w-0 rounded-3xl border border-white/10 bg-black/20 p-4"
        >
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="rounded-full bg-amber-300 px-3 py-1 text-xs font-black text-stone-950">
              {log.actor_type}
            </span>
            <span className="min-w-0 break-words text-sm font-black text-white">
              {log.action}
            </span>
            <span className="w-full text-xs text-stone-500 sm:ml-auto sm:w-auto">
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

function metricSummary(metrics: ContentMetric[]): {
  views: number;
  saves: number;
  ratings: number;
  modalClicks: number;
} {
  return metrics.reduce(
    (sum, metric) => ({
      views: sum.views + metric.content_view_count,
      saves: sum.saves + metric.save_count,
      ratings: sum.ratings + metric.rating_count,
      modalClicks: sum.modalClicks + metric.modal_click_count,
    }),
    { views: 0, saves: 0, ratings: 0, modalClicks: 0 },
  );
}

export function AdminPage({ user, notice, onSessionExpired }: AdminPageProps) {
  const [panel, setPanel] = useState<AdminPanel>("dashboard");
  const [beans, setBeans] = useState<Bean[]>([]);
  const [articles, setArticles] = useState<Article[]>([]);
  const [batchRuns, setBatchRuns] = useState<BatchRun[]>([]);
  const [latestRankingRun, setLatestRankingRun] = useState<BatchRun | null>(
    null,
  );
  const [latestInterestRun, setLatestInterestRun] = useState<BatchRun | null>(
    null,
  );
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([]);
  const [ranking, setRanking] = useState<RankingResult>({
    metrics: [],
    targets: [],
    beans: [],
    articles: [],
  });
  const [beanInput, setBeanInput] = useState<AdminBeanInput>(defaultBeanInput);
  const [articleInput, setArticleInput] =
    useState<AdminArticleInput>(defaultArticleInput);
  const [editingBeanID, setEditingBeanID] = useState<number | null>(null);
  const [editingArticleID, setEditingArticleID] = useState<number | null>(null);
  const [relationBeanID, setRelationBeanID] = useState<string>("");
  const [relationArticleID, setRelationArticleID] = useState<string>("");
  const [relationOrder, setRelationOrder] = useState<string>("0");
  const [relationArticleIDs, setRelationArticleIDs] = useState<string>("");
  const [auditRequestID, setAuditRequestID] = useState<string>("");
  const [rateLimitKey, setRateLimitKey] = useState<string>("");
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
  const metrics = useMemo(
    () => metricSummary(ranking.metrics),
    [ranking.metrics],
  );

  const handleError = useCallback(
    (error: unknown, fallback: string): void => {
      if (isAuthError(error)) {
        onSessionExpired(
          "管理者セッションが切れました。ログインし直してください。",
        );
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
      const [
        nextBeans,
        nextArticles,
        nextRuns,
        nextLogs,
        nextRanking,
        rankingRun,
        interestRun,
      ] = await Promise.all([
        adminListBeans(100, 0),
        adminListArticles(100, 0),
        adminListBatchRuns(20, 0).catch(() => []),
        adminListAuditLogs(20, 0).catch(() => []),
        listRankings("all", 100, 0).catch(() => ({
          metrics: [],
          targets: [],
          beans: [],
          articles: [],
        })),
        adminLatestBatchRun("ranking").catch(() => null),
        adminLatestBatchRun("interest").catch(() => null),
      ]);
      setBeans(nextBeans);
      setArticles(nextArticles);
      setBatchRuns(nextRuns);
      setAuditLogs(nextLogs);
      setRanking(nextRanking);
      setLatestRankingRun(rankingRun);
      setLatestInterestRun(interestRun);
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
      description:
        "content_metricsを再集計します。自動実行はAsia/Tokyo基準のAM2:00です。",
      onClick: async () => {
        const run = await adminRunRankingBatch();
        setBatchRuns((current) => [run, ...current]);
        setLatestRankingRun(run);
        setNoticeState({
          tone: "success",
          message: "ランキング再計算を開始しました",
        });
      },
    },
    {
      label: "興味スコア再計算",
      description: "User/Guestのinterest_profilesを再集計します。",
      onClick: async () => {
        const run = await adminRunInterestBatch();
        setBatchRuns((current) => [run, ...current]);
        setLatestInterestRun(run);
        setNoticeState({
          tone: "success",
          message: "興味スコア再計算を開始しました",
        });
      },
    },
    {
      label: "期限切れ削除",
      description: "期限切れRefreshToken / GuestSession等を掃除します。",
      onClick: async () => {
        await adminDeleteExpired();
        setNoticeState({
          tone: "success",
          message: "期限切れデータ削除を実行しました",
        });
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

  function resetBeanForm(): void {
    setEditingBeanID(null);
    setBeanInput(defaultBeanInput);
  }

  function resetArticleForm(): void {
    setEditingArticleID(null);
    setArticleInput(defaultArticleInput);
  }

  async function submitBean(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    if (beanInput.name.trim() === "") {
      setNoticeState({ tone: "error", message: "Bean名を入力してください" });
      return;
    }

    setLoading(true);
    try {
      const payload = buildBeanPayload(beanInput);
      const saved =
        editingBeanID === null
          ? await adminCreateBean(payload)
          : await adminUpdateBean(editingBeanID, payload);
      setBeans((current) => [
        saved,
        ...current.filter((bean) => bean.id !== saved.id),
      ]);
      resetBeanForm();
      setNoticeState({
        tone: "success",
        message:
          editingBeanID === null ? "Beanを作成しました" : "Beanを更新しました",
      });
    } catch (error) {
      handleError(error, "Bean保存に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function submitArticle(
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> {
    event.preventDefault();
    const payload = buildArticlePayload(articleInput);

    if (payload.title === "") {
      setNoticeState({
        tone: "error",
        message: "記事タイトルを入力してください",
      });
      return;
    }
    if (payload.summary === "") {
      setNoticeState({ tone: "error", message: "記事概要を入力してください" });
      return;
    }

    setArticleInput((current) => ({ ...current, slug: payload.slug }));
    setLoading(true);
    try {
      const saved =
        editingArticleID === null
          ? await adminCreateArticle(payload)
          : await adminUpdateArticle(editingArticleID, payload);
      setArticles((current) => [
        saved,
        ...current.filter((article) => article.id !== saved.id),
      ]);
      resetArticleForm();
      setNoticeState({
        tone: "success",
        message:
          editingArticleID === null
            ? "Articleを作成しました"
            : "Articleを更新しました",
      });
    } catch (error) {
      handleError(error, "Article保存に失敗しました");
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

  async function submitRelation(
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> {
    event.preventDefault();
    const beanID = Number(relationBeanID);
    const articleID = Number(relationArticleID);
    const order = Number(relationOrder);
    if (
      !Number.isInteger(beanID) ||
      !Number.isInteger(articleID) ||
      beanID <= 0 ||
      articleID <= 0
    ) {
      setNoticeState({
        tone: "error",
        message: "bean_idとarticle_idは1以上の整数で入力してください",
      });
      return;
    }
    setLoading(true);
    try {
      await adminCreateRelation(
        beanID,
        articleID,
        Number.isInteger(order) && order >= 0 ? order : 0,
      );
      setNoticeState({ tone: "success", message: "関連を追加しました" });
    } catch (error) {
      handleError(error, "関連追加に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function deleteRelation(): Promise<void> {
    const beanID = Number(relationBeanID);
    const articleID = Number(relationArticleID);
    if (
      !Number.isInteger(beanID) ||
      !Number.isInteger(articleID) ||
      beanID <= 0 ||
      articleID <= 0
    ) {
      setNoticeState({
        tone: "error",
        message: "削除するbean_idとarticle_idを入力してください",
      });
      return;
    }
    setLoading(true);
    try {
      await adminDeleteRelation(beanID, articleID);
      setNoticeState({ tone: "success", message: "関連を削除しました" });
    } catch (error) {
      handleError(error, "関連削除に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function submitRelationReplace(
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> {
    event.preventDefault();
    const beanID = Number(relationBeanID);
    const articleIDs = relationArticleIDs
      .split(",")
      .map((value) => Number(value.trim()))
      .filter((value) => Number.isInteger(value) && value > 0);
    if (!Number.isInteger(beanID) || beanID <= 0) {
      setNoticeState({
        tone: "error",
        message: "bean_idを1以上の整数で入力してください",
      });
      return;
    }
    setLoading(true);
    try {
      await adminReplaceBeanArticles(beanID, articleIDs);
      setNoticeState({
        tone: "success",
        message: "関連Articleを一括更新しました",
      });
    } catch (error) {
      handleError(error, "関連Articleの一括更新に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function searchAuditByRequest(
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> {
    event.preventDefault();
    const requestID = auditRequestID.trim();
    if (requestID === "") {
      setNoticeState({
        tone: "error",
        message: "request_idを入力してください",
      });
      return;
    }
    setLoading(true);
    try {
      const logs = await adminFindAuditLogsByRequestID(requestID);
      setAuditLogs(logs);
      setNoticeState({
        tone: "success",
        message: "request_idで監査ログを検索しました",
      });
    } catch (error) {
      handleError(error, "監査ログ検索に失敗しました");
    } finally {
      setLoading(false);
    }
  }

  async function resetRateLimit(
    event: FormEvent<HTMLFormElement>,
  ): Promise<void> {
    event.preventDefault();
    const key = rateLimitKey.trim();
    if (key === "") {
      setNoticeState({
        tone: "error",
        message: "RateLimit keyを入力してください",
      });
      return;
    }
    setLoading(true);
    try {
      await adminResetRateLimit(key);
      setNoticeState({
        tone: "success",
        message: "RateLimitをリセットしました",
      });
      setRateLimitKey("");
    } catch (error) {
      handleError(error, "RateLimit resetに失敗しました");
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="mx-auto w-full max-w-7xl overflow-hidden px-3 py-4 pb-24 sm:px-5 lg:px-8 lg:pb-6">
      <div className="grid min-w-0 gap-5 lg:grid-cols-[260px_minmax(0,1fr)] xl:grid-cols-[280px_minmax(0,1fr)]">
        <aside className="min-w-0 lg:sticky lg:top-6 lg:self-start">
          <div className="min-w-0 rounded-[1.5rem] border border-white/10 bg-white/[0.04] p-3 sm:rounded-[2rem] sm:p-4">
            <div className="mb-4 rounded-3xl bg-black/20 p-4">
              <p className="text-xs font-bold uppercase tracking-[0.18em] text-stone-500">
                Admin
              </p>
              <p className="mt-1 break-all text-sm font-black text-white">
                {user.email}
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-1">
              {adminMenu.map((item) => (
                <button
                  key={item.value}
                  type="button"
                  onClick={() => setPanel(item.value)}
                  className={`min-w-0 rounded-2xl border p-3 text-left transition sm:rounded-3xl sm:p-4 ${
                    panel === item.value
                      ? "border-amber-300/60 bg-amber-300/15"
                      : "border-white/10 bg-black/20 hover:bg-white/10"
                  }`}
                >
                  <span className="block break-words text-sm font-black text-white">
                    {item.title}
                  </span>
                  <span className="mt-1 block break-words text-xs leading-5 text-stone-400">
                    {item.description}
                  </span>
                </button>
              ))}
            </div>
          </div>
        </aside>

        <div className="grid min-w-0 gap-5">
          <Notice notice={noticeState} />

          {panel === "dashboard" ? (
            <>
              <div className="grid min-w-0 gap-3 sm:grid-cols-2 xl:grid-cols-4">
                <AdminCard title="Beans">
                  <p className="text-3xl font-black text-white">
                    {publishedBeans}/{beans.length}
                  </p>
                  <p className="mt-2 text-xs text-stone-400">公開 / 全件</p>
                </AdminCard>
                <AdminCard title="Articles">
                  <p className="text-3xl font-black text-white">
                    {publishedArticles}/{articles.length}
                  </p>
                  <p className="mt-2 text-xs text-stone-400">公開 / 全件</p>
                </AdminCard>
                <AdminCard title="Views">
                  <p className="text-3xl font-black text-white">
                    {metrics.views}
                  </p>
                  <p className="mt-2 text-xs text-stone-400">
                    content_view合計
                  </p>
                </AdminCard>
                <AdminCard title="Actions">
                  <p className="text-3xl font-black text-white">
                    {metrics.saves + metrics.ratings + metrics.modalClicks}
                  </p>
                  <p className="mt-2 text-xs text-stone-400">
                    save/rating/modal_click合計
                  </p>
                </AdminCard>
              </div>
              <AdminCard title="自動バッチ状態">
                <div className="grid min-w-0 gap-3 md:grid-cols-2">
                  <div className="rounded-3xl border border-white/10 bg-black/20 p-4">
                    <p className="text-sm font-black text-white">
                      ランキング自動実行
                    </p>
                    <p className="mt-2 text-xs leading-5 text-stone-400">
                      毎日AM2:00、Asia/Tokyo基準。BATCH_TIMEZONEで変更可能。
                    </p>
                    <p className="mt-3 text-xs text-stone-500">
                      最新:{" "}
                      {latestRankingRun === null
                        ? "-"
                        : `${latestRankingRun.status} / ${dateText(latestRankingRun.started_at)}`}
                    </p>
                  </div>
                  <div className="rounded-3xl border border-white/10 bg-black/20 p-4">
                    <p className="text-sm font-black text-white">
                      興味スコア手動実行
                    </p>
                    <p className="mt-2 text-xs leading-5 text-stone-400">
                      ユーザー・ゲストの興味プロフィール再計算。
                    </p>
                    <p className="mt-3 text-xs text-stone-500">
                      最新:{" "}
                      {latestInterestRun === null
                        ? "-"
                        : `${latestInterestRun.status} / ${dateText(latestInterestRun.started_at)}`}
                    </p>
                  </div>
                </div>
              </AdminCard>
              <AdminCard title="直近Batch">
                <BatchTable runs={batchRuns.slice(0, 5)} />
              </AdminCard>
            </>
          ) : null}

          {panel === "beans" ? (
            <>
              <AdminCard
                title={
                  editingBeanID === null
                    ? "Bean作成"
                    : `Bean編集 ID ${editingBeanID}`
                }
                action={
                  editingBeanID === null ? null : (
                    <button
                      type="button"
                      className={ghostButton}
                      onClick={resetBeanForm}
                    >
                      新規作成へ戻る
                    </button>
                  )
                }
              >
                <form
                  className="grid gap-3"
                  onSubmit={(event) => void submitBean(event)}
                >
                  <input
                    className={inputClass}
                    value={beanInput.name}
                    onChange={(event) =>
                      setBeanField("name", event.target.value)
                    }
                    placeholder="Bean名"
                    maxLength={100}
                  />
                  <div className="grid min-w-0 gap-3 sm:grid-cols-2">
                    <input
                      className={inputClass}
                      value={beanInput.roaster ?? ""}
                      onChange={(event) =>
                        setBeanField("roaster", event.target.value)
                      }
                      placeholder="Roaster"
                      maxLength={100}
                    />
                    <input
                      className={inputClass}
                      value={beanInput.origin ?? ""}
                      onChange={(event) =>
                        setBeanField("origin", event.target.value)
                      }
                      placeholder="Origin"
                      maxLength={50}
                    />
                  </div>
                  <select
                    className={selectClass}
                    value={beanInput.roast_level}
                    onChange={(event) =>
                      setBeanField(
                        "roast_level",
                        event.target.value as AdminBeanInput["roast_level"],
                      )
                    }
                  >
                    <option value="light">浅煎り</option>
                    <option value="medium">中煎り</option>
                    <option value="dark">深煎り</option>
                  </select>
                  <div className="grid min-w-0 gap-3 sm:grid-cols-5">
                    {(
                      [
                        "acidity",
                        "bitterness",
                        "flavor",
                        "aroma",
                        "body",
                      ] as const
                    ).map((key) => (
                      <label
                        key={key}
                        className="text-xs font-bold text-stone-400"
                      >
                        {key}
                        <input
                          className={`${inputClass} mt-1`}
                          type="number"
                          min={1}
                          max={5}
                          value={beanInput[key] ?? 3}
                          onChange={(event) =>
                            setBeanField(key, Number(event.target.value))
                          }
                        />
                      </label>
                    ))}
                  </div>
                  <textarea
                    className={inputClass}
                    value={beanInput.flavor_note ?? ""}
                    onChange={(event) =>
                      setBeanField("flavor_note", event.target.value)
                    }
                    placeholder="味わいメモ"
                    rows={2}
                    maxLength={500}
                  />
                  <textarea
                    className={inputClass}
                    value={beanInput.description ?? ""}
                    onChange={(event) =>
                      setBeanField("description", event.target.value)
                    }
                    placeholder="説明文"
                    rows={4}
                    maxLength={2000}
                  />
                  <input
                    className={inputClass}
                    value={beanInput.image_url ?? ""}
                    onChange={(event) =>
                      setBeanField("image_url", event.target.value)
                    }
                    placeholder="画像URL"
                  />
                  <button
                    type="submit"
                    className={primaryButton}
                    disabled={loading}
                  >
                    {editingBeanID === null ? "Beanを作成" : "Beanを更新"}
                  </button>
                </form>
              </AdminCard>
              <AdminCard title="Bean一覧">
                <div className="grid gap-3">
                  {beans.map((bean) => (
                    <article
                      key={bean.id}
                      className="min-w-0 rounded-3xl border border-white/10 bg-black/20 p-4"
                    >
                      <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center">
                        <div className="min-w-0 flex-1">
                          <h4 className="break-words font-black text-white">
                            {bean.name}
                          </h4>
                          <p className="mt-1 break-words text-xs text-stone-400">
                            ID {bean.id} / {bean.origin ?? "originなし"} /{" "}
                            {bean.roast_level} /{" "}
                            {bean.is_published ? "公開" : "下書き"}
                          </p>
                        </div>
                        <div className="grid w-full grid-cols-2 gap-2 sm:w-auto sm:flex sm:flex-wrap sm:justify-end">
                          <button
                            type="button"
                            className={ghostButton}
                            onClick={() => {
                              setEditingBeanID(bean.id);
                              setBeanInput(beanToInput(bean));
                            }}
                            disabled={loading}
                          >
                            編集
                          </button>
                          <button
                            type="button"
                            className={ghostButton}
                            onClick={() => void toggleBean(bean)}
                            disabled={loading}
                          >
                            {bean.is_published ? "非公開" : "公開"}
                          </button>
                        </div>
                      </div>
                    </article>
                  ))}
                </div>
              </AdminCard>
            </>
          ) : null}

          {panel === "articles" ? (
            <>
              <AdminCard
                title={
                  editingArticleID === null
                    ? "Article作成"
                    : `Article編集 ID ${editingArticleID}`
                }
                action={
                  editingArticleID === null ? null : (
                    <button
                      type="button"
                      className={ghostButton}
                      onClick={resetArticleForm}
                    >
                      新規作成へ戻る
                    </button>
                  )
                }
              >
                <form
                  className="grid gap-3"
                  onSubmit={(event) => void submitArticle(event)}
                >
                  <input
                    className={inputClass}
                    value={articleInput.title}
                    onChange={(event) => {
                      const title = event.target.value;
                      setArticleInput((current) => ({
                        ...current,
                        title,
                        slug:
                          current.slug === "" ? makeSlug(title) : current.slug,
                      }));
                    }}
                    placeholder="記事タイトル"
                    maxLength={120}
                  />
                  <input
                    className={inputClass}
                    value={articleInput.slug}
                    onChange={(event) =>
                      setArticleField("slug", makeSlug(event.target.value))
                    }
                    placeholder="slug 未入力時は自動生成"
                    maxLength={120}
                  />
                  <p className="-mt-1 text-xs leading-5 text-stone-500">
                    slugは英小文字・数字・ハイフンのみ。日本語タイトルの場合はarticle-年月日時分秒ミリ秒形式へ自動補正します。
                  </p>
                  <select
                    className={selectClass}
                    value={articleInput.category ?? "brewing"}
                    onChange={(event) =>
                      setArticleField(
                        "category",
                        event.target.value as AdminArticleInput["category"],
                      )
                    }
                  >
                    <option value="brewing">抽出</option>
                    <option value="roast">焙煎</option>
                    <option value="beans">豆知識</option>
                    <option value="recipe">レシピ</option>
                  </select>
                  <textarea
                    className={inputClass}
                    value={articleInput.summary}
                    onChange={(event) =>
                      setArticleField("summary", event.target.value)
                    }
                    placeholder="概要"
                    rows={3}
                    maxLength={300}
                  />
                  <textarea
                    className={inputClass}
                    value={articleInput.body ?? ""}
                    onChange={(event) =>
                      setArticleField("body", event.target.value)
                    }
                    placeholder="本文"
                    rows={6}
                    maxLength={5000}
                  />
                  <input
                    className={inputClass}
                    value={articleInput.image_url ?? ""}
                    onChange={(event) =>
                      setArticleField("image_url", event.target.value)
                    }
                    placeholder="画像URL"
                  />
                  <button
                    type="submit"
                    className={primaryButton}
                    disabled={loading}
                  >
                    {editingArticleID === null
                      ? "Articleを作成"
                      : "Articleを更新"}
                  </button>
                </form>
              </AdminCard>
              <AdminCard title="Article一覧">
                <div className="grid gap-3">
                  {articles.map((article) => (
                    <article
                      key={article.id}
                      className="min-w-0 rounded-3xl border border-white/10 bg-black/20 p-4"
                    >
                      <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center">
                        <div className="min-w-0 flex-1">
                          <h4 className="break-words font-black text-white">
                            {article.title}
                          </h4>
                          <p className="mt-1 break-all text-xs text-stone-400">
                            ID {article.id} / {article.slug} /{" "}
                            {article.category ?? "categoryなし"} /{" "}
                            {article.is_published ? "公開" : "下書き"}
                          </p>
                        </div>
                        <div className="grid w-full grid-cols-2 gap-2 sm:w-auto sm:flex sm:flex-wrap sm:justify-end">
                          <button
                            type="button"
                            className={ghostButton}
                            onClick={() => {
                              setEditingArticleID(article.id);
                              setArticleInput(articleToInput(article));
                            }}
                            disabled={loading}
                          >
                            編集
                          </button>
                          <button
                            type="button"
                            className={ghostButton}
                            onClick={() => void toggleArticle(article)}
                            disabled={loading}
                          >
                            {article.is_published ? "非公開" : "公開"}
                          </button>
                        </div>
                      </div>
                    </article>
                  ))}
                </div>
              </AdminCard>
            </>
          ) : null}

          {panel === "relations" ? (
            <AdminCard title="関連付け管理">
              <div className="grid min-w-0 gap-5 lg:grid-cols-2">
                <form
                  className="grid gap-3"
                  onSubmit={(event) => void submitRelation(event)}
                >
                  <p className="text-sm font-bold text-white">1件追加・削除</p>
                  <input
                    className={inputClass}
                    value={relationBeanID}
                    onChange={(event) => setRelationBeanID(event.target.value)}
                    placeholder="bean_id"
                    inputMode="numeric"
                  />
                  <input
                    className={inputClass}
                    value={relationArticleID}
                    onChange={(event) =>
                      setRelationArticleID(event.target.value)
                    }
                    placeholder="article_id"
                    inputMode="numeric"
                  />
                  <input
                    className={inputClass}
                    value={relationOrder}
                    onChange={(event) => setRelationOrder(event.target.value)}
                    placeholder="display_order"
                    inputMode="numeric"
                  />
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="submit"
                      className={primaryButton}
                      disabled={loading}
                    >
                      関連を追加
                    </button>
                    <button
                      type="button"
                      className={dangerButton}
                      disabled={loading}
                      onClick={() => void deleteRelation()}
                    >
                      関連を削除
                    </button>
                  </div>
                </form>
                <form
                  className="grid gap-3"
                  onSubmit={(event) => void submitRelationReplace(event)}
                >
                  <p className="text-sm font-bold text-white">
                    Beanごと一括差し替え
                  </p>
                  <input
                    className={inputClass}
                    value={relationBeanID}
                    onChange={(event) => setRelationBeanID(event.target.value)}
                    placeholder="bean_id"
                    inputMode="numeric"
                  />
                  <input
                    className={inputClass}
                    value={relationArticleIDs}
                    onChange={(event) =>
                      setRelationArticleIDs(event.target.value)
                    }
                    placeholder="article_ids 例: 1,2,3"
                  />
                  <button
                    type="submit"
                    className={primaryButton}
                    disabled={loading}
                  >
                    一括更新
                  </button>
                </form>
              </div>
            </AdminCard>
          ) : null}

          {panel === "batches" ? (
            <>
              <AdminCard title="バッチ手動実行">
                <div className="grid min-w-0 gap-3 lg:grid-cols-3">
                  {actions.map((action) => (
                    <button
                      key={action.label}
                      type="button"
                      className="rounded-3xl border border-white/10 bg-black/20 p-4 text-left transition hover:border-amber-300/40 hover:bg-white/10 disabled:opacity-50"
                      onClick={() => void runAdminAction(action)}
                      disabled={loading}
                    >
                      <span className="block font-black text-white">
                        {action.label}
                      </span>
                      <span className="mt-2 block text-xs leading-5 text-stone-400">
                        {action.description}
                      </span>
                    </button>
                  ))}
                </div>
              </AdminCard>
              <AdminCard title="バッチ実行履歴">
                <BatchTable runs={batchRuns} />
              </AdminCard>
            </>
          ) : null}

          {panel === "audit" ? (
            <>
              <AdminCard title="request_id検索">
                <form
                  className="grid min-w-0 gap-3 sm:grid-cols-[minmax(0,1fr)_auto_auto]"
                  onSubmit={(event) => void searchAuditByRequest(event)}
                >
                  <input
                    className={inputClass}
                    value={auditRequestID}
                    onChange={(event) => setAuditRequestID(event.target.value)}
                    placeholder="request_id"
                  />
                  <button
                    type="submit"
                    className={primaryButton}
                    disabled={loading}
                  >
                    検索
                  </button>
                  <button
                    type="button"
                    className={ghostButton}
                    disabled={loading}
                    onClick={() => void reloadAdminData()}
                  >
                    一覧へ戻る
                  </button>
                </form>
              </AdminCard>
              <AdminCard title="監査ログ">
                <AuditList logs={auditLogs} />
              </AdminCard>
            </>
          ) : null}

          {panel === "rate_limits" ? (
            <AdminCard title="RateLimit reset">
              <form
                className="grid gap-3"
                onSubmit={(event) => void resetRateLimit(event)}
              >
                <p className="text-sm leading-6 text-stone-400">
                  Redis上のTokenBucket
                  keyを指定して制限状態を解除します。誤ったkeyは何も解除しません。
                </p>
                <input
                  className={inputClass}
                  value={rateLimitKey}
                  onChange={(event) => setRateLimitKey(event.target.value)}
                  placeholder="例: rate:login:ip_hash"
                />
                <button
                  type="submit"
                  className={primaryButton}
                  disabled={loading}
                >
                  RateLimitをリセット
                </button>
              </form>
            </AdminCard>
          ) : null}
        </div>
      </div>
    </section>
  );
}
