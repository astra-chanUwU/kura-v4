import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { pageRangeLabel } from "../domain/paging";
import { PROFILE_PAGE_SIZE } from "../domain/profileSearch";
import type { Post, Profile } from "../types/api";
import { ProfilePage } from "./ProfilePage";

const hoisted = vi.hoisted(() => ({
  navigate: vi.fn(),
  pages: [] as number[],
  profile: null as Profile | null,
}));

const user: Profile["user"] = {
  id: "u1",
  role: "member",
  username: "ada",
  display_name: "Ada",
  bio: "",
  location: "",
  website: "",
  created_at: "2026-01-02T00:00:00Z",
  avatar_url: "",
};

function makePost(id: string): Post {
  return {
    id,
    owner_username: "ada",
    content_type: "image/png",
    size_bytes: 1,
    created_at: "2026-01-01T00:00:00Z",
    is_favorite: false,
    favorite_count: 0,
    media_url: `/media/${id}.jpg`,
    thumbnail_url: `/media/${id}-thumb.jpg`,
    artist: "",
    rating: "sfw",
    hidden: false,
    deleted: false,
    width: 10,
    height: 10,
    source_url: "",
    tags: [],
  };
}

function makeProfile(overrides: Partial<Profile> = {}): Profile {
  return {
    user,
    public_lists: [],
    contributions: overrides.contributions ?? overrides.upload_total ?? 0,
    upload_total: overrides.upload_total ?? 0,
    edit_count: 0,
    uploads: overrides.uploads ?? [],
    theme: "kura",
    font: "system",
    ...overrides,
  };
}

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children?: ReactNode }) => <a href="/lists/1">{children}</a>,
  useNavigate: () => hoisted.navigate,
}));
vi.mock("../components/PostCard", () => ({
  PostCard: ({ post }: { post: Post }) => (
    <article data-testid={`card-${post.id}`}>{post.id}</article>
  ),
}));
vi.mock("../hooks/useCollections", () => ({
  useProfile: (_username: string, page = 1) => {
    hoisted.pages.push(page);
    return { data: hoisted.profile, isPending: false, error: null };
  },
}));

describe("ProfilePage uploads", () => {
  beforeEach(() => {
    hoisted.pages = [];
    hoisted.profile = null;
  });

  it("shows the first 24 cards and pages the rest without calling them hidden", () => {
    const uploads = Array.from({ length: PROFILE_PAGE_SIZE }, (_, index) => makePost(`p${index}`));
    hoisted.profile = makeProfile({ uploads, upload_total: 30, contributions: 30 });
    render(<ProfilePage username="ada" page={1} />);

    expect(screen.getAllByTestId(/^card-/)).toHaveLength(24);
    expect(screen.getByText(pageRangeLabel(1, 30, PROFILE_PAGE_SIZE))).toBeInTheDocument();
    expect(screen.queryByText(/hidden pictures/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/on the board/i)).not.toBeInTheDocument();
    expect(hoisted.pages.at(-1)).toBe(1);

    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(hoisted.navigate).toHaveBeenCalledWith({
      to: "/u/$username",
      params: { username: "ada" },
      search: { page: 2 },
    });
  });

  it("mentions hidden pictures only when some uploads are not visible", () => {
    const uploads = Array.from({ length: PROFILE_PAGE_SIZE }, (_, index) => makePost(`p${index}`));
    hoisted.profile = makeProfile({ uploads, upload_total: 30, contributions: 40 });
    render(<ProfilePage username="ada" page={1} />);

    expect(screen.getByText("Hidden pictures stay off this profile.")).toBeInTheDocument();
    expect(screen.queryByText(/on the board/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/later page|next page|another page/i)).not.toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Pagination" })).toBeInTheDocument();
  });

  it("mentions hidden pictures on a single page and hides the pager when everything visible fits", () => {
    const uploads = Array.from({ length: 10 }, (_, index) => makePost(`p${index}`));
    hoisted.profile = makeProfile({ uploads, upload_total: 10, contributions: 12 });
    render(<ProfilePage username="ada" page={1} />);

    expect(screen.getAllByTestId(/^card-/)).toHaveLength(10);
    expect(screen.getByText("Hidden pictures stay off this profile.")).toBeInTheDocument();
    expect(screen.queryByRole("navigation", { name: "Pagination" })).not.toBeInTheDocument();
  });

  it("asks for page 2 and does not call an empty slice an empty board", () => {
    hoisted.profile = makeProfile({ uploads: [], upload_total: 30, contributions: 30 });
    render(<ProfilePage username="ada" page={2} />);

    expect(hoisted.pages.at(-1)).toBe(2);
    expect(screen.queryByText("No pictures on the board yet.")).not.toBeInTheDocument();
    expect(screen.queryByText(/hidden pictures/i)).not.toBeInTheDocument();
  });
});
