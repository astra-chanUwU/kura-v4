import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import { useCallback } from "react";
import { api } from "../api/kura";
import { queryKeys } from "../api/queryKeys";
import { DEFAULT_PAGE_SIZE, pageOffset } from "../domain/paging";
import type { ListDetail, Post, SearchPage } from "../types/api";
import { invalidatePostSurfaces, invalidateSearch } from "./useBoardData";

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

function patchPostContainer(data: unknown, patchPost: (post: Post) => Post): unknown {
  if (Array.isArray(data)) {
    return data.map((item) =>
      item && typeof item === "object" && "id" in item && "favorite_count" in item
        ? patchPost(item as Post)
        : item,
    );
  }
  if (!data || typeof data !== "object") return data;
  const record = data as Record<string, unknown>;
  let changed = false;
  const next: Record<string, unknown> = { ...record };
  for (const key of ["posts", "items", "uploads"]) {
    if (!Array.isArray(record[key])) continue;
    next[key] = (record[key] as Post[]).map(patchPost);
    changed = true;
  }
  return changed ? next : data;
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
      for (const prefix of ["post-related", "profile", "tag-page", "hidden-posts"] as const) {
        for (const [key, data] of client.getQueriesData({ queryKey: [prefix] })) {
          if (data == null) continue;
          const next = patchPostContainer(data, patchPost);
          if (next !== data) patch(key, next);
        }
      }
      return { touched };
    },
    onError: (_err, _vars, context) => {
      for (const [key, data] of context?.touched ?? []) client.setQueryData(key, data);
    },
    onSettled: (_data, _err, { postId }) => {
      invalidate();
      invalidatePostSurfaces(client, postId);
    },
  });
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
