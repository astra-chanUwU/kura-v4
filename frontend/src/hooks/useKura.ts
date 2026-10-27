import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
  type InfiniteData,
  type QueryKey,
} from "@tanstack/react-query";
import Fuse from "fuse.js";
import { useEffect, useMemo, useState } from "react";
import type { ChangeEntry, Post } from "../types/api";
import type { Role } from "../domain/roles";
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

export function usePosts() {
  const query = useInfiniteQuery({
    queryKey: queryKeys.posts(),
    queryFn: ({ pageParam, signal }) => api.posts(pageParam, signal),
    initialPageParam: 0,
    getNextPageParam: (lastPage, allPages) =>
      lastPage.length < 40 ? undefined : allPages.reduce((n, page) => n + page.length, 0),
    staleTime: 60_000,
  });
  return {
    posts: query.data?.pages.flat() ?? [],
    loading: query.isPending,
    error: query.error,
    retry: () => void query.refetch(),
    hasMore: query.hasNextPage ?? false,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void query.fetchNextPage(),
  };
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

// Mirror of the server's booru-style tag normalization: spaces and
// commas separate tags, underscores join words inside a tag.
export function normalizeTagToken(raw: string): string {
  return raw
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "_")
    .replace(/-/g, "_")
    .replace(/^_+|_+$/g, "");
}

export function tagTokens(text: string): string[] {
  return [
    ...new Set(
      text
        .split(/[\s,]+/)
        .map(normalizeTagToken)
        .filter(Boolean),
    ),
  ];
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
 * Search over the whole board. The server sends a lightweight index of
 * every post once; Fuse.js runs locally, so typos still find things
 * ("serraphyl" -> Serraphyll) without hammering the API.
 *
 * Selected tags chain with AND semantics (a post must carry every one),
 * like a booru; any remaining free text fuzzy-matches within that set.
 * Autocomplete suggestions come from the tags already in the index.
 */
export function usePostSearch(
  query: string,
  selectedTags: string[],
  excludedTags: string[],
  wantIndex: boolean,
  appliedQuery = "",
  stagedTags: string[] = [],
) {
  // `query` is live input: it only drives autocomplete suggestions.
  // Results are deliberate — they follow the applied tags plus the
  // query as it was when Filter was pressed (`appliedQuery`), so
  // staging several tags costs no re-filtering in between.
  const trimmed = query.trim();
  const appliedTrimmed = appliedQuery.trim();
  const active = appliedTrimmed.length > 0 || selectedTags.length > 0 || excludedTags.length > 0;
  const index = useQuery({
    queryKey: queryKeys.searchIndex(),
    queryFn: ({ signal }) => api.searchIndex(signal),
    enabled: wantIndex,
    staleTime: 5 * 60_000,
  });

  // Tag frequencies across the board, for autocomplete.
  const tagCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const post of index.data ?? []) {
      for (const tag of post.tags) counts.set(tag, (counts.get(tag) ?? 0) + 1);
    }
    return [...counts.entries()]
      .map(([tag, count]) => ({ tag, count }))
      .sort((a, b) => b.count - a.count);
  }, [index.data]);

  const suggestions = useMemo(() => {
    if (!index.data) return [];
    const available = tagCounts.filter(
      (t) =>
        !selectedTags.includes(t.tag) &&
        !excludedTags.includes(t.tag) &&
        !stagedTags.includes(t.tag),
    );
    if (trimmed.length < 1) return available.slice(0, 8);
    return new Fuse(available, { keys: ["tag"], threshold: 0.35, ignoreLocation: true })
      .search(trimmed)
      .slice(0, 8)
      .map((r) => r.item);
  }, [index.data, tagCounts, selectedTags, excludedTags, stagedTags, trimmed]);

  const results = useMemo(() => {
    if (!active || !index.data) return [];
    let base = index.data;
    if (selectedTags.length > 0) {
      base = base.filter((post) => selectedTags.every((tag) => post.tags.includes(tag)));
    }
    if (excludedTags.length > 0) {
      base = base.filter((post) => !excludedTags.some((tag) => post.tags.includes(tag)));
    }
    if (appliedTrimmed.length >= 2) {
      return searchFuse(base)
        .search(appliedTrimmed)
        .slice(0, 200)
        .map((r) => r.item);
    }
    return base.slice(0, 200);
  }, [active, index.data, selectedTags, excludedTags, appliedTrimmed]);

  return {
    active,
    searching: active && index.isPending,
    results,
    total: index.data?.length ?? 0,
    shown: results.length,
    suggestions,
  };
}

export function usePost(id: string) {
  return useQuery({ queryKey: queryKeys.post(id), queryFn: ({ signal }) => api.post(id, signal) });
}

export function useFavorites(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.favorites(),
    queryFn: ({ signal }) => api.favorites(signal),
    enabled,
  });
}

export function useMyPosts(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.myPosts(),
    queryFn: ({ signal }) => api.myPosts(signal),
    enabled,
  });
}

export function useMyLists(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.myLists(),
    queryFn: ({ signal }) => api.myLists(signal),
    enabled,
  });
}

export function useList(id: string) {
  return useQuery({ queryKey: queryKeys.list(id), queryFn: ({ signal }) => api.list(id, signal) });
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
    void client.invalidateQueries({ queryKey: queryKeys.searchIndex() });
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
      for (const key of [queryKeys.favorites(), queryKeys.myPosts()] as const) {
        const list = client.getQueryData<Post[]>(key);
        if (list) patch(key, list.map(patchPost));
      }
      for (const [key, detail] of client.getQueriesData<ListDetail>({ queryKey: ["list"] })) {
        if (detail) patch(key, { ...detail, items: detail.items.map(patchPost) });
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
        void client.invalidateQueries({ queryKey: queryKeys.searchIndex() });
        void client.invalidateQueries({ queryKey: queryKeys.postHistory(postId) });
      }),
    upload: (file: File, tags = "", rating = "sfw") =>
      api.upload(file, tags, rating).then((post) => {
        void client.invalidateQueries({ queryKey: queryKeys.posts() });
        void client.invalidateQueries({ queryKey: queryKeys.searchIndex() });
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
      api.setPreferences(showNsfw, over18).then(refresh),
    setTagPreferences: (includeTags: string[], excludeTags: string[]) =>
      api.setTagPreferences(includeTags, excludeTags).then(refresh),
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

export function useHiddenPosts(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.hiddenPosts(),
    queryFn: ({ signal }) => api.hiddenPosts(signal),
    enabled,
  });
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
    void client.invalidateQueries({ queryKey: queryKeys.searchIndex() });
    void client.invalidateQueries({ queryKey: queryKeys.postHistory(postId) });
    void client.invalidateQueries({ queryKey: queryKeys.hiddenPosts() });
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
