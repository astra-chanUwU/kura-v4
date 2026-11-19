import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  dismissCommentSafety,
  listCommentSafety,
  listCommentSafetyResolvers,
  removeCommentSafety,
  reopenCommentSafety,
} from "../api/commentSafety";
import { queryKeys } from "../api/queryKeys";
import { pageOffset } from "../domain/paging";
import type { CommentSafetyEvent, CommentSafetyQuery } from "../types/commentSafety";

export type CommentSafetyFilter = Pick<
  CommentSafetyQuery,
  "status" | "action" | "category" | "q" | "resolver" | "sort"
>;

const openFilter: CommentSafetyFilter = {
  status: "open",
  action: "",
  category: "",
  q: "",
  resolver: "",
  sort: "newest",
};

export function useCommentSafety(
  filter: CommentSafetyFilter = openFilter,
  page = 1,
  pageSize = 40,
) {
  const client = useQueryClient();
  const offset = pageOffset(page, pageSize);
  const query = useQuery({
    queryKey: queryKeys.commentSafety(
      filter.status,
      filter.action,
      filter.category,
      filter.q,
      filter.resolver,
      filter.sort,
      offset,
      pageSize,
    ),
    queryFn: ({ signal }) =>
      listCommentSafety({ ...filter, limit: pageSize, offset }, signal),
    placeholderData: keepPreviousData,
    retry: false,
  });
  const prefetchPage = useCallback(
    (target: number) => {
      const next = pageOffset(target, pageSize);
      void client.prefetchQuery({
        queryKey: queryKeys.commentSafety(
          filter.status,
          filter.action,
          filter.category,
          filter.q,
          filter.resolver,
          filter.sort,
          next,
          pageSize,
        ),
        queryFn: ({ signal }) =>
          listCommentSafety({ ...filter, limit: pageSize, offset: next }, signal),
      });
    },
    [client, filter.action, filter.category, filter.q, filter.resolver, filter.sort, filter.status, pageSize],
  );
  return {
    events: query.data?.events ?? [],
    total: query.data?.total ?? 0,
    loading: query.isPending,
    error: query.error,
    isSuccess: query.isSuccess,
    prefetchPage,
  };
}

export function useCommentSafetyResolvers() {
  const query = useQuery({
    queryKey: queryKeys.commentSafetyResolvers(),
    queryFn: ({ signal }) => listCommentSafetyResolvers(signal),
    retry: false,
  });
  return {
    people: query.data?.resolvers ?? [],
    loading: query.isPending,
    error: query.error,
  };
}

function refreshQueue(client: ReturnType<typeof useQueryClient>) {
  void client.invalidateQueries({ queryKey: ["comment-safety"] });
}

export function useCommentSafetyActions() {
  const client = useQueryClient();
  const dismiss = useMutation({
    mutationFn: (id: string) => dismissCommentSafety(id),
    onSuccess: () => refreshQueue(client),
  });
  const remove = useMutation({
    mutationFn: (event: CommentSafetyEvent) => removeCommentSafety(event.id),
    onSuccess: (_event, input) => {
      refreshQueue(client);
      if (input.comment_id !== "") {
        void client.invalidateQueries({ queryKey: queryKeys.comments(input.post_id) });
      }
    },
  });
  const reopen = useMutation({
    mutationFn: (event: CommentSafetyEvent) => reopenCommentSafety(event.id),
    onSuccess: (event, input) => {
      refreshQueue(client);
      if (event.comment_id !== "" || input.status === "removed") {
        void client.invalidateQueries({ queryKey: queryKeys.comments(input.post_id) });
      }
    },
  });
  return { dismiss, remove, reopen };
}
