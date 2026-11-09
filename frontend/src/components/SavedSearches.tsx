import { Link } from "@tanstack/react-router";
import { useId, useState, type FormEvent, type ReactNode } from "react";
import { boardSort, searchFromFilter, type BoardSort } from "../domain/boardFilter";
import {
  useCreateSavedSearch,
  useDeleteSavedSearch,
  useSavedSearches,
} from "../hooks/useSavedSearches";
import { useMe } from "../hooks/useKura";
import type { SavedSearch } from "../types/savedSearch";
import { ConfirmDialog } from "./ConfirmDialog";
import { shortError, useToasts } from "./Toasts";
import styles from "./SavedSearches.module.css";

export function SaveCurrentSearchButton({
  filter,
}: {
  filter: { tags: string[]; exclude: string[]; q: string; sort?: BoardSort };
}) {
  const { signedIn } = useMe();
  const toasts = useToasts();
  const create = useCreateSavedSearch();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const nameID = useId();
  const hasFilter = filter.tags.length > 0 || filter.exclude.length > 0 || filter.q.trim() !== "";
  if (!signedIn || !hasFilter) return null;

  const pending = create.isPending;
  const close = () => {
    if (pending) return;
    setName("");
    setOpen(false);
  };
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (pending) return;
    create.mutate(
      {
        name,
        include_tags: filter.tags,
        exclude_tags: filter.exclude,
        q: filter.q.trim(),
        sort: boardSort(filter.sort),
      },
      {
        onSuccess: () => {
          toasts.push("Search saved.", "success");
          setName("");
          setOpen(false);
        },
        onError: (err) => toasts.push(shortError(err), "error"),
      },
    );
  };

  if (!open) {
    return (
      <button type="button" className={styles.button} onClick={() => setOpen(true)}>
        Save search
      </button>
    );
  }

  return (
    <form className={styles.form} onSubmit={submit}>
      <label className={styles.label} htmlFor={nameID}>
        Name
      </label>
      <input
        id={nameID}
        className={styles.input}
        value={name}
        maxLength={60}
        required
        disabled={pending}
        onChange={(event) => setName(event.target.value)}
      />
      <button type="submit" className={styles.save} disabled={pending}>
        {pending ? "Saving…" : "Save"}
      </button>
      <button type="button" className={styles.cancel} disabled={pending} onClick={close}>
        Cancel
      </button>
    </form>
  );
}

function describeSearch(search: SavedSearch): string {
  const parts: string[] = [];
  if (search.include_tags.length > 0) parts.push(search.include_tags.join(", "));
  if (search.exclude_tags.length > 0) parts.push(`exclude ${search.exclude_tags.join(", ")}`);
  if (search.q.trim() !== "") parts.push(`“${search.q.trim()}”`);
  if (search.sort === "oldest") parts.push("oldest");
  if (search.sort === "favorites") parts.push("Favorites");
  return parts.join(" · ");
}

export function SavedSearchList() {
  const { signedIn, loading: accountLoading } = useMe();
  const searches = useSavedSearches();
  const remove = useDeleteSavedSearch();
  const toasts = useToasts();
  const [confirmID, setConfirmID] = useState<string | null>(null);
  const pendingDelete = searches.data?.find((item) => item.id === confirmID);

  let body: ReactNode;
  if (accountLoading) {
    body = <p className={styles.muted}>Loading…</p>;
  } else if (!signedIn) {
    body = <p className={styles.muted}>Sign in to save searches.</p>;
  } else if (searches.isPending) {
    body = <p className={styles.muted}>Loading saved searches…</p>;
  } else if (searches.isError) {
    body = (
      <p className={styles.error} role="alert">
        {shortError(searches.error)}
      </p>
    );
  } else if ((searches.data ?? []).length === 0) {
    body = <p className={styles.muted}>No saved searches yet.</p>;
  } else {
    body = (
      <ul className={styles.list}>
        {(searches.data ?? []).map((search) => {
          const summary = describeSearch(search);
          return (
            <li key={search.id} className={styles.row}>
              <Link
                to="/"
                search={searchFromFilter({
                  tags: search.include_tags,
                  exclude: search.exclude_tags,
                  q: search.q,
                  sort: boardSort(search.sort),
                })}
                className={styles.link}
              >
                <span className={styles.name}>{search.name}</span>
                {summary !== "" && <span className={styles.summary}>{summary}</span>}
              </Link>
              <button
                type="button"
                className={styles.delete}
                aria-label={`Delete ${search.name}`}
                disabled={remove.isPending}
                onClick={() => setConfirmID(search.id)}
              >
                Delete
              </button>
            </li>
          );
        })}
      </ul>
    );
  }

  return (
    <section className={styles.section}>
      <h2 className={styles.heading}>Saved searches</h2>
      {body}
      <ConfirmDialog
        open={confirmID !== null}
        title="Delete saved search"
        message={
          pendingDelete
            ? `Delete “${pendingDelete.name}”? This only removes the saved filter.`
            : "Delete this saved search?"
        }
        confirmLabel="Delete"
        destructive
        onConfirm={() => {
          if (confirmID === null) return;
          const id = confirmID;
          setConfirmID(null);
          remove.mutate(id, {
            onSuccess: () => toasts.push("Saved search deleted.", "success"),
            onError: (err) => toasts.push(shortError(err), "error"),
          });
        }}
        onCancel={() => setConfirmID(null)}
      />
    </section>
  );
}
