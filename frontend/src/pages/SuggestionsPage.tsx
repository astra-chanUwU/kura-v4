import { Link } from "@tanstack/react-router";
import { shortError, useToasts } from "../components/Toasts";
import { useSuggestionActions, useTagSuggestions } from "../hooks/useSuggestions";
import type { TagSuggestion } from "../types/suggestion";
import { StaffSection } from "./StaffLayout";
import simple from "./Simple.module.css";

function line(item: TagSuggestion): string {
  if (item.kind === "alias") {
    return `${item.source_tag} → ${item.target_tag}`;
  }
  return `${item.source_tag} implies ${item.target_tag}`;
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
      <ul className={simple.grid}>
        {items.map((item) => (
          <li key={item.id}>
            <strong>{item.kind}</strong>
            <span>
              {" "}
              <Link to="/tags/$tag" params={{ tag: item.target_tag }}>
                {line(item)}
              </Link>
            </span>
            <span className={simple.muted}>
              {" "}
              by @{item.author_username || "member"}
              {item.reason ? ` · ${item.reason}` : ""}
            </span>
            <div className={simple.actions}>
              <button
                type="button"
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
    </StaffSection>
  );
}
