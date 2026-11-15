/** Public profile uploads page. Page 1 is omitted from the URL. */
export interface ProfileSearch {
  page?: number;
}

/** Server page size for a public profile's uploads. Matches the profile handler. */
export const PROFILE_PAGE_SIZE = 24;

/** Positive integers only; strings must be whole digits (no floats). */
function parseProfilePage(value: unknown): number | undefined {
  if (typeof value === "number") {
    if (!Number.isInteger(value) || value < 1) return undefined;
    return value;
  }
  if (typeof value === "string") {
    if (!/^[1-9]\d*$/.test(value)) return undefined;
    return Number(value);
  }
  return undefined;
}

export function validateProfileSearch(search: Record<string, unknown>): ProfileSearch {
  const page = parseProfilePage(search.page);
  if (page === undefined || page === 1) return {};
  return { page };
}

/** 1-based profile uploads page; missing means page 1. */
export function profilePageFromSearch(search: ProfileSearch): number {
  return search.page ?? 1;
}
