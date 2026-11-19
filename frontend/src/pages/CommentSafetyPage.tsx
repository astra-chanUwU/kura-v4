import { Link } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { IconCheck, IconEye, IconTrash, IconUndo } from "../components/icons";
import { Pager } from "../components/Pager";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import { clampPage, DEFAULT_PAGE_SIZE, pageCount } from "../domain/paging";
import { useDebouncedValue } from "../hooks/useBoardData";
import {
  useCommentSafety,
  useCommentSafetyActions,
  useCommentSafetyResolvers,
  type CommentSafetyFilter,
} from "../hooks/useCommentSafety";
import { usePrefetchPage } from "../hooks/usePrefetchPage";
import type { CommentSafetyDecision, CommentSafetyEvent, CommentSafetyResolver } from "../types/commentSafety";
import { StaffSection } from "./StaffLayout";
import styles from "./CommentSafetyPage.module.css";
import simple from "./Simple.module.css";

const categories = [
  "phone",
  "telegram",
  "whatsapp",
  "discord",
  "signal",
  "snapchat",
  "kik",
  "email",
  "solicitation",
] as const;

function actionLabel(action: CommentSafetyEvent["action"]): string {
  if (action === "blocked") return "Blocked";
  return "Flagged";
}

function eventSubject(event: CommentSafetyEvent): string {
  const author = `@${event.author_username}`;
  if (event.categories.length === 0) return author;
  return `${event.categories.join(", ")} · ${author}`;
}

function stepVerb(event: CommentSafetyEvent, step: CommentSafetyDecision): string {
  if (step.action === "removed") return event.action === "flagged" ? "deleted" : "removed";
  if (step.action === "restored") return "restored";
  if (step.action === "reopened") return "reopened";
  return "dismissed";
}

function personVerb(status: string, gate: string): string {
  if (status === "removed") return gate === "flagged" ? "deleted" : "removed";
  if (status === "restored") return "restored";
  if (status === "reopened") return "reopened";
  return "dismissed";
}

function decisionPhrase(person: CommentSafetyResolver): string {
  const verb = personVerb(person.last_status, person.last_action);
  const what =
    person.last_categories.length > 0 ? person.last_categories.join(", ") : person.last_body.trim();
  return what === "" ? verb : `${verb} ${what}`;
}

function decisionWhen(iso: string): string {
  if (iso === "") return "";
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString();
}

function DecisionTrail({ event }: { event: CommentSafetyEvent }) {
  const steps = event.decisions ?? [];
  if (steps.length === 0) {
    if (event.status === "open" || event.resolver_username === "") return null;
    const verb = event.status === "removed" ? (event.action === "flagged" ? "deleted" : "removed") : "dismissed";
    const when = decisionWhen(event.resolved_at);
    return (
      <span className={styles.trailLatest}>
        {" · "}
        {verb} by @{event.resolver_username}
        {when !== "" && (
          <>
            {" · "}
            <time dateTime={event.resolved_at}>{when}</time>
          </>
        )}
      </span>
    );
  }
  const last = steps[steps.length - 1];
  const when = decisionWhen(last.created_at);
  return (
    <span className={styles.trail}>
      {" · "}
      {steps.map((step, index) => {
        const latest = index === steps.length - 1;
        return (
          <span key={`${step.created_at}-${step.actor_id}-${index}`}>
            {index > 0 && " · "}
            <span className={latest ? styles.trailLatest : undefined}>
              {stepVerb(event, step)} by @{step.actor_username}
            </span>
          </span>
        );
      })}
      {when !== "" && (
        <>
          {" · "}
          <time dateTime={last.created_at}>{when}</time>
        </>
      )}
    </span>
  );
}

function matchLabel(total: number): string {
  return total === 1 ? "1 match" : `${total} matches`;
}

function restoresComment(event: CommentSafetyEvent): boolean {
  return event.status === "removed" && event.action === "flagged";
}

