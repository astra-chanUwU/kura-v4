import type { BoardView } from "../hooks/useBoardView";
import { IconGrid, IconNarrow, IconRows, IconWide } from "./icons";
import styles from "./BoardViewControls.module.css";

/** Layout mode (Grid/Rows) and page-width pickers, as on the main board. */
export function BoardViewControls({
  view,
  onChange,
}: {
  view: BoardView;
  onChange: (next: BoardView) => void;
}) {
  return (
    <div className={styles.viewControls} role="group" aria-label="Board layout">
      <span className={styles.viewGroup} role="group" aria-label="View mode">
        <button
          type="button"
          aria-pressed={view.mode === "grid"}
          aria-label="Grid"
          className={view.mode === "grid" ? styles.viewActive : ""}
          title="Tile grid"
          onClick={() => onChange({ ...view, mode: "grid" })}
        >
          <IconGrid />
        </button>
        <button
          type="button"
          aria-pressed={view.mode === "rows"}
          aria-label="Rows"
          className={view.mode === "rows" ? styles.viewActive : ""}
          title="One row per picture, with details"
          onClick={() => onChange({ ...view, mode: "rows" })}
        >
          <IconRows />
        </button>
      </span>
      <span className={styles.viewGroup} role="group" aria-label="Page width">
        <button
          type="button"
          aria-pressed={!view.wide}
          aria-label="Narrow"
          className={!view.wide ? styles.viewActive : ""}
          title="Narrow, centered page"
          onClick={() => onChange({ ...view, wide: false })}
        >
          <IconNarrow />
        </button>
        <button
          type="button"
          aria-pressed={view.wide}
          aria-label="Wide"
          className={view.wide ? styles.viewActive : ""}
          title="Use the full width of the screen"
          onClick={() => onChange({ ...view, wide: true })}
        >
          <IconWide />
        </button>
      </span>
    </div>
  );
}
