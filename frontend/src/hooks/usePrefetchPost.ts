import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import { api } from "../api/kura";
import { queryKeys } from "../api/queryKeys";
import {
  HOVER_PREFETCH_DELAY_MS,
  shouldPrefetchMedia,
  type ConnectionInfo,
} from "../domain/prefetch";
import type { Post } from "../types/api";

// Media warmed this session. The Set keeps each URL one-shot; the
// element list keeps in-flight Images alive until they settle so the
// browser does not discard the download before the post page needs it.
const warmedMedia = new Set<string>();
const inFlight: HTMLImageElement[] = [];

function connectionInfo(): ConnectionInfo | undefined {
  const nav = navigator as Navigator & { connection?: ConnectionInfo };
  return nav.connection;
}

function reducedDataPreferred(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-reduced-data: reduce)").matches
  );
}

function warmMedia(url: string): void {
  if (warmedMedia.has(url)) return;
  warmedMedia.add(url);
  const img = new Image();
  img.decoding = "async";
  // Hover prefetch must yield to thumbnails already loading on the board.
  (img as HTMLImageElement & { fetchPriority?: string }).fetchPriority = "low";
  const settle = () => {
    const i = inFlight.indexOf(img);
    if (i >= 0) inFlight.splice(i, 1);
  };
  img.onload = settle;
  img.onerror = settle;
  inFlight.push(img);
  img.src = url;
}

/** Test seam: forget warmed media between tests. */
export function resetPrefetchCache(): void {
  warmedMedia.clear();
  inFlight.length = 0;
}

/**
 * Hover-intent prefetch for a post link. Spread the returned handlers
 * on the link: after the cursor (or keyboard focus) rests briefly, the
 * post JSON enters the React Query cache and — when
 * shouldPrefetchMedia allows — the full-size image enters the HTTP
 * cache, so opening the post page renders without a network wait.
 * Leaving before the delay cancels; prefetchQuery dedupes the JSON
 * and warmedMedia dedupes the image.
 */
export function usePrefetchPost(post: Post) {
  const client = useQueryClient();
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancel = useCallback(() => {
    if (timer.current !== null) {
      clearTimeout(timer.current);
      timer.current = null;
    }
  }, []);

  useEffect(() => cancel, [cancel]);

  const start = useCallback(() => {
    if (timer.current !== null) return;
    timer.current = setTimeout(() => {
      timer.current = null;
      void client.prefetchQuery({
        queryKey: queryKeys.post(post.id),
        queryFn: ({ signal }) => api.post(post.id, signal),
        staleTime: 60_000,
      });
      if (shouldPrefetchMedia(post, connectionInfo(), reducedDataPreferred())) {
        warmMedia(post.media_url);
      }
    }, HOVER_PREFETCH_DELAY_MS);
  }, [client, post]);

  return { onPointerEnter: start, onFocus: start, onPointerLeave: cancel, onBlur: cancel };
}
