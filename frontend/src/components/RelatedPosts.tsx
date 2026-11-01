import { PostCard } from "./PostCard";
import type { BoardSearch } from "../domain/boardFilter";
import { useRelatedPosts } from "../hooks/useKura";
import styles from "./RelatedPosts.module.css";

/** Pictures that share a tag with this one. An empty or still-loading strip stays quiet. */
export function RelatedPosts({ postId, search }: { postId: string; search?: BoardSearch }) {
  const related = useRelatedPosts(postId);
  if (related.isPending || related.isError || !related.data || related.data.length === 0) {
    return null;
  }
  return (
    <section className={styles.strip} aria-label="Related pictures">
      <h2 className={styles.heading}>Related</h2>
      <div className={styles.row}>
        {related.data.map((post) => (
          <PostCard key={post.id} post={post} search={search} />
        ))}
      </div>
    </section>
  );
}
