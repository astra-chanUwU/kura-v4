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
export interface User {
  id: string;
  username: string;
  display_name: string;
  bio: string;
  location: string;
  website: string;
  created_at: string;
  avatar_url: string;
}
export interface Post {
  id: string;
  owner_username: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
  is_favorite: boolean;
  media_url: string;
  thumbnail_url: string;
  artist: string;
  rating: "sfw" | "nsfw";
  width: number;
  height: number;
  source_url: string;
  tags: string[];
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
}
export interface Profile {
  user: User;
  public_lists: ListSummary[];
  contributions: number;
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
