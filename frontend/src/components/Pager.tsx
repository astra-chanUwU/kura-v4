import { useEffect, useState, type FormEvent } from "react";
import { clampPage, pageCount, pageRangeLabel, pageWindow } from "../domain/paging";
import styles from "./Pager.module.css";

type PagerProps = {
  page: number;
  total: number;
  pageSize: number;
  onPage: (page: number) => void;
  /** Warm a page when the viewer aims at its control (hover / focus). */
  onPrefetch?: (page: number) => void;
};

/** Numbered Previous / pages / Next for a server-paged list. */
export function Pager({ page, total, pageSize, onPage, onPrefetch }: PagerProps) {
  if (total <= pageSize) return null;

  const current = clampPage(page, total, pageSize);
  const last = pageCount(total, pageSize);
  const numbers = pageWindow(current, total, pageSize);

  const prefetch = (target: number) => {
    if (!onPrefetch) return;
    if (target < 1 || target > last || target === current) return;
    onPrefetch(target);
  };

  return (
    <nav className={styles.pager} aria-label="Pagination">
      <button
        type="button"
        className={styles.button}
        disabled={current <= 1}
        onClick={() => onPage(current - 1)}
        onPointerEnter={() => prefetch(current - 1)}
        onFocus={() => prefetch(current - 1)}
      >
        Previous
      </button>
      <ol className={styles.pages}>
        {numbers.map((n, index) =>
          n === null ? (
            <li key={`gap-${index}`} className={styles.gap} aria-hidden="true">
              …
            </li>
          ) : (
            <li key={n}>
              <button
                type="button"
                className={n === current ? styles.current : styles.button}
                aria-current={n === current ? "page" : undefined}
                disabled={n === current}
                onClick={() => onPage(n)}
                onPointerEnter={() => prefetch(n)}
                onFocus={() => prefetch(n)}
              >
                {n}
              </button>
            </li>
          ),
        )}
      </ol>
      <button
        type="button"
        className={styles.button}
        disabled={current >= last}
        onClick={() => onPage(current + 1)}
        onPointerEnter={() => prefetch(current + 1)}
        onFocus={() => prefetch(current + 1)}
      >
        Next
      </button>
      <JumpForm current={current} last={last} total={total} pageSize={pageSize} onPage={onPage} onPrefetch={prefetch} />
      <p className={styles.range}>{pageRangeLabel(current, total, pageSize)}</p>
    </nav>
  );
}

function JumpForm({
  current,
  last,
  total,
  pageSize,
  onPage,
  onPrefetch,
}: {
  current: number;
  last: number;
  total: number;
  pageSize: number;
  onPage: (page: number) => void;
  onPrefetch: (target: number) => void;
}) {
  const [draft, setDraft] = useState(String(current));

  useEffect(() => {
    setDraft(String(current));
  }, [current]);

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const trimmed = draft.trim();
    if (trimmed === "") return;
    const n = Number(trimmed);
    if (!Number.isFinite(n)) return;
    const next = clampPage(n, total, pageSize);
    setDraft(String(next));
    if (next !== current) onPage(next);
  };

  const resetIfInvalid = () => {
    const trimmed = draft.trim();
    if (trimmed === "") {
      setDraft(String(current));
      return;
    }
    const n = Number(trimmed);
    if (!Number.isFinite(n)) {
      setDraft(String(current));
      return;
    }
    setDraft(String(clampPage(n, total, pageSize)));
  };

  const draftPage = (() => {
    const n = Number(draft.trim());
    if (!Number.isFinite(n)) return null;
    return clampPage(n, total, pageSize);
  })();

  return (
    <form className={styles.jump} noValidate onSubmit={submit}>
      <label className={styles.jumpLabel}>
        <span className={styles.visuallyHidden}>Go to page</span>
        <input
          type="number"
          className={styles.jumpInput}
          min={1}
          max={last}
          inputMode="numeric"
          value={draft}
          aria-label="Go to page"
          onChange={(event) => setDraft(event.target.value)}
          onBlur={resetIfInvalid}
          onPointerEnter={() => {
            if (draftPage != null) onPrefetch(draftPage);
          }}
          onFocus={() => {
            if (draftPage != null) onPrefetch(draftPage);
          }}
        />
      </label>
      <button
        type="submit"
        className={styles.button}
        onPointerEnter={() => {
          if (draftPage != null) onPrefetch(draftPage);
        }}
        onFocus={() => {
          if (draftPage != null) onPrefetch(draftPage);
        }}
      >
        Go
      </button>
    </form>
  );
}
