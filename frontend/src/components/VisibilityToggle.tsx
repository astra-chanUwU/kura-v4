import { useEffect, useRef, useState } from "react";
import { IconGlobe, IconLock } from "./icons";
import styles from "./VisibilityToggle.module.css";

/** Private or public. A list has only these two, so this is a toggle, not a menu. */
export function VisibilityToggle({
  name,
  label = "Visibility",
  value,
  defaultValue = "private",
  onChange,
}: {
  name?: string;
  label?: string;
  value?: string;
  defaultValue?: string;
  onChange?: (value: string) => void;
}) {
  const controlled = value !== undefined;
  const [uncontrolled, setUncontrolled] = useState(defaultValue);
  const current = controlled ? value : uncontrolled;
  const rootRef = useRef<HTMLDivElement>(null);

  // A form reset (creating a list clears the row) should return to the default.
  useEffect(() => {
    if (controlled) return;
    const form = rootRef.current?.closest("form");
    if (!form) return;
    const onReset = () => setUncontrolled(defaultValue);
    form.addEventListener("reset", onReset);
    return () => form.removeEventListener("reset", onReset);
  }, [controlled, defaultValue]);

  function pick(next: string) {
    if (!controlled) setUncontrolled(next);
    onChange?.(next);
  }

  return (
    <div ref={rootRef} className={styles.toggle} role="group" aria-label={label}>
      {name && <input type="hidden" name={name} value={current} />}
      <button
        type="button"
        className={styles.option}
        aria-pressed={current === "private"}
        title="Only you"
        onClick={() => pick("private")}
      >
        <IconLock /> Private
      </button>
      <button
        type="button"
        className={styles.option}
        aria-pressed={current === "public"}
        title="Anyone with the link"
        onClick={() => pick("public")}
      >
        <IconGlobe /> Public
      </button>
    </div>
  );
}
