import {
  Children,
  Fragment,
  isValidElement,
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import type { CSSProperties, KeyboardEvent, ReactNode } from "react";
import styles from "./ThemedSelect.module.css";

/**
 * A fully themeable dropdown (ARIA listbox-button pattern) that keeps the
 * same public API as the old styled native <select>: options come from
 * <option> children, an optional `data-swatch` prop renders a color dot,
 * and `name` renders a hidden input so <form> + FormData callers keep
 * working. Supports controlled (`value`) and uncontrolled (`defaultValue`)
 * usage; `onChange` fires with the selected value.
 *
 * `popup` portals a centered dialog, because the header's backdrop-filter
 * would trap a position:fixed menu. That dialog can be searched. `compact`
 * draws the trigger as a square swatch or font sample. `embedded` draws the
 * same choices in place, for a preferences section.
 */
export function ThemedSelect({
  name,
  label,
  defaultValue,
  value,
  onChange,
  align = "start",
  popup = false,
  compact = false,
  embedded = false,
  children,
}: {
  name?: string;
  label: string;
  defaultValue?: string;
  value?: string;
  onChange?: (value: string) => void;
  /** Open toward the end edge so a control at the right of the header stays on screen. */
  align?: "start" | "end";
  /** Center the choices on the page instead of dropping a menu from the button. */
  popup?: boolean;
  /** Square trigger: swatch or "Ag" only, with no name and no chevron. */
  compact?: boolean;
  /** Render the choices in place instead of behind a button. */
  embedded?: boolean;
  children: React.ReactNode;
}) {
  const options = useMemo(() => childOptions(children), [children]);
  const controlled = value !== undefined;
  const [uncontrolled, setUncontrolled] = useState(() => defaultValue ?? options[0]?.value ?? "");
  const current = controlled ? value : uncontrolled;
  const selectedIndex = options.findIndex((o) => o.value === current);
  const selected = selectedIndex >= 0 ? options[selectedIndex] : undefined;

  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(selectedIndex >= 0 ? selectedIndex : 0);
  const [query, setQuery] = useState("");
  const shown = useMemo(
    () => (popup ? filterOptions(options, query) : options),
    [options, popup, query],
  );

  const wrapRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const optionRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const restoreFocus = useRef(false);
  const listId = useId();
  const dialogId = useId();
  const typeState = useRef({ buf: "", timer: 0 });

  useEffect(
    () => () => {
      window.clearTimeout(typeState.current.timer);
    },
    [],
  );

  // Click outside closes the anchored menu. The centered popup handles
  // that itself; a document listener would see a portal click as outside
  // and close before the choice registers.
  useEffect(() => {
    if (!open || popup) return;
    function onPointerDown(e: PointerEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
    };
  }, [open, popup]);

  useLayoutEffect(() => {
    if (!open || !popup) {
      if (restoreFocus.current) {
        restoreFocus.current = false;
        buttonRef.current?.focus();
      }
      return;
    }
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    searchRef.current?.focus();
    const index = options.findIndex((option) => option.value === current);
    optionRefs.current[index]?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
    return () => {
      document.body.style.overflow = previous;
    };
  }, [open, popup]);

  function openList() {
    setQuery("");
    setActive(selectedIndex >= 0 ? selectedIndex : 0);
    typeState.current.buf = "";
    setOpen(true);
  }

  function dismiss() {
    setQuery("");
    setOpen(false);
  }

  function closePopup() {
    restoreFocus.current = true;
    dismiss();
  }

  function select(next: string) {
    if (!controlled) setUncontrolled(next);
    onChange?.(next);
    if (embedded) return;
    if (popup) {
      closePopup();
      return;
    }
    dismiss();
    buttonRef.current?.focus();
  }

  function updateQuery(next: string) {
    setQuery(next);
    const list = filterOptions(options, next);
    const index = list.findIndex((option) => option.value === current);
    setActive(index >= 0 ? index : 0);
  }

  function typeahead(ch: string) {
    const s = typeState.current;
    window.clearTimeout(s.timer);
    s.buf = (s.buf + ch).toLowerCase();
    s.timer = window.setTimeout(() => {
      s.buf = "";
    }, 500);
    const i = options.findIndex((o) => o.label.toLowerCase().startsWith(s.buf));
    if (i < 0) return;
    if (open) {
      setActive(i);
    } else {
      const match = options[i];
      if (match) select(match.value);
    }
  }

  function moveActive(next: number) {
    const count = shown.length;
    if (count === 0) return;
    const index = ((next % count) + count) % count;
    setActive(index);
    optionRefs.current[index]?.focus();
  }

  function onButtonKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    switch (e.key) {
      case "Enter":
      case " ":
        e.preventDefault();
        if (!open) {
          openList();
        } else if (!popup) {
          const picked = options[active];
          if (picked) select(picked.value);
        }
        break;
      case "ArrowDown":
        e.preventDefault();
        if (!open) {
          openList();
        } else if (!popup && options.length > 0) {
          setActive((a) => (a + 1) % options.length);
        } else if (popup) {
          moveActive(active + 1);
        }
        break;
      case "ArrowUp":
        e.preventDefault();
        if (!open) {
          openList();
        } else if (!popup && options.length > 0) {
          setActive((a) => (a - 1 + options.length) % options.length);
        } else if (popup) {
          moveActive(active - 1);
        }
        break;
      case "Home":
        if (open && shown.length > 0) {
          e.preventDefault();
          setActive(0);
        }
        break;
      case "End":
        if (open && shown.length > 0) {
          e.preventDefault();
          setActive(shown.length - 1);
        }
        break;
      case "Escape":
        if (open) {
          e.preventDefault();
          if (popup) closePopup();
          else dismiss();
        }
        break;
      case "Tab":
        if (open) dismiss();
        break;
      default:
        if (e.key.length === 1 && !(open && popup)) typeahead(e.key);
        break;
    }
  }

  function onSearchKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      if (shown.length === 0) return;
      setActive(0);
      optionRefs.current[0]?.focus();
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      closePopup();
    }
  }

  function onOptionKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    if (!popup && !embedded) return;
    const columns = columnCount(listRef.current);
    if (event.key === "Escape") {
      event.preventDefault();
      closePopup();
      return;
    }
    if (event.key === "ArrowRight") {
      event.preventDefault();
      moveActive(index + 1);
    } else if (event.key === "ArrowLeft") {
      event.preventDefault();
      moveActive(index - 1);
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      moveActive(index + columns);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      moveActive(index - columns);
    }
  }

  const palette = options.length > 0 && options.every((option) => option.swatch);
  const fontChoices = options.length > 0 && options.every((option) => option.font) && !palette;
  const painted = popup || embedded;
  const selectedShown = shown.findIndex((option) => option.value === current);
  const wrapClass = [
    styles.wrap,
    embedded ? styles.embedded : "",
    open ? styles.open : "",
    align === "end" ? styles.alignEnd : "",
  ]
    .filter(Boolean)
    .join(" ");
  const buttonClass = [styles.button, compact ? styles.compact : ""].filter(Boolean).join(" ");
  const listClass = painted
    ? fontChoices
      ? styles.fontGrid
      : styles.dialogGrid
    : palette
      ? `${styles.menu} ${styles.palette}`
      : styles.menu;

  const list = (
    <ul
      ref={listRef}
      id={listId}
      // oxlint-disable-next-line jsx-a11y/no-noninteractive-element-to-interactive-role -- ul is the standard listbox container; its options are buttons
      role="listbox"
      aria-label={label}
      className={listClass}
    >
      {shown.map((o, i) => {
        const themeTile = Boolean(o.swatch) && painted;
        const fontChip = Boolean(o.font) && !o.swatch && painted;
        const optionClass = [
          styles.option,
          themeTile ? styles.themeTile : "",
          fontChip ? styles.fontChip : "",
          i === active ? styles.active : "",
        ]
          .filter(Boolean)
          .join(" ");
        return (
          <li key={`${o.value}-${i}`} role="presentation">
            <button
              ref={(node) => {
                optionRefs.current[i] = node;
              }}
              type="button"
              id={`${listId}-${i}`}
              role="option"
              aria-selected={i === selectedShown}
              tabIndex={painted ? 0 : -1}
              className={optionClass}
              data-theme={themeTile ? o.value : undefined}
              data-font={o.font}
              onClick={() => {
                select(o.value);
              }}
              onMouseEnter={() => {
                setActive(i);
              }}
              onKeyDown={(event) => onOptionKeyDown(event, i)}
            >
              {themeTile ? (
                <>
                  <span className={styles.themeName}>{o.label}</span>
                  <span className={styles.bubbles} aria-hidden="true">
                    <span />
                    <span />
                    <span />
                  </span>
                </>
              ) : (
                <>
                  {o.swatch ? (
                    <span className={styles.swatch} style={swatchStyle(o)} aria-hidden="true" />
                  ) : null}
                  {o.font && !fontChip ? (
                    <span className={styles.sample} aria-hidden="true">
                      Ag
                    </span>
                  ) : null}
                  <span className={styles.optionLabel}>{o.label}</span>
                  <svg
                    className={styles.check}
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden="true"
                  >
                    <path d="m5 12.5 5 5 9-11" />
                  </svg>
                </>
              )}
            </button>
          </li>
        );
      })}
    </ul>
  );

  const searchField = (
    <input
      ref={searchRef}
      type="search"
      className={styles.search}
      aria-label={`Search ${label}`}
      placeholder={`Search ${label.toLowerCase()}`}
      value={query}
      autoComplete="off"
      spellCheck={false}
      onChange={(event) => {
        updateQuery(event.target.value);
      }}
      onKeyDown={onSearchKeyDown}
    />
  );

  return (
    <div
      ref={wrapRef}
      className={wrapClass}
      onBlur={(e) => {
        // Portal focus sits outside this wrap. Closing on blur would
        // dismiss the dialog as soon as the search field takes focus.
        if (popup) return;
        if (open && (!e.relatedTarget || !wrapRef.current?.contains(e.relatedTarget as Node))) {
          dismiss();
        }
      }}
    >
      {embedded ? (
        <div className={styles.embeddedBody}>
          {palette ? searchField : null}
          {palette ? (
            <div className={styles.dialogScroll}>
              {shown.length === 0 ? <p className={styles.empty}>No matches</p> : list}
            </div>
          ) : shown.length === 0 ? (
            <p className={styles.empty}>No matches</p>
          ) : (
            list
          )}
        </div>
      ) : null}
      {embedded ? null : (
      <button
        ref={buttonRef}
        type="button"
        className={buttonClass}
        aria-haspopup={popup ? "dialog" : "listbox"}
        aria-expanded={open}
        aria-controls={popup ? dialogId : listId}
        // oxlint-disable-next-line jsx-a11y/role-supports-aria-props -- listbox-button pattern: focus stays on this button while aria-activedescendant tracks the active option
        aria-activedescendant={!popup && open ? `${listId}-${active}` : undefined}
        aria-label={label}
        data-font={selected?.font}
        onClick={() => {
          if (open) {
            dismiss();
          } else {
            openList();
          }
        }}
        onKeyDown={onButtonKeyDown}
      >
        {selected?.swatch ? (
          <span className={styles.swatch} style={swatchStyle(selected)} aria-hidden="true" />
        ) : null}
        {compact && selected?.font && !selected.swatch ? (
          <span className={styles.sample} aria-hidden="true">
            Ag
          </span>
        ) : null}
        {compact ? null : <span className={styles.label}>{selected?.label ?? ""}</span>}
        {compact ? null : (
          <svg
            className={styles.chevron}
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            aria-hidden="true"
          >
            <path d="m6 9.5 6 6 6-6" />
          </svg>
        )}
      </button>
      )}
      {name !== undefined ? <input type="hidden" name={name} value={current} /> : null}
      {open && popup
        ? createPortal(
            // oxlint-disable-next-line jsx-a11y/no-static-element-interactions -- click on the scrim, outside the dialog, closes it
            <div
              className={styles.backdrop}
              onMouseDown={(event) => {
                if (event.target === event.currentTarget) closePopup();
              }}
            >
              <div
                id={dialogId}
                className={styles.dialog}
                role="dialog"
                aria-modal="true"
                aria-label={label}
              >
                <h2 className={styles.dialogTitle}>{label}</h2>
                {searchField}
                <div className={styles.dialogScroll}>
                  {shown.length === 0 ? <p className={styles.empty}>No matches</p> : list}
                </div>
              </div>
            </div>,
            document.body,
          )
        : null}
      {open && !popup ? list : null}
    </div>
  );
}

