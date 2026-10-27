import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { HistoryList } from "../components/HistoryList";
import { PostActions } from "../components/PostCard";
import { shortError, useToasts } from "../components/Toasts";
import { atLeast } from "../domain/roles";
import { useMe, usePost, usePostActions, usePostHistory, useStaffActions } from "../hooks/useKura";
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
  const [metaError, setMetaError] = useState("");
  if (isPending) return <p>Loading…</p>;
  if (error || !post) return <p role="alert">That picture could not be found.</p>;
  const isVideo = post.content_type.startsWith("video/");

  const saveMeta = (artist: string, tags: string[], rating: string = post.rating) => {
    setMetaError("");
    void updatePostMeta(post.id, artist, tags, rating).catch((e: unknown) =>
      setMetaError(e instanceof Error ? e.message : "Could not save."),
    );
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
            <img className={styles.full} src={post.media_url} alt={post.artist || ""} />
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
          </div>
        </div>

        <div className={styles.details}>
          {post.hidden && (
            <p role="alert" className={styles.error}>
              This post is hidden from the board. Staff and the uploader can still open it here.
            </p>
          )}
          {(isModerator || isAdmin) && (
            <p className={styles.detailRow}>
              <span className={styles.detailLabel}>Staff</span>
              <span style={{ display: "inline-flex", gap: "0.4rem", flexWrap: "wrap" }}>
                {post.hidden ? (
                  <button
                    type="button"
                    onClick={() =>
                      void staff
                        .unhidePost(post.id)
                        .then(() => toasts.push("Post is back on the board.", "success"))
                        .catch((e: unknown) => toasts.push(shortError(e), "error"))
                    }
                  >
                    Unhide
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() =>
                      void staff
                        .hidePost(post.id)
                        .then(() => toasts.push("Post hidden from the board.", "success"))
                        .catch((e: unknown) => toasts.push(shortError(e), "error"))
                    }
                  >
                    Hide
                  </button>
                )}
                {isAdmin && !post.hidden && (
                  <button
                    type="button"
                    onClick={() => {
                      if (
                        window.confirm(
                          "Delete this post from the board? It can be restored from its history.",
                        )
                      )
                        void staff
                          .deletePost(post.id)
                          .then(() =>
                            toasts.push(
                              "Post deleted. Restore it from its history below.",
                              "success",
                            ),
                          )
                          .catch((e: unknown) => toasts.push(shortError(e), "error"));
                    }}
                  >
                    Delete
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
                    aria-pressed={post.rating === r}
                    onClick={() => saveMeta(post.artist, post.tags, r)}
                    style={{
                      padding: "0.25rem 0.6rem",
                      borderRadius: "999px",
                      border: "1px solid var(--border-strong)",
                      background: post.rating === r ? "var(--surface-soft)" : "var(--surface)",
                      color: post.rating === r ? "var(--text)" : "var(--muted)",
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
                  value={artistDraft ?? post.artist}
                  placeholder="Unknown artist"
                  aria-label="Artist"
                  onChange={(e) => setArtistDraft(e.target.value)}
                  onBlur={() => {
                    if (artistDraft !== null && artistDraft.trim() !== post.artist) {
                      saveMeta(artistDraft.trim(), post.tags);
                    }
                    setArtistDraft(null);
                  }}
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
              {post.tags.length === 0 && <span className={styles.muted}>No tags yet.</span>}
              {post.tags.map((tag) => (
                <span key={tag} className={styles.tag}>
                  {tag}
                  {signedIn && (
                    <button
                      type="button"
                      aria-label={`Remove tag ${tag}`}
                      onClick={() =>
                        saveMeta(
                          post.artist,
                          post.tags.filter((t) => t !== tag),
                        )
                      }
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
                const tag = newTag.trim().toLowerCase();
                if (tag && !post.tags.includes(tag)) {
                  saveMeta(post.artist, [...post.tags, tag]);
                }
                setNewTag("");
              }}
            >
              <input
                value={newTag}
                onChange={(e) => setNewTag(e.target.value)}
                placeholder="Add tags, space separated (anime girl white_hair)…"
                aria-label="Add a tag"
                maxLength={40}
              />
              <button type="submit">Add tag</button>
            </form>
          )}
          {metaError && (
            <p role="alert" className={styles.error}>
              {metaError}
            </p>
          )}
          {signedIn && (
            <div>
              <h2 style={{ fontSize: "1rem", margin: "1rem 0 0.5rem" }}>History</h2>
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
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
