import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Role } from "../domain/roles";
import { StaffLayout } from "./StaffLayout";

const hoisted = vi.hoisted(() => ({
  role: null as Role | null,
  hiddenCount: 0,
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    children,
    "aria-label": label,
  }: {
    to: string;
    children?: React.ReactNode;
    "aria-label"?: string;
  }) => (
    <a href={to} aria-label={label}>
      {children}
    </a>
  ),
  Outlet: () => <div data-testid="staff-outlet" />,
}));
vi.mock("../hooks/useKura", () => ({
  useMe: () => {
    const user = hoisted.role
      ? { id: "u1", username: "astra", display_name: "", role: hoisted.role }
      : null;
    return { me: user ? { user } : null, user, loading: false };
  },
  useHiddenPosts: () => ({ data: Array.from({ length: hoisted.hiddenCount }, (_, i) => i) }),
}));

describe("StaffLayout", () => {
  beforeEach(() => {
    hoisted.role = null;
    hoisted.hiddenCount = 0;
  });

  it("asks signed-out visitors to sign in instead of rendering staff tools", () => {
    render(<StaffLayout />);
    expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute("href", "/login");
    expect(screen.queryByTestId("staff-outlet")).not.toBeInTheDocument();
  });

  it("keeps members out", () => {
    hoisted.role = "member";
    render(<StaffLayout />);
    expect(screen.getByText(/Moderators and above can use this page/)).toBeInTheDocument();
    expect(screen.queryByTestId("staff-outlet")).not.toBeInTheDocument();
  });

  it("gives moderators the hidden queue and history, without People", () => {
    hoisted.role = "moderator";
    hoisted.hiddenCount = 3;
    render(<StaffLayout />);
    const tabs = within(screen.getByRole("navigation", { name: "Staff sections" }));
    expect(tabs.queryByRole("link", { name: /People/ })).not.toBeInTheDocument();
    expect(tabs.getByRole("link", { name: "Hidden, 3 off the board" })).toHaveAttribute(
      "href",
      "/admin/hidden",
    );
    expect(tabs.getByRole("link", { name: /History/ })).toHaveAttribute("href", "/admin/history");
    expect(screen.getByTestId("staff-outlet")).toBeInTheDocument();
  });

  it("adds the People tab for admins and hides an empty count", () => {
    hoisted.role = "admin";
    render(<StaffLayout />);
    const tabs = within(screen.getByRole("navigation", { name: "Staff sections" }));
    expect(tabs.getByRole("link", { name: /People/ })).toHaveAttribute("href", "/admin");
    expect(tabs.getByRole("link", { name: "Hidden" })).toBeInTheDocument();
  });
});
