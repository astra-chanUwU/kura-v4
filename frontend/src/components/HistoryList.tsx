import { Link } from "@tanstack/react-router";
import { diffStates } from "../domain/historyDiff";
import type { ChangeEntry } from "../types/api";
import { IconUndo } from "./icons";
import styles from "./HistoryList.module.css";

export function describeState(state: Record<string, unknown>): string {
  if (typeof state.role === "string") return `role: ${state.role}`;
  const parts: string[] = [];
  if (typeof state.artist === "string") parts.push(`artist: ${state.artist || "—"}`);
  if (Array.isArray(state.tags))
    parts.push(`tags: ${state.tags.length ? state.tags.join(", ") : "none"}`);
  if (typeof state.rating === "string") parts.push(state.rating.toUpperCase());
  if (state.deleted === true) parts.push("deleted");
  else if (state.hidden === true) parts.push("hidden");
  return parts.join(" · ") || "—";
}

const ACTION_LABELS: Record<string, string> = {
  "post.created": "Uploaded",
  "post.meta_updated": "Tags / artist / rating edited",
  "post.hidden": "Hidden",
  "post.unhidden": "Unhidden",
  "post.deleted": "Deleted (soft)",
  "user.role_changed": "Role changed",
  rollback: "Rollback",
};

export function HistoryList({
  entries,
  canRollback,
  onRollback,
  busyId,
}: {
  entries: ChangeEntry[];
  canRollback?: (entry: ChangeEntry) => boolean;
  onRollback?: (entry: ChangeEntry) => void;
  busyId?: string;
}) {
  if (entries.length === 0) return <p className={styles.muted}>No changes recorded yet.</p>;
  return (
    <>
      <p className={styles.legend} aria-hidden="true">
        <span className={styles.legendRemoved}>- removed</span>{" "}
        <span className={styles.legendAdded}>+ added</span>{" "}
        <span className={styles.legendUnchanged}>unchanged</span>
      </p>
      <ol className={styles.list}>
        {entries.map((entry) => (
          <li key={entry.id} className={styles.row}>
            <div className={styles.head}>
              <span className={styles.action}>{ACTION_LABELS[entry.action] ?? entry.action}</span>
              <span className={styles.meta}>
                {entry.actor_username ? (
                  <Link to="/u/$username" params={{ username: entry.actor_username }}>
                    @{entry.actor_username}
                  </Link>
                ) : (
                  "system"
                )}{" "}
                · {new Date(entry.created_at).toLocaleString()}
              </span>
              {entry.entity_type === "post" && (
                <Link
                  to="/posts/$id"
                  params={{ id: entry.entity_id }}
                  className={styles.postLink}
                  aria-label="Open the post this change belongs to"
                >
                  View post ↗
                </Link>
              )}
              {canRollback?.(entry) && onRollback && (
                <button
                  type="button"
                  className={styles.rollback}
                  disabled={busyId === entry.id}
                  onClick={() => onRollback(entry)}
                >
                  <IconUndo /> {busyId === entry.id ? "Rolling back…" : "Roll back"}
                </button>
              )}
            </div>
            <ul className={styles.diff} aria-label="What changed">
              {diffStates(entry.before, entry.after).map((line, index) => (
                <li
                  key={`${line.field}-${line.kind}-${index}`}
                  className={`${styles.diffLine} ${styles[line.kind]}`}
                  data-kind={line.kind}
                >
                  <span className={styles.sign} aria-hidden="true">
                    {line.kind === "added" ? "+" : line.kind === "removed" ? "-" : " "}
                  </span>
                  <span className={styles.diffText}>{line.text}</span>
                  <span className={styles.srOnly}>
                    {line.kind === "added" ? "(added)" : line.kind === "removed" ? "(removed)" : ""}
                  </span>
                </li>
              ))}
            </ul>
            {entry.rollback_of && <p className={styles.meta}>Undoes an earlier change</p>}
          </li>
        ))}
      </ol>
    </>
  );
}
