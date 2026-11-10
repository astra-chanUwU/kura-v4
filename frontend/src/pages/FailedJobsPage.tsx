import { useState } from "react";
import { shortError, useToasts } from "../components/Toasts";
import { useFailedJobs } from "../hooks/useFailedJobs";
import { useMe } from "../hooks/useKura";
import { StaffSection } from "./StaffLayout";
import styles from "./FailedJobsPage.module.css";
import simple from "./Simple.module.css";

export function FailedJobsPage() {
  const { user } = useMe();
  const toasts = useToasts();
  const [offset, setOffset] = useState(0);
  const query = useFailedJobs(!!user, offset);
  if (!user) return null;

  const jobs = query.data?.jobs ?? [];
  const total = query.data?.total ?? 0;

  return (
    <StaffSection
      id="failed-jobs-title"
      title="Failed jobs"
      intro="Background work that used every attempt. Retry puts a job back on the queue. Original files are not touched."
    >
      {query.isPending && <p className={simple.muted}>Loading…</p>}
      {query.error && (
        <p role="alert" className={simple.error}>
          {shortError(query.error)}
        </p>
      )}
      {!query.isPending && jobs.length === 0 && (
        <p className={simple.muted}>No failed jobs.</p>
      )}
      {jobs.length > 0 && (
        <ul className={styles.list}>
          {jobs.map((job) => (
            <li key={job.id} className={styles.row}>
              <div>
                <strong>{job.kind}</strong>
                <span className={simple.muted}>
                  {" "}
                  #{job.id} · {job.attempts}/{job.max_attempts} attempts · {job.updated_at}
                </span>
                {job.last_error && <p className={styles.error}>{job.last_error}</p>}
              </div>
              <button
                type="button"
                disabled={query.retry.isPending}
                onClick={() =>
                  query.retry.mutate(job.id, {
                    onSuccess: () => toasts.push("Job queued again.", "success"),
                    onError: (error: unknown) => toasts.push(shortError(error), "error"),
                  })
                }
              >
                Retry
              </button>
            </li>
          ))}
        </ul>
      )}
      {total > 40 && (
        <div className={simple.actions}>
          <button type="button" disabled={offset === 0} onClick={() => setOffset((n) => Math.max(0, n - 40))}>
            Previous
          </button>
          <button type="button" disabled={offset + jobs.length >= total} onClick={() => setOffset((n) => n + 40)}>
            Next
          </button>
        </div>
      )}
    </StaffSection>
  );
}
