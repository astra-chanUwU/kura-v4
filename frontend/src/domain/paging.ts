import type { Me } from "../types/api";

/** Server page size for every numbered list. */
export const DEFAULT_PAGE_SIZE = 40;

/** @deprecated Prefer DEFAULT_PAGE_SIZE. */
export const PAGE_SIZE = DEFAULT_PAGE_SIZE;

const LOCAL_POSTS_PER_PAGE_KEY = "kura-posts-per-page";

/** Treat non-finite or non-positive sizes as the default. */
function coercePageSize(pageSize: number): number {
  if (!Number.isFinite(pageSize) || pageSize < 1) return DEFAULT_PAGE_SIZE;
  return Math.floor(pageSize);
}

/** Keep posts-per-page inside the server range; anything else becomes the default. */
export function clampPostsPerPage(n: number): number {
  if (!Number.isFinite(n)) return DEFAULT_PAGE_SIZE;
  const v = Math.floor(n);
  if (v < 12 || v > 100) return DEFAULT_PAGE_SIZE;
  return v;
}

/** Effective page size: account preference, else localStorage, else default. */
export function resolvePageSize(me: Me | null | undefined): number {
  const fromMe = me?.preferences?.posts_per_page;
  if (fromMe != null) return clampPostsPerPage(fromMe);
  try {
    const raw = localStorage.getItem(LOCAL_POSTS_PER_PAGE_KEY);
    if (raw != null && raw !== "") {
      const n = Number(raw);
      if (Number.isFinite(n)) return clampPostsPerPage(n);
    }
  } catch {
    // Private mode: fall through to the default.
  }
  return DEFAULT_PAGE_SIZE;
}

/** Remember a page size for signed-out browsing. */
export function writeLocalPageSize(n: number): void {
  try {
    localStorage.setItem(LOCAL_POSTS_PER_PAGE_KEY, String(clampPostsPerPage(n)));
  } catch {
    // Private mode: the in-page size still applies for this session.
  }
}

/**
 * Nearest multiple of `columns` to `preferred`, clamped to 12–100.
 * When equally near two multiples, round up. `columns < 1` leaves preferred alone
 * (board grid not measured yet).
 */
export function snapPageSize(preferred: number, columns: number): number {
  if (!Number.isFinite(columns) || columns < 1) return preferred;
  const cols = Math.floor(columns);
  const pref = Number.isFinite(preferred) ? preferred : DEFAULT_PAGE_SIZE;

  const down = Math.floor(pref / cols) * cols;
  const up = Math.ceil(pref / cols) * cols;
  let snapped = down === up ? down : pref - down < up - pref ? down : up;

  if (snapped < 12) snapped = Math.ceil(12 / cols) * cols;
  if (snapped > 100) snapped = Math.floor(100 / cols) * cols;
  if (snapped < 12 || snapped > 100) return clampPostsPerPage(pref);
  return snapped;
}

/** Zero-based offset for a 1-based page number. */
export function pageOffset(page: number, pageSize: number): number {
  const size = coercePageSize(pageSize);
  const n = Number.isFinite(page) ? Math.floor(page) : 1;
  return Math.max(0, n - 1) * size;
}

/** How many pages a total fills. Empty lists are one empty page. */
export function pageCount(total: number, pageSize: number): number {
  const size = coercePageSize(pageSize);
  if (!Number.isFinite(total) || total <= 0) return 1;
  return Math.ceil(total / size);
}

/** Keep a requested page inside 1…pageCount(total, pageSize). */
export function clampPage(page: number, total: number, pageSize: number): number {
  const n = Number.isFinite(page) ? Math.floor(page) : 1;
  return Math.min(pageCount(total, pageSize), Math.max(1, n));
}

/**
 * Page numbers to show in the pager: current ± 2, plus first and last.
 * Gaps are represented as null (render as an ellipsis).
 */
export function pageWindow(page: number, total: number, pageSize: number): Array<number | null> {
  const count = pageCount(total, pageSize);
  const current = clampPage(page, total, pageSize);
  if (count <= 1) return [1];

  const wanted = new Set<number>([1, count]);
  for (let n = current - 2; n <= current + 2; n++) {
    if (n >= 1 && n <= count) wanted.add(n);
  }
  const sorted = [...wanted].sort((a, b) => a - b);
  const out: Array<number | null> = [];
  for (const n of sorted) {
    const prev = out.at(-1);
    if (typeof prev === "number" && n > prev + 1) out.push(null);
    out.push(n);
  }
  return out;
}

/** Inclusive label for the rows on this page, e.g. "41–80 of 120". */
export function pageRangeLabel(page: number, total: number, pageSize: number): string {
  const size = coercePageSize(pageSize);
  if (total <= 0) return "0 of 0";
  const current = clampPage(page, total, size);
  const start = pageOffset(current, size) + 1;
  const end = Math.min(total, pageOffset(current, size) + size);
  return `${start}–${end} of ${total}`;
}
