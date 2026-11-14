import { describe, expect, it } from "vitest";
import type { Post } from "../types/api";
import { countPageTags } from "./pageTags";

function post(id: string, tags: string[]): Post {
  return {
    id,
    owner_username: "ana",
    content_type: "image/png",
    size_bytes: 100,
    created_at: "2026-01-01T00:00:00Z",
    is_favorite: false,
    favorite_count: 0,
    media_url: `/media/posts/${id}`,
    thumbnail_url: `/media/posts/${id}/thumb`,
    artist: "ana",
    rating: "sfw",
    hidden: false,
    deleted: false,
    width: 10,
    height: 10,
    source_url: "",
    tags,
  };
}

describe("countPageTags", () => {
  it("returns empty when there are no posts", () => {
    expect(countPageTags([], [], [])).toEqual([]);
  });

  it("counts tags across every post on the page", () => {
    const posts = [
      post("1", ["cat", "blue"]),
      post("2", ["cat", "red"]),
      post("3", ["dog"]),
    ];
    expect(countPageTags(posts, [])).toEqual([
      { tag: "cat", count: 2 },
      { tag: "blue", count: 1 },
      { tag: "dog", count: 1 },
      { tag: "red", count: 1 },
    ]);
  });

  it("excludes selected and excluded tags", () => {
    const posts = [post("1", ["cat", "blue", "red"])];
    expect(countPageTags(posts, ["cat"], ["red"])).toEqual([{ tag: "blue", count: 1 }]);
  });

  it("sorts by count desc then name asc", () => {
    const posts = [
      post("1", ["zeta", "alpha"]),
      post("2", ["zeta", "beta"]),
      post("3", ["zeta"]),
    ];
    expect(countPageTags(posts, [])).toEqual([
      { tag: "zeta", count: 3 },
      { tag: "alpha", count: 1 },
      { tag: "beta", count: 1 },
    ]);
  });

  it("respects the limit", () => {
    const posts = [post("1", ["a", "b", "c", "d"])];
    expect(countPageTags(posts, [], [], 2)).toEqual([
      { tag: "a", count: 1 },
      { tag: "b", count: 1 },
    ]);
  });

  it("normalizes tags before counting and filtering", () => {
    const posts = [post("1", ["Blue Sky", "cat"])];
    expect(countPageTags(posts, ["blue_sky"])).toEqual([{ tag: "cat", count: 1 }]);
  });
});
