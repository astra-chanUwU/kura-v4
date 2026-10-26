import {
  Children,
  Fragment,
  isValidElement,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from "react";
import type { CSSProperties, KeyboardEvent, ReactNode } from "react";
import styles from "./ThemedSelect.module.css";

/**
 * A fully themeable dropdown (ARIA listbox-button pattern) that keeps the
 * same public API as the old styled native <select>: options come from
 * <option> children, an optional `data-swatch` prop renders a color dot,
 * and `name` renders a hidden input so <form> + FormData callers keep
 * working. Supports controlled (`value`) and uncontrolled (`defaultValue`)
 * usage; `onChange` fires with the selected value.
 */
export function ThemedSelect({
  name,
  label,
  defaultValue,
  value,
  onChange,
  children,
}: {
  name?: string;
  label: string;
  defaultValue?: string;
  value?: string;
  onChange?: (value: string) => void;
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

  const wrapRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const listId = useId();
  const typeState = useRef({ buf: "", timer: 0 });

  useEffect(
    () => () => {
      window.clearTimeout(typeState.current.timer);
    },
    [],
  );

  // Click outside closes the menu.
  useEffect(() => {
    if (!open) return;
    function onPointerDown(e: PointerEvent) {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
    };
  }, [open]);

  function openList() {
    setActive(selectedIndex >= 0 ? selectedIndex : 0);
    typeState.current.buf = "";
    setOpen(true);
  }

  function select(next: string) {
    if (!controlled) setUncontrolled(next);
    onChange?.(next);
    setOpen(false);
    buttonRef.current?.focus();
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

  function onButtonKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    switch (e.key) {
      case "Enter":
      case " ":
        e.preventDefault();
        if (!open) {
          openList();
        } else {
          const current = options[active];
          if (current) select(current.value);
        }
        break;
      case "ArrowDown":
        e.preventDefault();
        if (!open) {
          openList();
        } else if (options.length > 0) {
          setActive((a) => (a + 1) % options.length);
        }
        break;
      case "ArrowUp":
        e.preventDefault();
        if (!open) {
          openList();
        } else if (options.length > 0) {
          setActive((a) => (a - 1 + options.length) % options.length);
        }
        break;
      case "Home":
        if (open && options.length > 0) {
          e.preventDefault();
          setActive(0);
        }
        break;
      case "End":
        if (open && options.length > 0) {
          e.preventDefault();
          setActive(options.length - 1);
        }
        break;
      case "Escape":
        if (open) {
          e.preventDefault();
          setOpen(false);
        }
        break;
      case "Tab":
        if (open) setOpen(false);
        break;
      default:
        if (e.key.length === 1) typeahead(e.key);
        break;
    }
  }

  return (
    <div
      ref={wrapRef}
      className={styles.wrap}
      onBlur={(e) => {
        if (open && (!e.relatedTarget || !wrapRef.current?.contains(e.relatedTarget as Node))) {
          setOpen(false);
        }
      }}
    >
      <button
        ref={buttonRef}
        type="button"
        className={styles.button}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        // oxlint-disable-next-line jsx-a11y/role-supports-aria-props -- listbox-button pattern: focus stays on this button while aria-activedescendant tracks the active option
        aria-activedescendant={open ? `${listId}-${active}` : undefined}
        aria-label={label}
        onClick={() => {
          if (open) {
            setOpen(false);
          } else {
            openList();
          }
        }}
        onKeyDown={onButtonKeyDown}
      >
        {selected?.swatch ? (
          <span
            className={styles.swatch}
            style={{ "--swatch": selected.swatch } as CSSProperties}
            aria-hidden="true"
          />
        ) : null}
        <span className={styles.label}>{selected?.label ?? ""}</span>
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
      </button>
      {name !== undefined ? <input type="hidden" name={name} value={current} /> : null}
      {open ? (
        /* oxlint-disable-next-line jsx-a11y/no-noninteractive-element-to-interactive-role -- ul is the standard listbox container; its options are buttons */
        <ul id={listId} role="listbox" aria-label={label} className={styles.menu}>
          {options.map((o, i) => (
            <li key={`${o.value}-${i}`} role="presentation">
              <button
                type="button"
                id={`${listId}-${i}`}
                role="option"
                aria-selected={i === selectedIndex}
                tabIndex={-1}
                className={i === active ? `${styles.option} ${styles.active}` : styles.option}
                onClick={() => {
                  select(o.value);
                }}
                onMouseEnter={() => {
                  setActive(i);
                }}
              >
                {o.swatch ? (
                  <span
                    className={styles.swatch}
                    style={{ "--swatch": o.swatch } as CSSProperties}
                    aria-hidden="true"
                  />
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
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

type OptionData = { value: string; label: string; swatch?: string };

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
    };
    const value = props.value === undefined || props.value === null ? "" : String(props.value);
    const label = textOf(props.children).trim() || value;
    const swatch =
      typeof props["data-swatch"] === "string" ? (props["data-swatch"] as string) : undefined;
    out.push({ value, label, swatch });
  });
  return out;
}
