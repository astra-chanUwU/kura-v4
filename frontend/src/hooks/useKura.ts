import {
  keepPreviousData,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type QueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import type { ChangeEntry, Post, SearchPage } from "../types/api";
import type { Role } from "../domain/roles";
import {
  isFilterActive,
  normalizeTagToken,
  splitTagInput,
  type BoardFilter,
  type Neighbors,
} from "../domain/boardFilter";
import { clearBoardMemory } from "../domain/boardMemory";
import { api, DuplicatePostError } from "../api/kura";

/** The existing post when an upload was a duplicate, else undefined. */
export function duplicateOf(err: unknown): Post | undefined {
  return err instanceof DuplicatePostError ? err.post : undefined;
}
import type { ListDetail, Me } from "../types/api";
import { queryKeys } from "../api/queryKeys";
import { passkeysSupported } from "../api/webauthn";
import { ApiError } from "../api/client";

export { passkeysSupported };

export function useMe() {
  const query = useQuery({
    queryKey: queryKeys.me(),
    // A 401 is a signed-out answer, not an error to retry over: map it
    // to null so a stale cached user can never survive sign-out or an
    // expired session (React Query keeps old data on failed refetches).
    queryFn: async ({ signal }) => {
      try {
        return await api.me(signal);
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
    retry: false,
    staleTime: 30_000,
  });
  return {
    me: query.data,
    user: query.data?.user,
    loading: query.isPending,
    signedIn: !!query.data,
    error: query.error,
  };
}

export function isSignedOut(error: unknown): boolean {
  return error instanceof ApiError && error.status === 401;
}

function invalidateSearch(client: QueryClient) {
  void client.invalidateQueries({ queryKey: ["board"] });
  void client.invalidateQueries({ queryKey: ["tag-suggestions"] });
  void client.invalidateQueries({ queryKey: ["neighbors"] });
}

/** The board, filtered or not. Pages come from the server, 40 at a time. */
export function useBoard(filter: BoardFilter, enabled = true) {
  const query = useInfiniteQuery({
    queryKey: queryKeys.board(filter),
    queryFn: ({ pageParam, signal }) => api.search(filter, pageParam, signal),
    initialPageParam: "",
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((n, page) => n + page.posts.length, 0);
      const cursor = lastPage.posts.at(-1)?.id;
      return loaded < lastPage.total && cursor ? cursor : undefined;
    },
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  });
  return {
    posts: query.data?.pages.flatMap((page) => page.posts) ?? [],
    total: query.data?.pages.at(-1)?.total ?? 0,
    loading: query.isPending,
    searching: query.isFetching && !query.isPending,
    error: query.error,
    retry: () => void query.refetch(),
    hasMore: query.hasNextPage ?? false,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void query.fetchNextPage(),
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

function nextOffset(loaded: number, total: number): number | undefined {
  return loaded < total ? loaded : undefined;
}

/** Watches a sentinel and asks for the next page as it nears the viewport. */
export function useLoadMore(enabled: boolean, loadMore: () => void) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el || !enabled || typeof IntersectionObserver === "undefined") return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) loadMore();
      },
      { rootMargin: "900px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [enabled, loadMore]);
  return ref;
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

export { normalizeTagToken };

export function tagTokens(text: string): string[] {
  const { include, exclude } = splitTagInput(text);
  return [...new Set([...include, ...exclude])];
}

export function usePost(id: string) {
  return useQuery({ queryKey: queryKeys.post(id), queryFn: ({ signal }) => api.post(id, signal) });
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

function usePagedPosts(
  key: readonly unknown[],
  fetchPage: (offset: number, signal: AbortSignal) => Promise<SearchPage>,
  enabled: boolean,
) {
  const query = useInfiniteQuery({
    queryKey: key,
    queryFn: ({ pageParam, signal }) => fetchPage(pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((n, page) => n + page.posts.length, 0);
      return nextOffset(loaded, lastPage.total);
    },
    enabled,
  });
  return {
    posts: query.data?.pages.flatMap((page) => page.posts) ?? [],
    total: query.data?.pages.at(-1)?.total ?? 0,
    isPending: query.isPending,
    error: query.error,
    hasMore: query.hasNextPage ?? false,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void query.fetchNextPage(),
  };
}

export function useFavorites(enabled: boolean) {
  return usePagedPosts(
    queryKeys.favorites(),
    (offset, signal) => api.favorites(offset, signal),
    enabled,
  );
}

export function useMyPosts(enabled: boolean) {
  return usePagedPosts(
    queryKeys.myPosts(),
    (offset, signal) => api.myPosts(offset, signal),
    enabled,
  );
}

export function useMyLists(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.myLists(),
    queryFn: ({ signal }) => api.myLists(signal),
    enabled,
  });
}

export function useList(id: string) {
  const query = useInfiniteQuery({
    queryKey: queryKeys.list(id),
    queryFn: ({ pageParam, signal }) => api.list(id, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((n, page) => n + page.items.length, 0);
      return nextOffset(loaded, lastPage.total);
    },
    enabled: id !== "",
  });
  const items = query.data?.pages.flatMap((page) => page.items) ?? [];
  const first = query.data?.pages[0];
  return {
    data: first ? { list: first.list, items, total: first.total } : undefined,
    isPending: query.isPending,
    error: query.error,
    hasMore: query.hasNextPage ?? false,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void query.fetchNextPage(),
  };
}

export function useProfile(username: string) {
  return useQuery({
    queryKey: queryKeys.profile(username),
    queryFn: ({ signal }) => api.profile(username, signal),
  });
}

export function useInvalidate() {
  const client = useQueryClient();
  return () => {
    void client.invalidateQueries({ queryKey: queryKeys.posts() });
    invalidateSearch(client);
    void client.invalidateQueries({ queryKey: queryKeys.favorites() });
    void client.invalidateQueries({ queryKey: queryKeys.myPosts() });
    void client.invalidateQueries({ queryKey: queryKeys.myLists() });
    void client.invalidateQueries({ queryKey: queryKeys.me() });
  };
}

export function useToggleFavorite() {
  const client = useQueryClient();
  const invalidate = useInvalidate();
  return useMutation({
    mutationFn: ({ postId, favorite }: { postId: string; favorite: boolean }) =>
      favorite ? api.favorite(postId) : api.unfavorite(postId),
    // Patch every cache that shows this post right away — the board
    // (["posts"], plural) and the detail view (["post", id], singular)
    // are different keys, so invalidating only the board left the
    // detail view stale until reload, and repeat clicks kept sending
    // the same favorite/unfavorite instead of toggling.
    onMutate: async ({ postId, favorite }) => {
      const touched: [QueryKey, unknown][] = [];
      const patch = (key: QueryKey, data: unknown) => {
        touched.push([key, client.getQueryData(key)]);
        client.setQueryData(key, data);
      };
      const patchPost = (p: Post) => (p.id === postId ? { ...p, is_favorite: favorite } : p);
      await client.cancelQueries({ queryKey: queryKeys.post(postId) });
      const single = client.getQueryData<Post>(queryKeys.post(postId));
      if (single) patch(queryKeys.post(postId), patchPost(single));
      const board = client.getQueryData<InfiniteData<Post[]>>(queryKeys.posts());
      if (board) {
        patch(queryKeys.posts(), {
          ...board,
          pages: board.pages.map((page) => page.map(patchPost)),
        });
      }
      for (const [key, data] of client.getQueriesData<InfiniteData<SearchPage>>({
        queryKey: ["board"],
      })) {
        if (!data) continue;
        patch(key, {
          ...data,
          pages: data.pages.map((page) => ({ ...page, posts: page.posts.map(patchPost) })),
        });
      }
      for (const key of [queryKeys.favorites(), queryKeys.myPosts()] as const) {
        const list = client.getQueryData<InfiniteData<SearchPage>>(key);
        if (list) {
          patch(key, {
            ...list,
            pages: list.pages.map((page) => ({ ...page, posts: page.posts.map(patchPost) })),
          });
        }
      }
      for (const [key, detail] of client.getQueriesData<InfiniteData<ListDetail>>({
        queryKey: ["list"],
      })) {
        if (detail) {
          patch(key, {
            ...detail,
            pages: detail.pages.map((page) => ({ ...page, items: page.items.map(patchPost) })),
          });
        }
      }
      return { touched };
    },
    onError: (_err, _vars, context) => {
      for (const [key, data] of context?.touched ?? []) client.setQueryData(key, data);
    },
    onSettled: (_data, _err, { postId }) => {
      invalidate();
      void client.invalidateQueries({ queryKey: queryKeys.post(postId) });
      void client.invalidateQueries({ queryKey: ["list"] });
    },
  });
}

export function useAuthActions() {
  const client = useQueryClient();
  const invalidate = useInvalidate();
  // Drop everything that belongs to the previous account, then refresh
  // the public caches whose is_favorite flags depend on the viewer.
  const purgeUserCaches = () => {
    client.removeQueries({ queryKey: queryKeys.favorites() });
    client.removeQueries({ queryKey: queryKeys.myPosts() });
    client.removeQueries({ queryKey: queryKeys.myLists() });
  };
  const refreshMe = (me: Me) => {
    // Set the fresh session user directly instead of invalidating:
    // viewers update in the same render, with no stale in-between.
    purgeUserCaches();
    client.setQueryData(queryKeys.me(), me);
    void client.invalidateQueries({ queryKey: queryKeys.posts() });
    invalidateSearch(client);
    void client.invalidateQueries({ queryKey: ["list"] });
    return me;
  };
  return {
    registerPassword: (username: string, password: string, displayName: string) =>
      api.registerPassword(username, password, displayName).then(refreshMe),
    loginPassword: (username: string, password: string) =>
      api.loginPassword(username, password).then(refreshMe),
    registerPasskey: (username: string, displayName: string) =>
      api.registerPasskey(username, displayName).then(refreshMe),
    loginPasskey: () => api.loginPasskey().then(refreshMe),
    recoverAccount: (username: string, recoveryKey: string, newPassword: string) =>
      api.recoverAccount(username, recoveryKey, newPassword).then(refreshMe),
    logout: () =>
      api.logout().then(() => {
        // Clear the user synchronously — invalidating alone leaves the
        // old data visible until (and unless) the refetch lands.
        // Drop the tab's board too, so the next account does not inherit it.
        clearBoardMemory();
        client.setQueryData(queryKeys.me(), null);
        purgeUserCaches();
        invalidate();
      }),
  };
}

export function usePostActions() {
  const client = useQueryClient();
  return {
    addToList: (listId: string, postId: string) => api.addToList(listId, postId),
    updatePostMeta: (postId: string, artist: string, tags: string[], rating: string) =>
      api.updatePostMeta(postId, artist, tags, rating).then(() => {
        void client.invalidateQueries({ queryKey: queryKeys.post(postId) });
        void client.invalidateQueries({ queryKey: queryKeys.posts() });
        invalidateSearch(client);
        void client.invalidateQueries({ queryKey: queryKeys.postHistory(postId) });
      }),
    upload: (file: File, tags = "", rating = "sfw") =>
      api.upload(file, tags, rating).then((post) => {
        void client.invalidateQueries({ queryKey: queryKeys.posts() });
        invalidateSearch(client);
        void client.invalidateQueries({ queryKey: queryKeys.myPosts() });
        return post;
      }),
  };
}

export function useListActions(listId?: string) {
  const client = useQueryClient();
  const refresh = () => {
    void client.invalidateQueries({ queryKey: queryKeys.myLists() });
    if (listId) void client.invalidateQueries({ queryKey: queryKeys.list(listId) });
  };
  return {
    createList: (body: { title: string; description: string; visibility: string }) =>
      api.createList(body).then((created) => {
        refresh();
        return created;
      }),
    updateList: (id: string, body: { title: string; description: string; visibility: string }) =>
      api.updateList(id, body).then(refresh),
    deleteList: (id: string) => api.deleteList(id).then(refresh),
    removeFromList: (id: string, postId: string) => api.removeFromList(id, postId).then(refresh),
    reorderList: (id: string, postIds: string[]) => api.reorderList(id, postIds).then(refresh),
    moveListItem: (id: string, postId: string, afterId: string) =>
      api.moveListItem(id, postId, afterId).then(refresh),
    refresh,
  };
}

export function useAccountActions() {
  const client = useQueryClient();
  const refresh = () => {
    void client.invalidateQueries({ queryKey: queryKeys.me() });
  };
  return {
    updateProfile: (body: {
      display_name: string;
      bio: string;
      location: string;
      website: string;
    }) => api.updateProfile(body).then(refresh),
    setPreferences: (showNsfw: boolean, over18: boolean) =>
      api.setPreferences(showNsfw, over18).then((me) => {
        refresh();
        invalidateSearch(client);
        return me;
      }),
    setTagPreferences: (includeTags: string[], excludeTags: string[]) =>
      api.setTagPreferences(includeTags, excludeTags).then(refresh),
    setAppearance: (theme: string, font: string) => api.setAppearance(theme, font).then(refresh),
    setTagSuggestionRows: (rows: number) => api.setTagSuggestionRows(rows).then(refresh),
    setAvatar: (file: File) => api.setAvatar(file).then(refresh),
    removeAvatar: () => api.removeAvatar().then(refresh),
    setPassword: (password: string) => api.setPassword(password).then(refresh),
    removePassword: () => api.removePassword().then(refresh),
    addPasskey: () => api.addPasskey().then(refresh),
    removePasskey: (id: string) => api.removePasskey(id).then(refresh),
    regenerateRecoveryKey: () => api.regenerateRecoveryKey(),
  };
}

export function useAdminUsers(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.adminUsers(),
    queryFn: ({ signal }) => api.adminUsers(signal),
    enabled,
  });
}

