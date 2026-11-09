export interface SavedSearch {
  id: string;
  name: string;
  include_tags: string[];
  exclude_tags: string[];
  q: string;
  sort: "newest" | "oldest" | "favorites";
  created_at: string;
}

export interface CreateSavedSearch {
  name: string;
  include_tags: string[];
  exclude_tags: string[];
  q: string;
  sort: "newest" | "oldest" | "favorites";
}
