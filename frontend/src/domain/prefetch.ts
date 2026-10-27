import type { Post } from "../types/api";

/** Hover-intent delay: a passing cursor sweep must not trigger fetches. */
export const HOVER_PREFETCH_DELAY_MS = 65;

/** Full-size images above this are not prefetched on hover. */
export const MAX_PREFETCH_BYTES = 10 * 1024 * 1024;

export interface ConnectionInfo {
  saveData?: boolean;
  effectiveType?: string;
}

type MediaPost = Pick<Post, "content_type" | "size_bytes">;

/**
 * Whether hovering should also warm the full-size media, not just the
 * post JSON. Videos are excluded (the post page streams them with
 * preload="metadata"), as are oversized files and data-saver / slow
 * connections — prefetch must never cost more than the click saves.
 */
export function shouldPrefetchMedia(
  post: MediaPost,
  connection: ConnectionInfo | null | undefined,
  reducedData = false,
): boolean {
  if (!post.content_type.startsWith("image/")) return false;
  if (post.size_bytes > MAX_PREFETCH_BYTES) return false;
  if (reducedData) return false;
  if (connection?.saveData) return false;
  const type = connection?.effectiveType ?? "";
  if (type === "slow-2g" || type === "2g") return false;
  return true;
}
