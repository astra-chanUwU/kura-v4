import { describe, expect, it } from "vitest";
import type { Post } from "../types/api";
import {
  filterFromSearch,
  filterPosts,
  neighborsOf,
  searchFromFilter,
  validateBoardSearch,
} from "./boardFilter";

const post = (id: string, tags: string[], artist = ""): Post => ({
  id,
  owner_username: "",
  content_type: "image/jpeg",
  size_bytes: 0,
  created_at: "2026-10-27T00:00:00Z",
  is_favorite: false,
  media_url: `/media/posts/${id}`,
  thumbnail_url: `/media/posts/${id}/thumb`,
  artist,
  rating: "sfw",
  hidden: false,
  deleted: false,
  width: 0,
  height: 0,
  source_url: "",
  tags,
});

// Board order: newest first, as the server sends it.
const board = [
  post("a", ["yuri", "cute", "silver_hair"]),
  post("b", ["cute"]),
  post("c", ["yuri", "cute", "silver_hair", "uma_musume"]),
  post("d", ["yuri", "silver_hair"]),
  post("e", ["yuri", "cute", "silver_hair", "nsfw_ish"]),
];

describe("board filter URL round trip", () => {
  it("drops empty and non-string params", () => {
    expect(validateBoardSearch({ tags: "", exclude: 3, q: "x" })).toEqual({
      tags: undefined,
      exclude: undefined,
      q: "x",
    });
  });

  it("normalizes and dedupes tags read from the URL", () => {
    expect(filterFromSearch({ tags: "Yuri,cute,yuri, silver hair", q: "  hi " })).toEqual({
      tags: ["yuri", "cute", "silver_hair"],
      exclude: [],
      q: "hi",
    });
  });

  it("omits empty parts when writing the URL", () => {
    expect(searchFromFilter({ tags: ["yuri", "cute"], exclude: [], q: " " })).toEqual({
      tags: "yuri,cute",
      exclude: undefined,
      q: undefined,
    });
  });
});

describe("filterPosts", () => {
  it("requires every tag and drops excluded ones, keeping board order", () => {
    const result = filterPosts(board, {
      tags: ["yuri", "cute", "silver_hair"],
      exclude: ["nsfw_ish"],
      q: "",
    });
    expect(result.map((p) => p.id)).toEqual(["a", "c"]);
  });

  it("returns the whole board for an empty filter", () => {
    expect(filterPosts(board, { tags: [], exclude: [], q: "" })).toHaveLength(board.length);
  });
});

describe("neighborsOf", () => {
  const sequence = filterPosts(board, { tags: ["yuri"], exclude: [], q: "" });

  it("finds previous and next inside the filtered sequence", () => {
    const n = neighborsOf(sequence, "c");
    expect(n).toMatchObject({ index: 1, total: 4 });
    expect(n.prev?.id).toBe("a");
    expect(n.next?.id).toBe("d");
  });

  it("has no previous at the start and no next at the end", () => {
    expect(neighborsOf(sequence, "a").prev).toBeUndefined();
    expect(neighborsOf(sequence, "e").next).toBeUndefined();
  });

  it("reports a post outside the sequence", () => {
    expect(neighborsOf(sequence, "b")).toEqual({ index: -1, total: 4 });
  });
});
