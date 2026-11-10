import type { PostReport, PostReportPage } from "../types/api";
import { getJSON, sendJSON } from "./client";

export function listOpenReports(signal?: AbortSignal): Promise<PostReportPage> {
  return getJSON<PostReportPage>("/admin/reports", signal);
}

export function createPostReport(
  postId: string,
  reason: string,
  note: string,
): Promise<PostReport> {
  return sendJSON<PostReport>(`/posts/${encodeURIComponent(postId)}/reports`, "POST", {
    reason,
    note,
  });
}

export function dismissReport(id: string): Promise<PostReport> {
  return sendJSON<PostReport>(`/admin/reports/${encodeURIComponent(id)}/dismiss`, "POST");
}
