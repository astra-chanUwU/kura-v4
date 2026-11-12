import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPostReport } from "../api/reports";
import { PostReport } from "./PostReport";

vi.mock("../api/reports", () => ({
  createPostReport: vi.fn(),
  listOpenReports: vi.fn(),
  dismissReport: vi.fn(),
}));
vi.mock("./Toasts", () => ({
  shortError: (error: unknown) => (error instanceof Error ? error.message : "failed"),
}));

function renderReport(signedIn: boolean) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <PostReport postId="p1" signedIn={signedIn} />
    </QueryClientProvider>,
  );
}

describe("PostReport", () => {
  beforeEach(() => {
    vi.mocked(createPostReport).mockReset();
    vi.mocked(createPostReport).mockResolvedValue({
      id: "r1",
      post_id: "p1",
      thumbnail_url: "",
      reporter_username: "",
      reason: "spam",
      note: "",
      status: "open",
      created_at: "2026-11-10T12:00:00Z",
    });
  });

  it("hides the form until someone is signed in", () => {
    renderReport(false);
    expect(screen.queryByRole("button", { name: "Send report" })).not.toBeInTheDocument();
  });

  it("sends the chosen reason and note", async () => {
    renderReport(true);
    fireEvent.click(screen.getByRole("button", { name: "Reason" }));
    fireEvent.click(screen.getByRole("option", { name: /Wrong rating/ }));
    fireEvent.change(screen.getByLabelText("Note"), { target: { value: "duplicate" } });
    fireEvent.click(screen.getByRole("button", { name: "Send report" }));
    await waitFor(() =>
      expect(createPostReport).toHaveBeenCalledWith("p1", "wrong_rating", "duplicate"),
    );
    expect(await screen.findByText(/Report sent/)).toBeInTheDocument();
  });
});
