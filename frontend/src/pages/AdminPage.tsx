import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { IconEyeOff, IconHistory } from "../components/icons";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import {
  assignableRoles,
  atLeast,
  isRole,
  ROLE_DESCRIPTIONS,
  ROLE_LABELS,
  type Role,
} from "../domain/roles";
import { useAdminUsers, useHiddenPosts, useMe, useStaffActions } from "../hooks/useKura";
import styles from "./AdminPage.module.css";
import simple from "./Simple.module.css";

export function AdminPage() {
  const { me, user, loading } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const isStaff = !!user && atLeast(user.role, "moderator");
  const isAdmin = !!user && atLeast(user.role, "admin");
  const users = useAdminUsers(isAdmin);
  const hidden = useHiddenPosts(isStaff);
  const [pendingRole, setPendingRole] = useState<{
    targetId: string;
    targetName: string;
    next: Role;
  } | null>(null);

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

  const applyRole = (targetId: string, targetName: string, next: Role) => {
    void staff
      .setUserRole(targetId, next)
      .then(() => toasts.push(`@${targetName} is now ${ROLE_LABELS[next]}.`, "success"))
      .catch((e: unknown) => toasts.push(shortError(e), "error"));
  };

  const changeRole = (targetId: string, targetName: string, next: string, current: Role) => {
    if (!isRole(next) || next === current) return;
    if (next === "admin" || next === "super_admin") {
      setPendingRole({ targetId, targetName, next });
      return;
    }
    applyRole(targetId, targetName, next);
  };

  return (
    <section className={simple.page} style={{ maxWidth: "72rem" }} aria-labelledby="admin-title">
      <h1 id="admin-title">Staff</h1>
      <p className={simple.muted}>
        You are <strong>{ROLE_LABELS[user.role]}</strong> — {ROLE_DESCRIPTIONS[user.role]} Visitor
        means anonymous: SFW reading only, no account.
      </p>

      <div className={simple.card}>
        <h2>
          <IconEyeOff /> Hidden &amp; deleted posts
        </h2>
        <p className={simple.muted}>
          {hidden.isPending
            ? "Loading…"
            : `${hidden.data?.length ?? 0} post${(hidden.data?.length ?? 0) === 1 ? "" : "s"} off the board right now. Review them, restore the good ones, or roll a removal back from its history.`}
        </p>
        <div className={simple.actions}>
          <Link to="/admin/hidden">Review hidden &amp; deleted</Link>
        </div>
      </div>

      <div className={simple.card}>
        <h2>
          <IconHistory /> Change history
        </h2>
        <p className={simple.muted}>
          Tag, rating, hide, delete and role changes across the board — with rollback. It has its
          own page so this one stays about people.
        </p>
        <div className={simple.actions}>
          <Link to="/admin/history">Open change history</Link>
        </div>
      </div>

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
          <ul className={styles.userList}>
            {(users.data ?? []).map((u) => {
              const options = assignableRoles(user.role, u.role);
              return (
                <li key={u.id} className={styles.userRow}>
                  <span className={styles.userMain}>
                    <Link to="/u/$username" params={{ username: u.username }}>
                      @{u.username}
                    </Link>{" "}
                    <span className={simple.muted}>
                      {u.display_name}
                      {u.id === user.id ? " (you)" : ""} · joined{" "}
                      {new Date(u.created_at).toLocaleDateString()}
                    </span>
                  </span>
                  {options.length > 0 ? (
                    <ThemedSelect
                      label={`Role for @${u.username}`}
                      value={u.role}
                      onChange={(next) => changeRole(u.id, u.username, next, u.role)}
                    >
                      {[...new Set([u.role, ...options])].map((r) => (
                        <option key={r} value={r}>
                          {ROLE_LABELS[r]}
                        </option>
                      ))}
                    </ThemedSelect>
                  ) : (
                    <span className={styles.roleBadge}>{ROLE_LABELS[u.role]}</span>
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      )}
      <ConfirmDialog
        open={pendingRole !== null}
        title={
          pendingRole ? `Make @${pendingRole.targetName} a ${ROLE_LABELS[pendingRole.next]}?` : ""
        }
        message="They will gain staff powers."
        confirmLabel={pendingRole ? `Make ${ROLE_LABELS[pendingRole.next]}` : "Confirm"}
        onCancel={() => setPendingRole(null)}
        onConfirm={() => {
          const pending = pendingRole;
          setPendingRole(null);
          if (pending) applyRole(pending.targetId, pending.targetName, pending.next);
        }}
      />
    </section>
  );
}
