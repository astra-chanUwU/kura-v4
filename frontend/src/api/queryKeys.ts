export const queryKeys = {
  site: () => ["site"] as const,
  me: () => ["me"] as const,
  posts: () => ["posts"] as const,
  searchIndex: () => ["search-index"] as const,
  post: (id: string) => ["post", id] as const,
  favorites: () => ["favorites"] as const,
  myPosts: () => ["my-posts"] as const,
  myLists: () => ["my-lists"] as const,
  list: (id: string) => ["list", id] as const,
  profile: (username: string) => ["profile", username] as const,
};
