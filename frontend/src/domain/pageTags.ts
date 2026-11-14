import type { Post } from "../types/api";
import { normalizeTagToken } from "./boardFilter";

export interface PageTag {
  tag: string;
  count: number;
}

export const DEFAULT_PAGE_TAG_LIMIT = 24;

/**
 * Tag frequencies for the posts on the current board page.
 * Selected and excluded tags are omitted. Sorted by count desc, then name asc.
 */
export function countPageTags(
  posts: Post[],
  selectedTags: string[],
  excludedTags: string[] = [],
  limit = DEFAULT_PAGE_TAG_LIMIT,
): PageTag[] {
  if (posts.length === 0) return [];

  const skip = new Set<string>();
  for (const raw of selectedTags) {
    const tag = normalizeTagToken(raw);
    if (tag) skip.add(tag);
  }
  for (const raw of excludedTags) {
    const tag = normalizeTagToken(raw);
    if (tag) skip.add(tag);
  }

  const counts = new Map<string, number>();
  for (const post of posts) {
    for (const raw of post.tags) {
      const tag = normalizeTagToken(raw);
      if (!tag || skip.has(tag)) continue;
      counts.set(tag, (counts.get(tag) ?? 0) + 1);
    }
  }

  return [...counts.entries()]
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag))
    .slice(0, limit);
}
