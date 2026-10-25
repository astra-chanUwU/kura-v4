import { Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { RecoveryKeyPanel } from "../components/RecoveryKeyPanel";
import { useAuthActions } from "../hooks/useKura";
import styles from "./Simple.module.css";

export function RecoverPage() {
  const navigate = useNavigate();
  const auth = useAuthActions();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [newKey, setNewKey] = useState("");

  if (newKey) {
    return (
      <section className={styles.page} aria-labelledby="recover-title">
        <h1 id="recover-title">You're back in</h1>
        <p className={styles.muted}>
          Your password was reset, your old passkeys and sessions were revoked, and your old
          recovery key no longer works. Here's your new one:
        </p>
        <RecoveryKeyPanel recoveryKey={newKey} onDone={() => void navigate({ to: "/me" })} />
      </section>
    );
  }

  return (
    <section className={styles.page} aria-labelledby="recover-title">
      <h1 id="recover-title">Recover your account</h1>
      <p className={styles.muted}>
        Use the recovery key you saved when you created your account. This sets a new password,
        signs out everywhere, and removes your passkeys (you can add new ones after).
      </p>
      <form
        className={styles.form}
        onSubmit={(event) => {
          event.preventDefault();
          const form = new FormData(event.currentTarget);
          setBusy(true);
          setError("");
          void auth
            .recoverAccount(
              String(form.get("username") ?? ""),
              String(form.get("recovery_key") ?? ""),
              String(form.get("new_password") ?? ""),
            )
            .then((me) => {
              if (me?.recovery_key) setNewKey(me.recovery_key);
              else void navigate({ to: "/me" });
            })
            .catch((err: unknown) =>
              setError(err instanceof Error ? err.message : "Something went wrong."),
            )
            .finally(() => setBusy(false));
        }}
      >
        <label>
          Username
          <input name="username" required autoComplete="username" placeholder="e.g. astro_chan" />
        </label>
        <label>
          Recovery key
          <input
            name="recovery_key"
            required
            autoComplete="off"
            placeholder="XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX"
          />
        </label>
        <label>
          New password
          <input
            type="password"
            name="new_password"
            required
            minLength={15}
            autoComplete="new-password"
            placeholder="15+ characters — a short sentence works great"
          />
        </label>
        {error && (
          <p role="alert" className={styles.error}>
            {error}
          </p>
        )}
        <button type="submit" disabled={busy}>
          {busy ? "One moment…" : "Set new password and sign in"}
        </button>
      </form>
      <p className={styles.muted}>
        Remembered it after all? <Link to="/login">Back to sign in</Link>
      </p>
    </section>
  );
}
