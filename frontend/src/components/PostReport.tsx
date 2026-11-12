import { useState, type FormEvent } from "react";
import { useReportActions } from "../hooks/useReports";
import { ThemedSelect } from "./ThemedSelect";
import { shortError } from "./Toasts";
import styles from "./PostReport.module.css";

const REASONS = [
  ["spam", "Spam"],
  ["wrong_rating", "Wrong rating"],
  ["illegal", "Illegal"],
  ["underage", "Underage"],
  ["other", "Other"],
] as const;

export function PostReport({ postId, signedIn }: { postId: string; signedIn: boolean }) {
  const { create } = useReportActions();
  const [reason, setReason] = useState<string>("spam");
  const [note, setNote] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  if (!signedIn) return null;

  function submit(event: FormEvent) {
    event.preventDefault();
    if (create.isPending) return;
    setError("");
    setMessage("");
    create.mutate(
      { postId, reason, note },
      {
        onSuccess: () => {
          setMessage("Report sent. A moderator will see it.");
          setNote("");
        },
        onError: (err) => setError(shortError(err)),
      },
    );
  }

  return (
    <form className={styles.form} onSubmit={submit}>
      <h2>Report</h2>
      <label className={styles.reason}>
        Reason
        <ThemedSelect label="Reason" value={reason} onChange={setReason}>
          {REASONS.map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </ThemedSelect>
      </label>
      <label>
        Note
        <input value={note} maxLength={500} onChange={(event) => setNote(event.target.value)} />
      </label>
      <button type="submit" disabled={create.isPending}>
        Send report
      </button>
      {message && <p>{message}</p>}
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
    </form>
  );
}
