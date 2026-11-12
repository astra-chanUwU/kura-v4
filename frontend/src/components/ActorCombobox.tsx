import { useEffect, useId, useRef, useState } from "react";
import type { KeyboardEvent } from "react";
import type { HistoryActor } from "../types/api";
import styles from "./ActorCombobox.module.css";

/**
 * Person filter for the staff change log. The parent debounces the typed
 * query and supplies matches. Nothing here lists every account.
 */
export function ActorCombobox({
  value,
  onChange,
  query,
  onQueryChange,
  suggestions,
  loading = false,
}: {
  value: HistoryActor | null;
  onChange: (next: HistoryActor | null) => void;
  query: string;
  onQueryChange: (next: string) => void;
  suggestions: HistoryActor[];
  loading?: boolean;
}) {
  const [dismissed, setDismissed] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const wrapRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const focusAfterClear = useRef(false);
  const listId = useId();

  const suggestionKey = suggestions.map((actor) => actor.id).join("\0");
  const [trackedSuggestions, setTrackedSuggestions] = useState(suggestionKey);
  let highlightIndex = highlight;
  if (trackedSuggestions !== suggestionKey) {
    highlightIndex = 0;
    setTrackedSuggestions(suggestionKey);
    setHighlight(0);
  }

  const trimmed = query.trim();
  const browsing = value === null && !dismissed && trimmed !== "";
  const showOptions = browsing && suggestions.length > 0;
  const showEmpty = browsing && !loading && suggestions.length === 0;
  const popup = showOptions || showEmpty;
  const active = suggestions.length === 0 ? 0 : Math.min(highlightIndex, suggestions.length - 1);
  const activeActor = showOptions ? suggestions[active] : undefined;
  const suffix = activeActor ? prefixSuffix(query, activeActor.username) : "";

  useEffect(() => {
    if (value === null && focusAfterClear.current) {
      focusAfterClear.current = false;
      inputRef.current?.focus();
    }
  }, [value]);

  useEffect(() => {
    if (!popup) return;
    function onPointerDown(event: PointerEvent) {
      if (wrapRef.current && !wrapRef.current.contains(event.target as Node)) {
        setDismissed(true);
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [popup]);

  function choose(actor: HistoryActor) {
    setDismissed(true);
    setHighlight(0);
    onQueryChange("");
    onChange(actor);
  }

  function clear() {
    focusAfterClear.current = true;
    setDismissed(true);
    onQueryChange("");
    onChange(null);
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      if (trimmed === "" || suggestions.length === 0) return;
      event.preventDefault();
      const delta = event.key === "ArrowDown" ? 1 : -1;
      if (!showOptions) {
        setDismissed(false);
        setHighlight(event.key === "ArrowDown" ? 0 : suggestions.length - 1);
        return;
      }
      setHighlight((current) => {
        const index = Math.min(current, suggestions.length - 1);
        return (index + delta + suggestions.length) % suggestions.length;
      });
      return;
    }
    if (event.key === "Enter" || (event.key === "Tab" && suffix)) {
      if (!showOptions) return;
      event.preventDefault();
      const pick = suggestions[active];
      if (pick) choose(pick);
      return;
    }
    if (event.key === "ArrowRight" && suffix && caretAtEnd(event.currentTarget)) {
      event.preventDefault();
      const pick = suggestions[active];
      if (pick) choose(pick);
      return;
    }
    if (event.key === "Escape" && popup) {
      event.preventDefault();
      setDismissed(true);
      return;
    }
    if (event.key === "Tab") setDismissed(true);
  }

  return (
    <div ref={wrapRef} className={popup ? `${styles.wrap} ${styles.open}` : styles.wrap}>
      {value ? (
        <div className={styles.selected}>
          <span className={styles.username}>@{value.username}</span>
          <button
            type="button"
            className={styles.clear}
            aria-label="Show everyone's changes"
            onClick={clear}
          >
            ×
          </button>
        </div>
      ) : (
        <div className={styles.field}>
          <input
            ref={inputRef}
            type="text"
            className={styles.input}
            role="combobox"
            aria-label="Person"
            aria-autocomplete="both"
            aria-expanded={popup}
            aria-controls={showOptions ? listId : undefined}
            aria-activedescendant={showOptions ? `${listId}-${active}` : undefined}
            aria-busy={loading ? true : undefined}
            placeholder="Everyone"
            autoComplete="off"
            autoCorrect="off"
            spellCheck={false}
            value={query}
            onChange={(event) => {
              setDismissed(false);
              setHighlight(0);
              onQueryChange(event.target.value);
            }}
            onKeyDown={onKeyDown}
          />
          {suffix ? (
            <span className={styles.completion} aria-hidden="true">
              <span>{query}</span>
              <span className={styles.completionRest}>{suffix}</span>
            </span>
          ) : null}
        </div>
      )}
      {showOptions ? (
        /* oxlint-disable-next-line jsx-a11y/no-noninteractive-element-to-interactive-role -- ul is the standard listbox container; its options are buttons */
        <ul id={listId} role="listbox" aria-label="Person" className={styles.menu}>
          {suggestions.map((actor, index) => (
            <li key={actor.id} role="presentation">
              <button
                type="button"
                id={`${listId}-${index}`}
                role="option"
                aria-selected={index === active}
                tabIndex={-1}
                className={styles.option}
                onMouseEnter={() => setHighlight(index)}
                onMouseDown={(event) => event.preventDefault()}
                onClick={() => choose(actor)}
              >
                <ActorName username={actor.username} typed={trimmed} />
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      {showEmpty ? (
        <p className={styles.empty} role="status">
          No one named that.
        </p>
      ) : null}
    </div>
  );
}

// The rest of a username that continues what was typed, or "" when the
// highlighted person is not a prefix match (a match in the middle stays
// in the menu only).
function prefixSuffix(typed: string, username: string): string {
  if (typed === "" || typed !== typed.trim()) return "";
  if (username.length <= typed.length) return "";
  if (!username.toLowerCase().startsWith(typed.toLowerCase())) return "";
  return username.slice(typed.length);
}

function caretAtEnd(input: HTMLInputElement): boolean {
  return input.selectionStart === input.value.length && input.selectionEnd === input.value.length;
}

function ActorName({ username, typed }: { username: string; typed: string }) {
  const at = typed === "" ? -1 : username.toLowerCase().indexOf(typed.toLowerCase());
  if (at < 0) return <>@{username}</>;
  return (
    <>
      @{username.slice(0, at)}
      <mark className={styles.hit}>{username.slice(at, at + typed.length)}</mark>
      {username.slice(at + typed.length)}
    </>
  );
}
