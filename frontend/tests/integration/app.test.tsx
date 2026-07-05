import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "../../src/App";
import type { UseAuthStateResult } from "../../hooks/useAuthState";
import { adminUser, makeArticle, makeBatchRun, makeFeedItem, user } from "../fixtures";

vi.mock("../../hooks/useAuthState", () => ({
  useAuthState: vi.fn(),
}));

vi.mock("../../hooks/useFeedData", () => ({
  useFeedData: vi.fn(),
}));

vi.mock("../../src/api/client", () => ({
  adminCreateArticle: vi.fn(),
  adminCreateBean: vi.fn(),
  adminCreateRelation: vi.fn(),
  adminDeleteExpired: vi.fn(),
  adminDeleteRelation: vi.fn(),
  adminFindAuditLogsByRequestID: vi.fn(),
  adminLatestBatchRun: vi.fn(),
  adminListArticles: vi.fn(),
  adminListAuditLogs: vi.fn(),
  adminListBatchRuns: vi.fn(),
  adminListBeans: vi.fn(),
  adminPublishArticle: vi.fn(),
  adminPublishBean: vi.fn(),
  adminReplaceBeanArticles: vi.fn(),
  adminResetRateLimit: vi.fn(),
  adminRunInterestBatch: vi.fn(),
  adminRunRankingBatch: vi.fn(),
  adminUnpublishArticle: vi.fn(),
  adminUnpublishBean: vi.fn(),
  adminUpdateArticle: vi.fn(),
  adminUpdateBean: vi.fn(),
  clickModal: vi.fn(),
  closeModal: vi.fn(),
  getArticle: vi.fn(),
  getBean: vi.fn(),
  isAuthError: vi.fn(() => false),
  listRatings: vi.fn(),
  listRankings: vi.fn(),
  listSavedItems: vi.fn(),
  rateItem: vi.fn(),
  recordEvent: vi.fn(),
  removeSavedItem: vi.fn(),
  saveItem: vi.fn(),
  showModal: vi.fn(),
}));

import { useAuthState } from "../../hooks/useAuthState";
import { useFeedData } from "../../hooks/useFeedData";
import {
  adminLatestBatchRun,
  adminListArticles,
  adminListAuditLogs,
  adminListBatchRuns,
  adminListBeans,
  getBean,
  listRatings,
  listRankings,
  listSavedItems,
  recordEvent,
} from "../../src/api/client";

const useAuthStateMock = vi.mocked(useAuthState);
const useFeedDataMock = vi.mocked(useFeedData);
const adminLatestBatchRunMock = vi.mocked(adminLatestBatchRun);
const adminListArticlesMock = vi.mocked(adminListArticles);
const adminListAuditLogsMock = vi.mocked(adminListAuditLogs);
const adminListBatchRunsMock = vi.mocked(adminListBatchRuns);
const adminListBeansMock = vi.mocked(adminListBeans);
const getBeanMock = vi.mocked(getBean);
const listRatingsMock = vi.mocked(listRatings);
const listRankingsMock = vi.mocked(listRankings);
const listSavedItemsMock = vi.mocked(listSavedItems);
const recordEventMock = vi.mocked(recordEvent);

function authState(overrides: Partial<UseAuthStateResult> = {}): UseAuthStateResult {
  return {
    user: null,
    loading: false,
    notice: null,
    loginUser: vi.fn(async () => undefined),
    signupUser: vi.fn(async () => undefined),
    logoutUser: vi.fn(async () => undefined),
    markSessionExpired: vi.fn(),
    clearNotice: vi.fn(),
    ...overrides,
  };
}

