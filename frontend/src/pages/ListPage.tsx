import { Link } from "@tanstack/react-router";
import { ThemedSelect } from "../components/ThemedSelect";
import { useLayoutEffect, useRef, useState } from "react";
import type { Post } from "../types/api";
import { BoardViewControls } from "../components/BoardViewControls";
import { PostCard } from "../components/PostCard";
import { PostRow } from "../components/PostRow";
import { useBoardView } from "../hooks/useBoardView";
import { shortError, useToasts } from "../components/Toasts";
import { useList, useListActions, useMe } from "../hooks/useKura";
import styles from "./Simple.module.css";
import listStyles from "./ListPage.module.css";

export function ListPage({ id }: { id: string }) {
  const { data, isPending, error } = useList(id);
  const { user } = useMe();
  const { view, changeView } = useBoardView();
  const actions = useListActions(id);
  const toasts = useToasts();
  const [copied, setCopied] = useState(false);
  const [editError, setEditError] = useState("");
  const [editDirty, setEditDirty] = useState(false);
  const editFormRef = useRef<HTMLFormElement>(null);

  // Unsaved-changes check for the edit form (title/description inputs are
  // uncontrolled, so compare FormData against the saved list).
  const checkEditDirty = (visibility?: string) => {
    const form = editFormRef.current;
    if (!form || !data) return;
    const fd = new FormData(form);
    setEditDirty(
      String(fd.get("title") ?? "") !== data.list.title ||
        String(fd.get("description") ?? "") !== (data.list.description ?? "") ||
        (visibility ?? String(fd.get("visibility") ?? data.list.visibility)) !==
          data.list.visibility,
    );
  };

  // Local order shown while present; cleared back to the server order
  // whenever the query items change.
  // Local order from an arrow move, tagged with the server order it was
  // based on. Once the server order changes (our save landing, or an
  // edit elsewhere), the tag stops matching and the server order shows.
  const [local, setLocal] = useState<{ posts: Post[]; baseKey: string } | null>(null);
  const wrapRefs = useRef(new Map<string, HTMLElement>());

  // FLIP animation: capture wrapper positions before an order change,
  // then after render translate items back and let them glide home, so
  // reordering animates instead of snapping. Skipped for reduced motion.
  const flipRects = useRef(new Map<string, { left: number; top: number }>());
  const captureFlip = () => {
    const measured = new Map<string, { left: number; top: number }>();
    for (const [pid, el] of wrapRefs.current) {
      const r = el.getBoundingClientRect();
      measured.set(pid, { left: r.left, top: r.top });
    }
    flipRects.current = measured;
  };
  const orderKey = (local?.posts ?? data?.items ?? []).map((p) => p.id).join(",");
  useLayoutEffect(() => {
    const prev = flipRects.current;
    if (prev.size === 0) return;
    flipRects.current = new Map();
    const reduced = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
    for (const [pid, el] of wrapRefs.current) {
      const before = prev.get(pid);
      if (!before) continue;
      if (reduced) continue;
      const now = el.getBoundingClientRect();
      const dx = before.left - now.left;
      const dy = before.top - now.top;
      if (dx === 0 && dy === 0) continue;
      // WAAPI: a new animation replaces any in-flight one, and the
      // browser composites it — no transition restarts, no jank.
      el.animate(
        [{ transform: `translate(${dx}px, ${dy}px)` }, { transform: "translate(0px, 0px)" }],
        { duration: 190, easing: "cubic-bezier(0.2, 0.7, 0.2, 1)" },
      );
    }
  }, [orderKey]);

  if (isPending) return <p>Loading…</p>;
  if (error || !data) return <p role="alert">That list is private or could not be found.</p>;
  const { list, items } = data;
  const isOwner = user?.username === list.owner_username;
  const serverKey = items.map((p) => p.id).join(",");
  const visible = local && local.baseKey === serverKey ? local.posts : items;

  const refresh = actions.refresh;

  const saveOrder = (next: Post[]) => {
    const nextIds = next.map((p) => p.id);
    if (nextIds.join(",") === serverKey) {
      setLocal(null);
      return;
    }
    void actions.reorderList(id, nextIds).catch(() => {
      setLocal(null);
      setEditError("Could not reorder.");
    });
  };

  // One-step move used by the arrow buttons; animates via FLIP and
  // saves at once. (Drag-and-drop was removed: live-reflowing a grid
  // mid-drag never animated smoothly without a dedicated sortable lib.)
  const moveItem = (postId: string, target: number) => {
    const base = visible;
    const index = base.findIndex((p) => p.id === postId);
    if (index < 0 || target < 0 || target >= base.length || target === index) return;
    captureFlip();
    const next = [...base];
    const [moved] = next.splice(index, 1);
    if (!moved) return;
    next.splice(target, 0, moved);
    setLocal({ posts: next, baseKey: serverKey });
    saveOrder(next);
  };

  return (
    <section
      className={`${styles.page} ${view.wide ? listStyles.pageWide : ""}`}
      aria-labelledby="list-title"
      style={{ maxWidth: view.wide ? "none" : "64rem" }}
    >
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
          ref={editFormRef}
          className={styles.inlineForm}
          onChange={() => checkEditDirty()}
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
                setEditDirty(false);
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
          <ThemedSelect
            name="visibility"
            label="Visibility"
            defaultValue={list.visibility}
            onChange={(value) => checkEditDirty(value)}
          >
            <option value="private">Private — only me</option>
            <option value="public">Public — anyone with the link</option>
          </ThemedSelect>
          <button type="submit" className={editDirty ? styles.saveDirty : undefined}>
            Save list
          </button>
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
      {items.length > 0 && (
        <div className={listStyles.controlsRow}>
          <BoardViewControls view={view} onChange={changeView} />
        </div>
      )}
      <div className={view.mode === "rows" ? listStyles.rowsList : listStyles.listGrid}>
        {visible.map((post, index) => (
          <div
            key={post.id}
            ref={(el) => {
              if (!isOwner) return;
              if (el) wrapRefs.current.set(post.id, el);
              else wrapRefs.current.delete(post.id);
            }}
            className={isOwner ? listStyles.item : undefined}
          >
            <div className={listStyles.cardWrap}>
              {view.mode === "rows" ? <PostRow post={post} /> : <PostCard post={post} />}
              <span className={listStyles.rank} aria-label={`Pick number ${index + 1}`}>
                {index + 1}
              </span>
              {isOwner && (
                <div className={listStyles.moveControls}>
                  <button
                    type="button"
                    aria-label={view.mode === "rows" ? "Move up" : "Move earlier"}
                    title={view.mode === "rows" ? "Move up" : "Move earlier"}
                    disabled={index === 0}
                    onClick={() => moveItem(post.id, index - 1)}
                  >
                    {view.mode === "rows" ? "↑" : "←"}
                  </button>
                  <button
                    type="button"
                    aria-label={view.mode === "rows" ? "Move down" : "Move later"}
                    title={view.mode === "rows" ? "Move down" : "Move later"}
                    disabled={index === visible.length - 1}
                    onClick={() => moveItem(post.id, index + 1)}
                  >
                    {view.mode === "rows" ? "↓" : "→"}
                  </button>
                </div>
              )}
            </div>
            {isOwner && (
              <div className={listStyles.removeRow}>
                <button
                  type="button"
                  className={listStyles.remove}
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
