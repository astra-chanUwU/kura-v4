import { Link } from "@tanstack/react-router";
import { useState, type ReactNode } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { shortError, useToasts } from "../components/Toasts";
import { useSuggestionActions, useTagSuggestions } from "../hooks/useSuggestions";
import type { TagSuggestion } from "../types/suggestion";
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

function when(value: string | undefined): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString();
}

function pendingMeta(item: TagSuggestion): string {
  const author = `@${item.author_username || "member"}`;
  return item.reason ? `by ${author} · ${item.reason}` : `by ${author}`;
}

function historyMeta(item: TagSuggestion): string {
  const resolver = item.resolver_username ? ` by @${item.resolver_username}` : "";
  const at = when(item.resolved_at);
  const dated = at ? ` on ${at}` : "";
  const reason = item.reason ? ` · ${item.reason}` : "";
  return `${item.status}${resolver}${dated} · proposed by @${item.author_username || "member"}${reason}`;
}

function undoCopy(item: TagSuggestion): { title: string; message: string } {
  if (item.kind === "alias") {
    return {
      title: `Reject alias ${item.source_tag}?`,
      message: `${item.source_tag} stops pointing at ${item.target_tag}. Pictures already moved to ${item.target_tag} stay there.`,
    };
  }
  return {
    title: `Reject implication ${item.source_tag}?`,
    message: `${item.source_tag} will no longer add ${item.target_tag}. Pictures that already have ${item.target_tag} keep it.`,
  };
}

function GroupHeading({ level, children }: { level: "h3" | "h4"; children: ReactNode }) {
  const Tag = level;
  return <Tag>{children}</Tag>;
}

function SuggestionGroup({
  title,
  level,
  note,
  query,
  empty,
  meta,
  actions,
}: {
  title: string;
  level: "h3" | "h4";
  note?: string;
  query: ReturnType<typeof useTagSuggestions>;
  empty: string;
  meta: (item: TagSuggestion) => string;
  actions: (item: TagSuggestion) => ReactNode;
}) {
  const items = query.data ?? [];
  return (
    <div className={styles.group}>
      <GroupHeading level={level}>{title}</GroupHeading>
      {note && <p className={styles.note}>{note}</p>}
      {query.isPending && <p className={styles.note}>Loading…</p>}
      {query.error && (
        <p role="alert" className={simple.error}>
          {shortError(query.error)}
        </p>
      )}
      {!query.isPending && !query.error && items.length === 0 && (
        <p className={styles.note}>{empty}</p>
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
                <p className={styles.meta}>{meta(item)}</p>
              </div>
              <div className={styles.actions}>{actions(item)}</div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export function SuggestionsPage() {
  const toasts = useToasts();
  const pending = useTagSuggestions("pending");
  const accepted = useTagSuggestions("accepted");
  const rejected = useTagSuggestions("rejected");
  const { accept, reject } = useSuggestionActions();
  const [undo, setUndo] = useState<TagSuggestion | null>(null);
  const busy = accept.isPending || reject.isPending;
  const prompt = undo ? undoCopy(undo) : null;

  const runAccept = (item: TagSuggestion) => {
    accept.mutate(item.id, {
      onSuccess: () =>
        toasts.push(
          item.status === "rejected"
            ? "Suggestion accepted. The alias or implication is applied."
            : "Suggestion accepted.",
          "success",
        ),
      onError: (error: unknown) => toasts.push(shortError(error), "error"),
    });
  };
  const runReject = (item: TagSuggestion) => {
    reject.mutate(item.id, {
      onSuccess: () =>
        toasts.push(
          item.status === "accepted"
            ? "Suggestion rejected. The rule was removed."
            : "Suggestion rejected.",
          "success",
        ),
      onError: (error: unknown) => toasts.push(shortError(error), "error"),
    });
  };

  return (
    <StaffSection
      id="suggestions-title"
      title="Suggestions"
      intro="Alias and implication proposals from members. Accepting one applies it. Rejecting one leaves the tags alone. Decided suggestions stay in the history, and that decision can be changed."
    >
      <SuggestionGroup
        title="Pending"
        level="h3"
        query={pending}
        empty="No pending suggestions."
        meta={pendingMeta}
        actions={(item) => (
          <>
            <button
              type="button"
              className={styles.accept}
              disabled={busy}
              onClick={() => runAccept(item)}
            >
              Accept
            </button>
            <button
              type="button"
              className={styles.reject}
              disabled={busy}
              onClick={() => runReject(item)}
            >
              Reject
            </button>
          </>
        )}
      />
      <div className={styles.history}>
        <h3>History</h3>
        <SuggestionGroup
          title="Accepted"
          level="h4"
          note="Rejecting one removes that alias or implication. Pictures already moved onto a canonical tag stay there, and pictures that already gained an implied tag keep it."
          query={accepted}
          empty="No accepted suggestions."
          meta={historyMeta}
          actions={(item) => (
            <button
              type="button"
              className={styles.reject}
              disabled={busy}
              onClick={() => setUndo(item)}
            >
              Reject instead
            </button>
          )}
        />
        <SuggestionGroup
          title="Rejected"
          level="h4"
          note="Accepting one applies the alias or implication."
          query={rejected}
          empty="No rejected suggestions."
          meta={historyMeta}
          actions={(item) => (
            <button
              type="button"
              className={styles.accept}
              disabled={busy}
              onClick={() => runAccept(item)}
            >
              Accept instead
            </button>
          )}
        />
      </div>
      <ConfirmDialog
        open={undo !== null}
        title={prompt?.title ?? ""}
        message={prompt?.message ?? ""}
        confirmLabel="Reject instead"
        destructive
        onCancel={() => setUndo(null)}
        onConfirm={() => {
          const item = undo;
          setUndo(null);
          if (item) runReject(item);
        }}
      />
    </StaffSection>
  );
}
