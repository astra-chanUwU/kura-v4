import { useState } from "react";
import {
  boardSort,
  isFilterActive,
  tagTokens,
  type BoardFilter,
  type BoardSort,
} from "../domain/boardFilter";
import type { PageTag } from "../domain/pageTags";
import { useDebouncedValue, useTagSuggestions } from "../hooks/useBoardData";
import type { BoardView } from "../hooks/useBoardView";
import styles from "./board.module.css";
import { BoardTagPanel } from "./BoardTagPanel";
import { BoardViewControls } from "./BoardViewControls";
import { SaveCurrentSearchButton } from "./SavedSearches";

type BoardHeadProps = {
  selectedTags: string[];
  excludedTags: string[];
  appliedFilter: BoardFilter;
  commitFilter: (include: string[], exclude: string[], q: string, scroll?: boolean) => void;
  addTagsFromText: (text: string) => void;
  addTag: (tag: string) => void;
  addExcludedTag: (tag: string) => void;
  differsFromDefaults: boolean;
  applyDefaults: () => void;
  changeSort: (sort: BoardSort) => void;
  searching: boolean;
  total: number;
  view: BoardView;
  changeView: (next: BoardView) => void;
  pickRandom: () => void;
  randomBusy: boolean;
  signedIn: boolean;
  openUpload: () => void;
  showTagPanel: boolean;
  pageTags: PageTag[];
  activeFilterCount: number;
  stageInclude: (tag: string) => void;
  stageExclude: (tag: string) => void;
  loading: boolean;
  defaultsPending: boolean;
  failed: unknown;
  postCount: number;
};

