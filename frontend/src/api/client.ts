import type {
  ApiErrorBody,
  AuthResponse,
  Bean,
  ContentType,
  CSRFResponse,
  EventType,
  FeedFilter,
  GuestSessionResponse,
  Placement,
  Rating,
  RatingScore,
  RecommendationItem,
  SavedItem,
  SearchState,
  User,
} from "..//types";

const apiBase = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";
const accessTokenKey = "coffee_ranker_access_token";

let csrfToken: string | null = null;
let accessToken: string | null = sessionStorage.getItem(accessTokenKey);

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

type RequestOptions = {
  method?: Method;
  body?: unknown;
  auth?: boolean;
  csrf?: boolean;
  signal?: AbortSignal;
};

function normalizeBaseURL(value: string): string {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}

function setAccessToken(token: string | null): void {
  accessToken = token;
  if (token === null) {
    sessionStorage.removeItem(accessTokenKey);
    return;
  }
  sessionStorage.setItem(accessTokenKey, token);
}

export function getAccessToken(): string | null {
  return accessToken;
}

async function parseError(response: Response): Promise<ApiError> {
  let body: (ApiErrorBody & { error?: string }) | null;
  try {
    body = (await response.json()) as ApiErrorBody & { error?: string };
  } catch {
    body = null;
  }
  const code = body?.code ?? body?.error ?? "request_failed";
  return new ApiError(
    response.status,
    code,
    body?.message ?? errorMessageFor(code),
  );
}

function errorMessageFor(code: string): string {
  switch (code) {
    case "csrf_required":
    case "csrf_mismatch":
      return "CSRFトークンを更新してください";
    case "unauthorized":
      return "認証が必要です";
    case "forbidden":
      return "操作が許可されていません";
    default:
      return "API通信に失敗しました";
  }
}

function isCSRFError(error: ApiError): boolean {
  return (
    error.status === 403 &&
    (error.code === "csrf_required" || error.code === "csrf_mismatch")
  );
}

async function request<T>(
  path: string,
  options: RequestOptions = {},
  retriedCSRF = false,
): Promise<T> {
  if (options.csrf && csrfToken === null) {
    await issueCSRF();
  }

  const headers = new Headers();
  headers.set("Accept", "application/json");
  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
  }
  if (options.auth && accessToken !== null) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }
  if (options.csrf && csrfToken !== null) {
    headers.set("X-CSRF-Token", csrfToken);
  }

  const response = await fetch(`${normalizeBaseURL(apiBase)}${path}`, {
    method: options.method ?? "GET",
    headers,
    credentials: "include",
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    signal: options.signal,
  });

  if (!response.ok) {
    const error = await parseError(response);

    if (path === "/auth/refresh" || path === "/auth/logout") {
      csrfToken = null;
    }

    if (response.status === 401 && options.auth) {
      setAccessToken(null);
    }

    if (options.csrf && !retriedCSRF && isCSRFError(error)) {
      csrfToken = null;
      await issueCSRF();
      return request<T>(path, options, true);
    }

    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

function withQuery(
  path: string,
  query: Record<string, string | number | undefined>,
): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== "") {
      params.set(key, String(value));
    }
  }
  const text = params.toString();
  return text === "" ? path : `${path}?${text}`;
}

export async function issueCSRF(): Promise<string> {
  const response = await request<CSRFResponse>("/auth/csrf");
  csrfToken = response.csrf_token;
  return response.csrf_token;
}

export async function ensureGuestSession(): Promise<GuestSessionResponse> {
  return request<GuestSessionResponse>("/guest-session", { method: "POST" });
}

export async function signup(
  name: string,
  email: string,
  password: string,
): Promise<User> {
  return request<User>("/auth/signup", {
    method: "POST",
    body: { name, email, password },
  });
}

export async function login(
  email: string,
  password: string,
): Promise<AuthResponse> {
  const response = await request<AuthResponse>("/auth/login", {
    method: "POST",
    body: { email, password },
  });
  setAccessToken(response.access_token);
  return response;
}

export async function refreshAuth(): Promise<AuthResponse> {
  const response = await request<AuthResponse>("/auth/refresh", {
    method: "POST",
    csrf: true,
  });
  setAccessToken(response.access_token);
  return response;
}

