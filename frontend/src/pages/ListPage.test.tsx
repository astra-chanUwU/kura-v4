import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ListDetail, Post } from "../types/api";
import { ListPage } from "./ListPage";

const hoisted = vi.hoisted(() => ({
  reorderList: vi.fn((_id: string, _ids: string[]) => Promise.resolve()),
  items: [] as Post[],
}));

const makePost = (id: string): Post => ({
  id,
  owner_username: "ana",
  content_type: "image/png",
  size_bytes: 1,
  created_at: "2026-01-01T00:00:00Z",
  is_favorite: false,
  media_url: `/media/${id}.jpg`,
  thumbnail_url: `/media/${id}-thumb.jpg`,
  artist: "",
  rating: "sfw",
  width: 10,
  height: 10,
  source_url: "",
  tags: [],
});

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children?: React.ReactNode }) => <a href="/u/ana">{children}</a>,
}));
vi.mock("../components/PostCard", () => ({
  PostCard: ({ post }: { post: Post }) => <div data-testid={`card-${post.id}`}>{post.id}</div>,
  PostActions: () => <span />,
}));
vi.mock("../components/ThemedSelect", () => ({
  ThemedSelect: () => <span />,
}));
vi.mock("../components/Toasts", () => ({
  useToasts: () => ({ push: vi.fn() }),
  shortError: (e: unknown) => (e instanceof Error ? e.message : "error"),
}));
vi.mock("../hooks/useKura", () => ({
  useList: () => ({
    data: {
      list: {
        id: "l1",
        owner_username: "ana",
        title: "Faves",
        description: "",
        visibility: "private",
        item_count: hoisted.items.length,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
      },
      items: hoisted.items,
    } satisfies ListDetail,
    isPending: false,
    error: null,
  }),
  useMe: () => ({ user: { username: "ana" } }),
  useListActions: () => ({
    reorderList: hoisted.reorderList,
    removeFromList: vi.fn(() => Promise.resolve()),
    updateList: vi.fn(() => Promise.resolve()),
    deleteList: vi.fn(() => Promise.resolve()),
    refresh: vi.fn(),
  }),
}));

describe("list reorder controls", () => {
  it("moves a picture one step with the bottom-left arrows and saves", () => {
    hoisted.items = [makePost("p1"), makePost("p2"), makePost("p3")];
    hoisted.reorderList.mockClear();
    render(<ListPage id="l1" />);
    expect(screen.queryByRole("button", { name: "Drag to reorder" })).not.toBeInTheDocument();
    expect(screen.getByLabelText("Pick number 1")).toHaveTextContent("1");
    expect(screen.getByLabelText("Pick number 3")).toHaveTextContent("3");
    expect(screen.getByRole("group", { name: "View mode" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Page width" })).toBeInTheDocument();
    const later = screen.getAllByRole("button", { name: "Move later" });
    const earlier = screen.getAllByRole("button", { name: "Move earlier" });
    expect(earlier[0]).toBeDisabled();
    expect(later[later.length - 1]).toBeDisabled();
    fireEvent.click(later[0]!);
    expect(hoisted.reorderList).toHaveBeenCalledWith("l1", ["p2", "p1", "p3"]);
    // Local order updates in place (this is what FLIP then animates).
    const cards = screen.getAllByTestId(/^card-/).map((el) => el.getAttribute("data-testid"));
    expect(cards).toEqual(["card-p2", "card-p1", "card-p3"]);
  });

  it("uses up/down arrows in Rows mode", () => {
    localStorage.setItem("kura-board-view", JSON.stringify({ mode: "rows", wide: false }));
    hoisted.items = [makePost("p1"), makePost("p2"), makePost("p3")];
    hoisted.reorderList.mockClear();
    render(<ListPage id="l1" />);
    expect(screen.queryByRole("button", { name: "Move later" })).not.toBeInTheDocument();
    const down = screen.getAllByRole("button", { name: "Move down" });
    const up = screen.getAllByRole("button", { name: "Move up" });
    expect(up[0]).toBeDisabled();
    expect(down[down.length - 1]).toBeDisabled();
    fireEvent.click(down[0]!);
    expect(hoisted.reorderList).toHaveBeenCalledWith("l1", ["p2", "p1", "p3"]);
    localStorage.removeItem("kura-board-view");
  });

  it("highlights Save list only while edits are unsaved", () => {
    localStorage.removeItem("kura-board-view");
    hoisted.items = [makePost("p1")];
    render(<ListPage id="l1" />);
    const save = screen.getByRole("button", { name: "Save list" });
    expect(save.className).not.toMatch(/saveDirty/);
    fireEvent.change(screen.getByLabelText("List title"), { target: { value: "New title" } });
    expect(save.className).toMatch(/saveDirty/);
    fireEvent.change(screen.getByLabelText("List title"), { target: { value: "Faves" } });
    expect(save.className).not.toMatch(/saveDirty/);
  });
});
