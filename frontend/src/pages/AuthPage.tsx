import { Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { RecoveryKeyPanel } from "../components/RecoveryKeyPanel";
import { passkeysSupported, useAuthActions } from "../hooks/useKura";
import styles from "./Simple.module.css";

export function AuthPage({ mode }: { mode: "login" | "register" }) {
  const navigate = useNavigate();
  const auth = useAuthActions();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [method, setMethod] = useState<"passkey" | "password">("passkey");
  const [busy, setBusy] = useState(false);
  const [issuedKey, setIssuedKey] = useState("");
  const [error, setError] = useState("");
  const supported = passkeysSupported();
  const isRegister = mode === "register";

  const done = () => {
    void navigate({ to: "/" });
  };

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    const action =
      method === "passkey"
        ? isRegister
          ? auth.registerPasskey(username, displayName)
          : auth.loginPasskey()
        : isRegister
          ? auth.registerPassword(username, password, displayName)
          : auth.loginPassword(username, password);
    void action
      .then((me) => {
        if (isRegister && me?.recovery_key) {
          setIssuedKey(me.recovery_key);
        } else {
          done();
        }
      })
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : "Something went wrong."),
      )
      .finally(() => setBusy(false));
  };

  if (issuedKey) {
    return (
      <section className={styles.page} aria-labelledby="auth-title">
        <h1 id="auth-title">Account created</h1>
        <RecoveryKeyPanel recoveryKey={issuedKey} onDone={done} />
      </section>
    );
  }

  return (
    <section className={styles.page} aria-labelledby="auth-title">
      <h1 id="auth-title">{isRegister ? "Create your account" : "Welcome back"}</h1>
      <p className={styles.muted}>
        {isRegister
          ? "Pick how you'd like to sign in — you can add the other later."
          : "Use your passkey or password, whichever you set up."}
      </p>

      <div className={styles.tabs} role="tablist" aria-label="Sign-in method">
        <button
          type="button"
          role="tab"
          aria-selected={method === "passkey"}
          className={method === "passkey" ? styles.activeTab : ""}
          onClick={() => setMethod("passkey")}
        >
          Passkey {supported ? "" : "(not supported here)"}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={method === "password"}
          className={method === "password" ? styles.activeTab : ""}
          onClick={() => setMethod("password")}
        >
          Password
        </button>
      </div>
      {method === "passkey" && (
        <p className={styles.hint}>
          {isRegister
            ? "Recommended: no password to remember or leak — your device keeps the key."
            : "No username needed — pick the saved passkey for this site and you're in."}
        </p>
      )}

      <form onSubmit={submit} className={styles.form}>
        {(isRegister || method === "password") && (
          <label>
            Username
            <input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              required
              placeholder="e.g. astro_chan"
            />
          </label>
        )}
        {isRegister && (
          <label>
            Display name (optional)
            <input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              autoComplete="name"
              placeholder="How people see you"
            />
          </label>
        )}
        {method === "password" && (
          <label>
            Password
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete={isRegister ? "new-password" : "current-password"}
              required
              minLength={isRegister ? 15 : undefined}
              placeholder={isRegister ? "15+ characters — a short sentence works great" : undefined}
            />
          </label>
        )}
        {error && (
          <p role="alert" className={styles.error}>
            {error}
          </p>
        )}
        <button type="submit" disabled={busy || (method === "passkey" && !supported)}>
          {busy
            ? "One moment…"
            : method === "passkey"
              ? isRegister
                ? "Create account with passkey"
                : "Sign in with passkey"
              : isRegister
                ? "Create account"
                : "Sign in"}
        </button>
      </form>

      <p className={styles.muted}>
        {isRegister ? (
          <>
            Already have an account? <Link to="/login">Sign in</Link>
          </>
        ) : (
          <>
            New here? <Link to="/register">Create an account</Link>
            <br />
            Lost your password and passkeys? <Link to="/recover">Use a recovery key</Link>
          </>
        )}
      </p>
    </section>
  );
}
