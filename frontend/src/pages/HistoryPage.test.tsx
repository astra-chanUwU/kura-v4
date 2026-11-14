import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DEFAULT_PAGE_SIZE } from "../domain/paging";
import type { ChangeEntry, User } from "../types/api";
import { HistoryPage } from "./HistoryPage";

const hoisted = vi.hoisted(() => ({
  useRecentHistory: vi.fn(),
  useHistoryActors: vi.fn(),
}));

const moderator: User = {
  id: "mod-id",
  role: "moderator",
  username: "mod",
  display_name: "Mod",
  bio: "",
  location: "",
  website: "",
  created_at: "2026-11-10T00:00:00Z",
  avatar_url: "",
};

const entry: ChangeEntry = {
  id: "h1",
  entity_type: "post",
  entity_id: "p1",
  thumbnail_url: "/media/posts/p1/thumb",
  actor_id: "ada-id",
  actor_username: "ada",
  action: "post.hidden",
  before: { hidden: false },
  after: { hidden: true },
  rollback_of: null,
  created_at: "2026-10-27T00:00:00Z",
};

function historyResult(overrides: Record<string, unknown> = {}) {
  return {
    entries: [],
    total: 0,
    isPending: false,
    error: null,
    isSuccess: true,
    prefetchPage: vi.fn(),
    ...overrides,
  };
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    children,
    ...props
  }: {
    to?: string;
    children?: ReactNode;
    params?: Record<string, string>;
  }) => {
    const { params: _params, ...rest } = props;
    return (
      <a href={to} {...rest}>
        {children}
      </a>
    );
  },
}));

vi.mock("../hooks/useKura", async () => {
  const actual = await vi.importActual<typeof import("../hooks/useKura")>("../hooks/useKura");
  return {
    useDebouncedValue: actual.useDebouncedValue,
    useMe: () => ({ user: moderator }),
    useRecentHistory: (...args: unknown[]) => hoisted.useRecentHistory(...args),
    useHistoryActors: (...args: unknown[]) => hoisted.useHistoryActors(...args),
    useStaffActions: () => ({ rollback: vi.fn(() => Promise.resolve()) }),
  };
});

vi.mock("../components/Toasts", () => ({
  useToasts: () => ({ push: vi.fn() }),
  shortError: (err: unknown) => (err instanceof Error ? err.message : "error"),
}));

const emptyFilter = {
  q: "",
  actorId: "",
  action: "",
  entity: "",
  sort: "newest",
};

