import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, getJSON } from "./client";

afterEach(() => vi.unstubAllGlobals());
describe("API failures", () => {
  it("keeps the server error and request identifier", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ code: "unavailable", message: "Try later", request_id: "r1" }),
            { status: 503 },
          ),
        ),
    );
    await expect(getJSON("/site")).rejects.toMatchObject({
      status: 503,
      details: { request_id: "r1" },
      message: "Try later",
    });
  });
  it("handles a proxy returning an HTML error page", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("<html>bad gateway</html>", { status: 502 })),
    );
    await expect(getJSON("/site")).rejects.toBeInstanceOf(ApiError);
  });
  it("passes cancellation through to fetch", async () => {
    const controller = new AbortController();
    const fetchMock = vi.fn().mockResolvedValue(new Response("{}"));
    vi.stubGlobal("fetch", fetchMock);
    await getJSON("/site", controller.signal);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/site",
      expect.objectContaining({ signal: controller.signal, credentials: "same-origin" }),
    );
  });
});
