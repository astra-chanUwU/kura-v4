import { useMemo } from "react";
import {
  countPageTags,
  DEFAULT_PAGE_TAG_LIMIT,
  type PageTag,
} from "../domain/pageTags";
import type { Post } from "../types/api";

export type { PageTag };

/**
 * Suggested tags from the current board page's posts.
 * Stable until the page's post set (or filters) changes.
 */
export function usePageTags(
  posts: Post[],
  selectedTags: string[],
  excludedTags: string[] = [],
  limit = DEFAULT_PAGE_TAG_LIMIT,
): { tags: PageTag[] } {
  const tags = useMemo(
    () => countPageTags(posts, selectedTags, excludedTags, limit),
    [posts, selectedTags, excludedTags, limit],
  );
  return { tags };
}