export function useHiddenPosts(enabled: boolean, kind: "all" | "hidden" | "deleted" = "all") {
  const query = useInfiniteQuery({
    queryKey: queryKeys.hiddenPosts(kind),
    queryFn: ({ pageParam, signal }) => api.hiddenPosts(kind, pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((n, page) => n + page.posts.length, 0);
      return nextOffset(loaded, lastPage.total);
    },
    enabled,
  });
  const first = query.data?.pages[0];
  return {
    posts: query.data?.pages.flatMap((page) => page.posts) ?? [],
    total: first?.total ?? 0,
    hidden: first?.hidden ?? 0,
    deleted: first?.deleted ?? 0,
    isPending: query.isPending,
    error: query.error,
    hasMore: (query.hasNextPage ?? false) && enabled,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void query.fetchNextPage(),
  };
}

export function useRecentHistory(actorId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.adminHistory(actorId),
    queryFn: ({ signal }) => api.recentHistory(actorId, signal),
    enabled,
  });
}

export function usePostHistory(postId: string, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.postHistory(postId),
    queryFn: ({ signal }) => api.postHistory(postId, signal),
    enabled,
  });
}

export function useStaffActions() {
  const client = useQueryClient();
  const refreshPost = (postId: string) => {
    void client.invalidateQueries({ queryKey: queryKeys.post(postId) });
    void client.invalidateQueries({ queryKey: queryKeys.posts() });
    invalidateSearch(client);
    void client.invalidateQueries({ queryKey: queryKeys.postHistory(postId) });
    void client.invalidateQueries({ queryKey: ["hidden-posts"] });
    void client.invalidateQueries({ queryKey: ["admin-history"] });
  };
  return {
    setUserRole: (userId: string, role: Role) =>
      api.setUserRole(userId, role).then((u) => {
        void client.invalidateQueries({ queryKey: queryKeys.adminUsers() });
        void client.invalidateQueries({ queryKey: ["admin-history"] });
        return u;
      }),
    rollback: (entry: ChangeEntry) =>
      api.rollback(entry.id).then((created) => {
        if (entry.entity_type === "post") refreshPost(entry.entity_id);
        void client.invalidateQueries({ queryKey: queryKeys.adminUsers() });
        void client.invalidateQueries({ queryKey: ["admin-history"] });
        return created;
      }),
    hidePost: (postId: string) =>
      api.hidePost(postId).then((p) => {
        refreshPost(postId);
        return p;
      }),
    unhidePost: (postId: string) =>
      api.unhidePost(postId).then((p) => {
        refreshPost(postId);
        return p;
      }),
    deletePost: (postId: string) =>
      api.deletePost(postId).then((p) => {
        refreshPost(postId);
        return p;
      }),
  };
}
