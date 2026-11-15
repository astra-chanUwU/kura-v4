import type { Dispatch, SetStateAction } from "react";
import { boardSort, type BoardFilter, type BoardSort } from "../domain/boardFilter";
import type { PageTag } from "../domain/pageTags";
import type { BoardView } from "../hooks/useBoardView";
import { tagTokens } from "../hooks/useKura";
import type { TagCount } from "../types/api";
import styles from "../pages/HomePage.module.css";
import { BoardTagPanel } from "./BoardTagPanel";
import { BoardViewControls } from "./BoardViewControls";
import { SaveCurrentSearchButton } from "./SavedSearches";

type BoardHeadProps = {
  selectedTags: string[];
  excludedTags: string[];
  appliedTags: string[];
  appliedExcludedTags: string[];
  appliedQuery: string;
  searchText: string;
  setSearchText: Dispatch<SetStateAction<string>>;
  searchFocused: boolean;
  setSearchFocused: Dispatch<SetStateAction<boolean>>;
  highlight: number;
  setHighlight: Dispatch<SetStateAction<number>>;
  suggestions: TagCount[];
  commitFilter: (include: string[], exclude: string[], q: string, scroll?: boolean) => void;
  applyFilters: () => void;
  addTagsFromText: (text: string) => void;
  addTag: (tag: string) => void;
  addExcludedTag: (tag: string) => void;
  filtersDirty: boolean;
  pendingCount: number;
  clearFilters: () => void;
  differsFromDefaults: boolean;
  applyDefaults: () => void;
  urlSort: BoardSort | undefined;
  changeSort: (sort: BoardSort) => void;
  filterActive: boolean;
  searching: boolean;
  total: number;
  view: BoardView;
  changeView: (next: BoardView) => void;
  appliedFilter: BoardFilter;
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
};

/** Sticky board head: centered search → controls → page-tag panel. */
export function BoardHead({
  selectedTags,
  excludedTags,
  appliedTags,
  appliedExcludedTags,
  appliedQuery,
  searchText,
  setSearchText,
  searchFocused,
  setSearchFocused,
  highlight,
  setHighlight,
  suggestions,
  commitFilter,
  applyFilters,
  addTagsFromText,
  addTag,
  addExcludedTag,
  filtersDirty,
  pendingCount,
  clearFilters,
  differsFromDefaults,
  applyDefaults,
  urlSort,
  changeSort,
  filterActive,
  searching,
  total,
  view,
  changeView,
  appliedFilter,
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
}: BoardHeadProps) {
  return (
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
                  addTagsFromText(searchText);
                } else {
                  const pick = suggestions[highlight] ?? suggestions[0];
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
            <button type="button" className={styles.defaultsButton} onClick={applyDefaults}>
              My defaults
            </button>
          )}
        </div>
        <div className={styles.sortGroup} role="group" aria-label="Sort">
          <button
            type="button"
            aria-pressed={boardSort(urlSort) === "newest"}
            className={boardSort(urlSort) === "newest" ? styles.viewActive : ""}
            onClick={() => changeSort("newest")}
          >
            Newest
          </button>
          <button
            type="button"
            aria-pressed={boardSort(urlSort) === "oldest"}
            className={boardSort(urlSort) === "oldest" ? styles.viewActive : ""}
            onClick={() => changeSort("oldest")}
          >
            Oldest
          </button>
          <button
            type="button"
            aria-pressed={boardSort(urlSort) === "favorites"}
            className={boardSort(urlSort) === "favorites" ? styles.viewActive : ""}
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
            <button
              type="button"
              className={styles.uploadButton}
              onClick={openUpload}
            >
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
  );
}
