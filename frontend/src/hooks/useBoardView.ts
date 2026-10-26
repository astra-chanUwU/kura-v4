import { useState } from "react";

export interface BoardView {
  /** Layout mode: responsive tile grid, or one detail row per picture. */
  mode: "grid" | "rows";
  wide: boolean;
}

const KEY = "kura-board-view";

/** Board layout preference (Grid/Rows + page width), shared by the main
 * board and list pages and persisted under one localStorage key, so a
 * choice on one carries to the other. The old {cols, wide} shape
 * migrates to Grid mode. */
export function useBoardView(): {
  view: BoardView;
  changeView: (next: BoardView) => void;
} {
  const [view, setView] = useState<BoardView>(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(KEY) ?? "{}") as {
        mode?: string;
        wide?: boolean;
      };
      return { mode: saved.mode === "rows" ? "rows" : "grid", wide: !!saved.wide };
    } catch {
      return { mode: "grid", wide: false };
    }
  });
  const changeView = (next: BoardView) => {
    setView(next);
    localStorage.setItem(KEY, JSON.stringify(next));
  };
  return { view, changeView };
}
