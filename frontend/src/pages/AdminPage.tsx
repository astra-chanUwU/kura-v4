import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { HistoryList } from "../components/HistoryList";
import { shortError, useToasts } from "../components/Toasts";
import {
  assignableRoles,
  atLeast,
  isRole,
  ROLE_DESCRIPTIONS,
  ROLE_LABELS,
  type Role,
} from "../domain/roles";
import { useAdminUsers, useMe, useRecentHistory, useStaffActions } from "../hooks/useKura";
import type { ChangeEntry } from "../types/api";
import styles from "./AdminPage.module.css";
import simple from "./Simple.module.css";

export function AdminPage() {
  const { me, user, loading } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const [busyId, setBusyId] = useState("");
  const [actorFilter, setActorFilter] = useState<string | undefined>(undefined);
  const isStaff = !!user && atLeast(user.role, "moderator");
  const isAdmin = !!user && atLeast(user.role, "admin");
  const users = useAdminUsers(isAdmin);
  const history = useRecentHistory(actorFilter, isStaff);

  if (loading) return <p>Loading…</p>;
  if (!me || !user) {
    return (
      <section className={simple.page}>
        <h1>Staff</h1>
        <p className={simple.muted}>Sign in with a staff account to use this page.</p>
        <div className={simple.actions}>
          <Link to="/login">Sign in</Link>
        </div>
      </section>
    );
  }
  if (!isStaff) {
    return (
      <section className={simple.page}>
        <h1>Staff</h1>
        <p className={simple.muted}>
          Your account is a {ROLE_LABELS[user.role]}. Moderators and above can use this page — ask
          an admin if you need access.
        </p>
      </section>
    );
  }

  const changeRole = (targetId: string, targetName: string, next: string, current: Role) => {
    if (!isRole(next) || next === current) return;
    if (
      (next === "admin" || next === "super_admin") &&
      !window.confirm(`Make @${targetName} a ${ROLE_LABELS[next]}? They will gain staff powers.`)
    ) {
      return;
    }
    void staff
      .setUserRole(targetId, next)
      .then(() => toasts.push(`@${targetName} is now ${ROLE_LABELS[next]}.`, "success"))
      .catch((e: unknown) => toasts.push(shortError(e), "error"));
  };

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

  return (
    <section className={simple.page} style={{ maxWidth: "72rem" }} aria-labelledby="admin-title">
      <h1 id="admin-title">Staff</h1>
      <p className={simple.muted}>
        You are <strong>{ROLE_LABELS[user.role]}</strong> — {ROLE_DESCRIPTIONS[user.role]} Visitor
        means anonymous: SFW reading only, no account.
      </p>

      {isAdmin && (
        <div className={simple.card}>
          <h2>People & roles</h2>
          <p className={simple.muted}>
            Super Admins manage anyone. Admins only move people between Member and Moderator. The
            last Super Admin can never be demoted.
          </p>
          {users.isPending && <p className={simple.muted}>Loading…</p>}
          {users.error && (
            <p role="alert" className={simple.error}>
              {shortError(users.error)}
            </p>
          )}
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">User</th>
                  <th scope="col">Role</th>
                  <th scope="col">Joined</th>
                </tr>
              </thead>
              <tbody>
                {(users.data ?? []).map((u) => {
                  const options = assignableRoles(user.role, u.role);
                  return (
                    <tr key={u.id}>
                      <td>
                        <Link to="/u/$username" params={{ username: u.username }}>
                          @{u.username}
                        </Link>{" "}
                        <span className={simple.muted}>{u.display_name}</span>
                        {u.id === user.id && <span className={simple.muted}> (you)</span>}
                      </td>
                      <td>
                        {options.length > 0 ? (
                          <select
                            aria-label={`Role for @${u.username}`}
                            value={u.role}
                            onChange={(e) => changeRole(u.id, u.username, e.target.value, u.role)}
                          >
                            {[...new Set([u.role, ...options])].map((r) => (
                              <option key={r} value={r}>
                                {ROLE_LABELS[r]}
                              </option>
                            ))}
                          </select>
                        ) : (
                          ROLE_LABELS[u.role]
                        )}
                      </td>
                      <td className={simple.muted}>
                        {new Date(u.created_at).toLocaleDateString()}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div className={simple.card}>
        <h2>Change history</h2>
        <p className={simple.muted}>
          Every tag, rating, hide, delete and role change is recorded with who did it and what it
          was before. Roll back undoes one change and is itself recorded — use the actor filter to
          clean up after a rogue staff account, one change at a time.
        </p>
        <div
          style={{
            display: "flex",
            gap: "0.5rem",
            alignItems: "center",
            flexWrap: "wrap",
            marginBottom: "0.8rem",
          }}
        >
          <label>
            Actor{" "}
            <select
              aria-label="Filter history by actor"
              value={actorFilter ?? ""}
              onChange={(e) => setActorFilter(e.target.value || undefined)}
            >
              <option value="">Everyone</option>
              {[
                ...new Map(
                  (history.data ?? [])
                    .filter((e) => e.actor_id)
                    .map((e) => [e.actor_id, e.actor_username]),
                ).entries(),
              ].map(([id, name]) => (
                <option key={id} value={id}>
                  @{name}
                </option>
              ))}
            </select>
          </label>
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
      </div>
    </section>
  );
}
