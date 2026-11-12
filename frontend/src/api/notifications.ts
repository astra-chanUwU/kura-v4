import type { ApiErrorBody } from "../types/api";
import type { NotificationPage } from "../types/notification";
import { ApiError } from "./client";

export const notificationPageSize = 40;

async function handle<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const body: unknown = await response.json().catch(() => null);
    const details = body as Partial<ApiErrorBody> | null;
    throw new ApiError(response.status, {
      code: typeof details?.code === "string" ? details.code : "request_failed",
      message:
        typeof details?.message === "string"
          ? details.message
          : "Could not reach Kura. Please try again.",
      request_id: typeof details?.request_id === "string" ? details.request_id : "",
    });
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    credentials: "include",
    headers: { Accept: "application/json" },
    signal,
  });
  return handle<T>(response);
}

async function sendJSON<T>(path: string, method: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    method,
    credentials: "include",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  return handle<T>(response);
}

export function listNotifications(
  offset: number,
  signal?: AbortSignal,
  limit = notificationPageSize,
): Promise<NotificationPage> {
  const params = new URLSearchParams();
  params.set("limit", String(limit));
  params.set("offset", String(offset));
  return getJSON<NotificationPage>(`/notifications?${params.toString()}`, signal);
}

/** Empty ids marks every unread notice for the signed-in account. */
export function markNotificationsRead(ids: string[]): Promise<void> {
  return sendJSON<void>("/notifications/read", "POST", { ids });
}