describe("App integration", () => {
  const beanItem = makeFeedItem({ title: "Ethiopia Test Bean" });
  const article = makeArticle({ title: "初心者向け抽出ガイド", body: "記事本文" });
  const articleItem = makeFeedItem({
    key: "article-10",
    contentType: "article",
    contentId: 10,
    rankTargetId: 110,
    title: article.title,
    subtitle: "抽出 / Coffee Ranker Editorial",
    summary: article.summary,
    body: article.body,
    badge: "抽出",
    bean: undefined,
    article,
  });

  beforeEach(() => {
    useFeedDataMock.mockReturnValue({
      state: {
        items: [beanItem, articleItem],
        catalogItems: [beanItem, articleItem],
        loading: false,
        error: null,
      },
      searching: false,
      reload: vi.fn(async () => undefined),
      showCatalog: vi.fn(),
      runSearch: vi.fn(async () => undefined),
    });
    getBeanMock.mockResolvedValue(beanItem.bean!);
    listSavedItemsMock.mockResolvedValue([]);
    listRatingsMock.mockResolvedValue([]);
    recordEventMock.mockResolvedValue(undefined);
    adminListBeansMock.mockResolvedValue([]);
    adminListArticlesMock.mockResolvedValue([]);
    adminListBatchRunsMock.mockResolvedValue([]);
    adminListAuditLogsMock.mockResolvedValue([]);
    listRankingsMock.mockResolvedValue({
      metrics: [],
      targets: [],
      beans: [],
      articles: [],
    });
    adminLatestBatchRunMock.mockResolvedValue(makeBatchRun());
  });

  it("未ログインでもFeedを表示し、Bean詳細へ遷移できる", async () => {
    const tester = userEvent.setup();
    useAuthStateMock.mockReturnValue(authState());

    render(<App />);

    expect(screen.getByText("Coffee Ranker")).toBeInTheDocument();
    expect(screen.getByText("Guest")).toBeInTheDocument();
    expect(screen.getByText("Ethiopia Test Bean")).toBeInTheDocument();

    await tester.click(screen.getByLabelText("Ethiopia Test Bean の詳細を表示"));

    expect(await screen.findByText("Bean Detail")).toBeInTheDocument();
    expect(getBeanMock).toHaveBeenCalledWith(1);
  });

  it("ログイン済みユーザーはAccountで保存済み・Good済みを確認できる", async () => {
    const tester = userEvent.setup();
    useAuthStateMock.mockReturnValue(authState({ user }));
    listSavedItemsMock.mockResolvedValue([{ id: 1, user_id: 1, rank_target_id: 100, created_at: "2026-07-06T00:00:00Z", updated_at: "2026-07-06T00:00:00Z" }]);
    listRatingsMock.mockResolvedValue([{ id: 1, user_id: 1, rank_target_id: 110, score: 1, created_at: "2026-07-06T00:00:00Z", updated_at: "2026-07-06T00:00:00Z" }]);

    render(<App />);

    await waitFor(() => {
      expect(listSavedItemsMock).toHaveBeenCalled();
      expect(listRatingsMock).toHaveBeenCalled();
    });

    await tester.click(screen.getAllByRole("button", { name: "Account" })[0]);

    expect(screen.getByText("ログイン済み")).toBeInTheDocument();
    expect(screen.getAllByText("保存したコンテンツ").length).toBeGreaterThan(0);
    expect(screen.getByLabelText("Ethiopia Test Bean の詳細を表示")).toBeInTheDocument();

    await tester.click(screen.getByRole("button", { name: /Goodしたコンテンツ/ }));
    expect(screen.getByLabelText("初心者向け抽出ガイド の詳細を表示")).toBeInTheDocument();
  });

  it("adminユーザーはAdmin画面を開き、管理データ取得が走る", async () => {
    const tester = userEvent.setup();
    useAuthStateMock.mockReturnValue(authState({ user: adminUser }));

    render(<App />);

    await tester.click(screen.getByRole("button", { name: "Admin" }));

    expect(await screen.findByText("自動バッチ状態")).toBeInTheDocument();
    await waitFor(() => {
      expect(adminListBeansMock).toHaveBeenCalledWith(100, 0);
      expect(adminListArticlesMock).toHaveBeenCalledWith(100, 0);
      expect(listRankingsMock).toHaveBeenCalledWith("all", 100, 0);
    });
  });
});
