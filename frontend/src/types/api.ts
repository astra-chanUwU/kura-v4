// Keep these wire contracts in sync with internal/dto. They are handwritten.
export interface Site {
  name: string;
  description: string;
}
export interface ApiErrorBody {
  code: string;
  message: string;
  request_id: string;
}
import type { Role } from "../domain/roles";
export type { Role };
import type { FontId, ThemeId } from "../domain/appearance";
export type { FontId, ThemeId };
export interface User {
  id: string;
  role: Role;
  username: string;
  display_name: string;
  bio: string;
  location: string;
  website: string;
  created_at: string;
  avatar_url: string;
}
/** One filtered page of accounts. Role counts are global; `total` is how many match the filter. */
export interface UserPage {
  total: number;
  member: number;
  moderator: number;
  admin: number;
  super_admin: number;
  users: User[];
}
export interface Post {
  id: string;
  owner_username: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
  is_favorite: boolean;
  favorite_count: number;
  media_url: string;
  thumbnail_url: string;
  sample_url?: string;
  artist: string;
  rating: "sfw" | "nsfw";
  hidden: boolean;
  deleted: boolean;
  width: number;
  height: number;
  source_url: string;
  tags: string[];
  /** Latest person who saved tags, artist, rating, or source. Empty when nobody has. */
  last_tagger_username?: string;
}
export interface SearchPage {
  total: number;
  posts: Post[];
}
export interface TagCount {
  tag: string;
  count: number;
}
export interface PostNeighbors {
  index: number;
  total: number;
  prev_id: string;
  next_id: string;
}
export interface ListSummary {
  id: string;
  owner_username: string;
  title: string;
  description: string;
  visibility: "private" | "public";
  item_count: number;
  created_at: string;
  updated_at: string;
}
export interface ListDetail {
  list: ListSummary;
  items: Post[];
  total: number;
}
export interface HiddenPage {
  total: number;
  hidden: number;
  deleted: number;
  posts: Post[];
}
export interface FailedJob {
  id: number;
  kind: string;
  attempts: number;
  max_attempts: number;
  last_error: string;
  updated_at: string;
}
export interface FailedJobPage {
  total: number;
  jobs: FailedJob[];
}
export interface Profile {
  user: User;
  public_lists: ListSummary[];
  contributions: number;
  /** Posts this visitor can see. Hidden and deleted posts are omitted. */
  upload_total: number;
  /** Saves of tags, artist, rating, or source. */
  edit_count: number;
  uploads: Post[];
  theme: ThemeId;
  font: FontId;
}
export interface PasskeyInfo {
  id: string;
  name: string;
  created_at: string;
  last_used_at: string | null;
}
export interface Security {
  has_password: boolean;
  passkeys: PasskeyInfo[];
  has_recovery_key: boolean;
  recovery_key_created_at: string | null;
}
export interface Preferences {
  show_nsfw: boolean;
  default_include_tags: string[];
  default_exclude_tags: string[];
  theme: ThemeId;
  font: FontId;
  /** Rows of on-screen tag suggestions. 0 hides the strip. */
  tag_suggestion_rows: number;
}
export interface Me {
  user: User;
  security: Security;
  preferences: Preferences;
  /** Set only in the response that issues a key; shown once, never stored. */
  recovery_key?: string;
}
export interface WebAuthnCeremony {
  ceremony_id: string;
  options: unknown;
}

/** A member's report of one post. Dismissing it does not hide the post. */
export interface PostReport {
  id: string;
  post_id: string;
  thumbnail_url: string;
  reporter_username: string;
  reason: string;
  note: string;
  status: string;
  created_at: string;
  resolved_at?: string;
  resolver_id?: string;
}
export interface PostReportPage {
  total: number;
  reports: PostReport[];
}

/** One append-only history row. before/after are post snapshots ({artist,tags,rating,hidden}) or {"role": "..."} . */
export interface ChangeEntry {
  id: string;
  entity_type: "post" | "user";
  entity_id: string;
  /** Set for post entries; empty for user entries. */
  thumbnail_url: string;
  actor_id: string;
  actor_username: string;
  action: string;
  before: Record<string, unknown>;
  after: Record<string, unknown>;
  rollback_of: string | null;
  created_at: string;
}

/** One filtered page of the staff change log. `total` counts every match, not just this page. */
export interface HistoryPage {
  total: number;
  entries: ChangeEntry[];
}

/** A person match for the staff history filter. The server returns at most eight. */
export interface HistoryActor {
  id: string;
  username: string;
}
