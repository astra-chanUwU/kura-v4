import { Link, Outlet } from "@tanstack/react-router";
import styles from "./AppShell.module.css";

export function AppShell() {
  return (
    <div className={styles.shell}>
      <a className={styles.skip} href="#main">
        Skip to content
      </a>
      <header className={styles.header}>
        <Link className={styles.brand} to="/" aria-label="Kura home">
          <span className={styles.mark} aria-hidden="true">
            蔵
          </span>
          <span>
            Kura<span className={styles.version}>IV</span>
          </span>
        </Link>
        <nav aria-label="Main navigation">
          <Link to="/" activeProps={{ className: styles.active }} activeOptions={{ exact: true }}>
            Home
          </Link>
          <Link to="/about" activeProps={{ className: styles.active }}>
            About
          </Link>
        </nav>
        <span className={styles.note}>A place to keep what matters.</span>
      </header>
      <main id="main" className={styles.main} tabIndex={-1}>
        <Outlet />
      </main>
      <footer className={styles.footer}>
        <span>Your collection. Your space.</span>
        <span>Kura / V4</span>
      </footer>
    </div>
  );
}
