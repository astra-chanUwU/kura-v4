import { Link } from "@tanstack/react-router";
import { useId } from "react";
import type { BoardSearch } from "../domain/boardFilter";
import { usePrefetchPost } from "../hooks/usePrefetchPost";
import { useRelatedPosts } from "../hooks/useKura";
import type { Post } from "../types/api";
import styles from "./RelatedPosts.module.css";

/** Pictures that share a tag with this one. An empty or still-loading grid stays quiet. */
export function RelatedPosts({ postId, search }: { postId: string; search?: BoardSearch }) {
  const related = useRelatedPosts(postId);
  const headingId = useId();
  if (related.isPending || related.isError || !related.data || related.data.length === 0) {
    return null;
  }
  return (
    <section className={styles.strip} aria-labelledby={headingId}>
      <h2 id={headingId} className={styles.heading}>
        Related
      </h2>
      <div className={styles.grid}>
        {related.data.map((post) => (
          <RelatedThumb key={post.id} post={post} search={search} />
        ))}
      </div>
    </section>
  );
}

function RelatedThumb({ post, search }: { post: Post; search?: BoardSearch }) {
  const prefetch = usePrefetchPost(post);
  const who = post.artist.trim();
  const name = who || "related picture";
  return (
    <Link
      to="/posts/$id"
      params={{ id: post.id }}
      search={search}
      className={styles.thumb}
      aria-label={post.rating === "nsfw" ? `Open ${name}, NSFW` : `Open ${name}`}
      {...prefetch}
    >
      <img src={post.thumbnail_url} alt="" loading="lazy" />
      {post.rating === "nsfw" && <span className={styles.nsfw}>NSFW</span>}
    </Link>
  );
}
