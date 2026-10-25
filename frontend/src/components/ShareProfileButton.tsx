import { useState } from "react";
import { IconCheck, IconLink } from "./icons";
import styles from "./ShareProfileButton.module.css";

/** Icon button that copies the public profile link to the clipboard. */
export function ShareProfileButton({ username }: { username: string }) {
  const [copied, setCopied] = useState(false);
  const url = `${window.location.origin}/u/${username}`;
  return (
    <span className={styles.wrap}>
      <button
        type="button"
        className={styles.button}
        title={`Copy link to @${username}'s profile`}
        aria-label={`Copy link to @${username}'s profile`}
        onClick={() => {
          void navigator.clipboard?.writeText(url).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
          });
        }}
      >
        {copied ? <IconCheck /> : <IconLink />}
      </button>
      {copied && <span className={styles.copied}>Link copied</span>}
    </span>
  );
}
