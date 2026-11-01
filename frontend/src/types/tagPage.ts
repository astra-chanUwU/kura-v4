import type { Post } from "./api";

export type TagCategory = "general" | "artist" | "character" | "copyright" | "meta";

export const TAG_CATEGORIES: TagCategory[] = [
  "general",
  "artist",
  "character",
  "copyright",
  "meta",
];

export const TAG_CATEGORY_LABELS: Record<TagCategory, string> = {
  general: "General",
  artist: "Artist",
  character: "Character",
  copyright: "Copyright",
  meta: "Meta",
};

/** A tag wiki page. alias_of is set when this name points at another tag. */
export interface TagPage {
  tag: string;
  alias_of: string;
  aliases: string[];
  category: TagCategory;
  description: string;
  post_count: number;
  updated_by_username: string;
  updated_at: string;
  posts: Post[];
}
