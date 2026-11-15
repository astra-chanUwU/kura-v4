import { Link, Outlet } from "@tanstack/react-router";
import { useCallback } from "react";
import { NotificationBell } from "./NotificationBell";
import { ThemedSelect } from "./ThemedSelect";
import { useAccountActions, useMe } from "../hooks/useSession";
import { useAppearance } from "../hooks/useAppearance";
import {
  THEME_LABELS,
  THEME_PAPERS,
  THEME_SWATCHES,
  THEMES,
  normalizeTheme,
} from "../domain/appearance";
import { atLeast } from "../domain/roles";
import styles from "./AppShell.module.css";

// The Kura v3 shelf mark (web/static/icons/shelf.svg).
function KuraMark() {
  return (
    <svg
      className={styles.mark}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M3 3h14v14H3zM7 7h14v14H7z" />
      <path d="M7 7h10v10H7z" fill="currentColor" stroke="none" />
    </svg>
  );
}

export function AppShell() {
  const { me, user, loading } = useMe();
  const account = useAccountActions();
  const persist = useCallback(
    (
      theme: Parameters<typeof account.setAppearance>[0],
      font: Parameters<typeof account.setAppearance>[1],
    ) => {
      // Header picks are quick switches; the account save is
      // fire-and-forget (the local appearance already applied).
      void account.setAppearance(theme, font).catch(() => {});
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- account actions are stable wrappers
    [],
  );
  const { theme, change } = useAppearance(
    me?.preferences.theme,
    me?.preferences.font,
    !!me,
    persist,
  );
  const accountLabel = user ? user.display_name || `@${user.username}` : "";
  return (
    <div className={styles.shell}>
      <a className={styles.skip} href="#main">
        Skip to content
      </a>
      <header className={styles.header}>
        <Link className={styles.brand} to="/" search={{}} aria-label="Kura home">
          <KuraMark />
          <span className={styles.word}>Kura</span>
        </Link>
        {!loading && user && atLeast(user.role, "moderator") && (
          <nav aria-label="Main navigation" className={styles.nav}>
            <Link to="/admin" activeProps={{ className: styles.active }}>
              Staff
            </Link>
          </nav>
        )}
        <div className={styles.themeWrap}>
          {!loading && user && <NotificationBell />}
          <ThemedSelect
            label="Theme"
            popup
            compact
            value={theme}
            onChange={(id) => change({ theme: normalizeTheme(id) })}
          >
            {THEMES.map((id) => (
              <option
                key={id}
                value={id}
                data-swatch={THEME_SWATCHES[id]}
                data-paper={THEME_PAPERS[id]}
              >
                {THEME_LABELS[id]}
              </option>
            ))}
          </ThemedSelect>
          {!loading &&
            (user ? (
              <Link
                to="/me"
                activeProps={{ className: styles.active }}
                className={styles.userLink}
                aria-label={accountLabel}
              >
                {user.avatar_url ? (
                  <img src={user.avatar_url} alt="" className={styles.avatar} />
                ) : (
                  <span aria-hidden="true" className={styles.avatarFallback}>
                    {(user.display_name || user.username).charAt(0).toUpperCase()}
                  </span>
                )}
                <span className={styles.userName}>{accountLabel}</span>
              </Link>
            ) : (
              <>
                <Link to="/login" activeProps={{ className: styles.active }}>
                  Sign in
                </Link>
                <Link
                  to="/register"
                  activeProps={{ className: styles.active }}
                  className={styles.joinLink}
                >
                  Create account
                </Link>
              </>
            ))}
        </div>
      </header>
      <main id="main" className={styles.main} tabIndex={-1}>
        <Outlet />
      </main>
    </div>
  );
}
