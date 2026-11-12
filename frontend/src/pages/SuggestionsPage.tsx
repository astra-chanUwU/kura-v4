import { Link } from "@tanstack/react-router";
import { shortError, useToasts } from "../components/Toasts";
import { useSuggestionActions, useTagSuggestions } from "../hooks/useSuggestions";
import { StaffSection } from "./StaffLayout";
import styles from "./SuggestionsPage.module.css";
import simple from "./Simple.module.css";

function TagLink({ tag }: { tag: string }) {
  return (
    <Link to="/tags/$tag" params={{ tag }}>
      {tag}
    </Link>
  );
}

export function SuggestionsPage() {
  const toasts = useToasts();
  const query = useTagSuggestions(true);
  const { accept, reject } = useSuggestionActions();
  const items = query.data ?? [];

  return (
    <StaffSection
      id="suggestions-title"
      title="Suggestions"
      intro="Alias and implication proposals from members. Accepting one applies it. Rejecting one leaves the tags alone."
    >
      {query.isPending && <p className={simple.muted}>Loading…</p>}
      {query.error && (
        <p role="alert" className={simple.error}>
          {shortError(query.error)}
        </p>
      )}
      {!query.isPending && !query.error && items.length === 0 && (
        <p className={simple.muted}>No pending suggestions.</p>
      )}
      {items.length > 0 && (
        <ul className={styles.list}>
          {items.map((item) => (
            <li key={item.id} className={styles.row}>
              <div className={styles.body}>
                <p className={styles.rule}>
                  <span className={styles.kind}>{item.kind}</span>
                  <span>
                    <TagLink tag={item.source_tag} />
                    {item.kind === "alias" ? " → " : " implies "}
                    <TagLink tag={item.target_tag} />
                  </span>
                </p>
                <p className={styles.meta}>
                  by @{item.author_username || "member"}
                  {item.reason ? ` · ${item.reason}` : ""}
                </p>
              </div>
              <div className={styles.actions}>
                <button
                  type="button"
                  className={styles.accept}
                  disabled={accept.isPending || reject.isPending}
                  onClick={() =>
                    accept.mutate(item.id, {
                      onSuccess: () => toasts.push("Suggestion accepted.", "success"),
                      onError: (error: unknown) => toasts.push(shortError(error), "error"),
                    })
                  }
                >
                  Accept
                </button>
                <button
                  type="button"
                  className={styles.reject}
                  disabled={accept.isPending || reject.isPending}
                  onClick={() =>
                    reject.mutate(item.id, {
                      onSuccess: () => toasts.push("Suggestion rejected.", "success"),
                      onError: (error: unknown) => toasts.push(shortError(error), "error"),
                    })
                  }
                >
                  Reject
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </StaffSection>
  );
}