type OptionData = {
  value: string;
  label: string;
  swatch?: string;
  paper?: string;
  font?: string;
};

function filterOptions(options: OptionData[], query: string): OptionData[] {
  const q = query.trim().toLowerCase();
  if (q === "") return options;
  return options.filter(
    (option) => option.label.toLowerCase().includes(q) || option.value.toLowerCase().includes(q),
  );
}

function columnCount(list: HTMLElement | null): number {
  if (!list || typeof window.getComputedStyle !== "function") return 3;
  const columns = window.getComputedStyle(list).gridTemplateColumns;
  if (!columns || columns === "none") return 1;
  const count = columns.split(" ").filter((part) => part.trim() !== "").length;
  return count > 0 ? count : 1;
}

function swatchStyle(option: OptionData): CSSProperties {
  return {
    "--swatch": option.swatch,
    "--paper": option.paper ?? "var(--surface-soft)",
  } as CSSProperties;
}

function textOf(node: ReactNode): string {
  if (node === null || node === undefined || typeof node === "boolean") {
    return "";
  }
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }
  if (Array.isArray(node)) return node.map(textOf).join("");
  if (isValidElement(node)) {
    return textOf((node.props as { children?: ReactNode }).children);
  }
  return "";
}

function childOptions(nodes: ReactNode): OptionData[] {
  const out: OptionData[] = [];
  Children.forEach(nodes, (child) => {
    if (!isValidElement(child)) return;
    if (child.type === Fragment) {
      out.push(...childOptions((child.props as { children?: ReactNode }).children));
      return;
    }
    if (child.type !== "option") return;
    const props = child.props as {
      value?: unknown;
      children?: ReactNode;
      "data-swatch"?: unknown;
      "data-paper"?: unknown;
      "data-font"?: unknown;
    };
    const value = props.value === undefined || props.value === null ? "" : String(props.value);
    const label = textOf(props.children).trim() || value;
    const swatch = typeof props["data-swatch"] === "string" ? props["data-swatch"] : undefined;
    const paper = typeof props["data-paper"] === "string" ? props["data-paper"] : undefined;
    const font = typeof props["data-font"] === "string" ? props["data-font"] : undefined;
    out.push({ value, label, swatch, paper, font });
  });
  return out;
}
