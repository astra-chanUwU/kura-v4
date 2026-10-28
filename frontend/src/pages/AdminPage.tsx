import { Link, Navigate } from "@tanstack/react-router";
import { useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import { assignableRoles, atLeast, isRole, ROLE_LABELS, type Role } from "../domain/roles";
import { useAdminUsers, useMe, useStaffActions } from "../hooks/useKura";
import { StaffSection } from "./StaffLayout";
import styles from "./AdminPage.module.css";
import simple from "./Simple.module.css";

// People & roles, the Staff section's landing tab. Moderators have no
// people tools, so they land on the hidden-posts queue instead.
export function AdminPage() {
  const { user } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const isAdmin = !!user && atLeast(user.role, "admin");
  const users = useAdminUsers(isAdmin);
  const [pendingRole, setPendingRole] = useState<{
    targetId: string;
    targetName: string;
    next: Role;
  } | null>(null);

  if (!user) return null;
  if (!isAdmin) return <Navigate to="/admin/hidden" replace />;

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
    <StaffSection
      id="people-title"
      title="People & roles"
      intro={
        <>
          Super Admins manage anyone. Admins only move people between Member and Moderator. The last
          Super Admin can never be demoted. Visitor means anonymous: SFW reading only, no account.
        </>
      }
    >
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
    </StaffSection>
  );
}
