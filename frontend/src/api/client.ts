import type { ApiErrorBody } from "../types/api";

export class ApiError extends Error {
  constructor(
    public status: number,
    public details: ApiErrorBody,
  ) {
    super(details.message);
    this.name = "ApiError";
  }
}

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

export async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
    signal,
  });
  return handle<T>(response);
}

export async function sendJSON<T>(path: string, method: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    method,
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  return handle<T>(response);
}

export async function sendRaw<T>(path: string, body: string): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body,
  });
  return handle<T>(response);
}
