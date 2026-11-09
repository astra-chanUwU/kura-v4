import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Post } from "../types/api";
import type { TagPage as TagPageData } from "../types/tagPage";
import { TagPage } from "./TagPage";

const post: Post = {
  id: "p1",
  owner_username: "ana",
  content_type: "image/png",
  size_bytes: 10,
  created_at: "2026-11-01T00:00:00Z",
  is_favorite: false,
  favorite_count: 0,
  media_url: "/media/posts/p1",
  thumbnail_url: "/media/posts/p1/thumb",
  artist: "ana",
  rating: "sfw",
  hidden: false,
  deleted: false,
  width: 8,
  height: 8,
  source_url: "",
  tags: ["gray_hair"],
};

const page: TagPageData = {
  tag: "grey_hair",
  alias_of: "gray_hair",
  aliases: ["grey_hair"],
  category: "general",
  description: "Hair that is gray.",
  post_count: 4,
  updated_by_username: "ada",
  updated_at: "2026-11-01T00:00:00Z",
  posts: [post],
};

vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    Link: ({
      to,
      params,
      search,
      children,
    }: {
      to: string;
      params?: Record<string, string>;
      search?: Record<string, string | undefined>;
      children?: React.ReactNode;
    }) => {
      let href = to;
      for (const [key, value] of Object.entries(params ?? {}))
        href = href.replace(`$${key}`, value);
      const query = new URLSearchParams(
        Object.entries(search ?? {}).filter(
          (entry): entry is [string, string] => entry[1] !== undefined,
        ),
      ).toString();
      if (query) href += `?${query}`;
      return React.createElement("a", { href }, children);
    },
  };
});
vi.mock("../hooks/useKura", () => ({
  useMe: () => ({ user: undefined, signedIn: false, loading: false }),
}));
vi.mock("../hooks/useTagPage", () => ({
  useTagPage: () => ({ data: page, isPending: false, error: null }),
  useTagPageActions: () => ({
    update: { mutate: vi.fn(), isPending: false },
    addAlias: { mutate: vi.fn(), isPending: false },
    removeAlias: { mutate: vi.fn(), isPending: false },
  }),
}));
vi.mock("../components/PostCard", () => ({
  PostCard: ({ post: item }: { post: Post }) => <article>{item.tags[0]}</article>,
}));
vi.mock("../components/Toasts", () => ({
  useToasts: () => ({ push: vi.fn() }),
  shortError: () => "error",
}));

describe("TagPage", () => {
  it("shows an alias notice, the description, and a search for the canonical tag", () => {
    render(<TagPage name="grey_hair" />);
    expect(screen.getByRole("heading", { name: "grey_hair" })).toBeInTheDocument();
    expect(screen.getByText(/is an alias of/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "gray_hair" })).toHaveAttribute(
      "href",
      "/tags/gray_hair",
    );
    expect(screen.getByText("Hair that is gray.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Search this tag" })).toHaveAttribute(
      "href",
      "/?tags=gray_hair",
    );
    expect(screen.getByRole("article")).toHaveTextContent("gray_hair");
    expect(screen.queryByRole("button", { name: "Save description" })).not.toBeInTheDocument();
  });
});
