import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { IconEye, IconTrash } from "../components/icons";
import { ThemedSelect } from "../components/ThemedSelect";
import { shortError, useToasts } from "../components/Toasts";
import { atLeast } from "../domain/roles";
import { useHiddenPosts, useMe, useStaffActions } from "../hooks/useKura";
import styles from "./HiddenPostsPage.module.css";
import simple from "./Simple.module.css";

type Filter = "all" | "hidden" | "deleted";

export function HiddenPostsPage() {
  const { me, user, loading } = useMe();
  const toasts = useToasts();
  const staff = useStaffActions();
  const [filter, setFilter] = useState<Filter>("all");
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
  const isStaff = !!user && atLeast(user.role, "moderator");
  const isAdmin = !!user && atLeast(user.role, "admin");
  const hidden = useHiddenPosts(isStaff);

  if (loading) return <p>Loading…</p>;
  if (!me || !user) {
    return (
      <section className={simple.page}>
        <h1>Hidden &amp; deleted</h1>
        <p className={simple.muted}>Sign in with a staff account to review removed posts.</p>
        <div className={simple.actions}>
          <Link to="/login">Sign in</Link>
        </div>
      </section>
    );
  }
  if (!isStaff) {
    return (
      <section className={simple.page}>
        <h1>Hidden &amp; deleted</h1>
        <p className={simple.muted}>
          Moderators and above can review posts that are off the board.
        </p>
      </section>
    );
  }

  const all = hidden.data ?? [];
  const shown = all.filter((p) =>
    filter === "all" ? true : filter === "deleted" ? p.deleted : !p.deleted,
  );
  const hiddenCount = all.filter((p) => !p.deleted).length;
  const deletedCount = all.filter((p) => p.deleted).length;

  const run = (action: Promise<unknown>, ok: string) =>
    void action
      .then(() => toasts.push(ok, "success"))
      .catch((e: unknown) => toasts.push(shortError(e), "error"));

  return (
    <section className={simple.page} style={{ maxWidth: "80rem" }} aria-labelledby="hidden-title">
      <h1 id="hidden-title">Hidden &amp; deleted</h1>
      <p className={simple.muted}>
        Everything currently off the board, newest removal first. Hidden posts can be restored by
        any moderator; deleted posts are the stronger admin removal. Nothing here is gone for good —
        restore puts it back, and every step stays in the post&apos;s history. Back to{" "}
        <Link to="/admin">Staff</Link>.
      </p>
      <div className={styles.filterRow}>
        <ThemedSelect label="Show" value={filter} onChange={(next) => setFilter(next as Filter)}>
          <option value="all">All removed ({all.length})</option>
          <option value="hidden">Hidden only ({hiddenCount})</option>
          <option value="deleted">Deleted only ({deletedCount})</option>
        </ThemedSelect>
      </div>
      {hidden.isPending && <p className={simple.muted}>Loading…</p>}
      {hidden.error && (
        <p role="alert" className={simple.error}>
          {shortError(hidden.error)}
        </p>
      )}
      {!hidden.isPending && shown.length === 0 && (
        <p className={simple.muted}>Nothing here. The board is fully visible.</p>
      )}
      <div className={simple.grid}>
        {shown.map((post) => (
          <article key={post.id} className={styles.card}>
            <Link
              to="/posts/$id"
              params={{ id: post.id }}
              className={styles.media}
              aria-label="Open post"
            >
              <img src={post.thumbnail_url} alt={post.artist || ""} loading="lazy" />
              <span className={post.deleted ? styles.badgeDeleted : styles.badgeHidden}>
                {post.deleted ? "Deleted" : "Hidden"}
              </span>
            </Link>
            <div className={styles.info}>
              <span className={simple.muted}>
                {post.artist || "Unknown artist"}
                {post.owner_username ? ` · by @${post.owner_username}` : ""} ·{" "}
                {new Date(post.created_at).toLocaleDateString()}
              </span>
              {post.tags.length > 0 && (
                <span className={styles.tags}>{post.tags.slice(0, 6).join(", ")}</span>
              )}
            </div>
            <div className={styles.actions}>
              <button
                type="button"
                className={styles.restoreBtn}
                onClick={() => run(staff.unhidePost(post.id), "Post restored to the board.")}
              >
                <IconEye /> Restore
              </button>
              {isAdmin && !post.deleted && (
                <button
                  type="button"
                  className={styles.deleteBtn}
                  onClick={() => setPendingDeleteId(post.id)}
                >
                  <IconTrash /> Delete
                </button>
              )}
              <Link to="/posts/$id" params={{ id: post.id }} className={styles.viewLink}>
                View &amp; history
              </Link>
            </div>
          </article>
        ))}
      </div>
      <ConfirmDialog
        open={pendingDeleteId !== null}
        title="Delete this post?"
        message="Delete this post? It stays restorable from here and its history."
        confirmLabel="Delete"
        destructive
        onCancel={() => setPendingDeleteId(null)}
        onConfirm={() => {
          const id = pendingDeleteId;
          setPendingDeleteId(null);
          if (id) run(staff.deletePost(id), "Post deleted. Restore it from here anytime.");
        }}
      />
    </section>
  );
}