export function CommentSafetyPage() {
  const toasts = useToasts();
  const [status, setStatus] = useState<CommentSafetyFilter["status"]>("open");
  const [action, setAction] = useState<CommentSafetyFilter["action"]>("");
  const [category, setCategory] = useState("");
  const [sort, setSort] = useState<CommentSafetyFilter["sort"]>("newest");
  const [q, setQ] = useState("");
  const [resolver, setResolver] = useState("");
  const [page, setPage] = useState(1);
  const [removeID, setRemoveID] = useState<string | null>(null);
  const [reopenID, setReopenID] = useState<string | null>(null);
  const pageSize = DEFAULT_PAGE_SIZE;
  const debouncedQ = useDebouncedValue(q, 300);
  const submittedQ = debouncedQ.trim();
  const filter: CommentSafetyFilter = {
    status,
    action,
    category,
    q: submittedQ,
    resolver,
    sort,
  };
  const queue = useCommentSafety(filter, page, pageSize);
  const deciders = useCommentSafetyResolvers();
  const { dismiss, remove, reopen } = useCommentSafetyActions();
  usePrefetchPage(queue.prefetchPage, page, pageCount(queue.total, pageSize), queue.isSuccess);
  useEffect(() => {
    if (!queue.isSuccess) return;
    const clamped = clampPage(page, queue.total, pageSize);
    if (clamped !== page) setPage(clamped);
  }, [queue.isSuccess, queue.total, page, pageSize]);
  const lastPage = useRef(page);
  useEffect(() => {
    if (lastPage.current === page) return;
    lastPage.current = page;
    window.scrollTo({ top: 0, behavior: "smooth" });
  }, [page]);

  const pendingRemove = queue.events.find((event) => event.id === removeID) ?? null;
  const pendingReopen = queue.events.find((event) => event.id === reopenID) ?? null;
  const filtered =
    status !== "open" || action !== "" || category !== "" || submittedQ !== "" || resolver !== "" || sort !== "newest";

  return (
    <StaffSection
      id="comment-safety-title"
      title="Comment safety"
      intro="Blocked comments were not published. Flagged comments stay visible until a moderator removes them. A review keeps every step, so a comment deleted, restored by one moderator, and deleted again still shows the person who deleted it last."
    >
      <div className={styles.filterRow}>
        <input
          type="text"
          className={styles.search}
          value={q}
          onChange={(event) => {
            setQ(event.target.value);
            setPage(1);
          }}
          placeholder="Comment or author"
          aria-label="Search reviews"
        />
        <ThemedSelect
          label="Status"
          value={status}
          onChange={(next) => {
            setStatus(next === "dismissed" || next === "removed" || next === "any" ? next : "open");
            setPage(1);
          }}
        >
          <option value="open">Open</option>
          <option value="dismissed">Dismissed</option>
          <option value="removed">Removed</option>
          <option value="any">Any status</option>
        </ThemedSelect>
        <ThemedSelect
          label="Kind"
          value={action}
          onChange={(next) => {
            setAction(next === "blocked" || next === "flagged" ? next : "");
            setPage(1);
          }}
        >
          <option value="">Any kind</option>
          <option value="blocked">Blocked</option>
          <option value="flagged">Flagged</option>
        </ThemedSelect>
        <ThemedSelect
          label="Category"
          value={category}
          onChange={(next) => {
            setCategory(categories.includes(next as (typeof categories)[number]) ? next : "");
            setPage(1);
          }}
        >
          <option value="">Any category</option>
          {categories.map((item) => (
            <option key={item} value={item}>
              {item}
            </option>
          ))}
        </ThemedSelect>
        <ThemedSelect
          label="Order"
          value={sort}
          onChange={(next) => {
            const order = next === "oldest" || next === "decided" ? next : "newest";
            setSort(order);
            if (order === "decided" && status === "open") setStatus("any");
            setPage(1);
          }}
        >
          <option value="newest">Newest</option>
          <option value="oldest">Oldest</option>
          <option value="decided">Last decided</option>
        </ThemedSelect>
        {!queue.loading && <p className={styles.matchCount}>{matchLabel(queue.total)}</p>}
      </div>
      <div className={styles.people} role="group" aria-label="Decided by">
        <span className={styles.peopleLabel}>Decided by</span>
        <button
          type="button"
          className={styles.person}
          aria-pressed={resolver === ""}
          onClick={() => {
            setResolver("");
            setPage(1);
          }}
        >
          Anyone
        </button>
        <button
          type="button"
          className={styles.person}
          aria-pressed={resolver === "me"}
          onClick={() => {
            const next = resolver === "me" ? "" : "me";
            setResolver(next);
            if (next !== "" && status === "open") setStatus("any");
            setPage(1);
          }}
        >
          Me
        </button>
        {deciders.people.map((person) => {
          const when = decisionWhen(person.last_at);
          const phrase = decisionPhrase(person);
          return (
            <button
              key={person.id}
              type="button"
              className={styles.person}
              aria-pressed={resolver === person.id}
              title={when === "" ? phrase : `${phrase} · ${when}`}
              onClick={() => {
                const next = resolver === person.id ? "" : person.id;
                setResolver(next);
                if (next !== "" && status === "open") setStatus("any");
                setPage(1);
              }}
            >
              @{person.username}
            </button>
          );
        })}
      </div>
      {deciders.error && <p className={simple.muted}>Couldn't load who decided.</p>}
      {queue.loading && <p className={simple.muted}>Loading…</p>}
      {queue.error && (
        <p role="alert" className={simple.error}>
          {shortError(queue.error)}
        </p>
      )}
      {!queue.loading && !queue.error && queue.events.length === 0 && (
        <p className={simple.muted}>
          {filtered ? "Nothing matches." : "No open safety reviews."}
        </p>
      )}
      <ul className={styles.list}>
        {queue.events.map((event) => (
          <li key={event.id} className={styles.row}>
            <div className={styles.main}>
              <p className={styles.meta}>
                <span
                  className={`${styles.status} ${event.action === "blocked" ? styles.blocked : styles.flagged}`}
                >
                  {actionLabel(event.action)}
                </span>
                <span>{eventSubject(event)}</span>
                <DecisionTrail event={event} />
              </p>
              <p className={styles.comment}>{event.body}</p>
            </div>
            <div className={styles.actions}>
              <Link
                to="/posts/$id"
                params={{ id: event.post_id }}
                className={styles.icon}
                aria-label="View post"
                title="View post"
              >
                <IconEye />
              </Link>
              {event.status === "open" && (
                <button
                  type="button"
                  className={styles.icon}
                  aria-label="Dismiss"
                  title="Dismiss"
                  disabled={dismiss.isPending && dismiss.variables === event.id}
                  onClick={() => {
                    dismiss.mutate(event.id, {
                      onSuccess: () => toasts.push("Review dismissed.", "success"),
                      onError: (err) => toasts.push(shortError(err), "error"),
                    });
                  }}
                >
                  <IconCheck />
                </button>
              )}
              {event.status !== "removed" && (
                <button
                  type="button"
                  className={`${styles.icon} ${styles.danger}`}
                  aria-label="Remove"
                  title="Remove"
                  disabled={remove.isPending && remove.variables?.id === event.id}
                  onClick={() => setRemoveID(event.id)}
                >
                  <IconTrash />
                </button>
              )}
              {event.status !== "open" && (
                <button
                  type="button"
                  className={styles.icon}
                  aria-label={restoresComment(event) ? "Restore" : "Reopen"}
                  title={restoresComment(event) ? "Restore" : "Reopen"}
                  disabled={reopen.isPending && reopen.variables?.id === event.id}
                  onClick={() => setReopenID(event.id)}
                >
                  <IconUndo />
                </button>
              )}
            </div>
          </li>
        ))}
      </ul>
      <Pager
        page={page}
        total={queue.total}
        pageSize={pageSize}
        onPage={setPage}
        onPrefetch={queue.prefetchPage}
      />
      <ConfirmDialog
        open={pendingRemove !== null}
        title="Remove this comment?"
        message="The safety review closes. A published comment is deleted."
        confirmLabel="Remove"
        destructive
        onCancel={() => setRemoveID(null)}
        onConfirm={() => {
          if (pendingRemove === null || remove.isPending) return;
          remove.mutate(pendingRemove, {
            onSuccess: () => {
              setRemoveID(null);
              toasts.push("Comment removed.", "success");
            },
            onError: (err) => {
              setRemoveID(null);
              toasts.push(shortError(err), "error");
            },
          });
        }}
      />
      <ConfirmDialog
        open={pendingReopen !== null}
        title={pendingReopen && restoresComment(pendingReopen) ? "Restore this comment?" : "Reopen this review?"}
        message={
          pendingReopen && restoresComment(pendingReopen)
            ? "The comment is published again and the review returns to the open queue."
            : "The review returns to the open queue."
        }
        confirmLabel={pendingReopen && restoresComment(pendingReopen) ? "Restore" : "Reopen"}
        onCancel={() => setReopenID(null)}
        onConfirm={() => {
          if (pendingReopen === null || reopen.isPending) return;
          const restoring = restoresComment(pendingReopen);
          reopen.mutate(pendingReopen, {
            onSuccess: () => {
              setReopenID(null);
              toasts.push(restoring ? "Comment restored." : "Review reopened.", "success");
            },
            onError: (err) => {
              setReopenID(null);
              toasts.push(shortError(err), "error");
            },
          });
        }}
      />
    </StaffSection>
  );
}
