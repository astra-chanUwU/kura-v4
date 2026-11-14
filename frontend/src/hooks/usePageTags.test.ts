import { describe, expect, it } from "vitest";
import { renderHook } from "@testing-library/react";
import type { Post } from "../types/api";
import { usePageTags } from "./usePageTags";

const post = (id: string, tags: string[]): Post => ({
  id,
  owner_username: "a",
  content_type: "image/jpeg",
  size_bytes: 1,
  created_at: "2026-01-01T00:00:00Z",
  is_favorite: false,
  favorite_count: 0,
  media_url: `/m/${id}`,
  thumbnail_url: `/m/${id}/t`,
  artist: "",
  rating: "sfw",
  hidden: false,
  deleted: false,
  width: 1,
  height: 1,
  source_url: "",
  tags,
});

describe("usePageTags", () => {
  it("counts tags across the whole page and ranks by frequency", () => {
    const { result } = renderHook(() =>
      usePageTags(
        [post("1", ["yuri", "cute"]), post("2", ["yuri", "glasses"]), post("3", ["cute"])],
        [],
        [],
      ),
    );
    expect(result.current.tags.map((t) => t.tag)).toEqual(["cute", "yuri", "glasses"]);
    expect(result.current.tags[0]?.count).toBe(2);
  });

  it("skips tags already selected or excluded", () => {
    const { result } = renderHook(() =>
      usePageTags([post("1", ["yuri", "cute", "gif"])], ["yuri"], ["gif"]),
    );
    expect(result.current.tags.map((t) => t.tag)).toEqual(["cute"]);
  });

  it("returns nothing for an empty page", () => {
    const { result } = renderHook(() => usePageTags([], [], []));
    expect(result.current.tags).toEqual([]);
  });
});