/** Sticky board head: centered search → controls → page-tag panel. */
export function BoardHead({
  selectedTags,
  excludedTags,
  appliedFilter,
  commitFilter,
  addTagsFromText,
  addTag,
  addExcludedTag,
  differsFromDefaults,
  applyDefaults,
  changeSort,
  searching,
  total,
  view,
  changeView,
  pickRandom,
  randomBusy,
  signedIn,
  openUpload,
  showTagPanel,
  pageTags,
  activeFilterCount,
  stageInclude,
  stageExclude,
  loading,
  defaultsPending,
  failed,
  postCount,
}: BoardHeadProps) {
  const appliedTags = appliedFilter.tags;
  const appliedExcludedTags = appliedFilter.exclude;
  const appliedQuery = appliedFilter.q;
  // The address is the applied filter. A new filter replaces the draft;
  // keystrokes between those updates stay local.
  const appliedKey = [
    appliedTags.join("\0"),
    appliedExcludedTags.join("\0"),
    appliedQuery,
    appliedFilter.sort ?? "",
  ].join("\u001f");
  const [searchText, setSearchText] = useState(appliedQuery);
  const [searchFocused, setSearchFocused] = useState(false);
  const [highlight, setHighlight] = useState(0);
  const [seenKey, setSeenKey] = useState(appliedKey);
  if (seenKey !== appliedKey) {
    setSeenKey(appliedKey);
    setSearchText(appliedQuery);
    setHighlight(0);
  }
  const debouncedSearchText = useDebouncedValue(searchText, 200);
  const tagSuggestions = useTagSuggestions(debouncedSearchText, appliedFilter, searchFocused);
  const suggestions = tagSuggestions.suggestions.filter(
    (item) => !selectedTags.includes(item.tag) && !excludedTags.includes(item.tag),
  );
  const pendingCount =
    selectedTags.filter((t) => !appliedTags.includes(t)).length +
    appliedTags.filter((t) => !selectedTags.includes(t)).length +
    excludedTags.filter((t) => !appliedExcludedTags.includes(t)).length +
    appliedExcludedTags.filter((t) => !excludedTags.includes(t)).length +
    (searchText.trim() !== appliedQuery ? 1 : 0);
  const filtersDirty = pendingCount > 0;
  const filterActive = isFilterActive(appliedFilter);
  const applyFilters = () => {
    const q = searchText.trim();
    setSearchText(q);
    commitFilter(selectedTags, excludedTags, q, true);
  };
  const clearFilters = () => {
    setSearchText("");
    commitFilter([], [], "", true);
  };
  const submitTypedTags = (text: string) => {
    setSearchText("");
    setHighlight(0);
    addTagsFromText(text);
  };
  const chooseTag = (tag: string, exclude: boolean) => {
    setSearchText("");
    setHighlight(0);
    if (exclude) addExcludedTag(tag);
    else addTag(tag);
  };

  return (
    <>
      <div className={styles.boardHead}>
        <div className={styles.searchRow}>
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
                  setHighlight((h) => Math.max(0, Math.min(suggestions.length - 1, h + delta)));
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
                    submitTypedTags(searchText);
                  } else {
                    const pick = suggestions[highlight] ?? suggestions[0];
                    if (pick) chooseTag(pick.tag, false);
                    else submitTypedTags(searchText);
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
            {searchFocused && suggestions.length > 0 && (
              <ul className={styles.suggestions} aria-label="Tag suggestions">
                {suggestions.map((s, i) => (
                  <li key={s.tag} className={styles.suggestionRow}>
                    <button
                      type="button"
                      aria-label={`Exclude tag ${s.tag}`}
                      title={`Exclude ${s.tag}`}
                      className={`${styles.suggestionSide} ${styles.suggestionExclude}`}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        chooseTag(s.tag, true);
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
                        chooseTag(s.tag, false);
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
                        chooseTag(s.tag, false);
                      }}
                    >
                      +
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div className={styles.controlsRow}>
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
              <button
                type="button"
                className={styles.defaultsButton}
                onClick={() => {
                  setSearchText("");
                  applyDefaults();
                }}
              >
                My defaults
              </button>
            )}
          </div>
          <div className={styles.sortGroup} role="group" aria-label="Sort">
            <button
              type="button"
              aria-pressed={boardSort(appliedFilter.sort) === "newest"}
              className={boardSort(appliedFilter.sort) === "newest" ? styles.viewActive : ""}
              onClick={() => changeSort("newest")}
            >
              Newest
            </button>
            <button
              type="button"
              aria-pressed={boardSort(appliedFilter.sort) === "oldest"}
              className={boardSort(appliedFilter.sort) === "oldest" ? styles.viewActive : ""}
              onClick={() => changeSort("oldest")}
            >
              Oldest
            </button>
            <button
              type="button"
              aria-pressed={boardSort(appliedFilter.sort) === "favorites"}
              className={boardSort(appliedFilter.sort) === "favorites" ? styles.viewActive : ""}
              onClick={() => changeSort("favorites")}
            >
              Favorites
            </button>
          </div>
          {filterActive &&
            (searching ? (
              <span className={styles.searchMeta}>Searching…</span>
            ) : (
              <span className={styles.searchMeta}>
                {total} result{total === 1 ? "" : "s"}
                {appliedTags.length > 0 ? " with all selected tags" : ""}
                {appliedExcludedTags.length > 0
                  ? `, excluding ${appliedExcludedTags.length} tag${appliedExcludedTags.length === 1 ? "" : "s"}`
                  : ""}
                {filtersDirty ? " · unapplied changes" : ""}
              </span>
            ))}
          <div className={styles.viewControls}>
            <BoardViewControls view={view} onChange={changeView} />
            <SaveCurrentSearchButton filter={appliedFilter} />
            <button
              type="button"
              className={styles.uploadButton}
              onClick={pickRandom}
              disabled={randomBusy}
              aria-busy={randomBusy}
            >
              Random
            </button>
            {signedIn && (
              <button type="button" className={styles.uploadButton} onClick={openUpload}>
                ⬆ Upload
              </button>
            )}
          </div>
        </div>

        {showTagPanel && (
          <div className={styles.tagPanel}>
            <BoardTagPanel
              tags={pageTags}
              activeCount={activeFilterCount}
              onInclude={stageInclude}
              onExclude={stageExclude}
              onClear={clearFilters}
              loading={loading || defaultsPending}
            />
          </div>
        )}
      </div>
      {!loading && !failed && !defaultsPending && !filterActive && postCount === 0 && (
        <div className={styles.empty}>
          <p>No images yet</p>
        </div>
      )}
      {filterActive && !loading && !defaultsPending && !searching && postCount === 0 && (
        <div className={styles.empty}>
          <p>
            {searchText.trim()
              ? `Nothing matches “${searchText.trim()}”`
              : "Nothing matches your tag filters"}
          </p>
          <p>Try removing a tag, a different spelling, or a tag like “gif”.</p>
        </div>
      )}
    </>
  );
}
