import type { Site } from "../types/api";
import { getJSON } from "./client";

export function getSite(signal?: AbortSignal): Promise<Site> {
  return getJSON<Site>("/site", signal);
}
