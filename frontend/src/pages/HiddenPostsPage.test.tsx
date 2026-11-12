import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "../types/api";
import { HiddenPostsPage } from "./HiddenPostsPage";

const hoisted = vi.hoisted(() => ({
  useHiddenPosts: vi.fn(),
}));

const moderator: User = {
  id: "mod",
  role: "moderator",
  username: "mod",
  display_name: "Mod",
  bio: "",
  location: "",
  website: "",
  created_at: "2026-11-10T00:00:00Z",
  avatar_url: "",
};

const removedPost = {
  id: "p1",
  thumbnail_url: "/media/p1",
  artist: "Miku",
  owner_username: "ada",
  created_at: "2026-11-01T00:00:00Z",
  deleted: false,
  tags: ["vocaloid"],
};

function queue(overrides: Record<string, unknown> = {}) {
  return {
    posts: [],
    total: 0,
    hidden: 4,
    deleted: 5,
    isPending: false,
    error: null,
    hasMore: false,
    loadingMore: false,
    loadMore: vi.fn(),
    ...overrides,
  };
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    children,
    className,
    "aria-label": ariaLabel,
  }: {
    to: string;
    children?: React.ReactNode;
    className?: string;
    "aria-label"?: string;
    params?: { id: string };
  }) => (
    <a href={to} className={className} aria-label={ariaLabel}>
      {children}
    </a>
  ),
}));

vi.mock("../hooks/useKura", async () => {
  const actual = await vi.importActual<typeof import("../hooks/useKura")>("../hooks/useKura");
  return {
    useDebouncedValue: actual.useDebouncedValue,
    useMe: () => ({ user: moderator }),
    useHiddenPosts: (...args: unknown[]) => hoisted.useHiddenPosts(...args),
    useStaffActions: () => ({
      unhidePost: vi.fn(() => Promise.resolve()),
      deletePost: vi.fn(() => Promise.resolve()),
    }),
    useLoadMore: () => ({ current: null }),
  };
});

describe("HiddenPostsPage filters", () => {
  beforeEach(() => {
    hoisted.useHiddenPosts.mockReset();
    hoisted.useHiddenPosts.mockReturnValue(queue());
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows search, rating, order, and the match count", () => {
    hoisted.useHiddenPosts.mockReturnValue(
      queue({ total: 9, posts: [removedPost], hidden: 4, deleted: 5 }),
    );
    render(<HiddenPostsPage />);
    expect(screen.getByRole("textbox", { name: "Search removed posts" })).toHaveAttribute(
      "placeholder",
      "Artist, tag, uploader, or who removed it",
    );
    expect(screen.getByRole("button", { name: "Rating" })).toHaveTextContent("Any rating");
    expect(screen.getByRole("button", { name: "Order" })).toHaveTextContent("Newest removal");
    expect(screen.getByText("9 matches")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show" }));
    expect(screen.getByRole("option", { name: "All removed (9)" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Hidden only (4)" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Deleted only (5)" })).toBeInTheDocument();
    expect(hoisted.useHiddenPosts).toHaveBeenCalledWith(true, {
      kind: "all",
      q: "",
      rating: "",
      sort: "newest",
    });
  });

  it("uses the singular match label and hides the count while the first page is pending", () => {
    hoisted.useHiddenPosts.mockReturnValue(queue({ isPending: true, total: 1 }));
    const view = render(<HiddenPostsPage />);
    expect(screen.queryByText("1 match")).not.toBeInTheDocument();
    expect(screen.queryByText("0 matches")).not.toBeInTheDocument();
    expect(screen.getByText("Loading…")).toBeInTheDocument();
    hoisted.useHiddenPosts.mockReturnValue(queue({ total: 1, posts: [removedPost] }));
    view.rerender(<HiddenPostsPage />);
    expect(screen.getByText("1 match")).toBeInTheDocument();
  });

  it("keeps the fully visible copy when the queue is empty and unfiltered", () => {
    render(<HiddenPostsPage />);
    expect(screen.getByText("Nothing here. The board is fully visible.")).toBeInTheDocument();
    expect(screen.queryByText("Nothing matches.")).not.toBeInTheDocument();
    expect(screen.getByText("0 matches")).toBeInTheDocument();
  });

  it("debounces search before the empty copy and the hook update", () => {
    vi.useFakeTimers();
    render(<HiddenPostsPage />);
    const search = screen.getByRole("textbox", { name: "Search removed posts" });
    fireEvent.change(search, { target: { value: "miku" } });
    expect(search).toHaveValue("miku");
    expect(screen.getByText("Nothing here. The board is fully visible.")).toBeInTheDocument();
    expect(hoisted.useHiddenPosts).not.toHaveBeenCalledWith(
      true,
      expect.objectContaining({ q: "miku" }),
    );

    act(() => {
      vi.advanceTimersByTime(300);
    });

    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    expect(screen.queryByText("Nothing here. The board is fully visible.")).not.toBeInTheDocument();
    expect(hoisted.useHiddenPosts).toHaveBeenLastCalledWith(true, {
      kind: "all",
      q: "miku",
      rating: "",
      sort: "newest",
    });

    fireEvent.change(search, { target: { value: "" } });
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(screen.getByText("Nothing here. The board is fully visible.")).toBeInTheDocument();
    expect(hoisted.useHiddenPosts).toHaveBeenLastCalledWith(true, {
      kind: "all",
      q: "",
      rating: "",
      sort: "newest",
    });
  });

  it("passes rating, order, and kind through to the hook", () => {
    render(<HiddenPostsPage />);
    fireEvent.click(screen.getByRole("button", { name: "Rating" }));
    fireEvent.click(screen.getByRole("option", { name: "NSFW" }));
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    expect(hoisted.useHiddenPosts).toHaveBeenLastCalledWith(true, {
      kind: "all",
      q: "",
      rating: "nsfw",
      sort: "newest",
    });

    fireEvent.click(screen.getByRole("button", { name: "Order" }));
    fireEvent.click(screen.getByRole("option", { name: "Oldest removal" }));
    expect(hoisted.useHiddenPosts).toHaveBeenLastCalledWith(true, {
      kind: "all",
      q: "",
      rating: "nsfw",
      sort: "oldest",
    });

    fireEvent.click(screen.getByRole("button", { name: "Show" }));
    fireEvent.click(screen.getByRole("option", { name: "Hidden only (4)" }));
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    expect(hoisted.useHiddenPosts).toHaveBeenLastCalledWith(true, {
      kind: "hidden",
      q: "",
      rating: "nsfw",
      sort: "oldest",
    });
  });
});
