import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { ApiError } from "../api/client";
import { api } from "../api/kura";
import type { Me, Post } from "../types/api";
import { useAuthActions, useMe, useToggleFavorite } from "./useKura";

vi.mock("../api/kura", () => ({
  api: {
    favorite: vi.fn().mockResolvedValue(undefined),
    unfavorite: vi.fn().mockResolvedValue(undefined),
    me: vi.fn(),
    logout: vi.fn().mockResolvedValue(undefined),
  },
  DuplicatePostError: class extends Error {},
}));

const post: Post = {
  id: "p1",
  owner_username: "someone",
  content_type: "image/png",
  size_bytes: 10,
  created_at: "2026-01-01T00:00:00Z",
  is_favorite: false,
  media_url: "/media/posts/p1",
  thumbnail_url: "/media/posts/p1/thumb",
  artist: "",
  rating: "sfw",
  hidden: false,
  deleted: false,
  width: 10,
  height: 10,
  source_url: "",
  tags: [],
};

const meFixture = {
  user: { id: "u1", username: "someone", display_name: "Someone" },
} as unknown as Me;

function wrapperFor(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("auth cache", () => {
  it("drops the cached user when /me starts returning 401 (signed out)", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    vi.mocked(api.me).mockResolvedValueOnce(meFixture);
    const { result } = renderHook(() => useMe(), { wrapper: wrapperFor(client) });
    await waitFor(() => expect(result.current.signedIn).toBe(true));
    vi.mocked(api.me).mockRejectedValue(
      new ApiError(401, { code: "unauthorized", message: "Sign in required.", request_id: "" }),
    );
    await act(async () => {
      await client.refetchQueries({ queryKey: ["me"] });
    });
    await waitFor(() => expect(result.current.signedIn).toBe(false));
    expect(result.current.me ?? null).toBeNull();
  });

  it("logout clears the me cache immediately, without a refetch", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(["me"], meFixture);
    client.setQueryData(["favorites"], [post]);
    const { result } = renderHook(() => useAuthActions(), { wrapper: wrapperFor(client) });
    await act(async () => {
      await result.current.logout();
    });
    expect(client.getQueryData(["me"]) ?? null).toBeNull();
    expect(client.getQueryData(["favorites"])).toBeUndefined();
  });
});

describe("useToggleFavorite", () => {
  it("updates the single-post cache the detail view reads", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(["post", "p1"], post);
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
    const { result } = renderHook(() => useToggleFavorite(), { wrapper });
    await act(async () => {
      result.current.mutate({ postId: "p1", favorite: true });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(api.favorite).toHaveBeenCalledWith("p1");
    expect(client.getQueryData<Post>(["post", "p1"])?.is_favorite).toBe(true);
  });
});
