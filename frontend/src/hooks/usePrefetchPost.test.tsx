import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "../api/kura";
import { HOVER_PREFETCH_DELAY_MS } from "../domain/prefetch";
import type { Post } from "../types/api";
import { resetPrefetchCache, usePrefetchPost } from "./usePrefetchPost";

vi.mock("../api/kura", () => ({ api: { post: vi.fn() } }));

const post: Post = {
  id: "p1",
  owner_username: "ana",
  content_type: "image/png",
  size_bytes: 2048,
  created_at: "2026-01-01T00:00:00Z",
  is_favorite: false,
  media_url: "/media/posts/p1",
  thumbnail_url: "/media/posts/p1/thumb",
  artist: "ana",
  rating: "sfw",
  hidden: false,
  deleted: false,
  width: 10,
  height: 10,
  source_url: "",
  tags: [],
};

const images: { src: string }[] = [];

class FakeImage {
  decoding = "";
  fetchPriority = "";
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  #src = "";
  constructor() {
    images.push(this as unknown as { src: string });
  }
  set src(value: string) {
    this.#src = value;
  }
  get src() {
    return this.#src;
  }
}

function wrapperFor(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

function setup(p: Post = post) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const hook = renderHook(() => usePrefetchPost(p), { wrapper: wrapperFor(client) });
  return { client, ...hook };
}

beforeEach(() => {
  vi.useFakeTimers();
  images.length = 0;
  resetPrefetchCache();
  vi.mocked(api.post).mockReset().mockResolvedValue(post);
  vi.stubGlobal("Image", FakeImage);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe("usePrefetchPost", () => {
  it("prefetches post JSON and full-size image only after the hover delay", () => {
    const { client, result } = setup();
    act(() => result.current.onPointerEnter());
    act(() => vi.advanceTimersByTime(HOVER_PREFETCH_DELAY_MS - 1));
    expect(api.post).not.toHaveBeenCalled();
    expect(images).toHaveLength(0);

    act(() => vi.advanceTimersByTime(1));
    expect(api.post).toHaveBeenCalledWith("p1", expect.anything());
    expect(images.map((i) => i.src)).toEqual(["/media/posts/p1"]);
    expect(client.getQueryState(["post", "p1"])?.status).toBe("pending");
  });

  it("cancels when the cursor leaves before the delay", () => {
    const { result } = setup();
    act(() => result.current.onPointerEnter());
    act(() => result.current.onPointerLeave());
    act(() => vi.advanceTimersByTime(HOVER_PREFETCH_DELAY_MS + 50));
    expect(api.post).not.toHaveBeenCalled();
    expect(images).toHaveLength(0);
  });

  it("focus also prefetches (keyboard), blur cancels", () => {
    const { result } = setup();
    act(() => result.current.onFocus());
    act(() => vi.advanceTimersByTime(HOVER_PREFETCH_DELAY_MS));
    expect(api.post).toHaveBeenCalledTimes(1);
  });

  it("does not warm the same media twice", () => {
    const first = setup();
    act(() => first.result.current.onPointerEnter());
    act(() => vi.advanceTimersByTime(HOVER_PREFETCH_DELAY_MS));
    const second = setup();
    act(() => second.result.current.onPointerEnter());
    act(() => vi.advanceTimersByTime(HOVER_PREFETCH_DELAY_MS));
    expect(images).toHaveLength(1);
  });

  it("prefetches JSON but not media for videos", () => {
    const { result } = setup({
      ...post,
      id: "v1",
      content_type: "video/mp4",
      media_url: "/media/posts/v1",
    });
    act(() => result.current.onPointerEnter());
    act(() => vi.advanceTimersByTime(HOVER_PREFETCH_DELAY_MS));
    expect(api.post).toHaveBeenCalledWith("v1", expect.anything());
    expect(images).toHaveLength(0);
  });
});
