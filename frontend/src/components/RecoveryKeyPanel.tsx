import { useState } from "react";
import styles from "../pages/Simple.module.css";

/**
 * Shows a freshly issued recovery key exactly once. The key is never
 * stored client-side; the user saves it somewhere safe themselves.
 */
export function RecoveryKeyPanel({
  recoveryKey,
  onDone,
  doneLabel = "I saved it — continue",
}: {
  recoveryKey: string;
  onDone: () => void;
  doneLabel?: string;
}) {
  const [copied, setCopied] = useState(false);
  return (
    <div className={styles.card} role="alert">
      <h2>Save your recovery key</h2>
      <p className={styles.muted}>
        If you ever lose your password and your passkeys, this key is the only way back into your
        account. Kura can't show it again — save it somewhere safe, like a password manager or on
        paper.
      </p>
      <p>
        <code style={{ fontSize: "1.05rem", letterSpacing: "0.04em", wordBreak: "break-all" }}>
          {recoveryKey}
        </code>
      </p>
      <div className={styles.actions}>
        <button
          type="button"
          onClick={() => {
            void navigator.clipboard?.writeText(recoveryKey).then(() => setCopied(true));
          }}
        >
          {copied ? "Copied" : "Copy key"}
        </button>
        <button type="button" onClick={onDone}>
          {doneLabel}
        </button>
      </div>
    </div>
  );
}
