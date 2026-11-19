import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  dismissCommentSafety,
  listCommentSafety,
  listCommentSafetyResolvers,
  removeCommentSafety,
  reopenCommentSafety,
} from "../api/commentSafety";
import { queryKeys } from "../api/queryKeys";
import type { CommentSafetyEvent, CommentSafetyPage } from "../types/commentSafety";
import { useCommentSafety, useCommentSafetyActions, useCommentSafetyResolvers } from "./useCommentSafety";

vi.mock("../api/commentSafety", () => ({
  listCommentSafety: vi.fn(),
  listCommentSafetyResolvers: vi.fn(),
  dismissCommentSafety: vi.fn(),
  removeCommentSafety: vi.fn(),
  reopenCommentSafety: vi.fn(),
}));

const blocked: CommentSafetyEvent = {
  id: "e1",
  comment_id: "",
  post_id: "p1",
  author_username: "ada",
  action: "blocked",
  categories: ["phone"],
  body: "call me",
  status: "open",
  created_at: "2026-11-10T12:00:00Z",
  resolved_at: "",
  resolver_id: "",
  resolver_username: "",
  decisions: [],
};

const flagged: CommentSafetyEvent = {
  ...blocked,
  id: "e2",
  comment_id: "c2",
  post_id: "p2",
  action: "flagged",
  categories: ["telegram"],
};

const page: CommentSafetyPage = { total: 2, events: [blocked, flagged] };

function wrapperFor(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

function client() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

describe("useCommentSafety", () => {
  beforeEach(() => {
    vi.mocked(listCommentSafety).mockReset();
  });

  it("keeps the staff queue under the comment-safety key", async () => {
    vi.mocked(listCommentSafety).mockResolvedValue(page);
    const cache = client();
    const { result } = renderHook(() => useCommentSafety(), { wrapper: wrapperFor(cache) });
    await waitFor(() => expect(result.current.events).toEqual([blocked, flagged]));
    expect(result.current.total).toBe(2);
    expect(cache.getQueryData(queryKeys.commentSafety())).toEqual(page);
  });

  it("loads the people who decided under the comment-safety key", async () => {
    const people = {
      resolvers: [
        {
          id: "22222222-2222-4222-8222-222222222222",
          username: "kiri",
          decisions: 2,
          last_status: "dismissed" as const,
          last_action: "blocked" as const,
          last_categories: ["telegram"],
          last_body: "https://t.me/example_user",
          last_at: "2026-11-19T02:03:04Z",
        },
      ],
    };
    vi.mocked(listCommentSafetyResolvers).mockResolvedValue(people);
    const cache = client();
    const { result } = renderHook(() => useCommentSafetyResolvers(), { wrapper: wrapperFor(cache) });
    await waitFor(() => expect(result.current.people).toEqual(people.resolvers));
    expect(cache.getQueryData(queryKeys.commentSafetyResolvers())).toEqual(people);
  });
});

describe("useCommentSafetyActions", () => {
  beforeEach(() => {
    vi.mocked(dismissCommentSafety).mockReset();
    vi.mocked(removeCommentSafety).mockReset();
    vi.mocked(reopenCommentSafety).mockReset();
  });

  it("marks the safety queue stale after a dismiss", async () => {
    vi.mocked(dismissCommentSafety).mockResolvedValue({ ...blocked, status: "dismissed" });
    const cache = client();
    cache.setQueryData(queryKeys.commentSafety(), page);
    cache.setQueryData(queryKeys.commentSafetyResolvers(), { resolvers: [] });
    const { result } = renderHook(() => useCommentSafetyActions(), { wrapper: wrapperFor(cache) });
    act(() => {
      result.current.dismiss.mutate("e1");
    });
    await waitFor(() => expect(result.current.dismiss.isSuccess).toBe(true));
    expect(dismissCommentSafety).toHaveBeenCalledWith("e1");
    expect(cache.getQueryState(queryKeys.commentSafety())?.isInvalidated).toBe(true);
    expect(cache.getQueryState(queryKeys.commentSafetyResolvers())?.isInvalidated).toBe(true);
  });

  it("drops a published comment from the post after remove", async () => {
    vi.mocked(removeCommentSafety).mockResolvedValue({ ...flagged, status: "removed" });
    const cache = client();
    cache.setQueryData(queryKeys.commentSafety(), page);
    cache.setQueryData(queryKeys.comments("p2"), { total: 1, comments: [] });
    const { result } = renderHook(() => useCommentSafetyActions(), { wrapper: wrapperFor(cache) });
    act(() => {
      result.current.remove.mutate(flagged);
    });
    await waitFor(() => expect(result.current.remove.isSuccess).toBe(true));
    expect(removeCommentSafety).toHaveBeenCalledWith("e2");
    expect(cache.getQueryState(queryKeys.commentSafety())?.isInvalidated).toBe(true);
    expect(cache.getQueryState(queryKeys.comments("p2"))?.isInvalidated).toBe(true);
  });

  it("leaves the post comments alone when the review has no comment", async () => {
    vi.mocked(removeCommentSafety).mockResolvedValue({ ...blocked, status: "removed" });
    const cache = client();
    cache.setQueryData(queryKeys.commentSafety(), page);
    cache.setQueryData(queryKeys.comments("p1"), { total: 0, comments: [] });
    const { result } = renderHook(() => useCommentSafetyActions(), { wrapper: wrapperFor(cache) });
    act(() => {
      result.current.remove.mutate(blocked);
    });
    await waitFor(() => expect(result.current.remove.isSuccess).toBe(true));
    expect(cache.getQueryState(queryKeys.commentSafety())?.isInvalidated).toBe(true);
    expect(cache.getQueryState(queryKeys.comments("p1"))?.isInvalidated).toBe(false);
  });

  it("publishes a removed comment back onto the post when the review reopens", async () => {
    vi.mocked(reopenCommentSafety).mockResolvedValue({
      ...flagged,
      status: "open",
      comment_id: "c9",
    });
    const removed = { ...flagged, status: "removed" as const, comment_id: "" };
    const cache = client();
    cache.setQueryData(queryKeys.commentSafety(), page);
    cache.setQueryData(queryKeys.comments("p2"), { total: 0, comments: [] });
    const { result } = renderHook(() => useCommentSafetyActions(), { wrapper: wrapperFor(cache) });
    act(() => {
      result.current.reopen.mutate(removed);
    });
    await waitFor(() => expect(result.current.reopen.isSuccess).toBe(true));
    expect(reopenCommentSafety).toHaveBeenCalledWith("e2");
    expect(cache.getQueryState(queryKeys.commentSafety())?.isInvalidated).toBe(true);
    expect(cache.getQueryState(queryKeys.comments("p2"))?.isInvalidated).toBe(true);
  });
});
