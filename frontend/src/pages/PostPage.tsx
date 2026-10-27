import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { HistoryList } from "../components/HistoryList";
import { IconEye, IconEyeOff, IconTrash } from "../components/icons";
import { PostActions } from "../components/PostCard";
import { shortError, useToasts } from "../components/Toasts";
import { atLeast } from "../domain/roles";
import {
  tagTokens,
  useMe,
  usePost,
  usePostActions,
  usePostHistory,
  useStaffActions,
} from "../hooks/useKura";
import type { ChangeEntry } from "../types/api";
import styles from "./PostPage.module.css";

const TYPE_LABELS: Record<string, string> = {
  "image/gif": "GIF",
  "image/jpeg": "JPEG",
  "image/png": "PNG",
  "image/webp": "WebP",
  "image/avif": "AVIF",
  "video/mp4": "MP4",
  "video/webm": "WebM",
  "video/quicktime": "MOV",
};

function formatType(contentType: string): string {
  return TYPE_LABELS[contentType] ?? contentType;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function PostPage({ id }: { id: string }) {
  const { data: post, isPending, error } = usePost(id);
  const { signedIn, user } = useMe();
  const { updatePostMeta } = usePostActions();
  const staff = useStaffActions();
  const toasts = useToasts();
  const history = usePostHistory(id, signedIn);
  const [busyId, setBusyId] = useState("");
  const isModerator = !!user && atLeast(user.role, "moderator");
  const isAdmin = !!user && atLeast(user.role, "admin");
  const [newTag, setNewTag] = useState("");
  const [artistDraft, setArtistDraft] = useState<string | null>(null);
  const [tagsDraft, setTagsDraft] = useState<string[] | null>(null);
  const [ratingDraft, setRatingDraft] = useState<string | null>(null);
  const [metaError, setMetaError] = useState("");
  const [saving, setSaving] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  // Drafts belong to one post; never carry them across pictures.
  // (Adjusting state while rendering is the React-recommended reset
  // when a prop changes — no effect needed.)
  const [draftPostId, setDraftPostId] = useState(id);
  if (draftPostId !== id) {
    setDraftPostId(id);
    setArtistDraft(null);
    setTagsDraft(null);
    setRatingDraft(null);
    setNewTag("");
    setMetaError("");
  }

  if (isPending) return <p>Loading…</p>;
  if (error || !post) return <p role="alert">That picture could not be found.</p>;
  const isVideo = post.content_type.startsWith("video/");

  // Staged editing: tag/artist/rating changes sit in drafts and only hit
  // the server on Save — one save, one history entry, one rollback point.
  const artist = artistDraft ?? post.artist;
  const tags = tagsDraft ?? post.tags;
  const rating = ratingDraft ?? post.rating;
  const dirty =
    (artistDraft !== null && artistDraft.trim() !== post.artist) ||
    (tagsDraft !== null && JSON.stringify(tagsDraft) !== JSON.stringify(post.tags)) ||
    (ratingDraft !== null && ratingDraft !== post.rating);

  const discardDrafts = () => {
    setArtistDraft(null);
    setTagsDraft(null);
    setRatingDraft(null);
    setNewTag("");
    setMetaError("");
  };

  const saveChanges = () => {
    if (!dirty || saving) return;
    setMetaError("");
    setSaving(true);
    void updatePostMeta(post.id, artist.trim(), tags, rating)
      .then(() => {
        discardDrafts();
        toasts.push("Saved. One history entry covers all of it.", "success");
      })
      .catch((e: unknown) => setMetaError(e instanceof Error ? e.message : "Could not save."))
      .finally(() => setSaving(false));
  };

  return (
    <section className={styles.page} aria-labelledby="post-title">
      <h1 id="post-title">{post.artist || "Picture"}</h1>
      <div className={styles.layout}>
        <div className={styles.mediaCol}>
          {isVideo ? (
            <>
              {/* oxlint-disable-next-line jsx-a11y/media-has-caption -- user uploads have no caption track */}
              <video
                className={styles.full}
                src={post.media_url}
                controls
                muted
                loop
                playsInline
                preload="metadata"
              />
            </>
          ) : (
            <a
              className={styles.fullLink}
              href={post.media_url}
              target="_blank"
              rel="noreferrer"
              aria-label="Open full size in a new tab (picture only, no page)"
              title="Open full size in a new tab"
            >
              <img className={styles.full} src={post.media_url} alt={post.artist || ""} />
            </a>
          )}

          <div className={styles.actionBar}>
            <PostActions post={post} />
          </div>

          <div className={styles.meta}>
            {post.owner_username && (
              <span>
                by{" "}
                <Link to="/u/$username" params={{ username: post.owner_username }}>
                  @{post.owner_username}
                </Link>
              </span>
            )}
            <span>{new Date(post.created_at).toLocaleDateString()}</span>
            <a href={post.media_url} target="_blank" rel="noreferrer">
              Full size ↗
            </a>
          </div>
        </div>

        <div className={styles.details}>
          {post.hidden && (
            <p role="alert" className={styles.error}>
              {post.deleted
                ? "This post is deleted (soft). Staff and the uploader can still open it here, and Restore brings it back."
                : "This post is hidden from the board. Staff and the uploader can still open it here."}
            </p>
          )}
          {(isModerator || isAdmin) && (
            <p className={styles.detailRow}>
              <span className={styles.detailLabel}>Staff</span>
              <span style={{ display: "inline-flex", gap: "0.4rem", flexWrap: "wrap" }}>
                {post.hidden ? (
                  <button
                    type="button"
                    className={`${styles.staffBtn} ${styles.restoreBtn}`}
                    onClick={() =>
                      void staff
                        .unhidePost(post.id)
                        .then(() => toasts.push("Post is back on the board.", "success"))
                        .catch((e: unknown) => toasts.push(shortError(e), "error"))
                    }
                  >
                    <IconEye /> Restore
                  </button>
                ) : (
                  <button
                    type="button"
                    className={`${styles.staffBtn} ${styles.hideBtn}`}
                    onClick={() =>
                      void staff
                        .hidePost(post.id)
                        .then(() => toasts.push("Post hidden from the board.", "success"))
                        .catch((e: unknown) => toasts.push(shortError(e), "error"))
                    }
                  >
                    <IconEyeOff /> Hide
                  </button>
                )}
                {isAdmin && !post.deleted && (
                  <button
                    type="button"
                    className={`${styles.staffBtn} ${styles.deleteBtn}`}
                    onClick={() => setConfirmDelete(true)}
                  >
                    <IconTrash /> Delete
                  </button>
                )}
              </span>
            </p>
          )}
          <p className={styles.detailRow}>
            <span className={styles.detailLabel}>Rating</span>
            {signedIn ? (
              <span style={{ display: "inline-flex", gap: "0.4rem" }}>
                {(["sfw", "nsfw"] as const).map((r) => (
                  <button
                    key={r}
                    type="button"
                    aria-pressed={rating === r}
                    onClick={() => setRatingDraft(r)}
                    style={{
                      padding: "0.25rem 0.6rem",
                      borderRadius: "999px",
                      border: "1px solid var(--border-strong)",
                      background: rating === r ? "var(--surface-soft)" : "var(--surface)",
                      color: rating === r ? "var(--text)" : "var(--muted)",
                      cursor: "pointer",
                      textTransform: "uppercase",
                      fontSize: "0.75rem",
                    }}
                  >
                    {r}
                  </button>
                ))}
              </span>
            ) : (
              <span style={{ textTransform: "uppercase" }}>{post.rating}</span>
            )}
          </p>
          <p className={styles.detailRow}>
            <span className={styles.detailLabel}>Type</span>
            <span>{formatType(post.content_type)}</span>
            <span className={styles.detailLabel}>Resolution</span>
            <span>
              {post.width > 0 && post.height > 0 ? `${post.width} × ${post.height}` : "Unknown"}
            </span>
            <span className={styles.detailLabel}>Size</span>
            <span>{formatBytes(post.size_bytes)}</span>
          </p>
          {(post.artist || signedIn) && (
            <p className={styles.detailRow}>
              <span className={styles.detailLabel}>Artist</span>
              {signedIn ? (
                <input
                  className={styles.artistInput}
                  value={artist}
                  placeholder="Unknown artist"
                  aria-label="Artist"
                  onChange={(e) => setArtistDraft(e.target.value)}
                />
              ) : (
                <span>{post.artist || "Unknown"}</span>
              )}
              {post.source_url && (
                <a href={post.source_url} target="_blank" rel="noreferrer">
                  View original ↗
                </a>
              )}
            </p>
          )}
          <div className={styles.detailRow}>
            <span className={styles.detailLabel}>Tags</span>
            <div className={styles.tagList}>
              {tags.length === 0 && <span className={styles.muted}>No tags yet.</span>}
              {tags.map((tag) => (
                <span
                  key={tag}
                  className={
                    tagsDraft !== null && !post.tags.includes(tag)
                      ? `${styles.tag} ${styles.tagStaged}`
                      : styles.tag
                  }
                >
                  {tag}
                  {signedIn && (
                    <button
                      type="button"
                      aria-label={`Remove tag ${tag}`}
                      onClick={() => setTagsDraft(tags.filter((t) => t !== tag))}
                    >
                      ×
                    </button>
                  )}
                </span>
              ))}
            </div>
          </div>
          {signedIn && (
            <form
              className={styles.tagForm}
              onSubmit={(event) => {
                event.preventDefault();
                const added = tagTokens(newTag);
                if (added.length > 0) {
                  setTagsDraft([...new Set([...tags, ...added])].sort());
                }
                setNewTag("");
              }}
            >
              <input
                value={newTag}
                onChange={(e) => setNewTag(e.target.value)}
                placeholder="Add tags, space separated (anime girl white_hair)…"
                aria-label="Add a tag"
              />
              <button type="submit">Add</button>
            </form>
          )}
          {signedIn && (
            <div className={styles.saveRow}>
              <button
                type="button"
                className={dirty ? `${styles.saveBtn} ${styles.saveDirty}` : styles.saveBtn}
                disabled={!dirty || saving}
                onClick={saveChanges}
              >
                {saving ? "Saving…" : "Save changes"}
              </button>
              {dirty && (
                <button type="button" className={styles.discardBtn} onClick={discardDrafts}>
                  Discard
                </button>
              )}
              <span className={styles.muted}>
                {dirty
                  ? "Unsaved changes — Save writes tags, artist and rating as one history entry."
                  : "Tag, artist and rating edits are staged here until you Save."}
              </span>
            </div>
          )}
          {metaError && (
            <p role="alert" className={styles.error}>
              {metaError}
            </p>
          )}
        </div>
      </div>
      {signedIn && (
        <section className={styles.historySection} aria-labelledby="post-history-title">
          <h2 id="post-history-title" className={styles.historyTitle}>
            History
          </h2>
          {history.isPending && <p className={styles.muted}>Loading…</p>}
          {history.error && <p className={styles.muted}>History could not be loaded.</p>}
          <HistoryList
            entries={history.data ?? []}
            busyId={busyId}
            canRollback={(entry: ChangeEntry) => isModerator && entry.action !== "post.created"}
            onRollback={(entry: ChangeEntry) => {
              setBusyId(entry.id);
              void staff
                .rollback(entry)
                .then(() => toasts.push("Change rolled back.", "success"))
                .catch((e: unknown) => toasts.push(shortError(e), "error"))
                .finally(() => setBusyId(""));
            }}
          />
          {!isModerator && (
            <p className={styles.muted}>Moderators and above can roll a change back.</p>
          )}
        </section>
      )}
      <ConfirmDialog
        open={confirmDelete}
        title="Delete this post?"
        message="Delete this post from the board? It can be restored from Hidden & deleted, or its history."
        confirmLabel="Delete"
        destructive
        onCancel={() => setConfirmDelete(false)}
        onConfirm={() => {
          setConfirmDelete(false);
          void staff
            .deletePost(post.id)
            .then(() =>
              toasts.push("Post deleted. Find it under Staff → Hidden & deleted.", "success"),
            )
            .catch((e: unknown) => toasts.push(shortError(e), "error"));
        }}
      />
    </section>
  );
}
