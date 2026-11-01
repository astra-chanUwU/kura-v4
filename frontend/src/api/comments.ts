import type { Comment, CommentPage } from "../types/comments";
import { getJSON, sendJSON } from "./client";

export function listComments(postId: string, signal?: AbortSignal): Promise<CommentPage> {
  return getJSON<CommentPage>(`/posts/${encodeURIComponent(postId)}/comments?limit=100`, signal);
}

export function createComment(postId: string, body: string): Promise<Comment> {
  return sendJSON<Comment>(`/posts/${encodeURIComponent(postId)}/comments`, "POST", { body });
}

export function updateComment(id: string, body: string): Promise<Comment> {
  return sendJSON<Comment>(`/comments/${encodeURIComponent(id)}`, "PATCH", { body });
}

export function deleteComment(id: string): Promise<void> {
  return sendJSON<void>(`/comments/${encodeURIComponent(id)}`, "DELETE");
}
