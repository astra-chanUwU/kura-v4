import type { CreateSavedSearch, SavedSearch } from "../types/savedSearch";
import { getJSON, sendJSON } from "./client";

export function listSavedSearches(signal?: AbortSignal): Promise<SavedSearch[]> {
  return getJSON<SavedSearch[]>("/me/searches", signal);
}

export function createSavedSearch(input: CreateSavedSearch): Promise<SavedSearch> {
  return sendJSON<SavedSearch>("/me/searches", "POST", input);
}

export function deleteSavedSearch(id: string): Promise<void> {
  return sendJSON<void>(`/me/searches/${encodeURIComponent(id)}`, "DELETE");
}
