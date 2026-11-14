export const queryKeys = {
  site: () => ["site"] as const,
  me: () => ["me"] as const,
  board: (
    filter: { tags: string[]; exclude: string[]; q: string; sort?: string },
    offset = 0,
    pageSize = 40,
  ) =>
    [
      "board",
      filter.tags.join(","),
      filter.exclude.join(","),
      filter.q,
      filter.sort ?? "newest",
      offset,
      pageSize,
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
  comments: (postId: string) => ["comments", postId] as const,
  savedSearches: () => ["saved-searches"] as const,
  tagPage: (name: string) => ["tag-page", name] as const,
  tagPages: () => ["tag-page"] as const,
  related: (id: string) => ["post-related", id] as const,
  favorites: (offset?: number, pageSize?: number) =>
    offset === undefined
      ? (["favorites"] as const)
      : (["favorites", offset, pageSize ?? 40] as const),
  myPosts: (offset?: number, pageSize?: number) =>
    offset === undefined
      ? (["my-posts"] as const)
      : (["my-posts", offset, pageSize ?? 40] as const),
  myLists: () => ["my-lists"] as const,
  list: (id: string, offset?: number, pageSize?: number) =>
    offset === undefined
      ? (["list", id] as const)
      : (["list", id, offset, pageSize ?? 40] as const),
  profile: (username: string) => ["profile", username] as const,
  adminUsers: (q: string, role: string, sort: string, offset = 0, pageSize = 40) =>
    ["admin-users", q, role, sort, offset, pageSize] as const,
  postHistory: (id: string) => ["post-history", id] as const,
  hiddenPosts: (kind: string, q: string, rating: string, sort: string, offset = 0, pageSize = 40) =>
    ["hidden-posts", kind, q, rating, sort, offset, pageSize] as const,
  adminHistory: (
    actorId: string,
    q: string,
    action: string,
    entity: string,
    sort: string,
    offset = 0,
    pageSize = 40,
  ) => ["admin-history", actorId, q, action, entity, sort, offset, pageSize] as const,
  historyActors: (q: string) => ["history-actors", q] as const,
  openReports: () => ["open-reports"] as const,
  failedJobs: (offset?: number) =>
    offset === undefined ? (["failed-jobs"] as const) : (["failed-jobs", offset] as const),
};
