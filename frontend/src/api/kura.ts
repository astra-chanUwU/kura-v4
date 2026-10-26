import type { ListDetail, ListSummary, Me, Post, Profile, WebAuthnCeremony } from "../types/api";
import { getJSON, sendJSON, sendRaw } from "./client";
import { createPasskey, getPasskey } from "./webauthn";

/** The uploaded bytes already exist as this post (HTTP 409). */
export class DuplicatePostError extends Error {
  readonly post: Post;
  constructor(post: Post) {
    super("Already on the board.");
    this.name = "DuplicatePostError";
    this.post = post;
  }
}

export const api = {
  me: (signal?: AbortSignal) => getJSON<Me>("/me", signal),
  registerPassword: (username: string, password: string, display_name: string) =>
    sendJSON<Me>("/auth/register", "POST", { username, password, display_name }),
  loginPassword: (username: string, password: string) =>
    sendJSON<Me>("/auth/login", "POST", { username, password }),
  logout: () => sendJSON<void>("/auth/logout", "POST"),
  async registerPasskey(username: string, display_name: string): Promise<Me> {
    const ceremony = await sendJSON<WebAuthnCeremony>("/auth/passkey/register/begin", "POST", {
      username,
      display_name,
    });
    const credential = await createPasskey(ceremony.options);
    return sendRaw<Me>(
      `/auth/passkey/register/finish?ceremony=${ceremony.ceremony_id}`,
      credential,
    );
  },
  async addPasskey(): Promise<Me> {
    const ceremony = await sendJSON<WebAuthnCeremony>("/auth/passkey/register/begin", "POST", {});
    const credential = await createPasskey(ceremony.options);
    return sendRaw<Me>(
      `/auth/passkey/register/finish?ceremony=${ceremony.ceremony_id}`,
      credential,
    );
  },
  async loginPasskey(): Promise<Me> {
    // No username: the passkey itself identifies the account
    // (discoverable credential — the manager lists saved passkeys).
    const ceremony = await sendJSON<WebAuthnCeremony>("/auth/passkey/login/begin", "POST", {});
    const credential = await getPasskey(ceremony.options);
    return sendRaw<Me>(`/auth/passkey/login/finish?ceremony=${ceremony.ceremony_id}`, credential);
  },
  async setAvatar(file: File): Promise<Me> {
    const form = new FormData();
    form.append("file", file);
    const response = await fetch("/api/v1/me/avatar", {
      method: "POST",
      credentials: "same-origin",
      body: form,
    });
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as { message?: string } | null;
      throw new Error(body?.message ?? "Could not save that picture.");
    }
    return response.json() as Promise<Me>;
  },
  removeAvatar: () => sendJSON<Me>("/me/avatar", "DELETE"),
  updateProfile: (body: { display_name: string; bio: string; location: string; website: string }) =>
    sendJSON<Me>("/me/profile", "PATCH", body),
  regenerateRecoveryKey: () => sendJSON<Me>("/me/recovery-key", "POST"),
  recoverAccount: (username: string, recovery_key: string, new_password: string) =>
    sendJSON<Me>("/auth/recover", "POST", { username, recovery_key, new_password }),
  updatePostMeta: (postId: string, artist: string, tags: string[], rating: string) =>
    sendJSON<Post>(`/posts/${postId}/meta`, "PATCH", { artist, tags, rating }),
  setPreferences: (show_nsfw: boolean, over_18: boolean) =>
    sendJSON<Me>("/me/preferences", "PATCH", { show_nsfw, over_18 }),
  setTagPreferences: (include_tags: string[], exclude_tags: string[]) =>
    sendJSON<Me>("/me/tag-preferences", "PATCH", { include_tags, exclude_tags }),
  setPassword: (password: string) => sendJSON<Me>("/me/password", "POST", { password }),
  removePassword: () => sendJSON<Me>("/me/password", "DELETE"),
  removePasskey: (id: string) => sendJSON<Me>(`/me/passkeys/${id}`, "DELETE"),
  profile: (username: string, signal?: AbortSignal) =>
    getJSON<Profile>(`/users/${username}`, signal),
  searchIndex: (signal?: AbortSignal) => getJSON<Post[]>("/search-index", signal),
  posts: (offset: number, signal?: AbortSignal) =>
    getJSON<Post[]>(`/posts?limit=40&offset=${offset}`, signal),
  post: (id: string, signal?: AbortSignal) => getJSON<Post>(`/posts/${id}`, signal),
  favorites: (signal?: AbortSignal) => getJSON<Post[]>("/me/favorites", signal),
  myPosts: (signal?: AbortSignal) => getJSON<Post[]>("/me/posts", signal),
  favorite: (postId: string) => sendJSON<void>(`/me/favorites/${postId}`, "POST"),
  unfavorite: (postId: string) => sendJSON<void>(`/me/favorites/${postId}`, "DELETE"),
  myLists: (signal?: AbortSignal) => getJSON<ListSummary[]>("/me/lists", signal),
  list: (id: string, signal?: AbortSignal) => getJSON<ListDetail>(`/lists/${id}`, signal),
  createList: (body: { title: string; description: string; visibility: string }) =>
    sendJSON<ListSummary>("/lists", "POST", body),
  updateList: (id: string, body: { title: string; description: string; visibility: string }) =>
    sendJSON<ListSummary>(`/lists/${id}`, "PATCH", body),
  deleteList: (id: string) => sendJSON<void>(`/lists/${id}`, "DELETE"),
  addToList: (listId: string, postId: string) =>
    sendJSON<void>(`/lists/${listId}/items`, "POST", { post_id: postId }),
  removeFromList: (listId: string, postId: string) =>
    sendJSON<void>(`/lists/${listId}/items/${postId}`, "DELETE"),
  reorderList: (listId: string, post_ids: string[]) =>
    sendJSON<void>(`/lists/${listId}/reorder`, "PUT", { post_ids }),
  async upload(file: File, tags = "", rating = "sfw"): Promise<Post> {
    const form = new FormData();
    form.append("file", file);
    if (tags.trim()) form.append("tags", tags);
    form.append("rating", rating);
    const response = await fetch("/api/v1/posts", {
      method: "POST",
      credentials: "same-origin",
      body: form,
    });
    if (!response.ok) {
      const body = (await response.json().catch(() => null)) as {
        message?: string;
        post?: Post;
      } | null;
      if (response.status === 409 && body?.post) {
        throw new DuplicatePostError(body.post);
      }
      throw new Error(body?.message ?? "Upload failed.");
    }
    return response.json() as Promise<Post>;
  },
};
