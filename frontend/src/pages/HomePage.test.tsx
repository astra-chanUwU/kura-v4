import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { writeBoardMemory } from "../domain/boardMemory";
import { useSite } from "../hooks/useSite";
import type { Post } from "../types/api";
import { HomePage } from "./HomePage";

const board = vi.hoisted(() => ({
  posts: [] as Post[],
  signedIn: false,
  includeTags: [] as string[],
  excludeTags: [] as string[],
  urlSearch: {} as { tags?: string; exclude?: string; q?: string },
  historyState: {} as { boardCleared?: boolean },
  navigate: vi.fn(),
}));

vi.mock("../hooks/useSite");
vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    useSearch: () => board.urlSearch,
    useNavigate: () => board.navigate,
    useRouterState: ({
      select,
    }: {
      select: (s: { location: { state: { boardCleared?: boolean } } }) => unknown;
    }) => select({ location: { state: board.historyState } }),
    Link: ({
      to,
      params,
      children,
      ...props
    }: {
      to: string;
      params?: { id: string };
      children?: React.ReactNode;
    }) =>
      React.createElement("a", { href: to.replace("$id", params?.id ?? ""), ...props }, children),
  };
});
vi.mock("../hooks/usePrefetchPost", () => ({ usePrefetchPost: () => ({}) }));
vi.mock("@tanstack/react-query", async () => {
  const actual = await vi.importActual<typeof import("@tanstack/react-query")>(
    "@tanstack/react-query",
  );
  return {
    ...actual,
    useQueryClient: () => ({ prefetchQuery: vi.fn() }),
  };
});
vi.mock("../components/SavedSearches", () => ({
  SaveCurrentSearchButton: () => null,
  SavedSearchList: () => null,
}));
vi.mock("../hooks/useKura", () => ({
  normalizeTagToken: (raw: string) =>
    raw
      .toLowerCase()
      .replace(/[^a-z0-9_-]+/g, "_")
      .replace(/-/g, "_")
      .replace(/^_+|_+$/g, ""),
  tagTokens: (text: string) => [
    ...new Set(
      text
        .split(/[\s,]+/)
        .map((token) =>
          token
            .toLowerCase()
            .replace(/[^a-z0-9_-]+/g, "_")
            .replace(/-/g, "_")
            .replace(/^_+|_+$/g, ""),
        )
        .filter(Boolean),
    ),
  ],
  useDebouncedValue: (value: unknown) => value,
  prefetchBoardPage: vi.fn(),
  useMe: () => ({
    me: board.signedIn
      ? {
          user: { id: "u1", username: "ana" },
          preferences: {
            show_nsfw: false,
            default_include_tags: board.includeTags,
            default_exclude_tags: board.excludeTags,
            tag_suggestion_rows: 1,
            theme: "kura",
            font: "system",
            posts_per_page: 40,
          },
        }
      : undefined,
    user: undefined,
    loading: false,
    signedIn: board.signedIn,
    error: null,
  }),
  useBoard: (filter: { tags: string[]; exclude: string[]; q: string }) => {
    const active = filter.tags.length > 0 || filter.exclude.length > 0 || filter.q.trim() !== "";
    const posts = active
      ? board.posts.filter(
          (post) =>
            filter.tags.every((tag) => post.tags.includes(tag)) &&
            !filter.exclude.some((tag) => post.tags.includes(tag)),
        )
      : board.posts;
    return {
      posts,
      total: posts.length,
      loading: false,
      searching: false,
      error: null,
      isSuccess: true,
      retry: vi.fn(),
      active,
    };
  },
  useTagSuggestions: () => ({ suggestions: [], loading: false }),
  useRandomPost: () => vi.fn(),
  useInvalidate: () => vi.fn(),
  useMyLists: () => ({ data: [] }),
  useToggleFavorite: () => ({ mutate: vi.fn(), isPending: false }),
  usePostActions: () => ({ upload: vi.fn(), addToList: vi.fn() }),
  useListActions: () => ({ createList: vi.fn() }),
  duplicateOf: () => undefined,
}));
const mockUseSite = vi.mocked(useSite);

const untaggedPost: Post = {
  id: "p1",
  owner_username: "ana",
  content_type: "image/png",
  size_bytes: 1,
  created_at: "2026-01-01T00:00:00Z",
  is_favorite: false,
  favorite_count: 0,
  media_url: "/media/p1.jpg",
  thumbnail_url: "/media/p1-thumb.jpg",
  artist: "",
  rating: "sfw",
  hidden: false,
  deleted: false,
  width: 10,
  height: 10,
  source_url: "",
  tags: [],
};

