import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FailedJobsPage } from "./FailedJobsPage";

const hoisted = vi.hoisted(() => ({
  failedJobs: vi.fn(),
  retryFailedJob: vi.fn(),
}));

vi.mock("../hooks/useKura", () => ({
  useMe: () => ({ user: { id: "u1", username: "ada", role: "moderator" }, loading: false }),
}));
vi.mock("../components/Toasts", () => ({
  shortError: (error: unknown) => (error instanceof Error ? error.message : "failed"),
  useToasts: () => ({ push: vi.fn() }),
}));
vi.mock("../api/kura", () => ({
  api: {
    failedJobs: hoisted.failedJobs,
    retryFailedJob: hoisted.retryFailedJob,
  },
}));

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <FailedJobsPage />
    </QueryClientProvider>,
  );
}

describe("FailedJobsPage", () => {
  beforeEach(() => {
    hoisted.failedJobs.mockReset();
    hoisted.retryFailedJob.mockReset();
    hoisted.failedJobs.mockResolvedValue({
      total: 1,
      jobs: [
        {
          id: 4,
          kind: "post.derive",
          attempts: 3,
          max_attempts: 3,
          last_error: "job execution failed",
          updated_at: "2026-11-10T12:00:00Z",
        },
      ],
    });
    hoisted.retryFailedJob.mockResolvedValue(undefined);
  });

  it("lists a failed job and retries it", async () => {
    renderPage();
    expect(await screen.findByText("post.derive")).toBeInTheDocument();
    expect(screen.getByText(/job execution failed/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(hoisted.retryFailedJob).toHaveBeenCalledWith(4));
  });
});
