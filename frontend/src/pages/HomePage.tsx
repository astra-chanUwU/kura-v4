import { Link } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { PostCard } from "../components/PostCard";
import { UploadDialog } from "../components/UploadDialog";
import uploadStyles from "../components/UploadDialog.module.css";
import {
  normalizeTagToken,
  tagTokens,
  useDebouncedValue,
  useMe,
  usePostSearch,
  usePosts,
} from "../hooks/useKura";
import { useSite } from "../hooks/useSite";
import { useViewportTags } from "../hooks/useViewportTags";
import styles from "./HomePage.module.css";

export function HomePage() {
  const { signedIn } = useMe();
  const [uploadOpen, setUploadOpen] = useState(false);
  const [droppedFiles, setDroppedFiles] = useState<File[]>([]);
  const [uploadNotice, setUploadNotice] = useState("");
  const [pageDragging, setPageDragging] = useState(false);
  const { loading: siteLoading, error: siteError, retry: siteRetry } = useSite();
  const { posts, loading, error, retry, hasMore, loadingMore, loadMore } = usePosts();
  const sentinelRef = useRef<HTMLDivElement>(null);
  const [showToTop, setShowToTop] = useState(false);
  useEffect(() => {
    const onScroll = () => setShowToTop(window.scrollY > 600);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  // Infinite scroll: load the next page before the sentinel nears view.
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || typeof IntersectionObserver === "undefined") return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting) && hasMore && !loadingMore) loadMore();
      },
      { rootMargin: "900px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasMore, loadingMore, loadMore]);
  // Board layout preference: column count (0 = auto) and page width.
  const [view, setView] = useState<{ cols: number; wide: boolean }>(() => {
    try {
      const saved = JSON.parse(localStorage.getItem("kura-board-view") ?? "{}") as {
        cols?: number;
        wide?: boolean;
      };
      return { cols: typeof saved.cols === "number" ? saved.cols : 0, wide: !!saved.wide };
    } catch {
      return { cols: 0, wide: false };
    }
  });
  const changeView = (next: { cols: number; wide: boolean }) => {
    setView(next);
    localStorage.setItem("kura-board-view", JSON.stringify(next));
  };
  // Dragging files anywhere over the board offers the upload dialog.
  useEffect(() => {
    if (!signedIn) return;
    const hasFiles = (e: DragEvent) => e.dataTransfer?.types.includes("Files");
    const onDragOver = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      setPageDragging(true);
    };
    const onDragLeave = (e: DragEvent) => {
      if (e.relatedTarget === null) setPageDragging(false);
    };
    const onDrop = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      e.preventDefault();
      setPageDragging(false);
      const files = [...(e.dataTransfer?.files ?? [])];
      if (files.length > 0) {
        setDroppedFiles(files);
        setUploadOpen(true);
      }
    };
    window.addEventListener("dragover", onDragOver);
    window.addEventListener("dragleave", onDragLeave);
    window.addEventListener("drop", onDrop);
    return () => {
      window.removeEventListener("dragover", onDragOver);
      window.removeEventListener("dragleave", onDragLeave);
      window.removeEventListener("drop", onDrop);
    };
  }, [signedIn]);

  const [searchText, setSearchText] = useState("");
  const [selectedTags, setSelectedTags] = useState<string[]>([]);
  const [searchFocused, setSearchFocused] = useState(false);
  const [highlight, setHighlight] = useState(0);
  // Keystrokes update the input instantly, but filtering (and anything
  // network-adjacent) only sees the debounced value, 200ms behind.
  const debouncedSearchText = useDebouncedValue(searchText, 200);
  const search = usePostSearch(
    debouncedSearchText,
    selectedTags,
    searchFocused || selectedTags.length > 0,
  );
  // Changing the tag set means a new result list: jump back to the
  // start of the board instead of leaving the user mid-scroll.
  const goToBoardTop = () => window.scrollTo({ top: 0, behavior: "smooth" });

  const addTag = (tag: string) => {
    const normalized = normalizeTagToken(tag);
    if (normalized) {
      setSelectedTags((prev) => (prev.includes(normalized) ? prev : [...prev, normalized]));
    }
    setSearchText("");
    setHighlight(0);
    goToBoardTop();
  };
  // Booru-style entry: "anime girl white_hair yuri" becomes one chip per tag.
  const addTagsFromText = (text: string) => {
    const tokens = tagTokens(text);
    if (tokens.length > 0) {
      setSelectedTags((prev) => [...new Set([...prev, ...tokens])]);
    }
    setSearchText("");
    setHighlight(0);
    goToBoardTop();
  };
  const shownPosts = search.active ? search.results : posts;

  // Dynamic viewport-based recommendations: observe which cards are on screen
  // and surface their most frequent tags (like Safebooru but for infinite scroll).
  // Limited to 12, excludes already-selected tags, re-computes as you scroll.
  const { recommended: viewportTags, observe: observePost } = useViewportTags(
    shownPosts,
    selectedTags,
    12,
  );

  const failed = error ?? siteError;

  return (
    <section
      aria-label="Image board"
      className={`${styles.page} ${view.wide ? styles.pageWide : ""}`}
    >
      {pageDragging && (
        <div className={uploadStyles.pageDrop} aria-hidden="true">
          Drop pictures to upload
        </div>
      )}
      {uploadOpen && (
        <UploadDialog
          initialFiles={droppedFiles}
          onUploaded={(count, hiddenFromUploader) => {
            setUploadNotice(
              hiddenFromUploader
                ? `Added ${count} picture${count === 1 ? "" : "s"}, but NSFW is hidden for your account — turn on “Show NSFW posts” in Me → Sign-in & security to see it.`
                : `Added ${count} picture${count === 1 ? "" : "s"} — at the top of the board, and in Me → Contributions.`,
            );
            if (!hiddenFromUploader) window.scrollTo({ top: 0, behavior: "smooth" });
          }}
          onClose={() => {
            setUploadOpen(false);
            setDroppedFiles([]);
          }}
        />
      )}
      {uploadNotice && (
        <p role="status" className={styles.message}>
          {uploadNotice}{" "}
          <button type="button" onClick={() => setUploadNotice("")}>
            Dismiss
          </button>
        </p>
      )}
      {/* One sticky head for search + tag strip: as a single block it grows
          with the selected-tag row, so the strip is never covered by it. */}
      <div className={styles.boardHead}>
        <div className={styles.searchBar}>
          {selectedTags.length > 0 && (
            <div className={styles.chipRow} aria-label="Selected tags (posts must match all)">
              {selectedTags.map((tag) => (
                <span key={tag} className={styles.chip}>
                  {tag}
                  <button
                    type="button"
                    aria-label={`Remove tag ${tag}`}
                    onClick={() => {
                      setSelectedTags((prev) => prev.filter((t) => t !== tag));
                      goToBoardTop();
                    }}
                  >
                    ×
                  </button>
                </span>
              ))}
            </div>
          )}
          <div className={styles.searchBox}>
            <input
              type="search"
              value={searchText}
              onChange={(e) => {
                setSearchText(e.target.value);
                setHighlight(0);
              }}
              onFocus={() => setSearchFocused(true)}
              onBlur={() => {
                // Delay so a suggestion click registers first.
                setTimeout(() => setSearchFocused(false), 120);
              }}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown" || e.key === "ArrowUp") {
                  e.preventDefault();
                  const delta = e.key === "ArrowDown" ? 1 : -1;
                  setHighlight((h) =>
                    Math.max(0, Math.min(search.suggestions.length - 1, h + delta)),
                  );
                } else if (e.key === "Enter" && searchFocused) {
                  e.preventDefault();
                  if (tagTokens(searchText).length > 1) {
                    addTagsFromText(searchText);
                  } else {
                    const pick = search.suggestions[highlight] ?? search.suggestions[0];
                    if (pick) addTag(pick.tag);
                    else addTagsFromText(searchText);
                  }
                } else if (e.key === "Backspace" && searchText === "" && selectedTags.length > 0) {
                  setSelectedTags((prev) => prev.slice(0, -1));
                  goToBoardTop();
                } else if (e.key === "Escape") {
                  setSearchFocused(false);
                }
              }}
              placeholder={selectedTags.length > 0 ? "Add a tag…" : "Search tags…"}
              aria-label="Search the board"
            />
            {searchFocused && search.suggestions.length > 0 && (
              <ul className={styles.suggestions} aria-label="Tag suggestions">
                {search.suggestions.map((s, i) => (
                  <li key={s.tag}>
                    <button
                      type="button"
                      aria-label={`Add tag ${s.tag}`}
                      className={i === highlight ? styles.suggestionActive : ""}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        addTag(s.tag);
                      }}
                    >
                      <span>{s.tag}</span>
                      <span className={styles.suggestionCount}>{s.count}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
          {search.active &&
            (search.searching ? (
              <span className={styles.searchMeta}>Searching…</span>
            ) : (
              <>
                <span className={styles.searchMeta}>
                  {search.results.length} result{search.results.length === 1 ? "" : "s"}
                  {selectedTags.length > 0 ? " with all selected tags" : ""}
                </span>
                <button
                  type="button"
                  onClick={() => {
                    setSearchText("");
                    setSelectedTags([]);
                    goToBoardTop();
                  }}
                >
                  Clear
                </button>
              </>
            ))}
          <div className={styles.viewControls} role="group" aria-label="Board layout">
            <span className={styles.viewGroup} role="group" aria-label="Columns">
              {[0, 1, 2, 3, 4, 6].map((cols) => (
                <button
                  key={cols}
                  type="button"
                  aria-pressed={view.cols === cols}
                  className={view.cols === cols ? styles.viewActive : ""}
                  title={
                    cols === 0 ? "Automatic columns" : `${cols} column${cols === 1 ? "" : "s"}`
                  }
                  onClick={() => changeView({ ...view, cols })}
                >
                  {cols === 0 ? "Auto" : cols}
                </button>
              ))}
            </span>
            {signedIn && (
              <button
                type="button"
                className={styles.uploadButton}
                onClick={() => {
                  setDroppedFiles([]);
                  setUploadOpen(true);
                }}
              >
                ⬆ Upload
              </button>
            )}
            <span className={styles.viewGroup} role="group" aria-label="Page width">
              <button
                type="button"
                aria-pressed={!view.wide}
                className={!view.wide ? styles.viewActive : ""}
                title="Narrow, centered page"
                onClick={() => changeView({ ...view, wide: false })}
              >
                Narrow
              </button>
              <button
                type="button"
                aria-pressed={view.wide}
                className={view.wide ? styles.viewActive : ""}
                title="Use the full width of the screen"
                onClick={() => changeView({ ...view, wide: true })}
              >
                Wide
              </button>
            </span>
          </div>
        </div>

        {/* Viewport-aware tag strip: top tags from visible posts. It lives in
          the sticky board head, so it stays visible while scrolling. When
          the current view has no tags it keeps its place with a default
          prompt instead of vanishing. */}
        {!loading && !failed && shownPosts.length > 0 && (
          <div className={styles.viewportTags} aria-label="Tags in current view">
            {viewportTags.length > 0 ? (
              viewportTags.map((t) => (
                <button
                  key={t.tag}
                  type="button"
                  className={styles.viewportChip}
                  aria-label={`Add tag ${t.tag}`}
                  onClick={() => addTag(t.tag)}
                >
                  {t.tag} <span className={styles.viewportChipCount}>· {t.count}</span>
                </button>
              ))
            ) : (
              <>
                <span className={styles.viewportEmpty}>No tags detected</span>
                {signedIn ? (
                  <button
                    type="button"
                    className={`${styles.viewportChip} ${styles.viewportCta}`}
                    onClick={() => {
                      setDroppedFiles([]);
                      setUploadOpen(true);
                    }}
                  >
                    Start contributing
                  </button>
                ) : (
                  <Link to="/register" className={`${styles.viewportChip} ${styles.viewportCta}`}>
                    Start contributing
                  </Link>
                )}
              </>
            )}
          </div>
        )}
      </div>

      {(loading || siteLoading) && <p className={styles.message}>Loading…</p>}
      {failed && !loading && (
        <div role="alert" className={styles.message}>
          <span>Couldn’t connect to Kura.</span>
          <button
            onClick={() => {
              retry();
              siteRetry();
            }}
          >
            Try again
          </button>
        </div>
      )}
      {!loading && !failed && !search.active && posts.length === 0 && (
        <div className={styles.empty}>
          <p>No images yet</p>
        </div>
      )}
      {search.active && !search.searching && search.results.length === 0 && (
        <div className={styles.empty}>
          <p>Nothing matches “{searchText.trim()}”</p>
          <p>Try a different spelling, an artist name, or a tag like “gif”.</p>
        </div>
      )}
      {shownPosts.length > 0 && (
        <div className={styles.grid} data-cols={view.cols || undefined}>
          {shownPosts.map((post) => (
            <div key={post.id} ref={(el) => observePost(el, post.id)} data-post-id={post.id}>
              <PostCard post={post} />
            </div>
          ))}
        </div>
      )}
      <button
        type="button"
        aria-label="Back to top"
        title="Back to top"
        className={`${styles.toTop} ${showToTop ? styles.toTopVisible : ""}`}
        onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })}
      >
        ↑
      </button>
      {!search.active && <div ref={sentinelRef} aria-hidden="true" />}
      {!search.active && loadingMore && <p className={styles.message}>Loading more…</p>}
      {!search.active && !loading && !failed && posts.length > 0 && !hasMore && (
        <p className={styles.message}>That's everything.</p>
      )}
    </section>
  );
}
