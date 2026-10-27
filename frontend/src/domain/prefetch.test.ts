import { describe, expect, it } from "vitest";
import { MAX_PREFETCH_BYTES, shouldPrefetchMedia } from "./prefetch";

const image = { content_type: "image/png", size_bytes: 2048 };

describe("shouldPrefetchMedia", () => {
  it("warms a small image on a normal connection", () => {
    expect(shouldPrefetchMedia(image, { effectiveType: "4g" })).toBe(true);
    expect(shouldPrefetchMedia(image, undefined)).toBe(true);
  });

  it("never warms videos (the post page streams them)", () => {
    expect(shouldPrefetchMedia({ content_type: "video/mp4", size_bytes: 10 }, undefined)).toBe(
      false,
    );
  });

  it("skips oversized images", () => {
    expect(
      shouldPrefetchMedia(
        { content_type: "image/png", size_bytes: MAX_PREFETCH_BYTES + 1 },
        undefined,
      ),
    ).toBe(false);
    expect(
      shouldPrefetchMedia({ content_type: "image/png", size_bytes: MAX_PREFETCH_BYTES }, undefined),
    ).toBe(true);
  });

  it("respects data-saver, reduced-data and slow connections", () => {
    expect(shouldPrefetchMedia(image, { saveData: true })).toBe(false);
    expect(shouldPrefetchMedia(image, undefined, true)).toBe(false);
    expect(shouldPrefetchMedia(image, { effectiveType: "2g" })).toBe(false);
    expect(shouldPrefetchMedia(image, { effectiveType: "slow-2g" })).toBe(false);
    expect(shouldPrefetchMedia(image, { effectiveType: "3g" })).toBe(true);
  });
});
