import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useFeedData } from "../../hooks/useFeedData";
import type { FeedFilter, SearchState } from "../../src/types";
import {
  makeArticle,
  makeBean,
  makeRankingResult,
  makeRecommendation,
} from "../fixtures";

vi.mock("../../src/api/client", () => ({
  ensureGuestSession: vi.fn(),
  listArticles: vi.fn(),
  listBeans: vi.fn(),
  listRankings: vi.fn(),
  listRecommendations: vi.fn(),
  recordEvent: vi.fn(),
  searchArticles: vi.fn(),
  searchBeans: vi.fn(),
  stableSearchHash: vi.fn(() => "search_mock"),
}));

import {
  ensureGuestSession,
  listArticles,
  listBeans,
  listRankings,
  listRecommendations,
  recordEvent,
  searchArticles,
  searchBeans,
} from "../../src/api/client";

const ensureGuestSessionMock = vi.mocked(ensureGuestSession);
const listArticlesMock = vi.mocked(listArticles);
const listBeansMock = vi.mocked(listBeans);
const listRankingsMock = vi.mocked(listRankings);
const listRecommendationsMock = vi.mocked(listRecommendations);
const recordEventMock = vi.mocked(recordEvent);
const searchArticlesMock = vi.mocked(searchArticles);
const searchBeansMock = vi.mocked(searchBeans);

function FeedProbe({ activeFilter }: { activeFilter: FeedFilter }) {
  const { state, searching, runSearch } = useFeedData(activeFilter);

  const searchState: SearchState = {
    q: "ブラジル",
    sort: "score",
    contentType: "bean",
    roastLevel: "medium",
    category: "",
  };

  return (
    <section>
      <p>loading:{String(state.loading)}</p>
      <p>searching:{String(searching)}</p>
      <p>count:{state.items.length}</p>
      {state.error !== null ? <p>{state.error}</p> : null}
      <ul>
        {state.items.map((item) => (
          <li key={item.key}>{item.title}</li>
        ))}
      </ul>
      <button type="button" onClick={() => void runSearch(searchState)}>
        run search
      </button>
    </section>
  );
}

describe("useFeedData integration", () => {
  beforeEach(() => {
    const ethiopia = makeBean({ id: 1, name: "Ethiopia Test Bean", roast_level: "light" });
    const brazil = makeBean({ id: 2, name: "Brazil Medium Bean", origin: "Brazil", roast_level: "medium" });
    const article = makeArticle({ id: 10, title: "初心者向け抽出ガイド" });

    ensureGuestSessionMock.mockResolvedValue({
      id: 1,
      created: false,
      expires_at: "2026-07-07T00:00:00Z",
    });
    listBeansMock.mockResolvedValue([ethiopia, brazil]);
    listArticlesMock.mockResolvedValue([article]);
    listRecommendationsMock.mockImplementation(async (contentType) => {
      if (contentType === "article") {
        return [
          makeRecommendation({
            rank_target_id: 110,
            content_type: "article",
            content_id: 10,
            score: 70,
          }),
        ];
      }
      return [makeRecommendation({ rank_target_id: 100, content_type: "bean", content_id: 1 })];
    });
    listRankingsMock.mockResolvedValue(
      makeRankingResult({
        beans: [ethiopia, brazil],
        articles: [article],
      }),
    );
    searchBeansMock.mockResolvedValue([brazil]);
    searchArticlesMock.mockResolvedValue([]);
    recordEventMock.mockResolvedValue(undefined);
  });

  it("初期ロードでGuestSession、Beans、Articles、Ranking、Recommendationを統合して表示する", async () => {
    render(<FeedProbe activeFilter="all" />);

    expect(await screen.findByText("Ethiopia Test Bean")).toBeInTheDocument();
    expect(screen.getByText("Brazil Medium Bean")).toBeInTheDocument();
    expect(screen.getByText("初心者向け抽出ガイド")).toBeInTheDocument();
    expect(screen.getByText("loading:false")).toBeInTheDocument();
    expect(screen.getByText("count:3")).toBeInTheDocument();

    expect(ensureGuestSessionMock).toHaveBeenCalledTimes(1);
    expect(listBeansMock).toHaveBeenCalledWith(100);
    expect(listArticlesMock).toHaveBeenCalledWith(100);
    expect(listRecommendationsMock).toHaveBeenCalledWith("all", 50);
    expect(listRankingsMock).toHaveBeenCalledWith("bean", 100, 0);
  });

  it("activeFilterがbeanならArticleを除外する", async () => {
    render(<FeedProbe activeFilter="bean" />);

    expect(await screen.findByText("Ethiopia Test Bean")).toBeInTheDocument();
    expect(screen.getByText("Brazil Medium Bean")).toBeInTheDocument();
    expect(screen.queryByText("初心者向け抽出ガイド")).not.toBeInTheDocument();
    expect(screen.getByText("count:2")).toBeInTheDocument();
  });

  it("検索実行時は検索API結果を表示し、re_searchイベントを記録する", async () => {
    const tester = userEvent.setup();
    render(<FeedProbe activeFilter="all" />);

    expect(await screen.findByText("Ethiopia Test Bean")).toBeInTheDocument();

    await tester.click(screen.getByRole("button", { name: "run search" }));

    await waitFor(() => {
      expect(screen.getByText("Brazil Medium Bean")).toBeInTheDocument();
    });

    expect(searchBeansMock).toHaveBeenCalledWith({
      q: "ブラジル",
      sort: "score",
      contentType: "bean",
      roastLevel: "medium",
      category: "",
    });
    expect(searchArticlesMock).not.toHaveBeenCalled();
    expect(recordEventMock).toHaveBeenCalledWith({
      event_type: "re_search",
      placement: "search_result",
      search_condition_hash: "search_mock",
      search_keyword: "ブラジル",
      search_roast_level: "medium",
      search_category: undefined,
      page_path: "/search",
    });
  });
});