export async function logout(): Promise<void> {
  if (accessToken === null) {
    csrfToken = null;
    setAccessToken(null);
    return;
  }

  try {
    await request<void>("/auth/logout", {
      method: "POST",
      auth: true,
      csrf: true,
    });
  } finally {
    csrfToken = null;
    setAccessToken(null);
  }
}

export async function me(): Promise<User> {
  return request<User>("/auth/me", { auth: true });
}

export async function listBeans(limit = 20, offset = 0): Promise<Bean[]> {
  return request<Bean[]>(withQuery("/beans", { limit, offset }));
}

export async function listArticles(
  limit = 20,
  offset = 0,
): Promise<import("..//types").Article[]> {
  return request<import("..//types").Article[]>(
    withQuery("/articles", { limit, offset }),
  );
}

export async function getBean(id: number): Promise<Bean> {
  return request<Bean>(`/beans/${id}`);
}

export async function getArticle(
  slug: string,
): Promise<import("..//types").Article> {
  return request<import("..//types").Article>(
    `/articles/${encodeURIComponent(slug)}`,
    { auth: true },
  );
}

export async function listRecommendations(
  contentType: FeedFilter,
  limit = 30,
): Promise<RecommendationItem[]> {
  const typed: ContentType | undefined =
    contentType === "all" ? undefined : contentType;
  return request<RecommendationItem[]>(
    withQuery("/recommendations", {
      content_type: typed,
      limit,
      offset: 0,
    }),
  );
}

export async function searchBeans(state: SearchState): Promise<Bean[]> {
  return request<Bean[]>(
    withQuery("/search/beans", {
      q: state.q,
      roast_level: state.roastLevel,
      sort: state.sort,
      limit: 30,
      offset: 0,
    }),
  );
}

export async function searchArticles(
  state: SearchState,
): Promise<import("..//types").Article[]> {
  return request<import("..//types").Article[]>(
    withQuery("/search/articles", {
      q: state.q,
      category: state.category,
      sort: state.sort,
      limit: 30,
      offset: 0,
    }),
  );
}

export async function recordEvent(payload: {
  event_type: EventType;
  rank_target_id?: number;
  placement: Placement;
  dwell_ms?: number;
  search_condition_hash?: string;
  previous_condition_hash?: string;
  search_keyword?: string;
  search_roast_level?: string;
  search_category?: string;
  page_path: string;
  referrer_path?: string;
  dedup_key?: string;
  dedup_ttl_seconds?: number;
}): Promise<void> {
  await request<void>("/events", { method: "POST", body: payload, csrf: true });
}

export async function saveItem(
  rankTargetId: number,
  placement: Placement,
  pagePath: string,
): Promise<SavedItem> {
  return request<SavedItem>("/saved", {
    method: "POST",
    auth: true,
    csrf: true,
    body: { rank_target_id: rankTargetId, placement, page_path: pagePath },
  });
}

export async function removeSavedItem(rankTargetId: number): Promise<void> {
  await request<void>(`/saved/${rankTargetId}`, {
    method: "DELETE",
    auth: true,
    csrf: true,
  });
}

export async function rateItem(
  rankTargetId: number,
  score: RatingScore,
  placement: Placement,
  pagePath: string,
): Promise<Rating> {
  return request<Rating>("/ratings", {
    method: "POST",
    auth: true,
    csrf: true,
    body: {
      rank_target_id: rankTargetId,
      score,
      placement,
      page_path: pagePath,
    },
  });
}

export async function showModal(
  rankTargetId: number,
  pagePath: string,
): Promise<unknown> {
  return request<unknown>("/modals", {
    method: "POST",
    csrf: true,
    body: {
      rank_target_id: rankTargetId,
      trigger: "scroll_end",
      page_path: pagePath,
    },
  });
}

export function stableSearchHash(state: SearchState): string {
  const source = JSON.stringify({
    q: state.q.trim(),
    sort: state.sort,
    contentType: state.contentType,
    roastLevel: state.roastLevel,
    category: state.category,
  });
  let hash = 0;
  for (let i = 0; i < source.length; i += 1) {
    hash = (hash << 5) - hash + source.charCodeAt(i);
    hash |= 0;
  }
  return `search_${Math.abs(hash).toString(36)}`;
}
