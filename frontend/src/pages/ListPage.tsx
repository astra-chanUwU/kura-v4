import { Link } from "@tanstack/react-router";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { ThemedSelect } from "../components/ThemedSelect";
import { useLayoutEffect, useRef, useState } from "react";
import type { Post } from "../types/api";
import { BoardViewControls } from "../components/BoardViewControls";
import { IconSave, IconTrash, IconUndo } from "../components/icons";
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
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [editError, setEditError] = useState("");
  const [editDirty, setEditDirty] = useState(false);
  const [editRevision, setEditRevision] = useState(0);
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

  // Drag-and-drop reorder state (owner only). Following the approach
  // that works smoothly in waifulist's tier-list builder: during a drag
  // the order NEVER changes — we only record where the drop would land
  // and paint an insertion indicator. The grid therefore never reflows
  // mid-drag (that live reflow, plus a FLIP pass per dragover, is what
  // made the old drag feel choppy). The reorder commits once, on drop.
  const [dragId, setDragId] = useState<string | null>(null);
  const [dropIndex, setDropIndex] = useState<number | null>(null);
  const dropIndexRef = useRef<number | null>(null);
  // Item that must not FLIP after a drop: the dragged picture should
  // simply appear where it was dropped, not glide from its old slot.
  const flipSkipRef = useRef<string | null>(null);
  const setDropTarget = (index: number | null) => {
    if (dropIndexRef.current === index) return;
    dropIndexRef.current = index;
    setDropIndex(index);
  };
  const endDrag = () => {
    setDragId(null);
    setDropTarget(null);
  };

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
    const skip = flipSkipRef.current;
    flipSkipRef.current = null;
    for (const [pid, el] of wrapRefs.current) {
      const before = prev.get(pid);
      if (!before) continue;
      if (pid === skip) continue;
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

  // Apply a new order: FLIP-animate it and save at once. Shared by the
  // arrow buttons and by drag-and-drop's single commit on drop.
  const applyOrder = (next: Post[], skipFlipId?: string) => {
    captureFlip();
    flipSkipRef.current = skipFlipId ?? null;
    setLocal({ posts: next, baseKey: serverKey });
    saveOrder(next);
  };

  // One-step move used by the arrow buttons (also the keyboard/touch
  // path, since HTML5 drag needs a mouse).
  const moveItem = (postId: string, target: number) => {
    const base = visible;
    const index = base.findIndex((p) => p.id === postId);
    if (index < 0 || target < 0 || target >= base.length || target === index) return;
    const next = [...base];
    const [moved] = next.splice(index, 1);
    if (!moved) return;
    next.splice(target, 0, moved);
    applyOrder(next);
  };

  // Where a drop over item `index` would insert: before it on its
  // first half, after it on its second. Rows stack vertically, the
  // grid flows left-to-right, so the split axis follows the layout.
  const insertionAt = (event: React.DragEvent<HTMLElement>, index: number) => {
    const rect = event.currentTarget.getBoundingClientRect();
    const after =
      view.mode === "rows"
        ? event.clientY > rect.top + rect.height / 2
        : event.clientX > rect.left + rect.width / 2;
    return index + (after ? 1 : 0);
  };

  // Commit the drag at insertion index `insertion` (an index into the
  // pre-removal order). No-op when dropped back where it started.
  const commitDrag = (insertion: number) => {
    const id = dragId;
    endDrag();
    if (!id) return;
    const from = visible.findIndex((p) => p.id === id);
    if (from < 0) return;
    const target = from < insertion ? insertion - 1 : insertion;
    if (target === from) return;
    const next = [...visible];
    const [moved] = next.splice(from, 1);
    if (!moved) return;
    next.splice(target, 0, moved);
    applyOrder(next, id);
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
          key={editRevision}
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
          <span className={styles.formActions}>
            <button
              type="submit"
              className={editDirty ? `${styles.withIcon} ${styles.saveDirty}` : styles.withIcon}
            >
              <IconSave /> Save
            </button>
            <button
              type="button"
              className={styles.withIcon}
              disabled={!editDirty}
              title="Put the title, description, and visibility back"
              onClick={() => {
                setEditRevision((revision) => revision + 1);
                setEditDirty(false);
              }}
            >
              <IconUndo /> Undo
            </button>
            <button
              type="button"
              className={`${styles.withIcon} ${styles.dangerAction}`}
              onClick={() => setConfirmDelete(true)}
            >
              <IconTrash /> Delete
            </button>
          </span>
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
          {isOwner && items.length > 1 ? (
            <p className={listStyles.dragHint}>Drag a picture to reorder it, or use its arrows.</p>
          ) : (
            <span />
          )}
          <BoardViewControls view={view} onChange={changeView} />
        </div>
      )}
      <div
        className={view.mode === "rows" ? listStyles.rowsList : listStyles.listGrid}
        onDragOver={(event) => {
          if (dragId) event.preventDefault();
        }}
        onDrop={(event) => {
          if (!dragId) return;
          event.preventDefault();
          commitDrag(visible.length);
        }}
        onDragLeave={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
            setDropTarget(null);
          }
        }}
      >
        {visible.map((post, index) => (
          <div
            key={post.id}
            ref={(el) => {
              if (!isOwner) return;
              if (el) wrapRefs.current.set(post.id, el);
              else wrapRefs.current.delete(post.id);
            }}
            className={
              isOwner
                ? [
                    listStyles.item,
                    view.mode === "rows" ? listStyles.itemRows : "",
                    dragId === post.id ? listStyles.dragging : "",
                    dropIndex === index ? listStyles.dropBefore : "",
                    index === visible.length - 1 && dropIndex === visible.length
                      ? listStyles.dropAfter
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" ")
                : undefined
            }
            draggable={isOwner || undefined}
            title={isOwner ? "Drag to reorder" : undefined}
            onDragStart={
              isOwner
                ? (event) => {
                    setDragId(post.id);
                    setDropTarget(null);
                    const dt = event.dataTransfer;
                    if (!dt) return;
                    dt.effectAllowed = "move";
                    try {
                      // Firefox only starts a drag once data is set.
                      dt.setData("text/plain", post.id);
                    } catch {
                      // Data is a drag-start token only; order is state.
                    }
                    try {
                      const el = event.currentTarget;
                      const rect = el.getBoundingClientRect();
                      dt.setDragImage(el, event.clientX - rect.left, event.clientY - rect.top);
                    } catch {
                      // Fall back to the browser's default drag ghost.
                    }
                  }
                : undefined
            }
            onDragEnd={isOwner ? endDrag : undefined}
            onDragOver={
              isOwner
                ? (event) => {
                    if (!dragId) return;
                    event.preventDefault();
                    if (event.dataTransfer) event.dataTransfer.dropEffect = "move";
                    // Guarded setter: state only changes when the
                    // insertion point actually moves, so dragover's
                    // high fire rate never re-renders the grid.
                    setDropTarget(insertionAt(event, index));
                  }
                : undefined
            }
            onDrop={
              isOwner
                ? (event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    commitDrag(insertionAt(event, index));
                  }
                : undefined
            }
          >
            <div className={listStyles.cardWrap}>
              {view.mode === "rows" ? <PostRow post={post} /> : <PostCard post={post} />}
              <span className={listStyles.rank} aria-label={`Pick number ${index + 1}`}>
                {index + 1}
              </span>
              {isOwner && (
                <button
                  type="button"
                  className={listStyles.remove}
                  aria-label={`Remove pick ${index + 1}`}
                  title="Remove from this list"
                  draggable={false}
                  onMouseDown={(event) => event.stopPropagation()}
                  onClick={() =>
                    void actions
                      .removeFromList(id, post.id)
                      .then(() => toasts.push("Removed from this list", "success"))
                      .catch((e: unknown) => toasts.push(shortError(e), "error"))
                  }
                >
                  ×
                </button>
              )}
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
          </div>
        ))}
      </div>
      <ConfirmDialog
        open={confirmDelete}
        title="Delete this list?"
        message="Delete this list? The pictures stay on the board."
        confirmLabel="Delete list"
        destructive
        onCancel={() => setConfirmDelete(false)}
        onConfirm={() => {
          setConfirmDelete(false);
          void actions.deleteList(id).then(() => {
            window.location.assign("/me");
          });
        }}
      />
    </section>
  );
}
