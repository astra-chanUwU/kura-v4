import { Link } from "@tanstack/react-router";
import { ThemedSelect } from "../components/ThemedSelect";
import { useState } from "react";
import { PostCard } from "../components/PostCard";
import { shortError, useToasts } from "../components/Toasts";
import { useList, useListActions, useMe } from "../hooks/useKura";
import styles from "./Simple.module.css";

export function ListPage({ id }: { id: string }) {
  const { data, isPending, error } = useList(id);
  const { user } = useMe();
  const actions = useListActions(id);
  const toasts = useToasts();
  const [copied, setCopied] = useState(false);
  const [editError, setEditError] = useState("");

  if (isPending) return <p>Loading…</p>;
  if (error || !data) return <p role="alert">That list is private or could not be found.</p>;
  const { list, items } = data;
  const isOwner = user?.username === list.owner_username;

  const refresh = actions.refresh;
  const move = (index: number, direction: -1 | 1) => {
    const next = [...items];
    const target = index + direction;
    if (target < 0 || target >= next.length) return;
    const current = next[index];
    const other = next[target];
    if (!current || !other) return;
    next[index] = other;
    next[target] = current;
    void actions
      .reorderList(
        id,
        next.map((p) => p.id),
      )
      .catch((e: unknown) => setEditError(e instanceof Error ? e.message : "Could not reorder."));
  };

  return (
    <section className={styles.page} aria-labelledby="list-title" style={{ maxWidth: "64rem" }}>
      <div>
        <h1 id="list-title">{list.title}</h1>
        <p className={styles.muted}>
          {list.visibility === "public" ? "Public list" : "Private list"} · {items.length} pictures
          {list.owner_username && (
            <>
              {" · by "}
              <Link to="/u/$username" params={{ username: list.owner_username }}>
                @{list.owner_username}
              </Link>
            </>
          )}
        </p>
        {list.description && <p>{list.description}</p>}
      </div>

      {list.visibility === "public" && (
        <div className={styles.actions}>
          <button
            type="button"
            onClick={() => {
              void navigator.clipboard.writeText(window.location.href).then(() => setCopied(true));
            }}
          >
            {copied ? "Link copied" : "Copy share link"}
          </button>
        </div>
      )}

      {isOwner && (
        <form
          className={styles.inlineForm}
          onSubmit={(event) => {
            event.preventDefault();
            const form = new FormData(event.currentTarget);
            void actions
              .updateList(id, {
                title: String(form.get("title") ?? list.title),
                description: String(form.get("description") ?? ""),
                visibility: String(form.get("visibility") ?? list.visibility),
              })
              .then(() => {
                refresh();
                toasts.push("List saved.", "success");
              })
              .catch((e: unknown) => toasts.push(shortError(e, "Could not save."), "error"));
          }}
        >
          <input name="title" defaultValue={list.title} aria-label="List title" required />
          <input
            name="description"
            defaultValue={list.description}
            aria-label="List description"
            placeholder="Description"
          />
          <ThemedSelect name="visibility" label="Visibility" defaultValue={list.visibility}>
            <option value="private">Private — only me</option>
            <option value="public">Public — anyone with the link</option>
          </ThemedSelect>
          <button type="submit">Save list</button>
          <button
            type="button"
            onClick={() => {
              if (window.confirm("Delete this list? The pictures stay on the board.")) {
                void actions.deleteList(id).then(() => {
                  window.location.assign("/me");
                });
              }
            }}
          >
            Delete list
          </button>
        </form>
      )}
      {editError && (
        <p role="alert" className={styles.error}>
          {editError}
        </p>
      )}

      {items.length === 0 && (
        <p className={styles.muted}>Nothing here yet. Use “+ List” on any picture to add it.</p>
      )}
      <div className={styles.grid}>
        {items.map((post, index) => (
          <div key={post.id}>
            <PostCard post={post} />
            {isOwner && (
              <div className={styles.actions}>
                <button
                  type="button"
                  onClick={() => move(index, -1)}
                  disabled={index === 0}
                  aria-label="Move this picture earlier"
                >
                  ← Earlier
                </button>
                <button
                  type="button"
                  onClick={() => move(index, 1)}
                  disabled={index === items.length - 1}
                  aria-label="Move this picture later"
                >
                  Later →
                </button>
                <button
                  type="button"
                  onClick={() =>
                    void actions
                      .removeFromList(id, post.id)
                      .then(() => toasts.push("Removed from this list", "success"))
                      .catch((e: unknown) => toasts.push(shortError(e), "error"))
                  }
                >
                  Remove
                </button>
              </div>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}
