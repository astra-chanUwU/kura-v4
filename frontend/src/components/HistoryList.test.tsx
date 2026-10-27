import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ChangeEntry } from "../types/api";
import { HistoryList } from "./HistoryList";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, ...props }: { children?: ReactNode }) => <a {...props}>{children}</a>,
}));

const entry: ChangeEntry = {
  id: "h1",
  entity_type: "post",
  entity_id: "p1",
  actor_id: "u1",
  actor_username: "ana",
  action: "post.meta_updated",
  before: {
    artist: "ana",
    tags: ["keep_me", "old_tag"],
    rating: "sfw",
    hidden: false,
    deleted: false,
  },
  after: {
    artist: "bob",
    tags: ["keep_me", "new_tag"],
    rating: "sfw",
    hidden: false,
    deleted: false,
  },
  rollback_of: null,
  created_at: "2026-10-27T00:00:00Z",
};

describe("HistoryList diff highlighting", () => {
  it("renders removed values with a minus and added values with a plus", () => {
    render(<HistoryList entries={[entry]} />);
    const removed = screen.getByText("tags: old_tag");
    const added = screen.getByText("tags: new_tag");
    expect(removed.closest("li")).toHaveAttribute("data-kind", "removed");
    expect(removed.closest("li")?.textContent).toContain("-");
    expect(added.closest("li")).toHaveAttribute("data-kind", "added");
    expect(added.closest("li")?.textContent).toContain("+");
    expect(screen.getByText("artist: ana").closest("li")).toHaveAttribute("data-kind", "removed");
    expect(screen.getByText("artist: bob").closest("li")).toHaveAttribute("data-kind", "added");
    expect(screen.getByText("tags: keep_me").closest("li")).toHaveAttribute(
      "data-kind",
      "unchanged",
    );
    expect(screen.queryByText(/ana.*→.*bob/)).not.toBeInTheDocument();
  });
});
