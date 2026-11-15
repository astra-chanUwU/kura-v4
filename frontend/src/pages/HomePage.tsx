import { useNavigate, useRouterState, useSearch } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { BoardHead } from "../components/BoardHead";
import { BoardResults } from "../components/BoardResults";
import { Pager } from "../components/Pager";
import { shortError, useToasts } from "../components/Toasts";
import { UploadDialog } from "../components/UploadDialog";
import uploadStyles from "../components/UploadDialog.module.css";
import {
  boardPageFromSearch,
  boardSort,
  normalizeTagToken,
  sameTagSet,
  searchFromFilter,
  splitTagInput,
  type BoardSearch,
  type BoardSort,
} from "../domain/boardFilter";
import { boardMemoryFor, writeBoardMemory } from "../domain/boardMemory";
import { clampPage, pageCount, resolvePageSize, snapPageSize } from "../domain/paging";
import { prefetchBoardPage, useBoard, useRandomPost } from "../hooks/useBoardData";
import { useMe } from "../hooks/useSession";
import { useBoardView } from "../hooks/useBoardView";
import { usePageTags } from "../hooks/usePageTags";
import { usePrefetchPage } from "../hooks/usePrefetchPage";
import { useSite } from "../hooks/useSite";
import styles from "./HomePage.module.css";

/** How many tracks the board grid currently paints. */
function gridColumnCount(el: Element): number {
  const raw = getComputedStyle(el).gridTemplateColumns;
  if (!raw || raw === "none") return 0;
  return raw.split(/\s+/).filter(Boolean).length;
}

function sortToken(sort: string | undefined): "" | "oldest" | "favorites" {
  if (sort === "oldest" || sort === "favorites") return sort;
  return "";
}

