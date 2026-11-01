import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createComment, listComments } from "../api/comments";
import type { Comment } from "../types/comments";
import { PostComments } from "./PostComments";

vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    Link: ({
      to,
      params,
      children,
      ...props
    }: {
      to: string;
      params?: Record<string, string>;
      children?: React.ReactNode;
    }) => {
      let href = to;
      for (const [k, v] of Object.entries(params ?? {})) href = href.replace(`$${k}`, v);
      return React.createElement("a", { href, ...props }, children);
    },
  };
});

vi.mock("../api/comments", () => ({
  listComments: vi.fn(),
  createComment: vi.fn(),
  updateComment: vi.fn(),
  deleteComment: vi.fn(),
}));

vi.mock("../hooks/useKura", () => ({
  useMe: () => ({ signedIn: true, user: { id: "u1", username: "ada" } }),
}));

const comment: Comment = {
  id: "c1",
  post_id: "p1",
  author_username: "ada",
  author_avatar_url: "",
  body: "Hello there",
  created_at: "2026-11-01T12:00:00Z",
  updated_at: "2026-11-01T12:00:00Z",
  can_edit: true,
  can_delete: true,
};

function renderComments() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <PostComments postId="p1" />
    </QueryClientProvider>,
  );
}

describe("PostComments", () => {
  it("shows an empty state", async () => {
    vi.mocked(listComments).mockResolvedValue({ total: 0, comments: [] });
    renderComments();
    expect(await screen.findByText("No comments yet.")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Comments" })).toBeInTheDocument();
  });

  it("renders a comment", async () => {
    vi.mocked(listComments).mockResolvedValue({ total: 1, comments: [comment] });
    renderComments();
    expect(await screen.findByText("Hello there")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "ada" })).toHaveAttribute("href", "/u/ada");
  });

  it("posts a comment", async () => {
    vi.mocked(listComments).mockResolvedValue({ total: 0, comments: [] });
    vi.mocked(createComment).mockResolvedValue(comment);
    renderComments();
    expect(await screen.findByRole("button", { name: "Post" })).toBeDisabled();
    fireEvent.change(screen.getByRole("textbox", { name: "Write a comment" }), {
      target: { value: "Nice shot" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Post" }));
    await waitFor(() => expect(createComment).toHaveBeenCalledWith("p1", "Nice shot"));
  });
});
