import { afterEach, describe, expect, it } from "vitest";
import type { Me } from "../types/api";
import {
  DEFAULT_PAGE_SIZE,
  PAGE_SIZE,
  clampPage,
  clampPostsPerPage,
  pageCount,
  pageOffset,
  pageRangeLabel,
  pageWindow,
  resolvePageSize,
  snapPageSize,
  writeLocalPageSize,
} from "./paging";

describe("DEFAULT_PAGE_SIZE", () => {
  it("is 40 and PAGE_SIZE aliases it", () => {
    expect(DEFAULT_PAGE_SIZE).toBe(40);
    expect(PAGE_SIZE).toBe(DEFAULT_PAGE_SIZE);
  });
});

describe("clampPostsPerPage", () => {
  it("keeps values in 12–100 and defaults otherwise", () => {
    expect(clampPostsPerPage(12)).toBe(12);
    expect(clampPostsPerPage(40)).toBe(40);
    expect(clampPostsPerPage(100)).toBe(100);
    expect(clampPostsPerPage(50)).toBe(50);
    expect(clampPostsPerPage(11)).toBe(DEFAULT_PAGE_SIZE);
    expect(clampPostsPerPage(101)).toBe(DEFAULT_PAGE_SIZE);
    expect(clampPostsPerPage(Number.NaN)).toBe(DEFAULT_PAGE_SIZE);
  });
});

describe("resolvePageSize / writeLocalPageSize", () => {
  afterEach(() => {
    localStorage.removeItem("kura-posts-per-page");
  });

  it("prefers the signed-in preference", () => {
    writeLocalPageSize(24);
    const me = { preferences: { posts_per_page: 60 } } as Me;
    expect(resolvePageSize(me)).toBe(60);
  });

  it("falls back to localStorage then the default", () => {
    expect(resolvePageSize(null)).toBe(DEFAULT_PAGE_SIZE);
    writeLocalPageSize(24);
    expect(resolvePageSize(undefined)).toBe(24);
    expect(resolvePageSize(null)).toBe(24);
  });
});

describe("snapPageSize", () => {
  it("returns preferred when columns are unknown", () => {
    expect(snapPageSize(40, 0)).toBe(40);
    expect(snapPageSize(40, -1)).toBe(40);
  });

  it("snaps to the nearest multiple of columns, preferring up when tied", () => {
    expect(snapPageSize(40, 5)).toBe(40);
    expect(snapPageSize(40, 6)).toBe(42);
    expect(snapPageSize(39, 6)).toBe(42);
    expect(snapPageSize(37, 6)).toBe(36);
  });

  it("clamps the result into 12–100", () => {
    expect(snapPageSize(12, 5)).toBe(15);
    expect(snapPageSize(100, 6)).toBe(96);
    expect(snapPageSize(40, 101)).toBe(40);
  });
});

describe("pageOffset", () => {
  it("maps 1-based pages onto multiples of pageSize", () => {
    expect(pageOffset(1, DEFAULT_PAGE_SIZE)).toBe(0);
    expect(pageOffset(2, DEFAULT_PAGE_SIZE)).toBe(DEFAULT_PAGE_SIZE);
    expect(pageOffset(3, DEFAULT_PAGE_SIZE)).toBe(DEFAULT_PAGE_SIZE * 2);
    expect(pageOffset(2, 20)).toBe(20);
  });

  it("treats non-positive and non-finite values as page 1", () => {
    expect(pageOffset(0, DEFAULT_PAGE_SIZE)).toBe(0);
    expect(pageOffset(-4, DEFAULT_PAGE_SIZE)).toBe(0);
    expect(pageOffset(Number.NaN, DEFAULT_PAGE_SIZE)).toBe(0);
  });

  it("falls back to DEFAULT_PAGE_SIZE when pageSize is invalid", () => {
    expect(pageOffset(2, 0)).toBe(DEFAULT_PAGE_SIZE);
    expect(pageOffset(2, -10)).toBe(DEFAULT_PAGE_SIZE);
    expect(pageOffset(2, Number.NaN)).toBe(DEFAULT_PAGE_SIZE);
  });
});

describe("pageCount", () => {
  it("rounds up and treats an empty list as one page", () => {
    expect(pageCount(0, DEFAULT_PAGE_SIZE)).toBe(1);
    expect(pageCount(1, DEFAULT_PAGE_SIZE)).toBe(1);
    expect(pageCount(DEFAULT_PAGE_SIZE, DEFAULT_PAGE_SIZE)).toBe(1);
    expect(pageCount(DEFAULT_PAGE_SIZE + 1, DEFAULT_PAGE_SIZE)).toBe(2);
    expect(pageCount(80, DEFAULT_PAGE_SIZE)).toBe(2);
    expect(pageCount(21, 20)).toBe(2);
  });

  it("falls back to DEFAULT_PAGE_SIZE when pageSize is invalid", () => {
    expect(pageCount(41, 0)).toBe(2);
  });
});

describe("clampPage", () => {
  it("keeps the page inside the available range", () => {
    expect(clampPage(1, 80, DEFAULT_PAGE_SIZE)).toBe(1);
    expect(clampPage(2, 80, DEFAULT_PAGE_SIZE)).toBe(2);
    expect(clampPage(9, 80, DEFAULT_PAGE_SIZE)).toBe(2);
    expect(clampPage(0, 80, DEFAULT_PAGE_SIZE)).toBe(1);
    expect(clampPage(3, 45, 20)).toBe(3);
    expect(clampPage(9, 45, 20)).toBe(3);
  });
});

describe("pageWindow", () => {
  it("returns only page 1 for a single page", () => {
    expect(pageWindow(1, 10, DEFAULT_PAGE_SIZE)).toEqual([1]);
  });

  it("includes current ± 2 plus first and last, with gaps as null", () => {
    expect(pageWindow(1, 400, DEFAULT_PAGE_SIZE)).toEqual([1, 2, 3, null, 10]);
    expect(pageWindow(5, 400, DEFAULT_PAGE_SIZE)).toEqual([1, null, 3, 4, 5, 6, 7, null, 10]);
    expect(pageWindow(10, 400, DEFAULT_PAGE_SIZE)).toEqual([1, null, 8, 9, 10]);
  });
});

describe("pageRangeLabel", () => {
  it("describes the inclusive row range on this page", () => {
    expect(pageRangeLabel(1, 120, DEFAULT_PAGE_SIZE)).toBe("1–40 of 120");
    expect(pageRangeLabel(2, 120, DEFAULT_PAGE_SIZE)).toBe("41–80 of 120");
    expect(pageRangeLabel(3, 120, DEFAULT_PAGE_SIZE)).toBe("81–120 of 120");
    expect(pageRangeLabel(1, 0, DEFAULT_PAGE_SIZE)).toBe("0 of 0");
    expect(pageRangeLabel(2, 45, 20)).toBe("21–40 of 45");
  });
});
