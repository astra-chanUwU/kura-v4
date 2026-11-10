import { shortError, useToasts } from "../components/Toasts";
import { useOpenReports, useReportActions } from "../hooks/useReports";
import { StaffSection } from "./StaffLayout";
import simple from "./Simple.module.css";

export function ReportsPage() {
  const toasts = useToasts();
  const { reports, loading, error } = useOpenReports();
  const { dismiss } = useReportActions();

  return (
    <StaffSection
      id="reports-title"
      title="Reports"
      intro="Open reports from members, newest first. Dismissing one does not hide the post."
    >
      {loading && <p className={simple.muted}>Loading…</p>}
      {error && (
        <p role="alert" className={simple.error}>
          {shortError(error)}
        </p>
      )}
      {!loading && reports.length === 0 && <p className={simple.muted}>No open reports.</p>}
      <ul className={simple.grid}>
        {reports.map((report) => (
          <li key={report.id}>
            <strong>{report.reason}</strong>
            <span className={simple.muted}>
              {" "}
              by @{report.reporter_username || "member"} · {report.note}
            </span>
            <div className={simple.actions}>
              <button
                type="button"
                disabled={dismiss.isPending && dismiss.variables === report.id}
                onClick={() => {
                  dismiss.mutate(report.id, {
                    onSuccess: () => toasts.push("Report dismissed.", "success"),
                    onError: (err) => toasts.push(shortError(err), "error"),
                  });
                }}
              >
                Dismiss
              </button>
            </div>
          </li>
        ))}
      </ul>
    </StaffSection>
  );
}
