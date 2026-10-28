import { Link } from "@tanstack/react-router";
import type { BoardSearch } from "../domain/boardFilter";
import type { Post } from "../types/api";
import { usePrefetchPost } from "../hooks/usePrefetchPost";
import { PostActions } from "./PostCard";
import styles from "./PostRow.module.css";

const TYPE_LABEL: Record<string, string> = {
  "image/jpeg": "JPEG",
  "image/png": "PNG",
  "image/gif": "GIF",
  "image/webp": "WebP",
  "video/mp4": "MP4",
};

function formatSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  if (bytes >= 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${bytes} B`;
}

/** One picture per row: thumbnail left, full details right. Used by the
 * board's and lists' Rows view. Tag chips stage the tag when
 * onTagClick is given (the board), otherwise they link to a board
 * search for that tag. `search` is the board filter the post page
 * steps through. */
export function PostRow({
  post,
  onTagClick,
  search,
  onOpen,
}: {
  post: Post;
  onTagClick?: (tag: string) => void;
  search?: BoardSearch;
  onOpen?: () => void;
}) {
  const prefetch = usePrefetchPost(post);
  const type = TYPE_LABEL[post.content_type] ?? post.content_type;
  const dims = post.width > 0 && post.height > 0 ? `${post.width}×${post.height}` : null;
  return (
    <article className={styles.row}>
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
      <div className={styles.details}>
        <div className={styles.head}>
          <span className={styles.artist}>{post.artist || "Unknown artist"}</span>
          <PostActions post={post} />
        </div>
        <p className={styles.meta}>
          <span className={post.rating === "nsfw" ? styles.nsfw : styles.sfw}>
            {post.rating.toUpperCase()}
          </span>
          {" · "}
          {type}
          {dims ? ` · ${dims}` : ""} · {formatSize(post.size_bytes)}
        </p>
        <p className={styles.meta}>
          {post.owner_username && (
            <>
              by{" "}
              <Link to="/u/$username" params={{ username: post.owner_username }}>
                @{post.owner_username}
              </Link>
              {" · "}
            </>
          )}
          {new Date(post.created_at).toLocaleDateString()}
        </p>
        {post.tags.length > 0 && (
          <div className={styles.tags}>
            {post.tags.map((tag) =>
              onTagClick ? (
                <button
                  key={tag}
                  type="button"
                  className={styles.tag}
                  aria-label={`Add tag ${tag}`}
                  onClick={() => onTagClick(tag)}
                >
                  {tag}
                </button>
              ) : (
                <Link
                  key={tag}
                  to="/"
                  search={{ tags: tag }}
                  className={styles.tag}
                  aria-label={`Search tag ${tag}`}
                >
                  {tag}
                </Link>
              ),
            )}
          </div>
        )}
      </div>
    </article>
  );
}
