import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { HistoryList } from "../components/HistoryList";
import { IconHistory } from "../components/icons";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import { atLeast, ROLE_LABELS } from "../domain/roles";
import { useMe, useRecentHistory, useStaffActions } from "../hooks/useKura";
import type { ChangeEntry } from "../types/api";
import styles from "./AdminPage.module.css";
import simple from "./Simple.module.css";

export function HistoryPage() {
  const { me, user, loading } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const [busyId, setBusyId] = useState("");
  const [actorFilter, setActorFilter] = useState<string | undefined>(undefined);
  const isStaff = !!user && atLeast(user.role, "moderator");
  const history = useRecentHistory(actorFilter, isStaff);

  if (loading) return <p>Loading…</p>;
  if (!me || !user) {
    return (
      <section className={simple.page}>
        <h1>Change history</h1>
        <p className={simple.muted}>Sign in with a staff account to see the change history.</p>
        <div className={simple.actions}>
          <Link to="/login">Sign in</Link>
        </div>
      </section>
    );
  }
  if (!isStaff) {
    return (
      <section className={simple.page}>
        <h1>Change history</h1>
        <p className={simple.muted}>
          Your account is a {ROLE_LABELS[user.role]}. Moderators and above can see the staff change
          history.
        </p>
      </section>
    );
  }

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
    <section className={simple.page} style={{ maxWidth: "72rem" }} aria-labelledby="history-title">
      <h1 id="history-title">
        <IconHistory /> Change history
      </h1>
      <p className={simple.muted}>
        Every tag, rating, hide, delete and role change across the board, newest first — who did it
        and what it was before. Roll back undoes one change and is itself recorded, so a rogue staff
        account can be cleaned up one change at a time. Each post change links to its post. People
        and roles live on the <Link to="/admin">Staff</Link> page.
      </p>

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
      <HistoryList
        entries={history.data ?? []}
        canRollback={canRollback}
        onRollback={rollback}
        busyId={busyId}
      />
    </section>
  );
}
