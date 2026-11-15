import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ListSummary, Me } from "../types/api";
import { MePage } from "./MePage";

const lists = vi.hoisted(() => ({
  current: {
    data: [] as ListSummary[],
    isPending: false,
    isError: false,
    error: null as Error | null,
  },
}));

const me: Me = {
  user: {
    id: "u1",
    role: "member",
    username: "ana",
    display_name: "Ana",
    bio: "",
    location: "",
    website: "",
    created_at: "2026-01-01T00:00:00Z",
    avatar_url: "",
  },
  security: {
    has_password: true,
    passkeys: [],
    has_recovery_key: false,
    recovery_key_created_at: null,
  },
  preferences: {
    show_nsfw: false,
    default_include_tags: [],
    default_exclude_tags: [],
    share_default_tags: false,
    theme: "kura",
    font: "system",
    tag_suggestion_rows: 8,
    posts_per_page: 40,
  },
};

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, to }: { children?: ReactNode; to: string }) => (
    <a href={to}>{children}</a>
  ),
  useNavigate: () => vi.fn(),
}));
vi.mock("../components/Toasts", () => ({
  useToasts: () => ({ push: vi.fn() }),
  shortError: (error: unknown) => (error instanceof Error ? error.message : "error"),
}));
vi.mock("../hooks/useSession", () => ({
  useMe: () => ({ me, user: me.user, loading: false, signedIn: true }),
  useAccountActions: () => ({
    removePassword: vi.fn(),
    removePasskey: vi.fn(),
  }),
  useAuthActions: () => ({ logout: vi.fn(() => Promise.resolve()) }),
}));
vi.mock("../hooks/useCollections", () => ({
  useFavorites: () => ({
    posts: [],
    total: 0,
    isPending: false,
    isSuccess: false,
    prefetchPage: vi.fn(),
  }),
  useMyPosts: () => ({
    posts: [],
    total: 0,
    isPending: false,
    isSuccess: false,
    prefetchPage: vi.fn(),
  }),
  useMyLists: () => lists.current,
  useListActions: () => ({ createList: vi.fn() }),
  useInvalidate: () => vi.fn(),
}));

function list(id: string, title: string): ListSummary {
  return {
    id,
    owner_username: "ana",
    title,
    description: "",
    visibility: "private",
    item_count: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  };
}

describe("Me lists empty states", () => {
  beforeEach(() => {
    localStorage.setItem("kura-me-tab", "lists");
    localStorage.setItem("kura-passkey-tip-dismissed", "1");
    lists.current = { data: [], isPending: false, isError: false, error: null };
  });

  it("says there are no lists", () => {
    render(<MePage />);
    expect(screen.getByText("No lists yet.")).toBeInTheDocument();
  });

  it("does not call a loading list empty", () => {
    lists.current = { data: [], isPending: true, isError: false, error: null };
    render(<MePage />);
    expect(screen.getByText("Loading…")).toBeInTheDocument();
    expect(screen.queryByText("No lists yet.")).not.toBeInTheDocument();
  });

  it("says when the filter matches nothing", () => {
    lists.current = {
      data: Array.from({ length: 6 }, (_, index) => list(String(index), `List ${index}`)),
      isPending: false,
      isError: false,
      error: null,
    };
    render(<MePage />);
    fireEvent.change(screen.getByRole("textbox", { name: "Filter your lists" }), {
      target: { value: "zzzz" },
    });
    expect(screen.getByText("No lists match “zzzz”.")).toBeInTheDocument();
  });
});
