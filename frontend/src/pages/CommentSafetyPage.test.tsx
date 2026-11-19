import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ToastsProvider } from "../components/Toasts";
import type { CommentSafetyEvent } from "../types/commentSafety";
import { CommentSafetyPage } from "./CommentSafetyPage";

const hoisted = vi.hoisted(() => ({
  useCommentSafety: vi.fn(),
  useCommentSafetyActions: vi.fn(),
  useCommentSafetyResolvers: vi.fn(),
}));

const blocked: CommentSafetyEvent = {
  id: "e1",
  comment_id: "",
  post_id: "p1",
  author_username: "ada",
  action: "blocked",
  categories: ["phone"],
  body: "call me at 555",
  status: "open",
  created_at: "2026-11-10T12:00:00Z",
  resolved_at: "",
  resolver_id: "",
  resolver_username: "",
  decisions: [],
};

const flagged: CommentSafetyEvent = {
  id: "e2",
  comment_id: "c2",
  post_id: "p2",
  author_username: "mina",
  action: "flagged",
  categories: ["telegram"],
  body: "find me on telegram",
  status: "open",
  created_at: "2026-11-10T12:05:00Z",
  resolved_at: "",
  resolver_id: "",
  resolver_username: "",
  decisions: [],
};

function queueResult(overrides: Record<string, unknown> = {}) {
  return {
    events: [],
    total: Array.isArray(overrides.events) ? overrides.events.length : 0,
    loading: false,
    error: null,
    isSuccess: overrides.error == null,
    prefetchPage: () => {},
    ...overrides,
  };
}

vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    Link: ({
      to,
      params,
      children,
      ...rest
    }: {
      to: string;
      params?: Record<string, string>;
      children?: React.ReactNode;
    } & React.AnchorHTMLAttributes<HTMLAnchorElement>) => {
      let href = to;
      for (const [key, value] of Object.entries(params ?? {}))
        href = href.replace(`$${key}`, value);
      return React.createElement("a", { ...rest, href }, children);
    },
  };
});

vi.mock("../hooks/useCommentSafety", () => ({
  useCommentSafety: (...args: unknown[]) => hoisted.useCommentSafety(...args),
  useCommentSafetyActions: (...args: unknown[]) => hoisted.useCommentSafetyActions(...args),
  useCommentSafetyResolvers: (...args: unknown[]) => hoisted.useCommentSafetyResolvers(...args),
}));

function renderPage() {
  return render(
    <ToastsProvider>
      <CommentSafetyPage />
    </ToastsProvider>,
  );
}

