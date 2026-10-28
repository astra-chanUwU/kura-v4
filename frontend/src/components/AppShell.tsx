import { Link, Outlet } from "@tanstack/react-router";
import { useCallback } from "react";
import { ThemedSelect } from "./ThemedSelect";
import { useAccountActions, useMe } from "../hooks/useKura";
import { useAppearance } from "../hooks/useAppearance";
import {
  FONT_LABELS,
  FONTS,
  THEME_LABELS,
  THEME_SWATCHES,
  THEMES,
  normalizeFont,
  normalizeTheme,
} from "../domain/appearance";
import { atLeast } from "../domain/roles";
import styles from "./AppShell.module.css";

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
  const { theme, font, change } = useAppearance(
    me?.preferences.theme,
    me?.preferences.font,
    !!me,
    persist,
  );
  return (
    <div className={styles.shell}>
      <a className={styles.skip} href="#main">
        Skip to content
      </a>
      <header className={styles.header}>
        <Link className={styles.brand} to="/" aria-label="Kura home">
          Kura
        </Link>
        <nav aria-label="Main navigation" className={styles.nav}>
          {!loading &&
            (user ? (
              <>
                {atLeast(user.role, "moderator") && (
                  <Link to="/admin" activeProps={{ className: styles.active }}>
                    Staff
                  </Link>
                )}
                <Link
                  to="/me"
                  activeProps={{ className: styles.active }}
                  className={styles.userLink}
                >
                  {user.avatar_url ? (
                    <img src={user.avatar_url} alt="" className={styles.avatar} />
                  ) : (
                    <span aria-hidden="true" className={styles.avatarFallback}>
                      {(user.display_name || user.username).charAt(0).toUpperCase()}
                    </span>
                  )}
                  <span className={styles.userName}>
                    {user.display_name || `@${user.username}`}
                  </span>
                </Link>
              </>
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
        </nav>
        <div className={styles.themeWrap}>
          <ThemedSelect
            label="Theme"
            value={theme}
            onChange={(id) => change({ theme: normalizeTheme(id) })}
          >
            {THEMES.map((id) => (
              <option key={id} value={id} data-swatch={THEME_SWATCHES[id]}>
                {THEME_LABELS[id]}
              </option>
            ))}
          </ThemedSelect>
          <ThemedSelect
            label="Font"
            value={font}
            onChange={(id) => change({ font: normalizeFont(id) })}
          >
            {FONTS.map((id) => (
              <option key={id} value={id}>
                {FONT_LABELS[id]}
              </option>
            ))}
          </ThemedSelect>
        </div>
      </header>
      <main id="main" className={styles.main} tabIndex={-1}>
        <Outlet />
      </main>
    </div>
  );
}
