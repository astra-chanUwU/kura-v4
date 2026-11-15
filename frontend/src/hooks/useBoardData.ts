import { keepPreviousData, useQuery, type QueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { api } from "../api/kura";
import { queryKeys } from "../api/queryKeys";
import {
  isFilterActive,
  splitTagInput,
  type BoardFilter,
  type Neighbors,
} from "../domain/boardFilter";
import { DEFAULT_PAGE_SIZE, pageOffset } from "../domain/paging";

export function invalidateSearch(client: QueryClient) {
  void client.invalidateQueries({ queryKey: ["board"] });
  void client.invalidateQueries({ queryKey: ["tag-suggestions"] });
  void client.invalidateQueries({ queryKey: ["neighbors"] });
}

/** Prefetch one board page with the same key/fn as useBoard. */
export function prefetchBoardPage(
  client: QueryClient,
  filter: BoardFilter,
  page: number,
  pageSize: number,
) {
  const size = pageSize ?? DEFAULT_PAGE_SIZE;
  const offset = pageOffset(page, size);
  return client.prefetchQuery({
    queryKey: queryKeys.board(filter, offset, size),
    queryFn: ({ signal }) => api.search(filter, offset, size, signal),
    staleTime: 60_000,
  });
}

/** The board, filtered or not. One server page for the given page number. */
export function useBoard(
  filter: BoardFilter,
  page = 1,
  pageSize = DEFAULT_PAGE_SIZE,
  enabled = true,
) {
  const size = pageSize ?? DEFAULT_PAGE_SIZE;
  const offset = pageOffset(page, size);
  const query = useQuery({
    queryKey: queryKeys.board(filter, offset, size),
    queryFn: ({ signal }) => api.search(filter, offset, size, signal),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  });
  return {
    posts: query.data?.posts ?? [],
    total: query.data?.total ?? 0,
    loading: query.isPending,
    searching: query.isFetching && !query.isPending,
    error: query.error,
    isSuccess: query.isSuccess,
    retry: () => void query.refetch(),
    active: isFilterActive(filter),
  };
}

/** Tag names for the search box, counted inside the board filter already on. */
export function useTagSuggestions(text: string, filter: BoardFilter, enabled: boolean) {
  const q = text.trim().toLowerCase();
  const query = useQuery({
    queryKey: queryKeys.tagSuggestions(q, filter),
    queryFn: ({ signal }) => api.tagSuggestions(q, filter, signal),
    enabled,
    staleTime: 60_000,
  });
  return { suggestions: query.data ?? [], loading: query.isFetching };
}

// Debounce a rapidly changing value (e.g. keystrokes) so downstream
// work — filtering, and any query keyed on it — runs at most every
// `delay` ms instead of on every keystroke.
export function useDebouncedValue<T>(value: T, delay = 200): T {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delay);
    return () => clearTimeout(timer);
  }, [value, delay]);
  return debounced;
}

export function tagTokens(text: string): string[] {
  const { include, exclude } = splitTagInput(text);
  return [...new Set([...include, ...exclude])];
}

export function usePost(id: string) {
  return useQuery({ queryKey: queryKeys.post(id), queryFn: ({ signal }) => api.post(id, signal) });
}

export function useRelatedPosts(id: string) {
  return useQuery({
    queryKey: queryKeys.related(id),
    queryFn: ({ signal }) => api.relatedPosts(id, signal),
    enabled: id !== "",
  });
}

export function useRandomPost() {
  return (filter: BoardFilter) => api.randomPost(filter);
}

/**
 * Where a post sits in the board sequence for `filter` — the same order
 * and matching the board uses — so the post page can step to the
 * previous/next picture without dropping the viewer's filters.
 */
export function usePostNeighbors(id: string, filter: BoardFilter): Neighbors & { ready: boolean } {
  const query = useQuery({
    queryKey: queryKeys.neighbors(id, filter),
    queryFn: ({ signal }) => api.neighbors(id, filter, signal),
    staleTime: 60_000,
  });
  if (!query.data) return { index: -1, total: 0, ready: false };
  return {
    index: query.data.index,
    total: query.data.total,
    prev: query.data.prev_id ? { id: query.data.prev_id } : undefined,
    next: query.data.next_id ? { id: query.data.next_id } : undefined,
    ready: true,
  };
}
