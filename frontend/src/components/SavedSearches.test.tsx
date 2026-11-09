import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SavedSearch } from "../types/savedSearch";
import { SaveCurrentSearchButton, SavedSearchList } from "./SavedSearches";

const state = vi.hoisted(() => ({
  signedIn: true,
  loading: false,
  list: vi.fn(),
  create: vi.fn(),
  remove: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    search,
    children,
  }: {
    to: string;
    search?: { tags?: string; exclude?: string; q?: string; sort?: string };
    children?: ReactNode;
  }) => (
    <a
      href={to}
      data-tags={search?.tags ?? ""}
      data-exclude={search?.exclude ?? ""}
      data-q={search?.q ?? ""}
      data-sort={search?.sort ?? ""}
    >
      {children}
    </a>
  ),
}));

vi.mock("../hooks/useKura", () => ({
  useMe: () => ({ signedIn: state.signedIn, loading: state.loading }),
}));

vi.mock("../api/savedSearches", () => ({
  listSavedSearches: (...args: unknown[]) => state.list(...args),
  createSavedSearch: (...args: unknown[]) => state.create(...args),
  deleteSavedSearch: (...args: unknown[]) => state.remove(...args),
}));

function renderUI(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

const saved: SavedSearch[] = [
  {
    id: "s1",
    name: "Soft",
    include_tags: ["white_hair", "smile"],
    exclude_tags: ["comic"],
    q: "yuri",
    sort: "oldest",
    created_at: "2026-11-01T00:00:00Z",
  },
  {
    id: "s2",
    name: "New",
    include_tags: ["solo"],
    exclude_tags: [],
    q: "",
    sort: "newest",
    created_at: "2026-11-01T01:00:00Z",
  },
  {
    id: "s3",
    name: "Loved",
    include_tags: ["smile"],
    exclude_tags: [],
    q: "",
    sort: "favorites",
    created_at: "2026-11-01T02:00:00Z",
  },
];

beforeEach(() => {
  state.signedIn = true;
  state.loading = false;
  state.list.mockReset();
  state.create.mockReset();
  state.remove.mockReset();
});

describe("SaveCurrentSearchButton", () => {
  it("renders nothing when the filter is empty", () => {
    const { container } = renderUI(
      <SaveCurrentSearchButton filter={{ tags: [], exclude: [], q: "  " }} />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("renders nothing when signed out", () => {
    state.signedIn = false;
    const { container } = renderUI(
      <SaveCurrentSearchButton filter={{ tags: ["white_hair"], exclude: [], q: "" }} />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("saves the current filter under the typed name", async () => {
    const created = saved[0];
    if (!created) throw new Error("missing fixture");
    state.create.mockResolvedValue(created);
    renderUI(
      <SaveCurrentSearchButton
        filter={{ tags: ["white_hair"], exclude: ["comic"], q: "yuri", sort: "oldest" }}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Save search" }));
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Night" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => {
      expect(state.create).toHaveBeenCalledWith({
        name: "Night",
        include_tags: ["white_hair"],
        exclude_tags: ["comic"],
        q: "yuri",
        sort: "oldest",
      });
    });
    await waitFor(() => {
      expect(screen.queryByLabelText("Name")).not.toBeInTheDocument();
    });
  });
});

describe("SavedSearchList", () => {
  it("asks visitors to sign in", async () => {
    state.signedIn = false;
    renderUI(<SavedSearchList />);
    expect(screen.getByText("Sign in to save searches.")).toBeInTheDocument();
    await waitFor(() => expect(state.list).not.toHaveBeenCalled());
  });

  it("shows the empty copy when the account has none", async () => {
    state.list.mockResolvedValue([]);
    renderUI(<SavedSearchList />);
    expect(await screen.findByText("No saved searches yet.")).toBeInTheDocument();
  });

  it("links each saved search to the board filter", async () => {
    state.list.mockResolvedValue(saved);
    renderUI(<SavedSearchList />);
    const oldest = await screen.findByRole("link", { name: /Soft/ });
    expect(oldest).toHaveAttribute("href", "/");
    expect(oldest).toHaveAttribute("data-tags", "white_hair,smile");
    expect(oldest).toHaveAttribute("data-exclude", "comic");
    expect(oldest).toHaveAttribute("data-q", "yuri");
    expect(oldest).toHaveAttribute("data-sort", "oldest");

    const newest = screen.getByRole("link", { name: /New/ });
    expect(newest).toHaveAttribute("data-tags", "solo");
    expect(newest).toHaveAttribute("data-exclude", "");
    expect(newest).toHaveAttribute("data-q", "");
    expect(newest).toHaveAttribute("data-sort", "");

    const favorites = screen.getByRole("link", { name: /Loved/ });
    expect(favorites).toHaveAttribute("data-sort", "favorites");
    expect(favorites).toHaveTextContent("Favorites");
  });
});
