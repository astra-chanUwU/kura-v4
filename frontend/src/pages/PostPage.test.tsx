import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Post } from "../types/api";
import { PostPage } from "./PostPage";

const updatePostMeta = vi.hoisted(() => vi.fn(() => Promise.resolve()));
const navigate = vi.hoisted(() => vi.fn());
const neighbors = vi.hoisted(() => ({
  current: { index: -1, total: 0, ready: false } as {
    index: number;
    total: number;
    ready: boolean;
    prev?: { id: string };
    next?: { id: string };
  },
}));

const post: Post = {
  id: "p1",
  owner_username: "ana",
  content_type: "image/png",
  size_bytes: 2048,
  created_at: "2026-10-27T00:00:00Z",
  is_favorite: false,
  media_url: "/media/posts/p1",
  thumbnail_url: "/media/posts/p1/thumb",
  artist: "ana",
  rating: "sfw",
  hidden: false,
  deleted: false,
  width: 100,
  height: 80,
  source_url: "",
  tags: ["keep_me"],
};

vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    Link: ({
      to,
      params,
      search,
      replace: _replace,
      children,
      ...props
    }: {
      to: string;
      params?: Record<string, string>;
      search?: Record<string, string | undefined>;
      replace?: boolean;
      children?: React.ReactNode;
    }) => {
      let href = to;
      for (const [k, v] of Object.entries(params ?? {})) href = href.replace(`$${k}`, v);
      const query = new URLSearchParams(
        Object.entries(search ?? {}).filter((e): e is [string, string] => e[1] !== undefined),
      ).toString();
      if (query) href += `?${query}`;
      return React.createElement("a", { href, ...props }, children);
    },
    useNavigate: () => navigate,
  };
});
vi.mock("../hooks/usePrefetchPost", () => ({ usePrefetchNeighbors: () => undefined }));
vi.mock("../components/PostCard", () => ({ PostActions: () => null }));
vi.mock("../components/HistoryList", () => ({ HistoryList: () => null }));
vi.mock("../components/Toasts", () => ({
  useToasts: () => ({ push: vi.fn() }),
  shortError: (e: unknown) => (e instanceof Error ? e.message : "error"),
}));
vi.mock("../hooks/useKura", () => ({
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
  useMe: () => ({
    signedIn: true,
    user: { id: "u1", username: "ana", role: "member" },
    me: { user: { id: "u1", username: "ana", role: "member" } },
    loading: false,
  }),
  usePost: () => ({ data: post, isPending: false, error: null }),
  usePostNeighbors: () => neighbors.current,
  usePostActions: () => ({ updatePostMeta }),
  usePostHistory: () => ({ data: [], isPending: false, error: null }),
  useStaffActions: () => ({
    hidePost: vi.fn(),
    unhidePost: vi.fn(),
    deletePost: vi.fn(),
    rollback: vi.fn(),
  }),
}));

describe("PostPage staged editing", () => {
  it("stages tag adds/removes and rating until one Save writes them together", async () => {
    updatePostMeta.mockClear();
    render(<PostPage id="p1" />);
    const save = screen.getByRole("button", { name: "Save changes" });
    expect(save).toBeDisabled();
    expect(screen.getByText(/staged here until you Save/)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Add a tag"), { target: { value: "new_tag extra" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(screen.getByText("new_tag")).toBeInTheDocument();
    expect(updatePostMeta).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Remove tag keep_me" }));
    fireEvent.click(screen.getByRole("button", { name: "nsfw" }));
    expect(updatePostMeta).not.toHaveBeenCalled();
    expect(save).toBeEnabled();

    fireEvent.click(save);
    await waitFor(() => expect(updatePostMeta).toHaveBeenCalledTimes(1));
    expect(updatePostMeta).toHaveBeenCalledWith("p1", "ana", ["extra", "new_tag"], "nsfw");
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled(),
    );
  });

  it("Discard drops staged changes without saving", () => {
    render(<PostPage id="p1" />);
    fireEvent.click(screen.getByRole("button", { name: "Remove tag keep_me" }));
    expect(screen.queryByText("keep_me")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Discard" }));
    expect(screen.getByText("keep_me")).toBeInTheDocument();
    expect(updatePostMeta).not.toHaveBeenCalled();
  });

  it("steps to previous/next inside the board filter, keeping it in the URL", () => {
    neighbors.current = {
      index: 4,
      total: 57,
      ready: true,
      prev: { id: "p0" },
      next: { id: "p2" },
    };
    navigate.mockClear();
    render(<PostPage id="p1" search={{ tags: "yuri,cute", exclude: "nsfw" }} />);

    expect(screen.getByText("5 / 57")).toBeInTheDocument();
    expect(screen.getByText("yuri cute −nsfw")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Previous picture" })).toHaveAttribute(
      "href",
      "/posts/p0?tags=yuri%2Ccute&exclude=nsfw",
    );
    expect(screen.getByRole("link", { name: "Next picture" })).toHaveAttribute(
      "href",
      "/posts/p2?tags=yuri%2Ccute&exclude=nsfw",
    );
    expect(screen.getByRole("link", { name: /Board/ })).toHaveAttribute(
      "href",
      "/?tags=yuri%2Ccute&exclude=nsfw",
    );

    fireEvent.keyDown(window, { key: "ArrowRight" });
    expect(navigate).toHaveBeenLastCalledWith({
      to: "/posts/$id",
      params: { id: "p2" },
      search: { tags: "yuri,cute", exclude: "nsfw" },
      replace: true,
    });
    fireEvent.keyDown(window, { key: "ArrowLeft" });
    expect(navigate).toHaveBeenLastCalledWith(expect.objectContaining({ params: { id: "p0" } }));
  });

  it("leaves arrow keys to text fields and asks before dropping unsaved edits", () => {
    neighbors.current = { index: 0, total: 2, ready: true, next: { id: "p2" } };
    navigate.mockClear();
    render(<PostPage id="p1" />);
    expect(screen.queryByRole("link", { name: "Previous picture" })).not.toBeInTheDocument();

    const input = screen.getByLabelText("Add a tag");
    fireEvent.keyDown(input, { key: "ArrowRight" });
    expect(navigate).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Remove tag keep_me" }));
    fireEvent.keyDown(window, { key: "ArrowRight" });
    expect(navigate).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Discard and leave" }));
    expect(navigate).toHaveBeenCalledWith(expect.objectContaining({ params: { id: "p2" } }));
  });

  it("links the picture and a Full size link to the raw file in a new tab", () => {
    render(<PostPage id="p1" />);
    const full = screen.getByRole("link", { name: /Full size/ });
    expect(full).toHaveAttribute("href", "/media/posts/p1");
    expect(full).toHaveAttribute("target", "_blank");
    expect(screen.getByRole("link", { name: /Open full size/ })).toHaveAttribute(
      "href",
      "/media/posts/p1",
    );
  });
});
