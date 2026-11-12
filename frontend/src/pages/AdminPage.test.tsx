import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "../types/api";
import { AdminPage } from "./AdminPage";

const setUserRole = vi.hoisted(() => vi.fn(() => Promise.resolve()));
const hoisted = vi.hoisted(() => ({
  useAdminUsers: vi.fn(),
}));

const actor: User = {
  id: "astra",
  role: "super_admin",
  username: "astra",
  display_name: "Astra Vale",
  bio: "",
  location: "",
  website: "",
  created_at: "2026-11-10T00:00:00Z",
  avatar_url: "",
};

const people: User[] = [
  actor,
  {
    ...actor,
    id: "noa",
    role: "admin",
    username: "noa",
    display_name: "Noa Lin",
  },
];

vi.mock("@tanstack/react-router", () => ({
  Link: ({ to, children }: { to: string; children?: React.ReactNode }) => (
    <a href={to}>{children}</a>
  ),
  Navigate: () => null,
}));

function usersResult(overrides: Record<string, unknown> = {}) {
  return {
    users: people,
    total: people.length,
    member: 4,
    moderator: 3,
    admin: 2,
    superAdmin: 1,
    isPending: false,
    error: null,
    hasMore: false,
    loadingMore: false,
    loadMore: () => {},
    ...overrides,
  };
}

vi.mock("../hooks/useKura", async () => {
  const actual = await vi.importActual<typeof import("../hooks/useKura")>("../hooks/useKura");
  return {
    useDebouncedValue: actual.useDebouncedValue,
    useLoadMore: () => ({ current: null }),
    useMe: () => ({ user: actor }),
    useAdminUsers: (...args: unknown[]) => hoisted.useAdminUsers(...args),
    useStaffActions: () => ({ setUserRole }),
  };
});

