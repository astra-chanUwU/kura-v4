import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import { useCallback, useEffect, useState } from "react";
import type { ChangeEntry, Post, SearchPage } from "../types/api";
import { isRole, type Role } from "../domain/roles";
import {
  isFilterActive,
  normalizeTagToken,
  splitTagInput,
  type BoardFilter,
  type Neighbors,
} from "../domain/boardFilter";
import { DEFAULT_PAGE_SIZE, pageOffset } from "../domain/paging";
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

export { normalizeTagToken };

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

function usePagedPosts(
  key: (offset: number, pageSize: number) => readonly unknown[],
  fetchPage: (offset: number, limit: number, signal: AbortSignal) => Promise<SearchPage>,
  enabled: boolean,
  page: number,
  pageSize: number,
) {
  const client = useQueryClient();
  const size = pageSize ?? DEFAULT_PAGE_SIZE;
  const offset = pageOffset(page, size);
  const query = useQuery({
    queryKey: key(offset, size),
    queryFn: ({ signal }) => fetchPage(offset, size, signal),
    enabled,
    placeholderData: keepPreviousData,
  });
  const prefetchPage = useCallback(
    (target: number) => {
      const off = pageOffset(target, size);
      void client.prefetchQuery({
        queryKey: key(off, size),
        queryFn: ({ signal }) => fetchPage(off, size, signal),
      });
    },
    [client, key, fetchPage, size],
  );
  return {
    posts: query.data?.posts ?? [],
    total: query.data?.total ?? 0,
    isPending: query.isPending,
    error: query.error,
    isSuccess: query.isSuccess,
    prefetchPage,
  };
}

export function useFavorites(enabled: boolean, page = 1, pageSize = DEFAULT_PAGE_SIZE) {
  return usePagedPosts(
    queryKeys.favorites,
    api.favorites,
    enabled,
    page,
    pageSize,
  );
}

export function useMyPosts(enabled: boolean, page = 1, pageSize = DEFAULT_PAGE_SIZE) {
  return usePagedPosts(
    queryKeys.myPosts,
    api.myPosts,
    enabled,
    page,
    pageSize,
  );
}

export function useMyLists(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.myLists(),
    queryFn: ({ signal }) => api.myLists(signal),
    enabled,
  });
}

