import styles from "./ThemedSelect.module.css";

/**
 * A native <select> (accessible, keyboard-friendly) restyled to the
 * theme — appearance: none plus a custom chevron — instead of the
 * browser's default chrome.
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
  return (
    <span className={styles.wrap}>
      <select
        name={name}
        aria-label={label}
        defaultValue={defaultValue}
        value={value}
        onChange={onChange ? (e) => onChange(e.target.value) : undefined}
      >
        {children}
      </select>
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
    </span>
  );
}