describe("AdminPage roles", () => {
  beforeEach(() => {
    setUserRole.mockClear();
    hoisted.useAdminUsers.mockReset();
    hoisted.useAdminUsers.mockReturnValue(usersResult());
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("stages a role behind a dot until Set", () => {
    render(<AdminPage />);
    const group = within(screen.getByRole("radiogroup", { name: "Role for @astra" }));
    expect(group.getByRole("radio", { name: "Super Admin" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    expect(screen.queryByRole("img", { name: "Role change not saved yet" })).toBeNull();
    expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();

    fireEvent.click(group.getByRole("radio", { name: "Moderator" }));
    expect(group.getByRole("radio", { name: "Moderator" })).toHaveAttribute("aria-checked", "true");
    expect(group.getByRole("radio", { name: "Super Admin" })).toHaveAttribute(
      "aria-checked",
      "false",
    );
    expect(screen.getAllByRole("img", { name: "Role change not saved yet" })).toHaveLength(1);
    expect(screen.getAllByRole("button", { name: "Save" })).toHaveLength(1);
    expect(setUserRole).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(setUserRole).toHaveBeenCalledWith("astra", "moderator");
  });

  it("lists each role change on its own line", () => {
    render(<AdminPage />);
    const astra = within(screen.getByRole("radiogroup", { name: "Role for @astra" }));
    fireEvent.click(astra.getByRole("radio", { name: "Member" }));
    fireEvent.click(screen.getAllByRole("radio", { name: "Super Admin" })[1]!);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    const dialog = within(screen.getByRole("dialog", { name: "Save these roles?" }));
    expect(
      dialog.getAllByRole("listitem").map((item) => item.textContent?.replace(/\s+/g, " ").trim()),
    ).toEqual(["@astra → Member", "@noa → Super Admin"]);
    expect(setUserRole).not.toHaveBeenCalled();
  });

  it("asks before setting Admin or Super Admin", () => {
    render(<AdminPage />);
    fireEvent.click(screen.getAllByRole("radio", { name: "Super Admin" })[1]!);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(setUserRole).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toHaveTextContent("Make @noa a Super Admin?");
    fireEvent.click(screen.getByRole("button", { name: "Make Super Admin" }));
    expect(setUserRole).toHaveBeenCalledWith("noa", "super_admin");
  });

  it("shows search, role, order, and the match count", () => {
    render(<AdminPage />);
    expect(screen.getByRole("textbox", { name: "Search people" })).toHaveAttribute(
      "placeholder",
      "Username or display name",
    );
    expect(screen.getByRole("button", { name: "Role" })).toHaveTextContent("Everyone (10)");
    expect(screen.getByRole("button", { name: "Order" })).toHaveTextContent("Name");
    expect(screen.getByText("2 matches")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Role" }));
    expect(screen.getByRole("option", { name: "Everyone (10)" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Member (4)" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Moderator (3)" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Admin (2)" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Super Admin (1)" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("option", { name: "Moderator (3)" }));
    expect(hoisted.useAdminUsers).toHaveBeenLastCalledWith(true, {
      q: "",
      role: "moderator",
      sort: "name",
    });

    fireEvent.click(screen.getByRole("button", { name: "Order" }));
    fireEvent.click(screen.getByRole("option", { name: "Oldest" }));
    expect(hoisted.useAdminUsers).toHaveBeenLastCalledWith(true, {
      q: "",
      role: "moderator",
      sort: "oldest",
    });
  });

  it("hides the match count while the first page is pending and uses the singular label", () => {
    hoisted.useAdminUsers.mockReturnValue(usersResult({ isPending: true, total: 1, users: [] }));
    const view = render(<AdminPage />);
    expect(screen.queryByText("1 match")).not.toBeInTheDocument();
    expect(screen.queryByText("0 matches")).not.toBeInTheDocument();
    expect(screen.getByText("Loading…")).toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();

    hoisted.useAdminUsers.mockReturnValue(usersResult({ total: 1, users: [people[0]] }));
    view.rerender(<AdminPage />);
    expect(screen.getByText("1 match")).toBeInTheDocument();
  });

  it("debounces search text before passing it to useAdminUsers", () => {
    vi.useFakeTimers();
    render(<AdminPage />);
    fireEvent.change(screen.getByRole("textbox", { name: "Search people" }), {
      target: { value: "kiri" },
    });
    expect(hoisted.useAdminUsers).not.toHaveBeenCalledWith(
      true,
      expect.objectContaining({ q: "kiri" }),
    );

    act(() => {
      vi.advanceTimersByTime(299);
    });
    expect(hoisted.useAdminUsers).not.toHaveBeenCalledWith(
      true,
      expect.objectContaining({ q: "kiri" }),
    );

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(hoisted.useAdminUsers).toHaveBeenLastCalledWith(true, {
      q: "kiri",
      role: "",
      sort: "name",
    });
  });

  it("says nothing matches when a query or role finds nobody", () => {
    vi.useFakeTimers();
    hoisted.useAdminUsers.mockReturnValue(usersResult({ users: [], total: 0 }));
    render(<AdminPage />);
    fireEvent.change(screen.getByRole("textbox", { name: "Search people" }), {
      target: { value: "kiri" },
    });
    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    expect(screen.queryByText("No accounts yet.")).not.toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();

    fireEvent.change(screen.getByRole("textbox", { name: "Search people" }), {
      target: { value: "" },
    });
    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(screen.getByText("No accounts yet.")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Role" }));
    fireEvent.click(screen.getByRole("option", { name: "Member (4)" }));
    expect(screen.getByText("Nothing matches.")).toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    expect(hoisted.useAdminUsers).toHaveBeenLastCalledWith(true, {
      q: "",
      role: "member",
      sort: "name",
    });
  });

  it("says there are no accounts when everyone is empty", () => {
    hoisted.useAdminUsers.mockReturnValue(usersResult({ users: [], total: 0 }));
    render(<AdminPage />);
    expect(screen.getByText("No accounts yet.")).toBeInTheDocument();
    expect(screen.queryByText("Nothing matches.")).not.toBeInTheDocument();
    expect(screen.queryByRole("list")).not.toBeInTheDocument();
    expect(screen.getByText("0 matches")).toBeInTheDocument();
  });

  it("saves a staged role after that person leaves the loaded page", () => {
    const view = render(<AdminPage />);
    fireEvent.click(
      within(screen.getByRole("radiogroup", { name: "Role for @astra" })).getByRole("radio", {
        name: "Moderator",
      }),
    );
    hoisted.useAdminUsers.mockReturnValue(usersResult({ users: [people[1]], total: 1 }));
    view.rerender(<AdminPage />);
    expect(screen.queryByRole("radiogroup", { name: "Role for @astra" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(setUserRole).toHaveBeenCalledWith("astra", "moderator");
  });
});
