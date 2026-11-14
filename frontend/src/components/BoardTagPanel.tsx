import { useId, useState } from "react";
import { IconTag } from "./icons";
import styles from "./BoardTagPanel.module.css";

const STORAGE_KEY = "kura-board-tag-panel";

export type BoardTagPanelTag = {
  tag: string;
  count: number;
};

export type BoardTagPanelProps = {
  tags: BoardTagPanelTag[];
  /** Selected include + exclude filters currently applied. */
  activeCount: number;
  onInclude: (tag: string) => void;
  onExclude: (tag: string) => void;
  onClear?: () => void;
  loading?: boolean;
};

function readOpen(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === "open";
  } catch {
    return false;
  }
}

function writeOpen(open: boolean) {
  try {
    localStorage.setItem(STORAGE_KEY, open ? "open" : "closed");
  } catch {
    // Private mode can block storage; toggle still works for this visit.
  }
}

/** Collapsible page-tag chip panel (WaifuList GenreFilter–inspired). */
export function BoardTagPanel({
  tags,
  activeCount,
  onInclude,
  onExclude,
  onClear,
  loading,
}: BoardTagPanelProps) {
  const [open, setOpen] = useState(readOpen);
  const [search, setSearch] = useState("");
  const panelId = useId();
  const searchId = useId();

  const toggle = () => {
    setOpen((prev) => {
      const next = !prev;
      writeOpen(next);
      return next;
    });
  };

  const query = search.trim().toLowerCase();
  const filtered = query
    ? tags.filter((t) => t.tag.toLowerCase().includes(query))
    : tags;

  const hasActive = activeCount > 0;
  const badgeValue = hasActive ? activeCount : tags.length;
  const showBadge = hasActive || tags.length > 0;

  return (
    <div className={styles.panel}>
      <div
        className={`${styles.header} ${open ? styles.headerOpen : ""}`}
        role="button"
        tabIndex={0}
        aria-expanded={open}
        aria-controls={panelId}
        onClick={toggle}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            toggle();
          }
        }}
      >
        <span className={styles.headerLeft}>
          <span className={styles.icon}>
            <IconTag />
          </span>
          <span className={styles.title}>Tags on this page</span>
          {showBadge && (
            <span
              className={hasActive ? styles.badgeActive : styles.badgeMuted}
              aria-label={
                hasActive
                  ? `${activeCount} active filter${activeCount === 1 ? "" : "s"}`
                  : `${tags.length} tag${tags.length === 1 ? "" : "s"} on this page`
              }
            >
              {badgeValue}
            </span>
          )}
        </span>
        <span className={styles.headerRight}>
          {hasActive && onClear && open && (
            <button
              type="button"
              className={styles.clearButton}
              onClick={(event) => {
                event.stopPropagation();
                onClear();
                setSearch("");
              }}
            >
              Clear
            </button>
          )}
          <span className={styles.chevron} aria-hidden="true">
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
            >
              <path d="m6 9.5 6 6 6-6" />
            </svg>
          </span>
        </span>
      </div>

      {open && (
        <div id={panelId} className={styles.content}>
          {loading ? (
            <p className={styles.loading}>Loading tags…</p>
          ) : (
            <>
              {tags.length > 0 && (
                <div className={styles.searchWrapper}>
                  <label className={styles.srOnly} htmlFor={searchId}>
                    Filter tags
                  </label>
                  <input
                    id={searchId}
                    type="search"
                    className={styles.searchInput}
                    placeholder="Search tags…"
                    value={search}
                    onChange={(event) => setSearch(event.target.value)}
                    autoComplete="off"
                  />
                  {search && (
                    <button
                      type="button"
                      className={styles.clearSearch}
                      aria-label="Clear tag search"
                      onClick={() => setSearch("")}
                    >
                      ×
                    </button>
                  )}
                </div>
              )}

              {tags.length === 0 ? (
                <p className={styles.empty}>No tags on this page</p>
              ) : filtered.length === 0 ? (
                <p className={styles.empty}>No tags match “{search.trim()}”</p>
              ) : (
                <div className={styles.chipList} role="list">
                  {filtered.map((t) => (
                    <span key={t.tag} className={styles.chip} role="listitem">
                      <button
                        type="button"
                        className={`${styles.chipSide} ${styles.chipExclude}`}
                        aria-label={`Exclude tag ${t.tag}`}
                        title={`Exclude ${t.tag}`}
                        onClick={() => onExclude(t.tag)}
                      >
                        −
                      </button>
                      <button
                        type="button"
                        className={styles.chipMain}
                        aria-label={`Include tag ${t.tag}`}
                        title={`Include ${t.tag}`}
                        onClick={() => onInclude(t.tag)}
                      >
                        {t.tag}
                        <span className={styles.chipCount}>· {t.count}</span>
                      </button>
                    </span>
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
