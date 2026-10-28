import { useState } from "react";
import { HistoryList } from "../components/HistoryList";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import { atLeast } from "../domain/roles";
import { useMe, useRecentHistory, useStaffActions } from "../hooks/useKura";
import type { ChangeEntry } from "../types/api";
import { StaffSection } from "./StaffLayout";
import styles from "./AdminPage.module.css";
import simple from "./Simple.module.css";

export function HistoryPage() {
  const { user } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const [busyId, setBusyId] = useState("");
  const [actorFilter, setActorFilter] = useState<string | undefined>(undefined);
  const isStaff = !!user && atLeast(user.role, "moderator");
  const history = useRecentHistory(actorFilter, isStaff);

  if (!user) return null;

  const rollback = (entry: ChangeEntry) => {
    setBusyId(entry.id);
    void staff
      .rollback(entry)
      .then(() =>
        toasts.push("Change rolled back. The rollback itself is in the history.", "success"),
      )
      .catch((e: unknown) => toasts.push(shortError(e), "error"))
      .finally(() => setBusyId(""));
  };

  const canRollback = (entry: ChangeEntry) => {
    if (entry.action === "post.created") return false;
    if (entry.entity_type === "post") return atLeast(user.role, "moderator");
    return atLeast(user.role, "admin");
  };

  const actorOptions = [
    ...new Map(
      (history.data ?? []).filter((e) => e.actor_id).map((e) => [e.actor_id, e.actor_username]),
    ).entries(),
  ];

  return (
    <StaffSection
      id="history-title"
      title="Change history"
      intro={
        <>
          Every tag, rating, hide, delete and role change across the board, newest first — who did
          it and what it was before. Roll back undoes one change and is itself recorded, so a rogue
          staff account can be cleaned up one change at a time. Each post change links to its post.
        </>
      }
    >
      <div className={styles.filterRow}>
        <span className={simple.muted}>Actor</span>
        <ThemedSelect
          label="Filter history by actor"
          value={actorFilter ?? ""}
          onChange={(next) => setActorFilter(next || undefined)}
        >
          <option value="">Everyone</option>
          {actorOptions.map(([id, name]) => (
            <option key={id} value={id}>
              @{name}
            </option>
          ))}
        </ThemedSelect>
        {actorFilter && (
          <span className={simple.muted}>
            Roll back each row to undo this actor&apos;s changes.
          </span>
        )}
      </div>

      {history.isPending && <p className={simple.muted}>Loading…</p>}
      {history.error && (
        <p role="alert" className={simple.error}>
          {shortError(history.error)}
        </p>
      )}
      {history.data && (
        <HistoryList
          entries={history.data}
          canRollback={canRollback}
          onRollback={rollback}
          busyId={busyId}
          showThumbnails
        />
      )}
    </StaffSection>
  );
}
