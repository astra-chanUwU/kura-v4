import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { Post } from "../types/api";

export interface ViewportTag {
  tag: string;
  count: number;
}

/**
 * Tracks which post cards are currently in the viewport and derives
 * a limited, scored tag recommendation from them.
 *
 * Like Safebooru's tag sidebar but for infinite scroll: as you scroll,
 * the suggested tags change to reflect what's actually on screen.
 * Tags already selected are excluded. Frequency wins, ties broken
 * alphabetically.
 */
export function useViewportTags(
  posts: Post[],
  selectedTags: string[],
  limit = 12,
): { recommended: ViewportTag[]; observe: (el: Element | null, id: string) => void } {
  const [visibleIds, setVisibleIds] = useState<Set<string>>(new Set());
  const visibleRef = useRef<Set<string>>(new Set());
  const idByElement = useRef<Map<Element, string>>(new Map());
  const observerRef = useRef<IntersectionObserver | null>(null);

  // Throttle state updates via rAF so rapid scroll doesn't spam renders.
  const pendingRef = useRef<number | null>(null);
  const flush = useCallback(() => {
    if (pendingRef.current !== null) return;
    pendingRef.current = requestAnimationFrame(() => {
      pendingRef.current = null;
      setVisibleIds(new Set(visibleRef.current));
    });
  }, []);

  // Create single observer once.
  useEffect(() => {
    if (typeof IntersectionObserver === "undefined") return;
    const obs = new IntersectionObserver(
      (entries) => {
        let changed = false;
        for (const e of entries) {
          const id = idByElement.current.get(e.target);
          if (!id) continue;
          if (e.isIntersecting && e.intersectionRatio > 0) {
            if (!visibleRef.current.has(id)) {
              visibleRef.current.add(id);
              changed = true;
            }
          } else {
            if (visibleRef.current.has(id)) {
              visibleRef.current.delete(id);
              changed = true;
            }
          }
        }
        if (changed) flush();
      },
      {
        threshold: [0, 0.1, 0.5],
        rootMargin: "0px",
      },
    );
    observerRef.current = obs;
    const mapRef = idByElement.current;
    return () => {
      obs.disconnect();
      observerRef.current = null;
      mapRef.clear();
    };
  }, [flush]);

  // Keep selected set fast for filtering.
  const selectedSet = useMemo(() => new Set(selectedTags), [selectedTags]);
  const postById = useMemo(() => new Map(posts.map((p) => [p.id, p])), [posts]);

  // Prune visibleIds when posts list changes (e.g. filter cleared).
  useEffect(() => {
    const ids = new Set(posts.map((p) => p.id));
    let pruned = false;
    for (const vid of visibleRef.current) {
      if (!ids.has(vid)) {
        visibleRef.current.delete(vid);
        pruned = true;
      }
    }
    if (pruned) flush();
  }, [posts, flush]);

  const observe = useCallback(
    (el: Element | null, id: string) => {
      const obs = observerRef.current;
      if (!obs) return;
      // Disconnect previous element for this id if any (callback refs can move).
      for (const [elem, eid] of idByElement.current) {
        if (eid === id && elem !== el) {
          obs.unobserve(elem);
          idByElement.current.delete(elem);
          visibleRef.current.delete(id);
        }
      }
      if (el) {
        idByElement.current.set(el, id);
        obs.observe(el);
      } else {
        // Cleanup on unmount: find element for id
        for (const [elem, eid] of idByElement.current) {
          if (eid === id) {
            obs.unobserve(elem);
            idByElement.current.delete(elem);
            visibleRef.current.delete(id);
            flush();
            break;
          }
        }
      }
    },
    [flush],
  );

  const recommended = useMemo<ViewportTag[]>(() => {
    // Source posts: visible ones if any, otherwise fall back to first page
    // so the bar isn't empty before observation kicks in.
    const sourceIds = visibleIds.size > 0 ? [...visibleIds] : posts.slice(0, 20).map((p) => p.id);
    const counts = new Map<string, number>();
    for (const id of sourceIds) {
      const post = postById.get(id);
      if (!post) continue;
      for (const t of post.tags) {
        const tag = t.toLowerCase();
        if (selectedSet.has(tag)) continue;
        if (!tag) continue;
        counts.set(tag, (counts.get(tag) ?? 0) + 1);
      }
    }
    // Also count artist pseudo-tags? Keep pure tags to match booru feel.
    return [...counts.entries()]
      .map(([tag, count]) => ({ tag, count }))
      .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag))
      .slice(0, limit);
  }, [visibleIds, posts, postById, selectedSet, limit]);

  return { recommended, observe };
}
