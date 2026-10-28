import { Link } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import type { BoardSearch } from "../domain/boardFilter";
import type { Post } from "../types/api";
import {
  useListActions,
  useMe,
  useMyLists,
  usePostActions,
  useToggleFavorite,
} from "../hooks/useKura";
import { usePrefetchPost } from "../hooks/usePrefetchPost";
import { ThemedSelect } from "./ThemedSelect";
import { shortError, useToasts } from "./Toasts";
import styles from "./PostCard.module.css";

/** Save / + List controls, shared by board tiles (overlay) and the post page. */
export function PostActions({ post }: { post: Post }) {
  const { signedIn } = useMe();
  const { data: lists } = useMyLists(signedIn);
  const toggle = useToggleFavorite();
  const { addToList } = usePostActions();
  const { createList } = useListActions();
  const toasts = useToasts();
  const [menuOpen, setMenuOpen] = useState(false);
  const [filter, setFilter] = useState("");
  const [newOpen, setNewOpen] = useState(false);
  const [newTitle, setNewTitle] = useState("");
  const [newVisibility, setNewVisibility] = useState("private");
  const [creating, setCreating] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  const closeMenu = () => {
    setMenuOpen(false);
    setNewOpen(false);
    setFilter("");
  };

  // The menu is a popover: it closes on any action, on Escape, and on
  // any click outside it. It never just lingers over the board.
  useEffect(() => {
    if (!menuOpen) return;
    const onPointerDown = (event: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) closeMenu();
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") closeMenu();
    };
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [menuOpen]);

  if (!signedIn) return null;
  const allLists = lists ?? [];
  const shownLists = allLists.filter((list) =>
    list.title.toLowerCase().includes(filter.trim().toLowerCase()),
  );
  return (
    <div ref={rootRef} className={styles.actionsRoot}>
      <div className={styles.actions}>
        <button
          type="button"
          aria-pressed={post.is_favorite}
          aria-label={post.is_favorite ? "Remove from favorites" : "Save to favorites"}
          title="Favorites are private — only you can see them"
          disabled={toggle.isPending}
          onClick={() =>
            toggle.mutate(
              { postId: post.id, favorite: !post.is_favorite },
              {
                onSuccess: (_data, vars) =>
                  toasts.push(
                    vars.favorite ? "Added to favorites" : "Removed from favorites",
                    "success",
                  ),
                onError: (e) => toasts.push(shortError(e), "error"),
              },
            )
          }
        >
          {post.is_favorite ? "♥ Saved" : "♡ Save"}
        </button>
        <button
          type="button"
          aria-expanded={menuOpen}
          onClick={() => (menuOpen ? closeMenu() : setMenuOpen(true))}
        >
          + List
        </button>
      </div>
      {menuOpen && (
        <div className={styles.menu} role="menu" aria-label="Add to a list">
          {allLists.length > 5 && (
            <input
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder={`Filter ${allLists.length} lists…`}
              aria-label="Filter lists"
            />
          )}
          {allLists.length === 0 && <p>No lists yet — make your first below.</p>}
          {allLists.length > 0 && shownLists.length === 0 && <p>No list matches “{filter}”.</p>}
          {shownLists.length > 0 && (
            <div className={styles.menuLists}>
              {shownLists.slice(0, 5).map((list) => (
                <button
                  key={list.id}
                  type="button"
                  onClick={() => {
                    void addToList(list.id, post.id)
                      .then(() => {
                        toasts.push(`Added to “${list.title}”`, "success");
                        closeMenu();
                      })
                      .catch((e: unknown) => toasts.push(shortError(e), "error"));
                  }}
                >
                  {list.title} {list.visibility === "public" ? "(public)" : "(private)"}
                </button>
              ))}
              {shownLists.length > 5 && (
                <p className={styles.moreHint}>
                  {shownLists.length - 5} more — filter above or manage lists in your profile.
                </p>
              )}
            </div>
          )}
          {newOpen ? (
            <form
              className={styles.newList}
              onSubmit={(event) => {
                event.preventDefault();
                const title = newTitle.trim();
                if (!title || creating) return;
                setCreating(true);
                void createList({ title, description: "", visibility: newVisibility })
                  .then((created) => addToList(created.id, post.id))
                  .then(() => {
                    toasts.push(`Created “${title}” and added this picture`, "success");
                    setNewTitle("");
                    closeMenu();
                  })
                  .catch((e: unknown) =>
                    toasts.push(shortError(e, "Could not create that list."), "error"),
                  )
                  .finally(() => setCreating(false));
              }}
            >
              <input
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
                placeholder="New list name…"
                aria-label="New list name"
                maxLength={80}
              />
              <ThemedSelect
                label="New list visibility"
                value={newVisibility}
                onChange={setNewVisibility}
              >
                <option value="private">Private</option>
                <option value="public">Public</option>
              </ThemedSelect>
              <button type="submit" disabled={!newTitle.trim() || creating}>
                {creating ? "Creating…" : "Create & add"}
              </button>
            </form>
          ) : (
            <button type="button" className={styles.newToggle} onClick={() => setNewOpen(true)}>
              + New list…
            </button>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * Board tile: just a thumbnail, with Save / + List overlaid on top.
 * No title or owner text on the board — those live on the post page.
 * `search` is the board filter the post page steps through.
 */
export function PostCard({
  post,
  search,
  onOpen,
}: {
  post: Post;
  search?: BoardSearch;
  onOpen?: () => void;
}) {
  const prefetch = usePrefetchPost(post);
  return (
    <article className={styles.card}>
      <Link
        to="/posts/$id"
        params={{ id: post.id }}
        search={search}
        className={styles.media}
        aria-label="Open post"
        onMouseDown={onOpen}
        {...prefetch}
      >
        <img src={post.thumbnail_url} alt="" loading="lazy" />
      </Link>
      {post.rating === "nsfw" && <span className={styles.nsfwBadge}>NSFW</span>}
      <PostActions post={post} />
    </article>
  );
}
