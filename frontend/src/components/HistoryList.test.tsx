import { fireEvent, render, screen } from "@testing-library/react";
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
  thumbnail_url: "/media/posts/p1/thumb",
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

describe("HistoryList post links", () => {
  it("opens the post in a new tab and only shows the picture when asked", () => {
    const { container, rerender } = render(<HistoryList entries={[entry]} />);
    expect(screen.getByLabelText(/in a new tab/)).toHaveAttribute("target", "_blank");
    expect(container.querySelector("img")).toBeNull();

    rerender(<HistoryList entries={[entry]} showThumbnails />);
    expect(container.querySelector("img")).toHaveAttribute("src", "/media/posts/p1/thumb");
  });

  it("labels a withheld NSFW thumbnail instead of showing a broken image", () => {
    const nsfw: ChangeEntry = { ...entry, after: { ...entry.after, rating: "nsfw" } };
    const { container } = render(<HistoryList entries={[nsfw]} showThumbnails />);
    fireEvent.error(container.querySelector("img")!);
    expect(container.querySelector("img")).toBeNull();
    expect(screen.getByText("NSFW")).toBeInTheDocument();
  });

  it("shows no picture for user role changes", () => {
    const roleChange: ChangeEntry = {
      ...entry,
      id: "h2",
      entity_type: "user",
      entity_id: "u2",
      thumbnail_url: "",
      action: "user.role_changed",
      before: { role: "member" },
      after: { role: "moderator" },
    };
    const { container } = render(<HistoryList entries={[roleChange]} showThumbnails />);
    expect(container.querySelector("img")).toBeNull();
  });
});
