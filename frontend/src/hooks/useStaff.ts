import { keepPreviousData, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { api } from "../api/kura";
import { queryKeys } from "../api/queryKeys";
import { DEFAULT_PAGE_SIZE, pageOffset } from "../domain/paging";
import { isRole, type Role } from "../domain/roles";
import type { ChangeEntry } from "../types/api";
import { invalidateSearch } from "./useBoardData";

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
