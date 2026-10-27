import Fuse from "fuse.js";
import type { Post } from "../types/api";

/** The board's applied filter: required tags, excluded tags, free text. */
export interface BoardFilter {
  tags: string[];
  exclude: string[];
  q: string;
}

/** The same filter as URL search params (?tags=a,b&exclude=c&q=x). */
export interface BoardSearch {
  tags?: string;
  exclude?: string;
  q?: string;
}

export const EMPTY_FILTER: BoardFilter = { tags: [], exclude: [], q: "" };

// Mirror of the server's booru-style tag normalization: spaces and
// commas separate tags, underscores join words inside a tag.
export function normalizeTagToken(raw: string): string {
  return raw
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "_")
    .replace(/-/g, "_")
    .replace(/^_+|_+$/g, "");
}

export function parseTagList(value?: string): string[] {
  if (!value) return [];
  return [...new Set(value.split(",").map(normalizeTagToken).filter(Boolean))];
}

const nonEmptyString = (value: unknown): string | undefined =>
  typeof value === "string" && value !== "" ? value : undefined;

/** Router validateSearch for every route that carries the board filter. */
export function validateBoardSearch(search: Record<string, unknown>): BoardSearch {
  return {
    tags: nonEmptyString(search.tags),
    exclude: nonEmptyString(search.exclude),
    q: nonEmptyString(search.q),
  };
}

export function filterFromSearch(search: BoardSearch): BoardFilter {
  return {
    tags: parseTagList(search.tags),
    exclude: parseTagList(search.exclude),
    q: (search.q ?? "").trim(),
  };
}

export function searchFromFilter(filter: BoardFilter): BoardSearch {
  const q = filter.q.trim();
  return {
    tags: filter.tags.length > 0 ? filter.tags.join(",") : undefined,
    exclude: filter.exclude.length > 0 ? filter.exclude.join(",") : undefined,
    q: q !== "" ? q : undefined,
  };
}

export function isFilterActive(filter: BoardFilter): boolean {
  return filter.tags.length > 0 || filter.exclude.length > 0 || filter.q.trim() !== "";
}

// Search is tag-based, not filename-based: free text matches tags
// (and the artist credit), never the file-derived title.
const searchFuse = (index: Post[]) =>
  new Fuse(index, {
    keys: [
      { name: "tags", weight: 0.7 },
      { name: "artist", weight: 0.3 },
    ],
    threshold: 0.4,
    ignoreLocation: true,
    minMatchCharLength: 2,
  });

/**
 * Posts matching the filter, in board order. Required tags chain with
 * AND semantics, excluded tags drop any post carrying one, and free
 * text of two or more characters fuzzy-ranks what remains.
 */
export function filterPosts(index: Post[], filter: BoardFilter): Post[] {
  let base = index;
  if (filter.tags.length > 0) {
    base = base.filter((post) => filter.tags.every((tag) => post.tags.includes(tag)));
  }
  if (filter.exclude.length > 0) {
    base = base.filter((post) => !filter.exclude.some((tag) => post.tags.includes(tag)));
  }
  const q = filter.q.trim();
  if (q.length >= 2)
    return searchFuse(base)
      .search(q)
      .map((r) => r.item);
  return base;
}

export interface Neighbors {
  /** Zero-based position of the post, or -1 when it is not in the sequence. */
  index: number;
  total: number;
  prev?: Post;
  next?: Post;
}

export function neighborsOf(sequence: Post[], id: string): Neighbors {
  const index = sequence.findIndex((post) => post.id === id);
  if (index < 0) return { index, total: sequence.length };
  return {
    index,
    total: sequence.length,
    prev: index > 0 ? sequence[index - 1] : undefined,
    next: index < sequence.length - 1 ? sequence[index + 1] : undefined,
  };
}