describe("CommentSafetyPage", () => {
  const dismiss = vi.fn();
  const remove = vi.fn();

  beforeEach(() => {
    dismiss.mockReset();
    remove.mockReset();
    hoisted.useCommentSafety.mockReset();
    hoisted.useCommentSafetyActions.mockReset();
    hoisted.useCommentSafetyResolvers.mockReset();
    hoisted.useCommentSafety.mockReturnValue(queueResult());
    hoisted.useCommentSafetyResolvers.mockReturnValue({ people: [], loading: false, error: null });
    hoisted.useCommentSafetyActions.mockReturnValue({
      dismiss: { mutate: dismiss, isPending: false },
      remove: { mutate: remove, isPending: false },
      reopen: { mutate: vi.fn(), isPending: false },
    });
  });

  it("shows only the alert when the queue fails to load", () => {
    hoisted.useCommentSafety.mockReturnValue(
      queueResult({ error: new Error("Something went wrong.") }),
    );
    renderPage();
    expect(screen.getByRole("alert")).toHaveTextContent("Something went wrong.");
    expect(screen.queryByText("No open safety reviews.")).not.toBeInTheDocument();
  });

  it("shows the empty queue after a successful load", () => {
    renderPage();
    expect(screen.getByText("No open safety reviews.")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("lists a blocked review and a flagged review", () => {
    hoisted.useCommentSafety.mockReturnValue(queueResult({ events: [blocked, flagged] }));
    renderPage();
    expect(screen.getByText("Blocked")).toBeInTheDocument();
    expect(screen.getByText("Flagged")).toBeInTheDocument();
    expect(screen.getByText("phone · @ada")).toBeInTheDocument();
    expect(screen.getByText("telegram · @mina")).toBeInTheDocument();
    expect(screen.getByText("call me at 555")).toBeInTheDocument();
    expect(screen.getByText("find me on telegram")).toBeInTheDocument();
    const links = screen.getAllByRole("link", { name: "View post" });
    expect(links[0]).toHaveAttribute("href", "/posts/p1");
    expect(links[1]).toHaveAttribute("href", "/posts/p2");
    expect(screen.getAllByRole("button", { name: "Dismiss" })).toHaveLength(2);
    expect(screen.getAllByRole("button", { name: "Remove" })).toHaveLength(2);
    expect(screen.queryByText("No open safety reviews.")).not.toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("dismisses a review and shows the success toast", () => {
    dismiss.mockImplementation((_id: string, options?: { onSuccess?: () => void }) => {
      options?.onSuccess?.();
    });
    hoisted.useCommentSafety.mockReturnValue(queueResult({ events: [blocked] }));
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(dismiss).toHaveBeenCalledWith("e1", expect.any(Object));
    expect(screen.getByRole("status")).toHaveTextContent("Review dismissed.");
  });

  it("asks for confirmation before removing a comment", () => {
    remove.mockImplementation((_event: unknown, options?: { onSuccess?: () => void }) => {
      options?.onSuccess?.();
    });
    hoisted.useCommentSafety.mockReturnValue(queueResult({ events: [flagged] }));
    renderPage();
    fireEvent.click(screen.getByRole("button", { name: "Remove" }));
    expect(remove).not.toHaveBeenCalled();
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveTextContent("Remove this comment?");
    expect(dialog).toHaveTextContent("The safety review closes. A published comment is deleted.");
    fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(remove).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Remove" }));
    fireEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Remove" }));
    expect(remove).toHaveBeenCalledTimes(1);
    expect(remove).toHaveBeenCalledWith(
      expect.objectContaining({ id: "e2", post_id: "p2", comment_id: "c2" }),
      expect.any(Object),
    );
    expect(screen.getByRole("status")).toHaveTextContent("Comment removed.");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows the latest person on a review that was deleted, restored, and deleted again", () => {
    hoisted.useCommentSafety.mockReturnValue(
      queueResult({
        events: [
          {
            ...flagged,
            status: "removed",
            comment_id: "",
            decisions: [
              {
                actor_id: "a",
                actor_username: "mina",
                action: "removed",
                created_at: "2026-11-19T01:00:00Z",
              },
              {
                actor_id: "b",
                actor_username: "kiri",
                action: "restored",
                created_at: "2026-11-19T02:00:00Z",
              },
              {
                actor_id: "c",
                actor_username: "mika",
                action: "removed",
                created_at: "2026-11-19T03:00:00Z",
              },
            ],
          },
        ],
      }),
    );
    renderPage();
    expect(screen.getByText("deleted by @mina")).toBeInTheDocument();
    expect(screen.getByText("restored by @kiri")).toBeInTheDocument();
    const latest = screen.getByText("deleted by @mika");
    expect(latest).toBeInTheDocument();
    expect(screen.queryByText("Latest")).not.toBeInTheDocument();
  });

  it("filters by a moderator with a button", () => {
    hoisted.useCommentSafetyResolvers.mockReturnValue({
      people: [
        {
          id: "22222222-2222-4222-8222-222222222222",
          username: "kiri",
          decisions: 2,
          last_status: "dismissed",
          last_action: "blocked",
          last_categories: ["phone", "solicitation"],
          last_body: "text me at 555-123-4567",
          last_at: "2026-11-19T02:16:31Z",
        },
      ],
      loading: false,
      error: null,
    });
    renderPage();
    expect(screen.queryByRole("button", { name: "Decided by" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "@kiri" }));
    expect(hoisted.useCommentSafety).toHaveBeenLastCalledWith(
      expect.objectContaining({
        resolver: "22222222-2222-4222-8222-222222222222",
        status: "any",
      }),
      1,
      expect.any(Number),
    );
    fireEvent.click(screen.getByRole("button", { name: "Anyone" }));
    expect(hoisted.useCommentSafety).toHaveBeenLastCalledWith(
      expect.objectContaining({ resolver: "", status: "any" }),
      1,
      expect.any(Number),
    );
  });
});
