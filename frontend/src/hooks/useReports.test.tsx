import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { dismissReport, listOpenReports } from "../api/reports";
import { queryKeys } from "../api/queryKeys";
import type { PostReport, PostReportPage } from "../types/api";
import { useOpenReports, useReportActions } from "./useReports";

vi.mock("../api/reports", () => ({
  listOpenReports: vi.fn(),
  dismissReport: vi.fn(),
  createPostReport: vi.fn(),
}));

const report: PostReport = {
  id: "r1",
  post_id: "p1",
  thumbnail_url: "/media/posts/p1/thumb",
  reporter_username: "ada",
  reason: "spam",
  note: "duplicate",
  status: "open",
  created_at: "2026-11-10T12:00:00Z",
};

const page: PostReportPage = { total: 1, reports: [report] };

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

describe("useOpenReports", () => {
  beforeEach(() => {
    vi.mocked(listOpenReports).mockReset();
  });

  it("keeps the staff queue under the open-reports key", async () => {
    vi.mocked(listOpenReports).mockResolvedValue(page);
    const cache = client();
    const { result } = renderHook(() => useOpenReports(), { wrapper: wrapperFor(cache) });
    await waitFor(() => expect(result.current.reports).toEqual([report]));
    expect(result.current.total).toBe(1);
    expect(cache.getQueryData(queryKeys.openReports())).toEqual(page);
  });
});

describe("useReportActions", () => {
  beforeEach(() => {
    vi.mocked(dismissReport).mockReset();
  });

  it("marks the open queue stale after a dismiss", async () => {
    vi.mocked(dismissReport).mockResolvedValue({ ...report, status: "dismissed" });
    const cache = client();
    cache.setQueryData(queryKeys.openReports(), page);
    const { result } = renderHook(() => useReportActions(), { wrapper: wrapperFor(cache) });
    act(() => {
      result.current.dismiss.mutate("r1");
    });
    await waitFor(() => expect(result.current.dismiss.isSuccess).toBe(true));
    expect(dismissReport).toHaveBeenCalledWith("r1");
    expect(cache.getQueryState(queryKeys.openReports())?.isInvalidated).toBe(true);
  });
});