export function HomePage() {
  const { signedIn, me, loading: accountLoading } = useMe();
  const [uploadOpen, setUploadOpen] = useState(false);
  const [randomBusy, setRandomBusy] = useState(false);
  const toasts = useToasts();
  const randomPost = useRandomPost();
  const [droppedFiles, setDroppedFiles] = useState<File[]>([]);
  const [uploadNotice, setUploadNotice] = useState("");
  const [pageDragging, setPageDragging] = useState(false);
  const { loading: siteLoading, error: siteError, retry: siteRetry } = useSite();
  const [showToTop, setShowToTop] = useState(false);
  useEffect(() => {
    const onScroll = () => setShowToTop(window.scrollY > 600);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);
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

  // The applied filter lives in the URL (?tags=a,b&exclude=c&q=x&page=2).
  // Tag chips write it immediately. Back/Forward restore that entry.
  // This tab also remembers the last board. A bare "/" (the logo) puts
  // that board back. Saved defaults seed a bare homepage only when
  // nothing is remembered yet.
  const urlSearch = useSearch({ strict: false }) as BoardSearch;
  const navigate = useNavigate();
  const page = boardPageFromSearch(urlSearch);
  // Set on an empty board the user asked for (Clear, or the last chip
  // removed). Back/Forward restore this with the history entry. The
  // remembered board stores the same choice, so the logo honors a Clear.
  // Defaults apply only when this tab has no remembered board.
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
  const urlKey = `${urlSearch.tags ?? ""}|${urlSearch.exclude ?? ""}|${urlSearch.q ?? ""}|${sortToken(urlSearch.sort)}`;
  const hasUrlFilter = !!(urlSearch.tags || urlSearch.exclude || urlSearch.q);
  // A chip edit or Clear sets this before navigate() resolves. The
  // history flag can land a render later; this stops that render from
  // treating the empty URL as a fresh homepage and putting defaults back.
  const skipSeed = useRef(false);
  const pendingWrite = useRef<string | null>(null);
  const viewerId = me?.user.id ?? "";
  const writeUrl = (
    include: string[],
    exclude: string[],
    q: string,
    opts?: { replace?: boolean; cleared?: boolean; sort?: BoardSort },
  ) => {
    const cleared = opts?.cleared === true;
    const sort = opts?.sort ?? boardSort(urlSearch.sort);
    writeBoardMemory({ userId: viewerId, tags: include, exclude, q, cleared });
    pendingWrite.current = `${include.join(",")}|${exclude.join(",")}|${q}|${sortToken(sort)}`;
    const search: { tags?: string; exclude?: string; q?: string; sort?: "oldest" | "favorites" } = {
      tags: include.length > 0 ? include.join(",") : undefined,
      exclude: exclude.length > 0 ? exclude.join(",") : undefined,
      q: q !== "" ? q : undefined,
    };
    if (sort === "oldest" || sort === "favorites") search.sort = sort;
    void navigate({
      to: "/",
      replace: opts?.replace ?? true,
      state: { boardCleared: cleared },
      search,
    });
  };

  // Chips and the applied filter are the same set. Filter commits the strip.
  // The search box draft lives in the board head.
  const [selectedTags, setSelectedTags] = useState<string[]>(() => parseUrlTags(urlSearch.tags));
  const [excludedTags, setExcludedTags] = useState<string[]>(() => parseUrlTags(urlSearch.exclude));
  const [appliedTags, setAppliedTags] = useState<string[]>(() => parseUrlTags(urlSearch.tags));
  const [appliedExcludedTags, setAppliedExcludedTags] = useState<string[]>(() =>
    parseUrlTags(urlSearch.exclude),
  );
  const [appliedQuery, setAppliedQuery] = useState(() => urlSearch.q ?? "");
  // Latest chips, including ones staged from the strip and not yet in
  // the URL. Leaving the board saves them so the logo can bring them back.
  const chipRef = useRef({ tags: selectedTags, exclude: excludedTags, q: appliedQuery });
  chipRef.current = { tags: selectedTags, exclude: excludedTags, q: appliedQuery };
  const urlFilterRef = useRef({
    tags: parseUrlTags(urlSearch.tags),
    exclude: parseUrlTags(urlSearch.exclude),
    q: urlSearch.q ?? "",
  });
  urlFilterRef.current = {
    tags: parseUrlTags(urlSearch.tags),
    exclude: parseUrlTags(urlSearch.exclude),
    q: urlSearch.q ?? "",
  };
  const viewerIdRef = useRef(viewerId);
  viewerIdRef.current = viewerId;
  useEffect(() => {
    return () => {
      const chips = chipRef.current;
      const url = urlFilterRef.current;
      if (
        sameTagSet(chips.tags, url.tags) &&
        sameTagSet(chips.exclude, url.exclude) &&
        chips.q === url.q
      ) {
        return;
      }
      writeBoardMemory({
        userId: viewerIdRef.current,
        tags: chips.tags,
        exclude: chips.exclude,
        q: chips.q,
        cleared: false,
      });
    };
  }, []);

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
    // Logo, a typed URL, Back/Forward. Our own write already decided
    // whether defaults are allowed.
    if (pendingWrite.current !== urlKey) skipSeed.current = false;
    pendingWrite.current = null;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlKey]);

  // A filter already in the address (refresh, Back) is the board to
  // restore the next time the logo opens a bare "/".
  useEffect(() => {
    if (accountLoading || (!hasUrlFilter && !boardCleared)) return;
    writeBoardMemory({
      userId: viewerId,
      tags: parseUrlTags(urlSearch.tags),
      exclude: parseUrlTags(urlSearch.exclude),
      q: urlSearch.q ?? "",
      cleared: boardCleared && !hasUrlFilter,
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlKey, boardCleared, accountLoading, viewerId]);

  // A bare homepage restores this tab's board. Saved tags seed one only
  // when nothing is remembered yet. A URL that already has a filter, and
  // an empty board the user just asked for, are left alone.
  const defaultInclude = me?.preferences.default_include_tags ?? [];
  const defaultExclude = me?.preferences.default_exclude_tags ?? [];
  const defaultsKey = `${me?.user.id ?? ""}|${defaultInclude.join(",")}|${defaultExclude.join(",")}`;
  const remembered =
    !hasUrlFilter && !boardCleared && !accountLoading ? boardMemoryFor(viewerId) : null;
  useEffect(() => {
    if (hasUrlFilter || boardCleared || skipSeed.current || accountLoading) return;
    const memory = boardMemoryFor(viewerId);
    const include = memory ? memory.tags : defaultInclude;
    const exclude = memory ? memory.exclude : defaultExclude;
    const query = memory ? memory.q : "";
    const cleared = memory?.cleared === true;
    if (!memory) {
      if (!me || (include.length === 0 && exclude.length === 0)) return;
    }
    setSelectedTags(include);
    setExcludedTags(exclude);
    setAppliedTags(include);
    setAppliedExcludedTags(exclude);
    setAppliedQuery(query);
    writeUrl(include, exclude, query, { replace: true, cleared });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [defaultsKey, hasUrlFilter, boardCleared, accountLoading, viewerId]);
  // Search-box edits commit straight into the URL (replace, so Back
  // from a picture returns here). Scroll only when the result list is
  // deliberately replaced: Filter, Clear, My defaults.
  const commitFilter = (include: string[], exclude: string[], q: string, scroll = false) => {
    skipSeed.current = true;
    const query = q.trim();
    const cleared = include.length === 0 && exclude.length === 0 && query === "";
    setSelectedTags(include);
    setExcludedTags(exclude);
    setAppliedTags(include);
    setAppliedExcludedTags(exclude);
    setAppliedQuery(query);
    writeUrl(include, exclude, query, { replace: true, cleared });
    if (scroll) goToBoardTop();
  };
  const applyDefaults = () => {
    if (!me) return;
    commitFilter(
      me.preferences.default_include_tags ?? [],
      me.preferences.default_exclude_tags ?? [],
      "",
      true,
    );
  };
  const hasDefaults =
    !!me &&
    ((me.preferences.default_include_tags ?? []).length > 0 ||
      (me.preferences.default_exclude_tags ?? []).length > 0);
  const differsFromDefaults =
    hasDefaults &&
    (!sameTagSet(selectedTags, defaultInclude) || !sameTagSet(excludedTags, defaultExclude));
  // Don't paint the unfiltered board for a frame while a remembered
  // board, or the saved defaults, are still on their way into the chips.
  const restorePending = remembered
    ? !remembered.cleared &&
      (!sameTagSet(appliedTags, remembered.tags) ||
        !sameTagSet(appliedExcludedTags, remembered.exclude) ||
        appliedQuery !== remembered.q)
    : (defaultInclude.length > 0 || defaultExclude.length > 0) &&
      (!sameTagSet(appliedTags, defaultInclude) ||
        !sameTagSet(appliedExcludedTags, defaultExclude) ||
        appliedQuery !== "");
  const defaultsPending = !hasUrlFilter && !boardCleared && (accountLoading || restorePending);
  const appliedFilter = {
    tags: appliedTags,
    exclude: appliedExcludedTags,
    q: appliedQuery,
    sort: boardSort(urlSearch.sort),
  };
  const pickRandom = () => {
    if (randomBusy) return;
    setRandomBusy(true);
    void randomPost(appliedFilter)
      .then((post) => {
        void navigate({
          to: "/posts/$id",
          params: { id: post.id },
          search: searchFromFilter(appliedFilter),
        });
      })
      .catch((err: unknown) => {
        const status = err && typeof err === "object" && "status" in err ? Number(err.status) : 0;
        if (status === 404) {
          toasts.push("No picture matches that search.", "error");
          return;
        }
        toasts.push(shortError(err), "error");
      })
      .finally(() => setRandomBusy(false));
  };
  const preferredPageSize = resolvePageSize(me);
  const [columns, setColumns] = useState(0);
  const gridRef = useRef<HTMLDivElement | null>(null);
  // Grid mode fills complete rows; rows mode keeps the preference as-is.
  // columns < 1 → snap leaves preferred alone until the board grid is measured.
  const pageSize =
    view.mode === "grid" ? snapPageSize(preferredPageSize, columns) : preferredPageSize;
  const board = useBoard(appliedFilter, page, pageSize, !defaultsPending);
  const { posts, loading, error, retry, total, isSuccess, searching } = board;
  const queryClient = useQueryClient();
  const prefetchBoard = useCallback(
    (target: number) => {
      void prefetchBoardPage(queryClient, appliedFilter, target, pageSize);
    },
    // Applied filter fields match useBoard's query key.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      queryClient,
      pageSize,
      appliedTags.join(","),
      appliedExcludedTags.join(","),
      appliedQuery,
      appliedFilter.sort,
    ],
  );
  usePrefetchPage(prefetchBoard, page, pageCount(total, pageSize), isSuccess && !defaultsPending);
  // Changing the tag set or page means a new result list: jump back to
  // the start of the board instead of leaving the user mid-scroll.
  const goToBoardTop = () => window.scrollTo({ top: 0, behavior: "smooth" });
  const lastBoardPage = useRef(page);
  useEffect(() => {
    if (lastBoardPage.current === page) return;
    lastBoardPage.current = page;
    goToBoardTop();
  }, [page]);
  // Filter writes omit page (reset to 1). Pager pushes history so
  // Back/Forward moves between pages. Page 1 omits `page` from the URL.
  const goToBoardPage = (nextPage: number, replace = false) => {
    if (!Number.isFinite(nextPage)) return;
    let target = Math.floor(nextPage);
    if (target < 1) return;
    // Clamp past the end only when total matches the current query.
    if (isSuccess && !searching) {
      target = clampPage(target, total, pageSize);
    }
    if (target === page) return;
    const search: BoardSearch = searchFromFilter(appliedFilter);
    if (target > 1) search.page = target;
    void navigate({
      to: "/",
      replace,
      state: { boardCleared },
      search,
    });
  };
  // Count CSS grid tracks from a stable board grid (real tiles or probe)
  // so page size can fill whole rows even while loading / empty.
  useLayoutEffect(() => {
    if (view.mode !== "grid") return;
    const el = gridRef.current;
    if (!el) return;
    const measure = () => {
      const n = gridColumnCount(el);
      setColumns((prev) => (prev === n ? prev : n));
    };
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [view.mode]);
  // Keep the same item under the cursor when the effective page size changes
  // (preference or column snap): newPage = floor(((page-1)*old)/new) + 1.
  // Wait for a real column measure before treating size as settled, so the
  // preferred→snapped transition does not remap / double-fetch a page jump.
  const lastPageSize = useRef<number | null>(null);
  useEffect(() => {
    if (view.mode === "grid" && columns < 1) return;
    if (lastPageSize.current === null) {
      lastPageSize.current = pageSize;
      return;
    }
    const oldSize = lastPageSize.current;
    if (oldSize === pageSize) return;
    lastPageSize.current = pageSize;
    const newPage = Math.floor(((page - 1) * oldSize) / pageSize) + 1;
    if (newPage !== page) goToBoardPage(newPage, true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pageSize, columns, view.mode]);
  // Once total is known for the current query, a URL page past the end
  // snaps to the last page. keepPreviousData can leave a stale total while
  // searching — never clamp against that.
  useEffect(() => {
    if (defaultsPending || !isSuccess || searching || error) return;
    const clamped = clampPage(page, total, pageSize);
    if (clamped === page) return;
    goToBoardPage(clamped, true);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, total, isSuccess, searching, error, defaultsPending, urlKey, boardCleared, pageSize]);

  // The on-screen tag strip only stages a chip. Applying here would
  // swap the pictures and jump to the top, so the next tag in that
  // strip is gone. Filter commits them.
  const stageInclude = (tag: string) => {
    const normalized = normalizeTagToken(tag);
    if (!normalized) return;
    setExcludedTags((prev) => prev.filter((t) => t !== normalized));
    setSelectedTags((prev) => (prev.includes(normalized) ? prev : [...prev, normalized]));
  };
  const stageExclude = (tag: string) => {
    const normalized = normalizeTagToken(tag);
    if (!normalized) return;
    setSelectedTags((prev) => prev.filter((t) => t !== normalized));
    setExcludedTags((prev) => (prev.includes(normalized) ? prev : [...prev, normalized]));
  };
  const addTag = (tag: string) => {
    const normalized = normalizeTagToken(tag);
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
    if (include.length === 0 && exclude.length === 0) return;
    const nextExclude = [
      ...new Set([...excludedTags.filter((t) => !include.includes(t)), ...exclude]),
    ];
    const nextInclude = [
      ...new Set([...selectedTags.filter((t) => !nextExclude.includes(t)), ...include]),
    ];
    commitFilter(nextInclude, nextExclude, appliedQuery);
  };
  const shownPosts = posts;
  // writeUrl omits page (resets to 1). Scroll here only when already on
  // page 1 — otherwise the page-change effect scrolls once.
  const changeSort = (sort: BoardSort) => {
    if (boardSort(urlSearch.sort) === sort) return;
    skipSeed.current = true;
    writeUrl(appliedTags, appliedExcludedTags, appliedQuery, { sort });
    if (page === 1) goToBoardTop();
  };
  // The picture link carries the chips, including ones staged from the
  // strip and not yet applied. Opening one writes that filter into the
  // board history entry first, so Back restores it. No scroll.
  const postSearch = searchFromFilter({
    tags: selectedTags,
    exclude: excludedTags,
    q: appliedQuery,
    sort: boardSort(urlSearch.sort),
  });
  const commitStaged = () => {
    if (sameTagSet(selectedTags, appliedTags) && sameTagSet(excludedTags, appliedExcludedTags)) {
      return;
    }
    commitFilter(selectedTags, excludedTags, appliedQuery);
  };

  // Tags from the current page's posts. Stable until the page set changes.
  const { tags: pageTags } = usePageTags(shownPosts, selectedTags, excludedTags);
  const activeFilterCount = selectedTags.length + excludedTags.length;
  const showTagPanel = shownPosts.length > 0 || activeFilterCount > 0 || loading || defaultsPending;

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
      <BoardHead
        selectedTags={selectedTags}
        excludedTags={excludedTags}
        appliedFilter={appliedFilter}
        commitFilter={commitFilter}
        addTagsFromText={addTagsFromText}
        addTag={addTag}
        addExcludedTag={addExcludedTag}
        differsFromDefaults={differsFromDefaults}
        applyDefaults={applyDefaults}
        changeSort={changeSort}
        searching={searching}
        total={total}
        view={view}
        changeView={changeView}
        pickRandom={pickRandom}
        randomBusy={randomBusy}
        signedIn={signedIn}
        openUpload={() => {
          setDroppedFiles([]);
          setUploadOpen(true);
        }}
        showTagPanel={showTagPanel}
        pageTags={pageTags}
        activeFilterCount={activeFilterCount}
        stageInclude={stageInclude}
        stageExclude={stageExclude}
        loading={loading}
        defaultsPending={defaultsPending}
        failed={failed}
        postCount={posts.length}
      />

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
      <BoardResults
        posts={posts}
        defaultsPending={defaultsPending}
        view={view}
        gridRef={gridRef}
        postSearch={postSearch}
        addTag={addTag}
        commitStaged={commitStaged}
      />
      <button
        type="button"
        aria-label="Back to top"
        title="Back to top"
        className={`${styles.toTop} ${showToTop ? styles.toTopVisible : ""}`}
        onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })}
      >
        ↑
      </button>
      {!loading && !defaultsPending && !failed && posts.length > 0 && (
        <Pager
          page={page}
          total={total}
          pageSize={pageSize}
          onPage={goToBoardPage}
          onPrefetch={prefetchBoard}
        />
      )}
    </section>
  );
}