describe("HistoryPage filters", () => {
  beforeEach(() => {
    hoisted.useRecentHistory.mockReset();
    hoisted.useHistoryActors.mockReset();
    hoisted.useRecentHistory.mockReturnValue(historyResult());
    hoisted.useHistoryActors.mockReturnValue({ data: [], isFetching: false });
    window.scrollTo = vi.fn();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows search, action, about, order, a person field, and Me — not a menu of loaded usernames", () => {
    hoisted.useRecentHistory.mockReturnValue(historyResult({ total: 1, entries: [entry] }));
    render(<HistoryPage />);

    expect(screen.getByRole("combobox", { name: "Person" })).toHaveAttribute(
      "placeholder",
      "Everyone",
    );
    expect(screen.getByRole("button", { name: "Me" })).toHaveAttribute("title", "Only my changes");
    expect(screen.getByRole("button", { name: "Me" })).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("textbox", { name: "Search changes" })).toHaveAttribute(
      "placeholder",
      "Tag, artist, role, or source",
    );
    expect(screen.getByRole("button", { name: "Action" })).toHaveTextContent("Any change");
    expect(screen.getByRole("button", { name: "About" })).toHaveTextContent("Everything");
    expect(screen.getByRole("button", { name: "Order" })).toHaveTextContent("Newest");

    expect(screen.getByRole("link", { name: "@ada" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "@ada" })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Filter history by actor" }),
    ).not.toBeInTheDocument();
    expect(document.querySelector("select")).toBeNull();
    expect(hoisted.useRecentHistory).toHaveBeenCalledWith(true, emptyFilter, 1, DEFAULT_PAGE_SIZE);
  });

  it("debounces search text before passing it to useRecentHistory", () => {
    vi.useFakeTimers();
    render(<HistoryPage />);
    fireEvent.change(screen.getByRole("textbox", { name: "Search changes" }), {
      target: { value: "hatsune" },
    });
    expect(hoisted.useRecentHistory).not.toHaveBeenCalledWith(
      true,
      expect.objectContaining({ q: "hatsune" }),
      expect.anything(),
      expect.anything(),
    );

    act(() => {
      vi.advanceTimersByTime(299);
    });
    expect(hoisted.useRecentHistory).not.toHaveBeenCalledWith(
      true,
      expect.objectContaining({ q: "hatsune" }),
      expect.anything(),
      expect.anything(),
    );

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      {
        ...emptyFilter,
        q: "hatsune",
      },
      1,
      DEFAULT_PAGE_SIZE,
    );
  });

  it("sends a chosen person id only after the person query settles", () => {
    vi.useFakeTimers();
    hoisted.useHistoryActors.mockImplementation((q: string) =>
      q === "ada"
        ? { data: [{ id: "ada-id", username: "ada" }], isFetching: false }
        : { data: [], isFetching: false },
    );
    render(<HistoryPage />);
    fireEvent.change(screen.getByRole("combobox", { name: "Person" }), {
      target: { value: "ada" },
    });

    act(() => {
      vi.advanceTimersByTime(119);
    });
    expect(hoisted.useHistoryActors).not.toHaveBeenCalledWith("ada", true);
    expect(screen.queryByRole("option", { name: "@ada" })).not.toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(hoisted.useHistoryActors).toHaveBeenCalledWith("ada", true);
    fireEvent.click(screen.getByRole("option", { name: "@ada" }));
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      {
        ...emptyFilter,
        actorId: "ada-id",
      },
      1,
      DEFAULT_PAGE_SIZE,
    );
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show everyone's changes" })).toBeInTheDocument();
  });

  it("limits the log to the signed-in user without searching for them", () => {
    render(<HistoryPage />);
    fireEvent.click(screen.getByRole("button", { name: "Me" }));
    expect(screen.getByRole("button", { name: "Me" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByText("@mod")).toBeInTheDocument();
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      {
        ...emptyFilter,
        actorId: "mod-id",
      },
      1,
      DEFAULT_PAGE_SIZE,
    );
    expect(hoisted.useHistoryActors).not.toHaveBeenCalledWith("mod", expect.anything());
    expect(hoisted.useHistoryActors).not.toHaveBeenCalledWith("mod-id", expect.anything());
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
  });

  it("says nothing is recorded when the unfiltered log is empty", () => {
    render(<HistoryPage />);
    expect(screen.getAllByText("No changes recorded yet.")).toHaveLength(1);
    expect(screen.queryByText("Nothing matches.")).not.toBeInTheDocument();
    expect(screen.getByText("0 matches")).toBeInTheDocument();
  });

  it("says nothing matches once a query filters the log to zero rows", () => {
    vi.useFakeTimers();
    render(<HistoryPage />);
    fireEvent.change(screen.getByRole("textbox", { name: "Search changes" }), {
      target: { value: "hatsune" },
    });
    expect(screen.getByText("No changes recorded yet.")).toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    expect(screen.queryByText("No changes recorded yet.")).not.toBeInTheDocument();
  });

  it("uses the filtered total for the match count and hides it while the first page loads", () => {
    hoisted.useRecentHistory.mockReturnValue(historyResult({ total: 12, entries: [entry] }));
    const view = render(<HistoryPage />);
    expect(screen.getByText("12 matches")).toBeInTheDocument();
    expect(screen.queryByText("1 match")).not.toBeInTheDocument();

    hoisted.useRecentHistory.mockReturnValue(historyResult({ isPending: true, total: 1 }));
    view.rerender(<HistoryPage />);
    expect(screen.queryByText("1 match")).not.toBeInTheDocument();
    expect(screen.queryByText("12 matches")).not.toBeInTheDocument();
    expect(screen.queryByText("No changes recorded yet.")).not.toBeInTheDocument();
    expect(screen.getByText("Loading…")).toBeInTheDocument();

    hoisted.useRecentHistory.mockReturnValue(historyResult({ total: 1, entries: [entry] }));
    view.rerender(<HistoryPage />);
    expect(screen.getByText("1 match")).toBeInTheDocument();
  });

  it("passes action, about, and order through, and does not treat order alone as an empty result", () => {
    render(<HistoryPage />);
    fireEvent.click(screen.getByRole("button", { name: "Order" }));
    fireEvent.click(screen.getByRole("option", { name: "Oldest" }));
    expect(screen.getByText("No changes recorded yet.")).toBeInTheDocument();
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      {
        ...emptyFilter,
        sort: "oldest",
      },
      1,
      DEFAULT_PAGE_SIZE,
    );

    fireEvent.click(screen.getByRole("button", { name: "Action" }));
    fireEvent.click(screen.getByRole("option", { name: "Hidden" }));
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      {
        ...emptyFilter,
        action: "post.hidden",
        sort: "oldest",
      },
      1,
      DEFAULT_PAGE_SIZE,
    );

    fireEvent.click(screen.getByRole("button", { name: "About" }));
    fireEvent.click(screen.getByRole("option", { name: "Posts" }));
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      {
        ...emptyFilter,
        action: "post.hidden",
        entity: "post",
        sort: "oldest",
      },
      1,
      DEFAULT_PAGE_SIZE,
    );
  });

  it("shows pagination when total exceeds one page", () => {
    hoisted.useRecentHistory.mockReturnValue(historyResult({ entries: [entry], total: 80 }));
    render(<HistoryPage />);
    expect(screen.getByRole("navigation", { name: "Pagination" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Roll back/ })).toBeInTheDocument();
  });

  it("clamps past the last page once a successful total is known", () => {
    hoisted.useRecentHistory.mockReturnValue(historyResult({ entries: [entry], total: 80 }));
    const view = render(<HistoryPage />);
    fireEvent.click(screen.getByRole("button", { name: "2" }));
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      emptyFilter,
      2,
      DEFAULT_PAGE_SIZE,
    );

    hoisted.useRecentHistory.mockReturnValue(historyResult({ entries: [entry], total: 5 }));
    view.rerender(<HistoryPage />);
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      emptyFilter,
      1,
      DEFAULT_PAGE_SIZE,
    );
  });

  it("does not clamp while the query is not successful", () => {
    hoisted.useRecentHistory.mockReturnValue(historyResult({ entries: [entry], total: 80 }));
    const view = render(<HistoryPage />);
    fireEvent.click(screen.getByRole("button", { name: "2" }));

    hoisted.useRecentHistory.mockReturnValue(
      historyResult({ entries: [], total: 0, isSuccess: false, isPending: true }),
    );
    view.rerender(<HistoryPage />);
    expect(hoisted.useRecentHistory).toHaveBeenLastCalledWith(
      true,
      emptyFilter,
      2,
      DEFAULT_PAGE_SIZE,
    );
  });
});
