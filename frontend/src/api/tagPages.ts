import { getJSON, sendJSON } from "./client";
import type { TagPage } from "../types/tagPage";

function tagPath(name: string): string {
  return `/tag/${encodeURIComponent(name)}`;
}

export function getTagPage(name: string, signal?: AbortSignal): Promise<TagPage> {
  return getJSON<TagPage>(tagPath(name), signal);
}

export function updateTagPage(
  name: string,
  category: string,
  description: string,
): Promise<TagPage> {
  return sendJSON<TagPage>(tagPath(name), "PUT", { category, description });
}

export function addTagAlias(target: string, alias: string): Promise<TagPage> {
  return sendJSON<TagPage>(`${tagPath(target)}/aliases`, "POST", { alias });
}

export function removeTagAlias(target: string, alias: string): Promise<TagPage> {
  return sendJSON<TagPage>(`${tagPath(target)}/aliases/${encodeURIComponent(alias)}`, "DELETE");
}

export function addTagImplication(antecedent: string, consequent: string): Promise<TagPage> {
  return sendJSON<TagPage>(`${tagPath(antecedent)}/implications`, "POST", { consequent });
}

export function removeTagImplication(antecedent: string, consequent: string): Promise<TagPage> {
  return sendJSON<TagPage>(
    `${tagPath(antecedent)}/implications/${encodeURIComponent(consequent)}`,
    "DELETE",
  );
}