describe("image board states", () => {
  beforeEach(() => {
    board.posts = [];
    board.signedIn = false;
    board.includeTags = [];
    board.excludeTags = [];
    board.urlSearch = {};
    board.historyState = {};
    board.navigate.mockClear();
    sessionStorage.clear();
    localStorage.removeItem("kura-board-tag-panel");
  });
  const openPageTags = () => {
    const toggle = screen.getByRole("button", { name: /Tags on this page/ });
    if (toggle.getAttribute("aria-expanded") !== "true") fireEvent.click(toggle);
  };
  it("offers recovery when the API is unavailable", () => {
    const retry = vi.fn();
    mockUseSite.mockReturnValue({
      site: undefined,
      loading: false,
      error: new Error("offline"),
      retry,
    });
    render(<HomePage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn’t connect to Kura.");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledOnce();
  });
  it("shows the empty board when the API responds", () => {
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByText("No images yet")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
  it("shows an empty page-tag panel when posts have no tags", () => {
    board.posts = [untaggedPost];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    openPageTags();
    expect(screen.getByText("No tags on this page")).toBeInTheDocument();
  });
  it("keeps page tags visible after staging a tag", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
    ];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    openPageTags();
    fireEvent.click(screen.getByRole("button", { name: "Include tag anime" }));
    // Staged chips stay in the search row; the panel still lists other page tags.
    expect(screen.getByRole("button", { name: "Include tag girl" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Include tag landscape" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Include tag anime" })).not.toBeInTheDocument();
    const panel = screen.getByText("Tags on this page").closest('[class*="boardHead"]');
    expect(panel).not.toBeNull();
    expect(panel).toContainElement(screen.getByLabelText("Search the board"));
  });
  it("pre-loads the signed-in user's saved default tags", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
      { ...untaggedPost, id: "p3", tags: ["girl"] },
    ];
    board.signedIn = true;
    board.includeTags = ["anime"];
    board.excludeTags = ["landscape"];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByRole("button", { name: "Remove tag anime" })).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Remove excluded tag landscape" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/1 result/)).toBeInTheDocument();
    expect(board.navigate).toHaveBeenCalledWith({
      to: "/",
      replace: true,
      state: { boardCleared: false },
      search: { tags: "anime", exclude: "landscape", q: undefined },
    });
  });
  it("restores the last filter when the address becomes a bare homepage", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
      { ...untaggedPost, id: "p3", tags: ["girl"] },
    ];
    board.signedIn = true;
    board.includeTags = ["anime"];
    board.excludeTags = ["landscape"];
    board.urlSearch = { tags: "girl" };
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    const { rerender } = render(<HomePage />);
    expect(screen.getByRole("button", { name: "Remove tag girl" })).toBeInTheDocument();
    expect(board.navigate).not.toHaveBeenCalled();
    board.urlSearch = {};
    rerender(<HomePage />);
    expect(screen.getByRole("button", { name: "Remove tag girl" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove tag anime" })).not.toBeInTheDocument();
    expect(board.navigate).toHaveBeenCalledWith({
      to: "/",
      replace: true,
      state: { boardCleared: false },
      search: { tags: "girl", exclude: undefined, q: undefined },
    });
  });
  it("restores a remembered clear instead of the saved defaults", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
    ];
    board.signedIn = true;
    board.includeTags = ["anime"];
    board.historyState = { boardCleared: true };
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    const { rerender } = render(<HomePage />);
    expect(screen.queryByRole("button", { name: "Remove tag anime" })).not.toBeInTheDocument();
    expect(board.navigate).not.toHaveBeenCalled();
    board.historyState = {};
    rerender(<HomePage />);
    expect(screen.queryByRole("button", { name: "Remove tag anime" })).not.toBeInTheDocument();
    expect(board.navigate).toHaveBeenCalledWith({
      to: "/",
      replace: true,
      state: { boardCleared: true },
      search: { tags: undefined, exclude: undefined, q: undefined },
    });
  });
  it("ignores another account's remembered board and seeds defaults", () => {
    board.posts = [{ ...untaggedPost, id: "p1", tags: ["anime", "girl"] }];
    board.signedIn = true;
    board.includeTags = ["anime"];
    writeBoardMemory({
      userId: "someone-else",
      tags: ["girl"],
      exclude: [],
      q: "",
      cleared: false,
    });
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByRole("button", { name: "Remove tag anime" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove tag girl" })).not.toBeInTheDocument();
  });
  it("brings staged tags back after leaving the board", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
    ];
    board.signedIn = true;
    board.includeTags = ["anime"];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    const first = render(<HomePage />);
    openPageTags();
    fireEvent.click(screen.getByRole("button", { name: "Include tag girl" }));
    expect(board.navigate).not.toHaveBeenCalledWith(
      expect.objectContaining({ search: expect.objectContaining({ tags: "anime,girl" }) }),
    );
    first.unmount();
    board.navigate.mockClear();
    render(<HomePage />);
    expect(screen.getByRole("button", { name: "Remove tag anime" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove tag girl" })).toBeInTheDocument();
  });
  it("keeps a cleared board from snapping back to the saved defaults", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
    ];
    board.signedIn = true;
    board.includeTags = ["anime"];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByRole("button", { name: "Remove tag anime" })).toBeInTheDocument();
    // Prefer the controls-row Clear (not the panel Clear when both exist).
    fireEvent.click(screen.getAllByRole("button", { name: "Clear" })[0]!);
    expect(screen.queryByRole("button", { name: "Remove tag anime" })).not.toBeInTheDocument();
    expect(screen.queryByText(/result/)).not.toBeInTheDocument();
    expect(board.navigate).toHaveBeenCalledWith({
      to: "/",
      replace: true,
      state: { boardCleared: true },
      search: { tags: undefined, exclude: undefined, q: undefined },
    });
  });
  it("leaves a cleared board empty when that history entry comes back", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
    ];
    board.signedIn = true;
    board.includeTags = ["anime"];
    board.historyState = { boardCleared: true };
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.queryByRole("button", { name: "Remove tag anime" })).not.toBeInTheDocument();
    expect(board.navigate).not.toHaveBeenCalled();
  });
  it("excludes a tag typed with a leading dash", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime"] },
      { ...untaggedPost, id: "p2", tags: ["gif"] },
    ];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    const input = screen.getByLabelText("Search the board");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "-gif" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(screen.getByRole("button", { name: "Remove excluded tag gif" })).toBeInTheDocument();
    expect(screen.getByText(/1 result/)).toBeInTheDocument();
    expect(board.navigate).toHaveBeenCalledWith({
      to: "/",
      replace: true,
      state: { boardCleared: false },
      search: { tags: undefined, exclude: "gif", q: undefined },
    });
  });
  it("stages page tags until Filter writes them into the URL", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["yuri", "cute", "white_hair"] },
      { ...untaggedPost, id: "p2", tags: ["yuri", "cute"] },
      { ...untaggedPost, id: "p3", tags: ["yuri", "white_hair"] },
    ];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    openPageTags();
    fireEvent.click(screen.getByRole("button", { name: "Include tag yuri" }));
    fireEvent.click(screen.getByRole("button", { name: "Include tag cute" }));
    expect(screen.getByRole("button", { name: "Remove tag yuri" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove tag cute" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Include tag white_hair" })).toBeInTheDocument();
    expect(screen.queryByText(/results with all selected tags/)).not.toBeInTheDocument();
    expect(board.navigate).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /Filter/ }));
    expect(screen.getByText(/2 results with all selected tags/)).toBeInTheDocument();
    expect(board.navigate).toHaveBeenLastCalledWith({
      to: "/",
      replace: true,
      state: { boardCleared: false },
      search: { tags: "yuri,cute", exclude: undefined, q: undefined },
    });
  });
  it("restores the filter from the URL and lets it win over defaults", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
      { ...untaggedPost, id: "p3", tags: ["girl"] },
    ];
    board.signedIn = true;
    board.includeTags = ["landscape"];
    board.urlSearch = { tags: "anime", exclude: "landscape" };
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByRole("button", { name: "Remove tag anime" })).toBeInTheDocument();
    expect(screen.getByText(/1 result/)).toBeInTheDocument();
    expect(board.navigate).not.toHaveBeenCalled();
  });
  it("excludes a tag from the page panel's minus button", () => {
    board.posts = [
      { ...untaggedPost, id: "p1", tags: ["anime", "girl"] },
      { ...untaggedPost, id: "p2", tags: ["anime", "landscape"] },
    ];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    openPageTags();
    expect(screen.getByRole("button", { name: "Include tag girl" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Exclude tag girl" }));
    expect(screen.getByRole("button", { name: "Remove excluded tag girl" })).toBeInTheDocument();
    // Staged from the panel: the pictures stay put until Filter.
    expect(screen.queryByText(/1 result/)).not.toBeInTheDocument();
    expect(board.navigate).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: /Filter/ }));
    expect(screen.getByText(/1 result/)).toBeInTheDocument();
  });
  it("shows detected page tags instead of the empty state", () => {
    board.posts = [{ ...untaggedPost, tags: ["anime"] }];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    openPageTags();
    expect(screen.queryByText("No tags on this page")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Include tag anime" })).toBeInTheDocument();
  });
  it("keeps page tags closed until the panel is opened, and remembers that choice", () => {
    board.posts = [{ ...untaggedPost, tags: ["anime"] }];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    const first = render(<HomePage />);
    const toggle = screen.getByRole("button", { name: /Tags on this page/ });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("button", { name: "Include tag anime" })).not.toBeInTheDocument();
    fireEvent.click(toggle);
    expect(screen.getByRole("button", { name: "Include tag anime" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Tags on this page/ }));
    expect(screen.queryByRole("button", { name: "Include tag anime" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Tags on this page/ }));
    first.unmount();
    render(<HomePage />);
    expect(screen.getByRole("button", { name: /Tags on this page/ })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(screen.getByRole("button", { name: "Include tag anime" })).toBeInTheDocument();
  });
});
