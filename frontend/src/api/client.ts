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

export async function getJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    credentials: "same-origin",
    headers: { Accept: "application/json" },
    signal,
  });
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
  return response.json() as Promise<T>;
}
