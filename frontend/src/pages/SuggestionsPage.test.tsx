import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SuggestionStatus, TagSuggestion } from "../types/suggestion";
import { SuggestionsPage } from "./SuggestionsPage";

const hoisted = vi.hoisted(() => ({
  useTagSuggestions: vi.fn(),
  useSuggestionActions: vi.fn(),
}));

function item(overrides: Partial<TagSuggestion> = {}): TagSuggestion {
  return {
    id: "s1",
    kind: "alias",
    source_tag: "silverhair",
    target_tag: "silver_hair",
    reason: "Same hair.",
    status: "pending",
    author_username: "hana",
    created_at: "2026-11-12T00:00:00Z",
    ...overrides,
  };
}

function queryResult(overrides: Record<string, unknown> = {}) {
  return { data: [], isPending: false, error: null, ...overrides };
}

vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    Link: ({
      to,
      params,
      children,
    }: {
      to: string;
      params?: Record<string, string>;
      children?: React.ReactNode;
    }) => {
      let href = to;
      for (const [key, value] of Object.entries(params ?? {}))
        href = href.replace(`$${key}`, value);
      return React.createElement("a", { href }, children);
    },
  };
});

vi.mock("../hooks/useSuggestions", () => ({
  useTagSuggestions: (status: SuggestionStatus) => hoisted.useTagSuggestions(status),
  useSuggestionActions: () => hoisted.useSuggestionActions(),
}));

describe("SuggestionsPage", () => {
  const accept = vi.fn();
  const reject = vi.fn();

  beforeEach(() => {
    accept.mockReset();
    reject.mockReset();
    hoisted.useTagSuggestions.mockReset();
    hoisted.useSuggestionActions.mockReset();
    hoisted.useTagSuggestions.mockImplementation((status: SuggestionStatus) => {
      if (status === "pending") return queryResult({ data: [item()] });
      if (status === "accepted") {
        return queryResult({
          data: [
            item({
              id: "s-accepted",
              status: "accepted",
              resolver_username: "kiri",
              resolved_at: "2026-11-12T01:00:00Z",
            }),
          ],
        });
      }
      return queryResult({
        data: [
          item({
            id: "s-rejected",
            kind: "implication",
            source_tag: "smile",
            target_tag: "open_mouth",
            status: "rejected",
            resolver_username: "kiri",
            resolved_at: "2026-11-12T02:00:00Z",
          }),
        ],
      });
    });
    hoisted.useSuggestionActions.mockReturnValue({
      accept: { mutate: accept, isPending: false },
      reject: { mutate: reject, isPending: false },
    });
  });

  it("keeps decided suggestions in the history and can change them", () => {
    render(<SuggestionsPage />);
    expect(screen.getByRole("heading", { name: "Pending" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "History" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Accepted" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Rejected" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Accept" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(screen.getByText(/accepted by @kiri/)).toBeInTheDocument();
    expect(screen.getByText(/rejected by @kiri/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Accept" }));
    expect(accept).toHaveBeenCalledWith("s1", expect.any(Object));

    fireEvent.click(screen.getByRole("button", { name: "Reject instead" }));
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveTextContent("stops pointing at silver_hair");
    fireEvent.click(within(dialog).getByRole("button", { name: "Reject instead" }));
    expect(reject).toHaveBeenCalledWith("s-accepted", expect.any(Object));

    fireEvent.click(screen.getByRole("button", { name: "Accept instead" }));
    expect(accept).toHaveBeenCalledWith("s-rejected", expect.any(Object));
  });

  it("does not change an accepted suggestion when the confirm is cancelled", () => {
    render(<SuggestionsPage />);
    fireEvent.click(screen.getByRole("button", { name: "Reject instead" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(reject).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows an empty history without hiding a failed pending queue", () => {
    hoisted.useTagSuggestions.mockImplementation((status: SuggestionStatus) => {
      if (status === "pending") return queryResult({ error: new Error("Something went wrong.") });
      return queryResult();
    });
    render(<SuggestionsPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Something went wrong.");
    expect(screen.queryByText("No pending suggestions.")).not.toBeInTheDocument();
    expect(screen.getByText("No accepted suggestions.")).toBeInTheDocument();
    expect(screen.getByText("No rejected suggestions.")).toBeInTheDocument();
  });
});
