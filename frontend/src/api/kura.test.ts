import { afterEach, describe, expect, it, vi } from "vitest";
import { api, DuplicatePostError } from "./kura";

afterEach(() => vi.unstubAllGlobals());

describe("api.upload", () => {
  it("surfaces the server's duplicate message instead of a generic failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: "request_error",
            message:
              "Already on the board — that exact picture is already here, so it wasn't added again.",
            request_id: "r1",
          }),
          { status: 409 },
        ),
      ),
    );
    const file = new File(["x"], "dup.png", { type: "image/png" });
    await expect(api.upload(file, "", "sfw")).rejects.toThrow("Already on the board");
  });

  it("surfaces the upload limit message", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: "request_error",
            message: "Upload limit reached. Try again later.",
            request_id: "r1",
          }),
          { status: 429 },
        ),
      ),
    );
    const file = new File(["x"], "cap.png", { type: "image/png" });
    await expect(api.upload(file, "", "sfw")).rejects.toThrow(
      "Upload limit reached. Try again later.",
    );
  });

  it("carries the existing post on a duplicate so the UI can show it", async () => {
    const existing = { id: "post-9", thumbnail_url: "/media/posts/post-9/thumb" };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: "duplicate_post",
            message: "Already on the board.",
            post: existing,
          }),
          {
            status: 409,
          },
        ),
      ),
    );
    const file = new File(["x"], "dup.png", { type: "image/png" });
    const err = await api.upload(file, "", "sfw").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(DuplicatePostError);
    expect((err as DuplicatePostError).post.id).toBe("post-9");
  });
});

describe("api.recentHistory", () => {
  it("always sends limit, offset, and sort, and omits empty filters", async () => {
    const fetchMock = vi.fn().mockImplementation(
      () =>
        new Response(JSON.stringify({ total: 0, entries: [] }), {
          status: 200,
        }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await api.recentHistory({ offset: 0 });
    await api.recentHistory({
      offset: 0,
      sort: "oldest",
      q: "hatsune",
      actorId: "6b1e0c3a-1111-4111-8111-111111111111",
      action: "post.hidden",
      entity: "post",
    });
    await api.recentHistory({
      offset: 40,
      q: "  ",
      actorId: "",
      action: "",
      entity: "",
      sort: "newest",
    });

    expect(fetchMock.mock.calls.map((call) => call[0])).toEqual([
      "/api/v1/admin/history?limit=40&offset=0&sort=newest",
      "/api/v1/admin/history?limit=40&offset=0&sort=oldest&q=hatsune&actor_id=6b1e0c3a-1111-4111-8111-111111111111&action=post.hidden&entity=post",
      "/api/v1/admin/history?limit=40&offset=40&sort=newest",
    ]);
  });
});

describe("api.adminUsers", () => {
  it("always sends limit, offset, and sort, and omits empty q and role", async () => {
    const fetchMock = vi.fn().mockImplementation(
      () =>
        new Response(
          JSON.stringify({
            total: 0,
            member: 0,
            moderator: 0,
            admin: 0,
            super_admin: 0,
            users: [],
          }),
          { status: 200 },
        ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await api.adminUsers({ offset: 0 });
    await api.adminUsers({
      offset: 0,
      sort: "newest",
      q: "kiri",
      role: "moderator",
    });
    await api.adminUsers({
      offset: 40,
      q: "  ",
      role: "",
      sort: "oldest",
    });

    expect(fetchMock.mock.calls.map((call) => call[0])).toEqual([
      "/api/v1/admin/users?limit=40&offset=0&sort=name",
      "/api/v1/admin/users?limit=40&offset=0&sort=newest&q=kiri&role=moderator",
      "/api/v1/admin/users?limit=40&offset=40&sort=oldest",
    ]);
  });
});

describe("api.historyActors", () => {
  it("encodes the typed query", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response("[]", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await api.historyActors("hatsune miku");
    expect(fetchMock.mock.calls.map((call) => call[0])).toEqual([
      "/api/v1/admin/history-actors?q=hatsune+miku",
    ]);
  });
});
