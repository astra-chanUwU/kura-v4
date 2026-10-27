export type DiffKind = "added" | "removed" | "unchanged";

export interface DiffLine {
  field: string;
  kind: DiffKind;
  text: string;
}

const FIELD_ORDER = ["role", "artist", "tags", "rating", "hidden", "deleted"] as const;

function isEmpty(value: unknown): boolean {
  return value === undefined || value === null || value === "";
}

function displayValue(field: string, value: unknown): string {
  if (typeof value === "boolean") return value ? "yes" : "no";
  if (field === "rating" && typeof value === "string") return value.toUpperCase();
  if (typeof value === "string") return value || "—";
  if (value === undefined || value === null) return "—";
  return JSON.stringify(value) ?? "—";
}

function tagList(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((tag): tag is string => typeof tag === "string");
}

function diffScalar(lines: DiffLine[], field: string, before: unknown, after: unknown): void {
  if (isEmpty(before) && isEmpty(after)) {
    lines.push({ field, kind: "unchanged", text: `${field}: —` });
    return;
  }
  if (displayValue(field, before) === displayValue(field, after)) {
    lines.push({ field, kind: "unchanged", text: `${field}: ${displayValue(field, after)}` });
    return;
  }
  if (!isEmpty(before))
    lines.push({ field, kind: "removed", text: `${field}: ${displayValue(field, before)}` });
  if (!isEmpty(after))
    lines.push({ field, kind: "added", text: `${field}: ${displayValue(field, after)}` });
}

function diffTags(lines: DiffLine[], before: unknown, after: unknown): void {
  const beforeTags = tagList(before);
  const afterTags = tagList(after);
  if (beforeTags.length === 0 && afterTags.length === 0) {
    lines.push({ field: "tags", kind: "unchanged", text: "tags: none" });
    return;
  }
  const afterSet = new Set(afterTags);
  const beforeSet = new Set(beforeTags);
  const kept = beforeTags.filter((tag) => afterSet.has(tag));
  const removed = beforeTags.filter((tag) => !afterSet.has(tag));
  const added = afterTags.filter((tag) => !beforeSet.has(tag));
  if (removed.length)
    lines.push({ field: "tags", kind: "removed", text: `tags: ${removed.join(", ")}` });
  if (added.length) lines.push({ field: "tags", kind: "added", text: `tags: ${added.join(", ")}` });
  if (kept.length)
    lines.push({ field: "tags", kind: "unchanged", text: `tags: ${kept.join(", ")}` });
}

/** Git-style line diff between two history snapshots. */
export function diffStates(
  before: Record<string, unknown>,
  after: Record<string, unknown>,
): DiffLine[] {
  const lines: DiffLine[] = [];
  for (const field of FIELD_ORDER) {
    if (!(field in before) && !(field in after)) continue;
    if (field === "tags") diffTags(lines, before[field], after[field]);
    else diffScalar(lines, field, before[field], after[field]);
  }
  const known = new Set<string>(FIELD_ORDER);
  const extra = [...new Set([...Object.keys(before), ...Object.keys(after)])]
    .filter((field) => !known.has(field))
    .sort();
  for (const field of extra) diffScalar(lines, field, before[field], after[field]);
  return lines;
}
