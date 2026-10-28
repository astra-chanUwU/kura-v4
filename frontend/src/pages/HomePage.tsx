import { Link, useNavigate, useRouterState, useSearch } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { BoardViewControls } from "../components/BoardViewControls";
import { PostCard } from "../components/PostCard";
import { PostRow } from "../components/PostRow";
import { UploadDialog } from "../components/UploadDialog";
import uploadStyles from "../components/UploadDialog.module.css";
import { sameTagSet, searchFromFilter, splitTagInput } from "../domain/boardFilter";
import {
  normalizeTagToken,
  tagTokens,
  useDebouncedValue,
  useMe,
  usePostSearch,
  usePosts,
} from "../hooks/useKura";
import { useBoardView } from "../hooks/useBoardView";
import { useSite } from "../hooks/useSite";
import { useViewportTags } from "../hooks/useViewportTags";
import styles from "./HomePage.module.css";

export function HomePage() {
  const { signedIn, me, loading: accountLoading } = useMe();
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
  const { view, changeView } = useBoardView();
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

  // The applied filter lives in the URL (?tags=a,b&exclude=c&q=x).
  // Tag chips write it immediately. Back/Forward restore that entry.
  // Saved defaults seed only a bare homepage (no filter, not a Clear).
  const urlSearch = useSearch({ strict: false }) as {
    tags?: string;
    exclude?: string;
    q?: string;
  };
  const navigate = useNavigate();
  // Set on an empty board the user asked for (Clear, or the last chip
  // removed). A bare homepage visit has no such state, so defaults apply.
  // Back/Forward restore this with the history entry. Nothing else does.
  const boardCleared = useRouterState({
    select: (s) => s.location.state.boardCleared === true,
  });
  const parseUrlTags = (value?: string): string[] =>
    value
      ? [
          ...new Set(
            value
              .split(",")
              .map((t) => normalizeTagToken(t))
              .filter(Boolean),
          ),
        ]
      : [];
  const urlKey = `${urlSearch.tags ?? ""}|${urlSearch.exclude ?? ""}|${urlSearch.q ?? ""}`;
  const hasUrlFilter = !!(urlSearch.tags || urlSearch.exclude || urlSearch.q);
  // A chip edit or Clear sets this before navigate() resolves. The
  // history flag can land a render later; this stops that render from
  // treating the empty URL as a fresh homepage and putting defaults back.
  const skipSeed = useRef(false);
  const pendingWrite = useRef<string | null>(null);
  const writeUrl = (
    include: string[],
    exclude: string[],
    q: string,
    opts?: { replace?: boolean; cleared?: boolean },
  ) => {
    pendingWrite.current = `${include.join(",")}|${exclude.join(",")}|${q}`;
    void navigate({
      to: "/",
      replace: opts?.replace ?? true,
      state: { boardCleared: opts?.cleared === true },
      search: {
        tags: include.length > 0 ? include.join(",") : undefined,
        exclude: exclude.length > 0 ? exclude.join(",") : undefined,
        q: q !== "" ? q : undefined,
      },
    });
  };

  const [searchText, setSearchText] = useState(() => urlSearch.q ?? "");
  // Chips and the applied filter are the same set. The text query is
  // the only thing that waits for Filter / Enter.
  const [selectedTags, setSelectedTags] = useState<string[]>(() => parseUrlTags(urlSearch.tags));
  const [excludedTags, setExcludedTags] = useState<string[]>(() => parseUrlTags(urlSearch.exclude));
  const [appliedTags, setAppliedTags] = useState<string[]>(() => parseUrlTags(urlSearch.tags));
  const [appliedExcludedTags, setAppliedExcludedTags] = useState<string[]>(() =>
    parseUrlTags(urlSearch.exclude),
  );
  const [appliedQuery, setAppliedQuery] = useState(() => urlSearch.q ?? "");
  const [searchFocused, setSearchFocused] = useState(false);

  // URL changed under us (Back/Forward, a link): adopt it wholesale.
  const lastUrlKey = useRef(urlKey);
  useEffect(() => {
    if (lastUrlKey.current === urlKey) return;
    lastUrlKey.current = urlKey;
    const include = parseUrlTags(urlSearch.tags);
    const exclude = parseUrlTags(urlSearch.exclude);
    setSelectedTags(include);
    setExcludedTags(exclude);
    setAppliedTags(include);
    setAppliedExcludedTags(exclude);
    setAppliedQuery(urlSearch.q ?? "");
    setSearchText(urlSearch.q ?? "");
    // Logo, a typed URL, Back/Forward. Our own write already decided
    // whether defaults are allowed.
    if (pendingWrite.current !== urlKey) skipSeed.current = false;
    pendingWrite.current = null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlKey]);

  // Saved tags seed a bare homepage only. A URL that already has a
  // filter, and an empty board the user just asked for, are left alone.
  const defaultInclude = me?.preferences.default_include_tags ?? [];
  const defaultExclude = me?.preferences.default_exclude_tags ?? [];
  const defaultsKey = `${me?.user.id ?? ""}|${defaultInclude.join(",")}|${defaultExclude.join(",")}`;
  useEffect(() => {
    if (!me || hasUrlFilter || boardCleared || skipSeed.current) return;
    if (defaultInclude.length === 0 && defaultExclude.length === 0) return;
    setSelectedTags(defaultInclude);
    setExcludedTags(defaultExclude);
    setAppliedTags(defaultInclude);
    setAppliedExcludedTags(defaultExclude);
    setAppliedQuery("");
    setSearchText("");
    writeUrl(defaultInclude, defaultExclude, "", { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [defaultsKey, hasUrlFilter, boardCleared]);
  // Tags commit straight into the URL (replace, so Back from a picture
  // returns here instead of walking every chip). The text query still
  // waits for Filter / Enter.
  const commitFilter = (include: string[], exclude: string[], q: string) => {
    skipSeed.current = true;
    const query = q.trim();
    const cleared = include.length === 0 && exclude.length === 0 && query === "";
    setSelectedTags(include);
    setExcludedTags(exclude);
    setAppliedTags(include);
    setAppliedExcludedTags(exclude);
    setAppliedQuery(query);
    writeUrl(include, exclude, query, { replace: true, cleared });
    goToBoardTop();
  };
  const applyDefaults = () => {
    if (!me) return;
    setSearchText("");
    commitFilter(
      me.preferences.default_include_tags ?? [],
      me.preferences.default_exclude_tags ?? [],
      "",
    );
  };
  const applyFilters = () => {
    const q = searchText.trim();
    setSearchText(q);
    commitFilter(selectedTags, excludedTags, q);
  };
  const clearFilters = () => {
    setSearchText("");
    commitFilter([], [], "");
  };
  const pendingCount =
    selectedTags.filter((t) => !appliedTags.includes(t)).length +
    appliedTags.filter((t) => !selectedTags.includes(t)).length +
    excludedTags.filter((t) => !appliedExcludedTags.includes(t)).length +
    appliedExcludedTags.filter((t) => !excludedTags.includes(t)).length +
    (searchText.trim() !== appliedQuery ? 1 : 0);
  const filtersDirty = pendingCount > 0;
  const hasDefaults =
    !!me &&
    ((me.preferences.default_include_tags ?? []).length > 0 ||
      (me.preferences.default_exclude_tags ?? []).length > 0);
  const differsFromDefaults =
    hasDefaults &&
    (!sameTagSet(selectedTags, defaultInclude) || !sameTagSet(excludedTags, defaultExclude));
  // Don't paint the unfiltered board for a frame while saved defaults
  // are still on their way into the chips.
  const defaultsPending =
    !hasUrlFilter &&
    !boardCleared &&
    (accountLoading ||
      (defaultInclude.length + defaultExclude.length > 0 &&
        (!sameTagSet(appliedTags, defaultInclude) ||
          !sameTagSet(appliedExcludedTags, defaultExclude) ||
          appliedQuery !== "")));
  const [highlight, setHighlight] = useState(0);
  // Keystrokes update the input instantly, but filtering (and anything
  // network-adjacent) only sees the debounced value, 200ms behind.
  const debouncedSearchText = useDebouncedValue(searchText, 200);
  const search = usePostSearch(
    debouncedSearchText,
    appliedTags,
    appliedExcludedTags,
    searchFocused ||
      selectedTags.length > 0 ||
      excludedTags.length > 0 ||
      appliedTags.length > 0 ||
      appliedExcludedTags.length > 0,
    appliedQuery,
    [...selectedTags, ...excludedTags],
  );
  // Changing the tag set means a new result list: jump back to the
  // start of the board instead of leaving the user mid-scroll.
  const goToBoardTop = () => window.scrollTo({ top: 0, behavior: "smooth" });

  const addTag = (tag: string) => {
    const normalized = normalizeTagToken(tag);
    setSearchText("");
    setHighlight(0);
    if (!normalized) return;
    const include = selectedTags.includes(normalized)
      ? selectedTags
      : [...selectedTags, normalized];
    commitFilter(
      include,
      excludedTags.filter((t) => t !== normalized),
      appliedQuery,
    );
  };
  const addExcludedTag = (tag: string) => {
    const normalized = normalizeTagToken(tag);
    setSearchText("");
    setHighlight(0);
    if (!normalized) return;
    const exclude = excludedTags.includes(normalized)
      ? excludedTags
      : [...excludedTags, normalized];
    commitFilter(
      selectedTags.filter((t) => t !== normalized),
      exclude,
      appliedQuery,
    );
  };
  // Booru-style entry: "anime girl white_hair -gif" — plain tokens are
  // required tags, "-tag" tokens are excluded tags, one chip each.
  const addTagsFromText = (text: string) => {
    const { include, exclude } = splitTagInput(text);
    setSearchText("");
    setHighlight(0);
    if (include.length === 0 && exclude.length === 0) return;
    const nextExclude = [
      ...new Set([...excludedTags.filter((t) => !include.includes(t)), ...exclude]),
    ];
    const nextInclude = [
      ...new Set([...selectedTags.filter((t) => !nextExclude.includes(t)), ...include]),
    ];
    commitFilter(nextInclude, nextExclude, appliedQuery);
  };
  const shownPosts = search.active ? search.results : posts;
  // Opening a picture carries the applied filter (defaults included),
  // so the post page's Previous/Next stay inside this result set.
  const postSearch = searchFromFilter({
    tags: appliedTags,
    exclude: appliedExcludedTags,
    q: appliedQuery,
  });

  // Dynamic viewport-based recommendations: observe which cards are on screen
  // and surface their most frequent tags (like Safebooru but for infinite scroll).
  // Limited to 12, excludes already-selected tags, re-computes as you scroll.
  const { recommended: viewportTags, observe: observePost } = useViewportTags(
    shownPosts,
    [...selectedTags, ...excludedTags, ...appliedTags, ...appliedExcludedTags],
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
          {(selectedTags.length > 0 || excludedTags.length > 0) && (
            <div className={styles.chipRow} aria-label="Tag filters">
              {selectedTags.map((tag) => (
                <span
                  key={`in-${tag}`}
                  className={`${styles.chip} ${appliedTags.includes(tag) ? "" : styles.chipPending}`}
                >
                  {tag}
                  <button
                    type="button"
                    aria-label={`Remove tag ${tag}`}
                    onClick={() =>
                      commitFilter(
                        selectedTags.filter((t) => t !== tag),
                        excludedTags,
                        appliedQuery,
                      )
                    }
                  >
                    ×
                  </button>
                </span>
              ))}
              {excludedTags.map((tag) => (
                <span
                  key={`ex-${tag}`}
                  className={`${styles.chip} ${styles.chipExcluded} ${appliedExcludedTags.includes(tag) ? "" : styles.chipPending}`}
                >
                  −{tag}
                  <button
                    type="button"
                    aria-label={`Remove excluded tag ${tag}`}
                    onClick={() =>
                      commitFilter(
                        selectedTags,
                        excludedTags.filter((t) => t !== tag),
                        appliedQuery,
                      )
                    }
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
                  if (searchText.trim() === "") {
                    // Enter on an empty box submits the staged filter.
                    applyFilters();
                  } else if (
                    searchText.trim().startsWith("-") ||
                    searchText.trim().includes(" -") ||
                    tagTokens(searchText).length > 1
                  ) {
                    addTagsFromText(searchText);
                  } else {
                    const pick = search.suggestions[highlight] ?? search.suggestions[0];
                    if (pick) addTag(pick.tag);
                    else addTagsFromText(searchText);
                  }
                } else if (
                  e.key === "Backspace" &&
                  searchText === "" &&
                  (selectedTags.length > 0 || excludedTags.length > 0)
                ) {
                  if (selectedTags.length > 0) {
                    commitFilter(selectedTags.slice(0, -1), excludedTags, appliedQuery);
                  } else {
                    commitFilter(selectedTags, excludedTags.slice(0, -1), appliedQuery);
                  }
                } else if (e.key === "Escape") {
                  setSearchFocused(false);
                }
              }}
              placeholder={
                selectedTags.length > 0 || excludedTags.length > 0
                  ? "Add a tag… (-tag excludes)"
                  : "Search tags… (-tag excludes)"
              }
              aria-label="Search the board"
            />
            {searchFocused && search.suggestions.length > 0 && (
              <ul className={styles.suggestions} aria-label="Tag suggestions">
                {search.suggestions.map((s, i) => (
                  <li key={s.tag} className={styles.suggestionRow}>
                    <button
                      type="button"
                      aria-label={`Exclude tag ${s.tag}`}
                      title={`Exclude ${s.tag}`}
                      className={`${styles.suggestionSide} ${styles.suggestionExclude}`}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        addExcludedTag(s.tag);
                      }}
                    >
                      −
                    </button>
                    <button
                      type="button"
                      aria-label={`Add tag ${s.tag}`}
                      className={`${styles.suggestionMain} ${i === highlight ? styles.suggestionActive : ""}`}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        addTag(s.tag);
                      }}
                    >
                      <span>{s.tag}</span>
                      <span className={styles.suggestionCount}>{s.count}</span>
                    </button>
                    <button
                      type="button"
                      aria-label={`Include tag ${s.tag}`}
                      title={`Include ${s.tag}`}
                      className={`${styles.suggestionSide} ${styles.suggestionInclude}`}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        addTag(s.tag);
                      }}
                    >
                      +
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div className={styles.filterControls}>
            <button
              type="button"
              className={filtersDirty ? styles.filterActive : styles.filterButton}
              onClick={applyFilters}
              aria-label={
                filtersDirty
                  ? `Filter (${pendingCount} unapplied change${pendingCount === 1 ? "" : "s"})`
                  : "Filter"
              }
            >
              Filter
            </button>
            {(selectedTags.length > 0 ||
              excludedTags.length > 0 ||
              appliedTags.length > 0 ||
              appliedExcludedTags.length > 0 ||
              searchText.trim() !== "" ||
              appliedQuery !== "") && (
              <button type="button" className={styles.clearButton} onClick={clearFilters}>
                Clear
              </button>
            )}
            {differsFromDefaults && (
              <button type="button" className={styles.defaultsButton} onClick={applyDefaults}>
                My defaults
              </button>
            )}
          </div>
          {search.active &&
            (search.searching ? (
              <span className={styles.searchMeta}>Searching…</span>
            ) : (
              <span className={styles.searchMeta}>
                {search.results.length} result{search.results.length === 1 ? "" : "s"}
                {appliedTags.length > 0 ? " with all selected tags" : ""}
                {appliedExcludedTags.length > 0
                  ? `, excluding ${appliedExcludedTags.length} tag${appliedExcludedTags.length === 1 ? "" : "s"}`
                  : ""}
                {filtersDirty ? " · unapplied changes" : ""}
              </span>
            ))}
          <div className={styles.viewControls}>
            <BoardViewControls view={view} onChange={changeView} />
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
          </div>
        </div>

        {/* Viewport-aware tag strip: top tags from visible posts. It lives in
          the sticky board head, so it stays visible while scrolling. When
          the current view has no tags it keeps its place with a default
          prompt instead of vanishing. */}
        {!loading && !failed && !defaultsPending && shownPosts.length > 0 && (
          <div className={styles.viewportTags} aria-label="Tags in current view">
            {viewportTags.length > 0 ? (
              viewportTags.map((t) => (
                <span key={t.tag} className={styles.viewportTag}>
                  <button
                    type="button"
                    className={`${styles.viewportTagSide} ${styles.viewportExclude}`}
                    aria-label={`Exclude tag ${t.tag}`}
                    title={`Exclude ${t.tag}`}
                    onClick={() => addExcludedTag(t.tag)}
                  >
                    −
                  </button>
                  <button
                    type="button"
                    className={styles.viewportTagMain}
                    aria-label={`Add tag ${t.tag}`}
                    onClick={() => addTag(t.tag)}
                  >
                    {t.tag} <span className={styles.viewportChipCount}>· {t.count}</span>
                  </button>
                  <button
                    type="button"
                    className={`${styles.viewportTagSide} ${styles.viewportInclude}`}
                    aria-label={`Include tag ${t.tag}`}
                    title={`Include ${t.tag}`}
                    onClick={() => addTag(t.tag)}
                  >
                    +
                  </button>
                </span>
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

      {(loading || siteLoading || defaultsPending) && <p className={styles.message}>Loading…</p>}
      {failed && !loading && !defaultsPending && (
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
      {!loading && !failed && !defaultsPending && !search.active && posts.length === 0 && (
        <div className={styles.empty}>
          <p>No images yet</p>
        </div>
      )}
      {search.active && !defaultsPending && !search.searching && search.results.length === 0 && (
        <div className={styles.empty}>
          <p>
            {searchText.trim()
              ? `Nothing matches “${searchText.trim()}”`
              : "Nothing matches your tag filters"}
          </p>
          <p>Try removing a tag, a different spelling, or a tag like “gif”.</p>
        </div>
      )}
      {shownPosts.length > 0 &&
        !defaultsPending &&
        (view.mode === "rows" ? (
          <div className={styles.rows}>
            {shownPosts.map((post) => (
              <div key={post.id} ref={(el) => observePost(el, post.id)} data-post-id={post.id}>
                <PostRow post={post} onTagClick={addTag} search={postSearch} />
              </div>
            ))}
          </div>
        ) : (
          <div className={styles.grid}>
            {shownPosts.map((post) => (
              <div key={post.id} ref={(el) => observePost(el, post.id)} data-post-id={post.id}>
                <PostCard post={post} search={postSearch} />
              </div>
            ))}
          </div>
        ))}
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
