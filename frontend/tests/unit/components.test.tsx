import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ActionRail } from "../../src/components/ActionRail";
import { AuthPage } from "../../src/components/AuthPage";
import { Notice } from "../../src/components/Notice";
import { SearchPage } from "../../src/components/SearchPage";
import { TasteMeter } from "../../src/components/TasteMeter";
import { adminUser, makeFeedItem, user } from "../fixtures";

describe("components", () => {
  it("Noticeはnoticeがnullなら何も表示せず、tone付きmessageを表示する", () => {
    const { container, rerender } = render(<Notice notice={null} />);
    expect(container).toBeEmptyDOMElement();

    rerender(<Notice notice={{ tone: "success", message: "保存しました" }} />);
    expect(screen.getByText("保存しました")).toBeInTheDocument();
  });

  it("TasteMeterは1〜5の表示幅にclampする", () => {
    const { container, rerender } = render(<TasteMeter label="酸味" value={7} />);

    expect(screen.getByText("酸味")).toBeInTheDocument();
    expect(screen.getByText("7")).toBeInTheDocument();
    expect(container.querySelector("span[style]")).toHaveStyle({ width: "100%" });

    rerender(<TasteMeter label="苦味" value={-1} />);
    expect(container.querySelector("span[style]")).toHaveStyle({ width: "0%" });
  });

  it("ActionRailは未ログイン時に保存・評価を無効化し、ログイン時だけcallbackを呼ぶ", async () => {
    const feedItem = makeFeedItem({ isSaved: false, ratingScore: null });
    const onSave = vi.fn(async () => undefined);
    const onRate = vi.fn(async () => undefined);
    const tester = userEvent.setup();

    const { rerender } = render(
      <ActionRail
        item={feedItem}
        user={null}
        onSave={onSave}
        onRate={onRate}
        notice={null}
      />,
    );

    expect(screen.getByTitle("保存する")).toBeDisabled();
    expect(screen.getByTitle("Good評価する")).toBeDisabled();
    expect(screen.getByTitle("Bad評価する")).toBeDisabled();

    rerender(
      <ActionRail
        item={feedItem}
        user={user}
        onSave={onSave}
        onRate={onRate}
        notice={null}
        layout="horizontal"
      />,
    );

    await tester.click(screen.getByTitle("保存する"));
    await tester.click(screen.getByTitle("Good評価する"));
    await tester.click(screen.getByTitle("Bad評価する"));

    expect(onSave).toHaveBeenCalledWith(feedItem);
    expect(onRate).toHaveBeenNthCalledWith(1, feedItem, 1);
    expect(onRate).toHaveBeenNthCalledWith(2, feedItem, -1);
  });

  it("SearchPageは入力した検索条件をonSearchへ渡し、検索結果選択をonSelectへ渡す", async () => {
    const feedItem = makeFeedItem({ title: "Brazil Medium Bean" });
    const onSearch = vi.fn(async () => undefined);
    const onSelect = vi.fn();
    const tester = userEvent.setup();

    render(
      <SearchPage
        activeFilter="all"
        items={[feedItem]}
        searching={false}
        restoreRevision={0}
        onSearch={onSearch}
        onSelect={onSelect}
      />,
    );

    await tester.type(
      screen.getByPlaceholderText("産地、抽出、味のキーワード"),
      "brazil",
    );

    const selects = screen.getAllByRole("combobox");
    await tester.selectOptions(selects[0], "bean");
    await tester.selectOptions(selects[1], "popular");
    await tester.selectOptions(selects[2], "medium");
    await tester.click(screen.getByRole("button", { name: "検索する" }));

    expect(onSearch).toHaveBeenCalledWith({
      q: "brazil",
      sort: "popular",
      contentType: "bean",
      roastLevel: "medium",
      category: "",
    });

    await tester.click(
      screen.getByRole("button", { name: /Brazil Medium Bean/ }),
    );
    expect(onSelect).toHaveBeenCalledWith(feedItem);
  });

  it("AuthPageはログイン・サインアップ・管理画面導線を呼び分ける", async () => {
    const onLogin = vi.fn(async () => undefined);
    const onSignup = vi.fn(async () => undefined);
    const onLogout = vi.fn(async () => undefined);
    const onSelectItem = vi.fn();
    const onOpenAdmin = vi.fn();
    const tester = userEvent.setup();

    const { rerender } = render(
      <AuthPage
        user={null}
        loading={false}
        notice={null}
        savedItems={[]}
        goodItems={[]}
        onLogin={onLogin}
        onSignup={onSignup}
        onLogout={onLogout}
        onSelectItem={onSelectItem}
        onOpenAdmin={onOpenAdmin}
      />,
    );

    await tester.type(screen.getByPlaceholderText("email@example.com"), "rin@example.com");
    await tester.type(screen.getByPlaceholderText("password"), "password123");
    await tester.click(screen.getAllByRole("button", { name: "ログイン" })[1]);
    expect(onLogin).toHaveBeenCalledWith("rin@example.com", "password123");

    await tester.click(screen.getAllByRole("button", { name: "サインアップ" })[0]);
    await tester.type(screen.getByPlaceholderText("表示名"), "Rin");
    await tester.clear(screen.getByPlaceholderText("email@example.com"));
    await tester.type(screen.getByPlaceholderText("email@example.com"), "new@example.com");
    await tester.clear(screen.getByPlaceholderText("password"));
    await tester.type(screen.getByPlaceholderText("password"), "password456");
    await tester.click(screen.getAllByRole("button", { name: "サインアップ" })[1]);
    expect(onSignup).toHaveBeenCalledWith("Rin", "new@example.com", "password456");

    rerender(
      <AuthPage
        user={adminUser}
        loading={false}
        notice={null}
        savedItems={[makeFeedItem({ title: "Saved Bean" })]}
        goodItems={[]}
        onLogin={onLogin}
        onSignup={onSignup}
        onLogout={onLogout}
        onSelectItem={onSelectItem}
        onOpenAdmin={onOpenAdmin}
      />,
    );

    await tester.click(screen.getByRole("button", { name: "管理画面を開く" }));
    await tester.click(screen.getByRole("button", { name: "ログアウト" }));
    await tester.click(screen.getByLabelText("Saved Bean の詳細を表示"));

    expect(onOpenAdmin).toHaveBeenCalledTimes(1);
    expect(onLogout).toHaveBeenCalledTimes(1);
    expect(onSelectItem).toHaveBeenCalledWith(
      expect.objectContaining({ title: "Saved Bean" }),
    );
  });
});
