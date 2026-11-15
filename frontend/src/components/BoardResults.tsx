import type { RefObject } from "react";
import type { BoardSearch } from "../domain/boardFilter";
import type { BoardView } from "../hooks/useBoardView";
import type { Post } from "../types/api";
import styles from "./board.module.css";
import { PostCard } from "./PostCard";
import { PostRow } from "./PostRow";

type BoardResultsProps = {
  posts: Post[];
  defaultsPending: boolean;
  view: BoardView;
  gridRef: RefObject<HTMLDivElement | null>;
  postSearch: BoardSearch;
  addTag: (tag: string) => void;
  commitStaged: () => void;
};

/** Detail rows, or the measured tile grid on the home board. */
export function BoardResults({
  posts,
  defaultsPending,
  view,
  gridRef,
  postSearch,
  addTag,
  commitStaged,
}: BoardResultsProps) {
  const shownPosts = posts;
  return (
    <>
      {view.mode === "rows" && shownPosts.length > 0 && !defaultsPending && (
        <div className={styles.rows}>
          {shownPosts.map((post) => (
            <div key={post.id} data-post-id={post.id}>
              <PostRow
                post={post}
                onTagClick={addTag}
                search={postSearch}
                onOpen={commitStaged}
                fromBoard
              />
            </div>
          ))}
        </div>
      )}
      {view.mode === "grid" && (
        <div
          ref={gridRef}
          className={
            shownPosts.length > 0 && !defaultsPending
              ? styles.grid
              : `${styles.grid} ${styles.gridProbe}`
          }
          aria-hidden={shownPosts.length === 0 || defaultsPending}
        >
          {shownPosts.length > 0 &&
            !defaultsPending &&
            shownPosts.map((post) => (
              <div key={post.id} data-post-id={post.id}>
                <PostCard post={post} search={postSearch} onOpen={commitStaged} fromBoard />
              </div>
            ))}
        </div>
      )}
    </>
  );
}
