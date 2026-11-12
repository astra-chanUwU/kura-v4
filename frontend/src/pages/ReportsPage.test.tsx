import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PostReport } from "../types/api";
import { ReportsPage } from "./ReportsPage";

const hoisted = vi.hoisted(() => ({
  useOpenReports: vi.fn(),
  useReportActions: vi.fn(),
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

function reportsResult(overrides: Record<string, unknown> = {}) {
  return {
    reports: [],
    loading: false,
    error: null,
    ...overrides,
  };
}

vi.mock("../hooks/useReports", () => ({
  useOpenReports: (...args: unknown[]) => hoisted.useOpenReports(...args),
  useReportActions: (...args: unknown[]) => hoisted.useReportActions(...args),
}));

describe("ReportsPage", () => {
  beforeEach(() => {
    hoisted.useOpenReports.mockReset();
    hoisted.useReportActions.mockReset();
    hoisted.useOpenReports.mockReturnValue(reportsResult());
    hoisted.useReportActions.mockReturnValue({
      dismiss: { mutate: vi.fn(), isPending: false },
    });
  });

  it("shows only the alert when the queue fails to load", () => {
    hoisted.useOpenReports.mockReturnValue(
      reportsResult({ error: new Error("Something went wrong.") }),
    );
    render(<ReportsPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Something went wrong.");
    expect(screen.queryByText("No open reports.")).not.toBeInTheDocument();
  });

  it("shows the empty queue after a successful load", () => {
    render(<ReportsPage />);
    expect(screen.getByText("No open reports.")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("lists reports after a successful load", () => {
    hoisted.useOpenReports.mockReturnValue(reportsResult({ reports: [report] }));
    render(<ReportsPage />);
    expect(screen.getByText("spam")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dismiss" })).toBeInTheDocument();
    expect(screen.queryByText("No open reports.")).not.toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
