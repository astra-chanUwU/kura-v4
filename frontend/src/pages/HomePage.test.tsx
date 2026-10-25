import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useSite } from "../hooks/useSite";
import { HomePage } from "./HomePage";

vi.mock("../hooks/useSite");
vi.mock("../hooks/useKura", () => ({
  useDebouncedValue: (value: unknown) => value,
  useMe: () => ({ me: undefined, user: undefined, loading: false, signedIn: false, error: null }),
  usePosts: () => ({
    posts: [],
    loading: false,
    error: null,
    retry: vi.fn(),
    hasMore: false,
    loadingMore: false,
    loadMore: vi.fn(),
  }),
  usePostSearch: () => ({
    active: false,
    searching: false,
    results: [],
    total: 0,
    shown: 0,
    suggestions: [],
  }),
  useInvalidate: () => vi.fn(),
  useMyLists: () => ({ data: [] }),
  useToggleFavorite: () => ({ mutate: vi.fn() }),
  usePostActions: () => ({ upload: vi.fn(), addToList: vi.fn() }),
}));
const mockUseSite = vi.mocked(useSite);
describe("image board states", () => {
  it("offers recovery when the API is unavailable", () => {
    const retry = vi.fn();
    mockUseSite.mockReturnValue({
      site: undefined,
      loading: false,
      error: new Error("offline"),
      retry,
    });
    render(<HomePage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Couldn’t connect to Kura.");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledOnce();
  });
  it("shows the empty board when the API responds", () => {
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByText("No images yet")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
