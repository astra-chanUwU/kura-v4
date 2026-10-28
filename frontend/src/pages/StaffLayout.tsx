import { Link, Outlet } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { IconEyeOff, IconHistory, IconUser } from "../components/icons";
import { atLeast, ROLE_DESCRIPTIONS, ROLE_LABELS } from "../domain/roles";
import { useHiddenPosts, useMe } from "../hooks/useKura";
import styles from "./StaffLayout.module.css";
import simple from "./Simple.module.css";

// The staff section: one access gate, one heading and a tab per tool.
// Child pages render inside the Outlet and can assume a staff account.
export function StaffLayout() {
  const { me, user, loading } = useMe();
  const isStaff = !!user && atLeast(user.role, "moderator");
  const isAdmin = !!user && atLeast(user.role, "admin");
  const hidden = useHiddenPosts(isStaff);

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

  const removed = hidden.data?.length ?? 0;
  return (
    <section className={styles.page} aria-labelledby="staff-title">
      <header className={styles.head}>
        <h1 id="staff-title">Staff</h1>
        <p className={simple.muted}>
          You are <strong>{ROLE_LABELS[user.role]}</strong> — {ROLE_DESCRIPTIONS[user.role]}
        </p>
      </header>
      <nav className={styles.tabs} aria-label="Staff sections">
        {isAdmin && (
          <Link
            to="/admin"
            activeOptions={{ exact: true }}
            activeProps={{ className: styles.active }}
          >
            <IconUser />
            People
          </Link>
        )}
        <Link
          to="/admin/hidden"
          activeProps={{ className: styles.active }}
          aria-label={removed > 0 ? `Hidden, ${removed} off the board` : undefined}
        >
          <IconEyeOff />
          Hidden
          {removed > 0 && (
            <span className={styles.count} aria-hidden="true">
              {removed}
            </span>
          )}
        </Link>
        <Link to="/admin/history" activeProps={{ className: styles.active }}>
          <IconHistory />
          History
        </Link>
      </nav>
      <Outlet />
    </section>
  );
}

export function StaffSection({
  id,
  title,
  intro,
  children,
}: {
  id: string;
  title: ReactNode;
  intro: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className={styles.section} aria-labelledby={id}>
      <h2 id={id}>{title}</h2>
      <p className={simple.muted}>{intro}</p>
      {children}
    </section>
  );
}
