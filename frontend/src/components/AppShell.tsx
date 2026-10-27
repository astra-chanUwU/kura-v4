import { Link, Outlet } from "@tanstack/react-router";
import { useState } from "react";
import { ThemedSelect } from "./ThemedSelect";
import { useMe } from "../hooks/useKura";
import { atLeast } from "../domain/roles";
import styles from "./AppShell.module.css";

const THEMES = [
  { id: "kura", label: "Kura Dark", swatch: "#b49afa" },
  { id: "safebooru", label: "Safebooru", swatch: "#0b6bcb" },
  { id: "rule34", label: "Rule34", swatch: "#8bc34a" },
  { id: "umineko", label: "Umineko", swatch: "#c9a227" },
] as const;

export function AppShell() {
  const { user, loading } = useMe();
  const [theme, setTheme] = useState(() => localStorage.getItem("kura-theme") ?? "kura");
  const applyTheme = (id: string) => {
    setTheme(id);
    localStorage.setItem("kura-theme", id);
    if (id === "kura") delete document.documentElement.dataset.theme;
    else document.documentElement.dataset.theme = id;
  };
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
                {atLeast(user.role, "moderator") && (
                  <Link to="/admin/history" activeProps={{ className: styles.active }}>
                    History
                  </Link>
                )}
                {atLeast(user.role, "moderator") && (
                  <Link to="/admin/hidden" activeProps={{ className: styles.active }}>
                    Hidden
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
          <ThemedSelect label="Theme" value={theme} onChange={applyTheme}>
            {THEMES.map((t) => (
              <option key={t.id} value={t.id} data-swatch={t.swatch}>
                {t.label}
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
