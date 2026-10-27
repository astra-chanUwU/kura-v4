import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useSite } from "../hooks/useSite";
import type { Post } from "../types/api";
import { HomePage } from "./HomePage";

const board = vi.hoisted(() => ({
  posts: [] as Post[],
  signedIn: false,
  includeTags: [] as string[],
  excludeTags: [] as string[],
  urlSearch: {} as { tags?: string; exclude?: string; q?: string },
  navigate: vi.fn(),
}));

vi.mock("../hooks/useSite");
vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    useSearch: () => board.urlSearch,
    useNavigate: () => board.navigate,
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
  useMe: () => ({
    me: board.signedIn
      ? {
          user: { id: "u1", username: "ana" },
          preferences: {
            show_nsfw: false,
            default_include_tags: board.includeTags,
            default_exclude_tags: board.excludeTags,
          },
        }
      : undefined,
    user: undefined,
    loading: false,
    signedIn: board.signedIn,
    error: null,
  }),
  usePosts: () => ({
    posts: board.posts,
    loading: false,
    error: null,
    retry: vi.fn(),
    hasMore: false,
    loadingMore: false,
    loadMore: vi.fn(),
  }),
  // Faithful to the real hook's contract: selected tags filter the board
  // with AND semantics, so adding a tag swaps in the filtered result list.
  usePostSearch: (_query: string, selectedTags: string[], excludedTags: string[] = []) => {
    const active = selectedTags.length > 0 || excludedTags.length > 0;
    const results = active
      ? board.posts.filter(
          (post) =>
            selectedTags.every((tag) => post.tags.includes(tag)) &&
            !excludedTags.some((tag) => post.tags.includes(tag)),
        )
      : [];
    return {
      active,
      searching: false,
      results,
      total: board.posts.length,
      shown: results.length,
      suggestions: [],
    };
  },
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
    board.navigate.mockClear();
    sessionStorage.clear();
  });
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
  it("shows the no-tags default in the scrolling tag strip, linking signed-out viewers to register", () => {
    board.posts = [untaggedPost];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    const strip = screen.getByLabelText("Tags in current view");
    expect(strip).toHaveTextContent("No tags detected");
    expect(screen.getByRole("link", { name: "Start contributing" })).toHaveAttribute(
      "href",
      "/register",
    );
  });
  it("opens the upload dialog from Start contributing when signed in", () => {
    board.posts = [untaggedPost];
    board.signedIn = true;
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByText("No tags detected")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start contributing" }));
    expect(screen.getByRole("dialog", { name: "Upload pictures" })).toBeInTheDocument();
  });
  it("keeps tag recommendations visible after adding a tag", () => {
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
    fireEvent.click(screen.getByRole("button", { name: "Add tag anime" }));
    // The filtered view still recommends its other tags, and the strip lives
    // in the same sticky board head as the search bar — a separate sticky
    // strip at a fixed offset gets covered once the selected-tag row appears.
    expect(screen.getByRole("button", { name: "Add tag girl" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add tag landscape" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Add tag anime" })).not.toBeInTheDocument();
    const strip = screen.getByLabelText("Tags in current view");
    const head = strip.closest('[class*="boardHead"]');
    expect(head).not.toBeNull();
    expect(head).toContainElement(screen.getByLabelText("Search the board"));
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
    // Staged only: nothing filters until Filter is pressed.
    expect(screen.queryByText(/result/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Filter/ }));
    expect(screen.getByText(/1 result/)).toBeInTheDocument();
  });
  it("stages several tags and filters once on Filter", () => {
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
    fireEvent.click(screen.getByRole("button", { name: "Add tag yuri" }));
    fireEvent.click(screen.getByRole("button", { name: "Add tag cute" }));
    expect(screen.getByRole("button", { name: "Remove tag yuri" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Remove tag cute" })).toBeInTheDocument();
    const filterButton = screen.getByRole("button", { name: /Filter/ });
    expect(filterButton).toHaveTextContent(/^Filter$/);
    expect(filterButton).toHaveAccessibleName("Filter (2 unapplied changes)");
    expect(screen.queryByText(/results with all selected tags/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Filter/ }));
    expect(screen.getByText(/2 results with all selected tags/)).toBeInTheDocument();
    // The applied filter is written to the URL for Back/Forward.
    expect(board.navigate).toHaveBeenCalledWith({
      to: "/",
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
  });
  it("excludes a tag from the viewport strip's minus button", () => {
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
    expect(screen.getByRole("button", { name: "Include tag girl" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Exclude tag girl" }));
    expect(screen.getByRole("button", { name: "Remove excluded tag girl" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Filter/ }));
    expect(screen.getByText(/1 result/)).toBeInTheDocument();
  });
  it("shows detected tags instead of the no-tags default", () => {
    board.posts = [{ ...untaggedPost, tags: ["anime"] }];
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.queryByText("No tags detected")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add tag anime" })).toBeInTheDocument();
  });
});