export function useList(id: string, page = 1, pageSize = DEFAULT_PAGE_SIZE) {
  const client = useQueryClient();
  const size = pageSize ?? DEFAULT_PAGE_SIZE;
  const offset = pageOffset(page, size);
  const query = useQuery({
    queryKey: queryKeys.list(id, offset, size),
    queryFn: ({ signal }) => api.list(id, offset, size, signal),
    enabled: id !== "",
    placeholderData: keepPreviousData,
  });
  const prefetchPage = useCallback(
    (target: number) => {
      const off = pageOffset(target, size);
      void client.prefetchQuery({
        queryKey: queryKeys.list(id, off, size),
        queryFn: ({ signal }) => api.list(id, off, size, signal),
      });
    },
    [client, id, size],
  );
  return {
    data: query.data
      ? { list: query.data.list, items: query.data.items, total: query.data.total }
      : undefined,
    isPending: query.isPending,
    error: query.error,
    isSuccess: query.isSuccess,
    prefetchPage,
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
    // Patch every cache that shows this post right away. The board
    // (["board", …]) and the detail view (["post", id]) are different
    // keys, so invalidating only the board left the detail view stale
    // until reload, and repeat clicks kept sending the same
    // favorite/unfavorite instead of toggling.
    onMutate: async ({ postId, favorite }) => {
      const touched: [QueryKey, unknown][] = [];
      const patch = (key: QueryKey, data: unknown) => {
        touched.push([key, client.getQueryData(key)]);
        client.setQueryData(key, data);
      };
      const patchPost = (p: Post): Post => {
        if (p.id !== postId) return p;
        const flipped = p.is_favorite !== favorite;
        return {
          ...p,
          is_favorite: favorite,
          favorite_count: flipped
            ? Math.max(0, p.favorite_count + (favorite ? 1 : -1))
            : p.favorite_count,
        };
      };
      await client.cancelQueries({ queryKey: queryKeys.post(postId) });
      const single = client.getQueryData<Post>(queryKeys.post(postId));
      if (single) patch(queryKeys.post(postId), patchPost(single));
      for (const [key, data] of client.getQueriesData<SearchPage>({ queryKey: ["board"] })) {
        if (!data) continue;
        patch(key, { ...data, posts: data.posts.map(patchPost) });
      }
      for (const prefix of [["favorites"], ["my-posts"]] as const) {
        for (const [key, list] of client.getQueriesData<SearchPage>({ queryKey: [...prefix] })) {
          if (!list) continue;
          patch(key, { ...list, posts: list.posts.map(patchPost) });
        }
      }
      for (const [key, detail] of client.getQueriesData<ListDetail>({ queryKey: ["list"] })) {
        if (!detail) continue;
        patch(key, { ...detail, items: detail.items.map(patchPost) });
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
    addToList: (listId: string, postId: string) =>
      api.addToList(listId, postId).then(() => {
        void client.invalidateQueries({ queryKey: queryKeys.myLists() });
        void client.invalidateQueries({ queryKey: queryKeys.list(listId) });
      }),
    updatePostMeta: (
      postId: string,
      artist: string,
      tags: string[],
      rating: string,
      sourceURL: string,
    ) =>
      api.updatePostMeta(postId, artist, tags, rating, sourceURL).then(() => {
        void client.invalidateQueries({ queryKey: queryKeys.post(postId) });
        void client.invalidateQueries({ queryKey: ["post-related"] });
        invalidateSearch(client);
        void client.invalidateQueries({ queryKey: queryKeys.postHistory(postId) });
      }),
    upload: (file: File, tags = "", rating = "sfw", sourceURL = "") =>
      api.upload(file, tags, rating, sourceURL).then((post) => {
        void client.invalidateQueries({ queryKey: ["post-related"] });
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
    setPostsPerPage: (postsPerPage: number) =>
      api.setPostsPerPage(postsPerPage).then((me) => {
        refresh();
        return me;
      }),
    setAvatar: (file: File) => api.setAvatar(file).then(refresh),
    removeAvatar: () => api.removeAvatar().then(refresh),
    setPassword: (password: string) => api.setPassword(password).then(refresh),
    removePassword: () => api.removePassword().then(refresh),
    addPasskey: () => api.addPasskey().then(refresh),
    removePasskey: (id: string) => api.removePasskey(id).then(refresh),
    regenerateRecoveryKey: () => api.regenerateRecoveryKey(),
  };
}

export type AdminUsersFilter = {
  q?: string;
  role?: string;
  sort?: "name" | "newest" | "oldest";
};

export function useAdminUsers(
  enabled: boolean,
  filter: AdminUsersFilter = {},
  page = 1,
  pageSize = DEFAULT_PAGE_SIZE,
) {
  const client = useQueryClient();
  const q = (filter.q ?? "").trim();
  const requestedRole = filter.role ?? "";
  const role = isRole(requestedRole) ? requestedRole : "";
  const sort = filter.sort === "newest" || filter.sort === "oldest" ? filter.sort : "name";
  const size = pageSize ?? DEFAULT_PAGE_SIZE;
  const offset = pageOffset(page, size);
  const query = useQuery({
    queryKey: queryKeys.adminUsers(q, role, sort, offset, size),
    queryFn: ({ signal }) => api.adminUsers({ offset, limit: size, q, role, sort }, signal),
    enabled,
    placeholderData: keepPreviousData,
  });
  const prefetchPage = useCallback(
    (target: number) => {
      const off = pageOffset(target, size);
      void client.prefetchQuery({
        queryKey: queryKeys.adminUsers(q, role, sort, off, size),
        queryFn: ({ signal }) => api.adminUsers({ offset: off, limit: size, q, role, sort }, signal),
      });
    },
    [client, q, role, sort, size],
  );
  return {
    users: query.data?.users ?? [],
    total: query.data?.total ?? 0,
    member: query.data?.member ?? 0,
    moderator: query.data?.moderator ?? 0,
    admin: query.data?.admin ?? 0,
    superAdmin: query.data?.super_admin ?? 0,
    isPending: query.isPending,
    error: query.error,
    isSuccess: query.isSuccess,
    prefetchPage,
  };
}

export type HiddenQueueFilter = {
  kind?: "all" | "hidden" | "deleted";
  q?: string;
  rating?: "" | "sfw" | "nsfw";
  sort?: "newest" | "oldest";
};

export function useHiddenPosts(
  enabled: boolean,
  filter: HiddenQueueFilter = {},
  page = 1,
  pageSize = DEFAULT_PAGE_SIZE,
) {
  const client = useQueryClient();
  const kind = filter.kind ?? "all";
  const q = (filter.q ?? "").trim();
  const rating = filter.rating === "sfw" || filter.rating === "nsfw" ? filter.rating : "";
  const sort = filter.sort === "oldest" ? "oldest" : "newest";
  const size = pageSize ?? DEFAULT_PAGE_SIZE;
  const offset = pageOffset(page, size);
  const query = useQuery({
    queryKey: queryKeys.hiddenPosts(kind, q, rating, sort, offset, size),
    queryFn: ({ signal }) =>
      api.hiddenPosts({ kind, offset, limit: size, q, rating, sort }, signal),
    enabled,
    placeholderData: keepPreviousData,
  });
  const prefetchPage = useCallback(
    (target: number) => {
      const off = pageOffset(target, size);
      void client.prefetchQuery({
        queryKey: queryKeys.hiddenPosts(kind, q, rating, sort, off, size),
        queryFn: ({ signal }) =>
          api.hiddenPosts({ kind, offset: off, limit: size, q, rating, sort }, signal),
      });
    },
    [client, kind, q, rating, sort, size],
  );
  return {
    posts: query.data?.posts ?? [],
    total: query.data?.total ?? 0,
    hidden: query.data?.hidden ?? 0,
    deleted: query.data?.deleted ?? 0,
    isPending: query.isPending,
    error: query.error,
    isSuccess: query.isSuccess,
    prefetchPage,
  };
}

export type HistoryFilter = {
  q?: string;
  actorId?: string;
  action?: string;
  entity?: string;
  sort?: "newest" | "oldest";
};

export function useRecentHistory(
  enabled: boolean,
  filter: HistoryFilter = {},
  page = 1,
  pageSize = DEFAULT_PAGE_SIZE,
) {
  const client = useQueryClient();
  const q = (filter.q ?? "").trim();
  const actorId = (filter.actorId ?? "").trim();
  const action = (filter.action ?? "").trim();
  const entity = (filter.entity ?? "").trim();
  const sort = filter.sort === "oldest" ? "oldest" : "newest";
  const size = pageSize ?? DEFAULT_PAGE_SIZE;
  const offset = pageOffset(page, size);
  const query = useQuery({
    queryKey: queryKeys.adminHistory(actorId, q, action, entity, sort, offset, size),
    queryFn: ({ signal }) =>
      api.recentHistory({ offset, limit: size, q, actorId, action, entity, sort }, signal),
    enabled,
    placeholderData: keepPreviousData,
  });
  const prefetchPage = useCallback(
    (target: number) => {
      const off = pageOffset(target, size);
      void client.prefetchQuery({
        queryKey: queryKeys.adminHistory(actorId, q, action, entity, sort, off, size),
        queryFn: ({ signal }) =>
          api.recentHistory(
            { offset: off, limit: size, q, actorId, action, entity, sort },
            signal,
          ),
      });
    },
    [client, actorId, q, action, entity, sort, size],
  );
  return {
    entries: query.data?.entries ?? [],
    total: query.data?.total ?? 0,
    isPending: query.isPending,
    error: query.error,
    isSuccess: query.isSuccess,
    prefetchPage,
  };
}

export function useHistoryActors(q: string, enabled: boolean) {
  const text = q.trim();
  return useQuery({
    queryKey: queryKeys.historyActors(text),
    queryFn: ({ signal }) => api.historyActors(text, signal),
    // An empty box means "everyone" — never ask for a list of people.
    enabled: enabled && text !== "",
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
    invalidateSearch(client);
    void client.invalidateQueries({ queryKey: queryKeys.postHistory(postId) });
    void client.invalidateQueries({ queryKey: ["hidden-posts"] });
    void client.invalidateQueries({ queryKey: ["admin-history"] });
  };
  return {
    setUserRole: (userId: string, role: Role) =>
      api.setUserRole(userId, role).then((u) => {
        void client.invalidateQueries({ queryKey: ["admin-users"] });
        void client.invalidateQueries({ queryKey: ["admin-history"] });
        return u;
      }),
    rollback: (entry: ChangeEntry) =>
      api.rollback(entry.id).then((created) => {
        if (entry.entity_type === "post") refreshPost(entry.entity_id);
        void client.invalidateQueries({ queryKey: ["admin-users"] });
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
