import type { RefObject } from "react";
import type { BoardSearch } from "../domain/boardFilter";
import type { BoardView } from "../hooks/useBoardView";
import type { Post } from "../types/api";
import styles from "../pages/HomePage.module.css";
import { PostCard } from "./PostCard";
import { PostRow } from "./PostRow";

type BoardResultsProps = {
  posts: Post[];
  loading: boolean;
  failed: unknown;
  defaultsPending: boolean;
  filterActive: boolean;
  searching: boolean;
  searchText: string;
  view: BoardView;
  gridRef: RefObject<HTMLDivElement | null>;
  postSearch: BoardSearch;
  addTag: (tag: string) => void;
  commitStaged: () => void;
};

/** Empty state, detail rows, or the measured tile grid on the home board. */
export function BoardResults({
  posts,
  loading,
  failed,
  defaultsPending,
  filterActive,
  searching,
  searchText,
  view,
  gridRef,
  postSearch,
  addTag,
  commitStaged,
}: BoardResultsProps) {
  const shownPosts = posts;
  return (
    <>
      {!loading && !failed && !defaultsPending && !filterActive && posts.length === 0 && (
        <div className={styles.empty}>
          <p>No images yet</p>
        </div>
      )}
      {filterActive && !loading && !defaultsPending && !searching && posts.length === 0 && (
        <div className={styles.empty}>
          <p>
            {searchText.trim()
              ? `Nothing matches “${searchText.trim()}”`
              : "Nothing matches your tag filters"}
          </p>
          <p>Try removing a tag, a different spelling, or a tag like “gif”.</p>
        </div>
      )}
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
