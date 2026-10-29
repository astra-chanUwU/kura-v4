export const queryKeys = {
  site: () => ["site"] as const,
  me: () => ["me"] as const,
  posts: () => ["posts"] as const,
  board: (filter: { tags: string[]; exclude: string[]; q: string; sort?: string }) =>
    [
      "board",
      filter.tags.join(","),
      filter.exclude.join(","),
      filter.q,
      filter.sort ?? "newest",
    ] as const,
  tagSuggestions: (q: string, filter: { tags: string[]; exclude: string[]; q: string }) =>
    ["tag-suggestions", q, filter.tags.join(","), filter.exclude.join(","), filter.q] as const,
  neighbors: (
    id: string,
    filter: { tags: string[]; exclude: string[]; q: string; sort?: string },
  ) =>
    [
      "neighbors",
      id,
      filter.tags.join(","),
      filter.exclude.join(","),
      filter.q,
      filter.sort ?? "newest",
    ] as const,
  post: (id: string) => ["post", id] as const,
  favorites: () => ["favorites"] as const,
  myPosts: () => ["my-posts"] as const,
  myLists: () => ["my-lists"] as const,
  list: (id: string) => ["list", id] as const,
  profile: (username: string) => ["profile", username] as const,
  adminUsers: () => ["admin-users"] as const,
  postHistory: (id: string) => ["post-history", id] as const,
  hiddenPosts: (kind: string) => ["hidden-posts", kind] as const,
  adminHistory: (actorId?: string) => ["admin-history", actorId ?? ""] as const,
};
