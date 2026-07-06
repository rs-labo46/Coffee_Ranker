import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  adminResetRateLimit,
  ApiError,
  clearAuthTokens,
  getAccessToken,
  getRating,
  isAuthError,
  login,
  me,
  recordEvent,
  stableSearchHash,
} from "../../src/api/client";
import { user } from "../fixtures";

function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
    ...init,
  });
}

function emptyResponse(init: ResponseInit = {}): Response {
  return new Response(null, { status: 204, ...init });
}

function fetchPath(call: [RequestInfo | URL, RequestInit?]): string {
  return String(call[0]).replace("http://localhost:8080", "");
}

function requestInit(call: [RequestInfo | URL, RequestInit?]): RequestInit {
  return call[1] ?? {};
}

describe("api/client", () => {
  beforeEach(() => {
    clearAuthTokens();
    sessionStorage.clear();
  });

  it("login後にaccess tokenを保持し、認証APIへBearer tokenを付与する", async () => {
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL): Promise<Response> => {
        const path = String(input).replace("http://localhost:8080", "");
        if (path === "/auth/login") {
          return jsonResponse({ user, access_token: "access-token-1" });
        }
        if (path === "/auth/me") {
          return jsonResponse(user);
        }
        return jsonResponse(
          { code: "not_found", message: "not found" },
          { status: 404 },
        );
      },
    );
    vi.stubGlobal("fetch", fetchMock);

    const response = await login("rin@example.com", "password123");
    const currentUser = await me();

    expect(response.user).toEqual(user);
    expect(currentUser).toEqual(user);
    expect(getAccessToken()).toBe("access-token-1");
    expect(sessionStorage.getItem("coffee_ranker_access_token")).toBe(
      "access-token-1",
    );

    const loginInit = requestInit(fetchMock.mock.calls[0]);
    expect(fetchPath(fetchMock.mock.calls[0])).toBe("/auth/login");
    expect(loginInit.method).toBe("POST");
    expect(loginInit.credentials).toBe("include");
    expect(loginInit.body).toBe(
      JSON.stringify({ email: "rin@example.com", password: "password123" }),
    );

    const meHeaders = requestInit(fetchMock.mock.calls[1]).headers as Headers;
    expect(fetchPath(fetchMock.mock.calls[1])).toBe("/auth/me");
    expect(meHeaders.get("Authorization")).toBe("Bearer access-token-1");
  });

  it("CSRFが必要なPOSTではCSRFを発行してから送信する", async () => {
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL): Promise<Response> => {
        const path = String(input).replace("http://localhost:8080", "");
        if (path === "/auth/csrf") {
          return jsonResponse({ csrf_token: "csrf-token-1" });
        }
        if (path === "/events") {
          return emptyResponse();
        }
        return jsonResponse(
          { code: "not_found", message: "not found" },
          { status: 404 },
        );
      },
    );
    vi.stubGlobal("fetch", fetchMock);

    await recordEvent({
      event_type: "impression",
      rank_target_id: 100,
      placement: "top",
      page_path: "/",
      dedup_key: "feed:100:impression",
      dedup_ttl_seconds: 3600,
    });

    expect(fetchPath(fetchMock.mock.calls[0])).toBe("/auth/csrf");
    expect(fetchPath(fetchMock.mock.calls[1])).toBe("/events");

    const eventInit = requestInit(fetchMock.mock.calls[1]);
    const headers = eventInit.headers as Headers;
    expect(eventInit.method).toBe("POST");
    expect(headers.get("X-CSRF-Token")).toBe("csrf-token-1");
    expect(eventInit.body).toBe(
      JSON.stringify({
        event_type: "impression",
        rank_target_id: 100,
        placement: "top",
        page_path: "/",
        dedup_key: "feed:100:impression",
        dedup_ttl_seconds: 3600,
      }),
    );
  });

  it("CSRF mismatch時はCSRFを再発行して1回だけ再試行する", async () => {
    let csrfCount = 0;
    let resetCount = 0;
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL): Promise<Response> => {
        const path = String(input).replace("http://localhost:8080", "");

        if (path === "/auth/csrf") {
          csrfCount += 1;
          return jsonResponse({ csrf_token: `csrf-token-${csrfCount}` });
        }

        if (path === "/admin/rate-limits/reset") {
          resetCount += 1;

          if (resetCount === 1) {
            return jsonResponse(
              { code: "csrf_mismatch", message: "csrf mismatch" },
              { status: 403 },
            );
          }

          return emptyResponse();
        }

        return jsonResponse(
          { code: "not_found", message: "not found" },
          { status: 404 },
        );
      },
    );
    vi.stubGlobal("fetch", fetchMock);

    await adminResetRateLimit("rate:login:test");

    const resetCalls = fetchMock.mock.calls.filter(
      (call) => fetchPath(call) === "/admin/rate-limits/reset",
    );
    expect(resetCalls).toHaveLength(2);

    const firstHeaders = requestInit(resetCalls[0]).headers as Headers;
    const secondHeaders = requestInit(resetCalls[1]).headers as Headers;
    expect(firstHeaders.get("X-CSRF-Token")).toBe("csrf-token-1");
    expect(secondHeaders.get("X-CSRF-Token")).toBe("csrf-token-2");
    expect(requestInit(resetCalls[1]).body).toBe(
      JSON.stringify({ key: "rate:login:test" }),
    );
  });

  //認証エラー
  it("getRatingは404だけnullに変換し、401だけ認証エラーとして判定する", async () => {
    const fetchMock = vi.fn(
      async (input: RequestInfo | URL): Promise<Response> => {
        const path = String(input).replace("http://localhost:8080", "");

        if (path === "/ratings/100") {
          return jsonResponse(
            { code: "not_found", message: "not found" },
            { status: 404 },
          );
        }

        return jsonResponse(
          { code: "not_found", message: "not found" },
          { status: 404 },
        );
      },
    );

    vi.stubGlobal("fetch", fetchMock);

    const rating = await getRating(100);
    const unauthorized = new ApiError(401, "unauthorized", "認証が必要です");
    const forbidden = new ApiError(403, "forbidden", "操作不可");
    const validation = new ApiError(400, "invalid_input", "入力エラー");

    expect(rating).toBeNull();
    expect(isAuthError(unauthorized)).toBe(true);
    expect(isAuthError(forbidden)).toBe(false);
    expect(isAuthError(validation)).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchPath(fetchMock.mock.calls[0])).toBe("/ratings/100");
  });

  it("stableSearchHashは同じ検索条件なら同じ値を返し、主要条件が変われば変わる", () => {
    const base = {
      q: " Ethiopia ",
      sort: "score" as const,
      contentType: "all" as const,
      roastLevel: "" as const,
      category: "" as const,
    };

    expect(stableSearchHash(base)).toBe(stableSearchHash(base));
    expect(stableSearchHash(base)).toBe(
      stableSearchHash({ ...base, q: "Ethiopia" }),
    );
    expect(stableSearchHash(base)).not.toBe(
      stableSearchHash({ ...base, contentType: "bean" }),
    );
    expect(stableSearchHash(base)).toMatch(/^search_[a-z0-9]+$/);
  });
});
