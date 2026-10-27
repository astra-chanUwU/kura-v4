import { useCallback, useEffect, useState } from "react";
import {
  DEFAULT_FONT,
  DEFAULT_THEME,
  normalizeFont,
  normalizeTheme,
  type FontId,
  type ThemeId,
} from "../domain/appearance";

/** Apply an appearance to the document and remember it for pre-paint. */
export function applyAppearance(theme: ThemeId, font: FontId) {
  const root = document.documentElement;
  if (theme === DEFAULT_THEME) delete root.dataset.theme;
  else root.dataset.theme = theme;
  if (font === DEFAULT_FONT) delete root.dataset.font;
  else root.dataset.font = font;
  try {
    localStorage.setItem("kura-theme", theme);
    localStorage.setItem("kura-font", font);
  } catch {
    // Private mode: the in-page appearance still applies.
  }
}

function storedAppearance(): { theme: ThemeId; font: FontId } {
  try {
    return {
      theme: normalizeTheme(localStorage.getItem("kura-theme")),
      font: normalizeFont(localStorage.getItem("kura-font")),
    };
  } catch {
    return { theme: DEFAULT_THEME, font: DEFAULT_FONT };
  }
}

/**
 * The viewer's current appearance. Anonymous choices live only in
 * localStorage; signed-in choices follow the account preference (the
 * default across devices) until the viewer picks a new one, which is
 * persisted back.
 */
export function useAppearance(
  serverTheme: unknown,
  serverFont: unknown,
  signedIn: boolean,
  persist: (theme: ThemeId, font: FontId) => void,
) {
  const [stored] = useState(storedAppearance);
  const [picked, setPicked] = useState<{ theme: ThemeId; font: FontId } | null>(null);

  // A newer account preference (saved here, in Me, or on another
  // device) replaces a stale local pick. Adjusting state during
  // render is the React-endorsed way to track a changed prop.
  const serverValue = {
    theme: normalizeTheme(serverTheme),
    font: normalizeFont(serverFont),
  };
  const [seenServer, setSeenServer] = useState(serverValue);
  if (
    signedIn &&
    (seenServer.theme !== serverValue.theme || seenServer.font !== serverValue.font)
  ) {
    setSeenServer(serverValue);
    if (picked && (picked.theme !== serverValue.theme || picked.font !== serverValue.font)) {
      setPicked(null);
    }
  }

  const appearance =
    picked ??
    (signedIn ? { theme: normalizeTheme(serverTheme), font: normalizeFont(serverFont) } : stored);

  // Sync the effective appearance to the document (external system).
  useEffect(() => {
    applyAppearance(appearance.theme, appearance.font);
  }, [appearance.theme, appearance.font]);

  const change = useCallback(
    (next: Partial<{ theme: ThemeId; font: FontId }>) => {
      setPicked((cur) => {
        const base =
          cur ??
          (signedIn
            ? { theme: normalizeTheme(serverTheme), font: normalizeFont(serverFont) }
            : stored);
        const value = { ...base, ...next };
        if (signedIn) persist(value.theme, value.font);
        return value;
      });
    },
    [signedIn, persist, serverTheme, serverFont, stored],
  );

  return { ...appearance, change };
}
