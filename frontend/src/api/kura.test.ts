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
