import Fuse from "fuse.js";
import type { Post } from "../types/api";

/** Newest is the board's default order. Oldest reverses it. */
export type BoardSort = "newest" | "oldest";

/** The board's applied filter: required tags, excluded tags, free text, order. */
export interface BoardFilter {
  tags: string[];
  exclude: string[];
  q: string;
  sort?: BoardSort;
}

/** The same filter as URL search params (?tags=a,b&exclude=c&q=x). */
export interface BoardSearch {
  tags?: string;
  exclude?: string;
  q?: string;
  sort?: BoardSort;
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

export function sameTagSet(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const other = new Set(b);
  return a.every((tag) => other.has(tag));
}

/**
 * One tag line, the way the board search box reads it. Plain tokens are
 * required; a leading "-" excludes. A token in both lists is excluded.
 */
export function splitTagInput(text: string): { include: string[]; exclude: string[] } {
  const include: string[] = [];
  const exclude: string[] = [];
  const seenIn = new Set<string>();
  const seenEx = new Set<string>();
  for (const raw of text.split(/[\s,]+/)) {
    if (!raw) continue;
    const isExclude = raw.startsWith("-") && raw.length > 1;
    const normalized = normalizeTagToken(isExclude ? raw.slice(1) : raw);
    if (!normalized) continue;
    if (isExclude) {
      if (seenEx.has(normalized)) continue;
      seenEx.add(normalized);
      exclude.push(normalized);
    } else if (!seenIn.has(normalized)) {
      seenIn.add(normalized);
      include.push(normalized);
    }
  }
  return { include: include.filter((tag) => !seenEx.has(tag)), exclude };
}

/** Saved default lists: at most one home per tag, and the edited list wins. */
export function mergeTagEdit(
  kind: "include" | "exclude",
  raw: string,
  include: string[],
  exclude: string[],
): { include: string[]; exclude: string[] } {
  const parsed = splitTagInput(raw);
  const addingExclude =
    kind === "exclude" ? [...parsed.include, ...parsed.exclude] : parsed.exclude;
  const addingInclude = (kind === "include" ? parsed.include : []).filter(
    (tag) => !addingExclude.includes(tag),
  );
  const nextExclude = [
    ...new Set([...exclude.filter((tag) => !addingInclude.includes(tag)), ...addingExclude]),
  ];
  const nextInclude = [
    ...new Set([...include.filter((tag) => !nextExclude.includes(tag)), ...addingInclude]),
  ];
  return { include: nextInclude.slice(0, 30), exclude: nextExclude.slice(0, 30) };
}

const nonEmptyString = (value: unknown): string | undefined =>
  typeof value === "string" && value !== "" ? value : undefined;

/** Router validateSearch for every route that carries the board filter. */
export function boardSort(value: unknown): BoardSort {
  return value === "oldest" ? "oldest" : "newest";
}

export function validateBoardSearch(search: Record<string, unknown>): BoardSearch {
  const out: BoardSearch = {
    tags: nonEmptyString(search.tags),
    exclude: nonEmptyString(search.exclude),
    q: nonEmptyString(search.q),
  };
  if (search.sort === "oldest") out.sort = "oldest";
  return out;
}

export function filterFromSearch(search: BoardSearch): BoardFilter {
  return {
    tags: parseTagList(search.tags),
    exclude: parseTagList(search.exclude),
    q: (search.q ?? "").trim(),
    sort: boardSort(search.sort),
  };
}

export function searchFromFilter(filter: BoardFilter): BoardSearch {
  const q = filter.q.trim();
  const out: BoardSearch = {
    tags: filter.tags.length > 0 ? filter.tags.join(",") : undefined,
    exclude: filter.exclude.length > 0 ? filter.exclude.join(",") : undefined,
    q: q !== "" ? q : undefined,
  };
  if (filter.sort === "oldest") out.sort = "oldest";
  return out;
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
  prev?: { id: string };
  next?: { id: string };
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
