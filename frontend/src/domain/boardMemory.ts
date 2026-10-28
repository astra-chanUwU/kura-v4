import { parseTagList } from "./boardFilter";

/** Last board this tab was looking at. A bare "/" restores it. */
export interface RememberedBoard {
  /** Account the filter belongs to. Empty when the viewer was signed out. */
  userId: string;
  tags: string[];
  exclude: string[];
  q: string;
  /** The viewer cleared the board. A later bare "/" stays empty. */
  cleared: boolean;
}

const STORAGE_KEY = "kura.board";

function storage(): Storage | null {
  try {
    return globalThis.sessionStorage ?? null;
  } catch {
    return null;
  }
}

function asTagList(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return parseTagList(value.filter((tag) => typeof tag === "string").join(","));
}

export function readBoardMemory(): RememberedBoard | null {
  const store = storage();
  if (!store) return null;
  try {
    const raw = store.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<RememberedBoard> | null;
    if (!parsed || typeof parsed !== "object") return null;
    const userId = typeof parsed.userId === "string" ? parsed.userId : "";
    if (parsed.cleared === true) return { userId, tags: [], exclude: [], q: "", cleared: true };
    const tags = asTagList(parsed.tags);
    const exclude = asTagList(parsed.exclude);
    const q = typeof parsed.q === "string" ? parsed.q.trim() : "";
    if (tags.length === 0 && exclude.length === 0 && q === "") return null;
    return { userId, tags, exclude, q, cleared: false };
  } catch {
    return null;
  }
}

/** The remembered board for this viewer, ignoring another account's filter. */
export function boardMemoryFor(userId: string): RememberedBoard | null {
  const memory = readBoardMemory();
  if (!memory || memory.userId !== userId) return null;
  return memory;
}

export function writeBoardMemory(board: RememberedBoard): void {
  const store = storage();
  if (!store) return;
  const next: RememberedBoard = board.cleared
    ? { userId: board.userId, tags: [], exclude: [], q: "", cleared: true }
    : {
        userId: board.userId,
        tags: board.tags,
        exclude: board.exclude,
        q: board.q.trim(),
        cleared: false,
      };
  // An empty board that was not cleared must not block the first-visit defaults.
  if (!next.cleared && next.tags.length === 0 && next.exclude.length === 0 && next.q === "") return;
  try {
    store.setItem(STORAGE_KEY, JSON.stringify(next));
  } catch {
    // Private mode and quota failures leave the previous memory in place.
  }
}

export function clearBoardMemory(): void {
  const store = storage();
  if (!store) return;
  try {
    store.removeItem(STORAGE_KEY);
  } catch {
    // Nothing to clear when storage is unavailable.
  }
}
