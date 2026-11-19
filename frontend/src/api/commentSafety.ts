import type { CommentSafetyEvent, CommentSafetyPage, CommentSafetyQuery, CommentSafetyResolvers } from "../types/commentSafety";
import { getJSON, sendJSON } from "./client";

export function listCommentSafety(
  query: CommentSafetyQuery,
  signal?: AbortSignal,
): Promise<CommentSafetyPage> {
  const params = new URLSearchParams();
  const entries: Array<[string, string | number]> = [
    ["status", query.status],
    ["action", query.action],
    ["category", query.category],
    ["q", query.q],
    ["resolver", query.resolver],
    ["sort", query.sort],
    ["limit", query.limit],
    ["offset", query.offset],
  ];
  for (const [key, value] of entries) {
    if (value !== "") params.set(key, String(value));
  }
  const search = params.toString();
  return getJSON<CommentSafetyPage>(`/admin/comment-safety${search ? `?${search}` : ""}`, signal);
}

export function dismissCommentSafety(id: string): Promise<CommentSafetyEvent> {
  return sendJSON<CommentSafetyEvent>(
    `/admin/comment-safety/${encodeURIComponent(id)}/dismiss`,
    "POST",
  );
}

export function removeCommentSafety(id: string): Promise<CommentSafetyEvent> {
  return sendJSON<CommentSafetyEvent>(
    `/admin/comment-safety/${encodeURIComponent(id)}/remove`,
    "POST",
  );
}

export function reopenCommentSafety(id: string): Promise<CommentSafetyEvent> {
  return sendJSON<CommentSafetyEvent>(
    `/admin/comment-safety/${encodeURIComponent(id)}/reopen`,
    "POST",
  );
}

export function listCommentSafetyResolvers(signal?: AbortSignal): Promise<CommentSafetyResolvers> {
  return getJSON<CommentSafetyResolvers>("/admin/comment-safety/resolvers", signal);
}
