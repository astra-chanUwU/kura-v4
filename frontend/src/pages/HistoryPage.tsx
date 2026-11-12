import { useState } from "react";
import { ActorCombobox } from "../components/ActorCombobox";
import { HistoryList } from "../components/HistoryList";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import { atLeast } from "../domain/roles";
import {
  useDebouncedValue,
  useHistoryActors,
  useLoadMore,
  useMe,
  useRecentHistory,
  useStaffActions,
} from "../hooks/useKura";
import type { ChangeEntry, HistoryActor } from "../types/api";
import { StaffSection } from "./StaffLayout";
import styles from "./HistoryPage.module.css";
import simple from "./Simple.module.css";

type HistoryAction =
  | ""
  | "post.created"
  | "post.meta_updated"
  | "post.hidden"
  | "post.unhidden"
  | "post.deleted"
  | "user.role_changed"
  | "rollback";
type HistoryEntity = "" | "post" | "user";
type HistorySort = "newest" | "oldest";

function matchLabel(total: number): string {
  return total === 1 ? "1 match" : `${total} matches`;
}

export function HistoryPage() {
  const { user } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const [busyId, setBusyId] = useState("");
  const [actor, setActor] = useState<HistoryActor | null>(null);
  const [personQuery, setPersonQuery] = useState("");
  const [q, setQ] = useState("");
  const [action, setAction] = useState<HistoryAction>("");
  const [entity, setEntity] = useState<HistoryEntity>("");
  const [sort, setSort] = useState<HistorySort>("newest");
  const debouncedQ = useDebouncedValue(q, 300);
  const debouncedPerson = useDebouncedValue(personQuery, 120);
  const submittedQ = debouncedQ.trim();
  const submittedPerson = debouncedPerson.trim();
  const isStaff = !!user && atLeast(user.role, "moderator");
  const history = useRecentHistory(isStaff, {
    q: submittedQ,
    actorId: actor?.id ?? "",
    action,
    entity,
    sort,
  });
  const actorQuery = useHistoryActors(submittedPerson, isStaff && actor === null);
  const moreRef = useLoadMore(history.hasMore && !history.loadingMore, history.loadMore);

  if (!user) return null;

  const personText = personQuery.trim();
  const suggestions =
    actor === null && personText === submittedPerson ? (actorQuery.data ?? []) : [];
  const actorLoading =
    actor === null &&
    personText !== "" &&
    (personText !== submittedPerson || actorQuery.isFetching);
  const filtered = submittedQ !== "" || actor !== null || action !== "" || entity !== "";

  const rollback = (entry: ChangeEntry) => {
    setBusyId(entry.id);
    void staff
      .rollback(entry)
      .then(() =>
        toasts.push("Change rolled back. The rollback itself is in the history.", "success"),
      )
      .catch((e: unknown) => toasts.push(shortError(e), "error"))
      .finally(() => setBusyId(""));
  };

  const canRollback = (entry: ChangeEntry) => {
    if (entry.action === "post.created") return false;
    if (entry.entity_type === "post") return atLeast(user.role, "moderator");
    return atLeast(user.role, "admin");
  };

  return (
    <StaffSection
      id="history-title"
      title="Change history"
      intro={
        <>
          Every tag, rating, hide, delete and role change across the board, newest first — who did
          it and what it was before. Search by person, action, and what changed. Roll back undoes
          one change and is itself recorded, so a rogue staff account can be cleaned up one change
          at a time. Each post change links to its post.
        </>
      }
    >
      <div className={styles.filterRow}>
        <ActorCombobox
          value={actor}
          onChange={(next) => {
            setActor(next);
            setPersonQuery("");
          }}
          query={personQuery}
          onQueryChange={setPersonQuery}
          suggestions={suggestions}
          loading={actorLoading}
        />
        <button
          type="button"
          className={styles.me}
          aria-pressed={actor?.id === user.id}
          title="Only my changes"
          onClick={() => {
            setActor({ id: user.id, username: user.username });
            setPersonQuery("");
          }}
        >
          Me
        </button>
        <input
          type="text"
          className={styles.search}
          value={q}
          onChange={(event) => setQ(event.target.value)}
          placeholder="Tag, artist, role, or source"
          aria-label="Search changes"
        />
        <ThemedSelect
          label="Action"
          value={action}
          onChange={(next) => setAction(next as HistoryAction)}
        >
          <option value="">Any change</option>
          <option value="post.created">Uploaded</option>
          <option value="post.meta_updated">Tags / artist / rating</option>
          <option value="post.hidden">Hidden</option>
          <option value="post.unhidden">Restored</option>
          <option value="post.deleted">Deleted</option>
          <option value="user.role_changed">Role changed</option>
          <option value="rollback">Rollbacks</option>
        </ThemedSelect>
        <ThemedSelect
          label="About"
          value={entity}
          onChange={(next) => setEntity(next === "post" || next === "user" ? next : "")}
        >
          <option value="">Everything</option>
          <option value="post">Posts</option>
          <option value="user">People</option>
        </ThemedSelect>
        <ThemedSelect
          label="Order"
          value={sort}
          onChange={(next) => setSort(next === "oldest" ? "oldest" : "newest")}
        >
          <option value="newest">Newest</option>
          <option value="oldest">Oldest</option>
        </ThemedSelect>
        {!history.isPending && <p className={styles.matchCount}>{matchLabel(history.total)}</p>}
      </div>
      {history.isPending && <p className={simple.muted}>Loading…</p>}
      {history.error && (
        <p role="alert" className={simple.error}>
          {shortError(history.error)}
        </p>
      )}
      {!history.isPending && !history.error && history.entries.length === 0 && (
        <p className={simple.muted}>{filtered ? "Nothing matches." : "No changes recorded yet."}</p>
      )}
      {history.entries.length > 0 && (
        <HistoryList
          entries={history.entries}
          canRollback={canRollback}
          onRollback={rollback}
          busyId={busyId}
          showThumbnails
        />
      )}
      <div ref={moreRef} aria-hidden="true" />
      {history.loadingMore && <p className={simple.muted}>Loading more…</p>}
    </StaffSection>
  );
}
