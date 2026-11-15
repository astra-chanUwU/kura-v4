// Pure appearance logic mirroring internal/accounts (Themes/Fonts).
// No React, no transport.
// Theme ids, labels, swatches, and papers live in themeCatalog.ts.
import {
  THEME_IDS,
  THEME_LABELS,
  THEME_PAPERS,
  THEME_SWATCHES,
  type ThemeId,
} from "./themeCatalog";

export type { ThemeId };
export { THEME_LABELS, THEME_PAPERS, THEME_SWATCHES };

export type FontId = "system" | "serif" | "rounded" | "mono" | "hack";

export const THEMES: ThemeId[] = [...THEME_IDS];
export const FONTS: FontId[] = ["system", "serif", "rounded", "mono", "hack"];

/** Real self-hosted fonts behind each id (see main.tsx imports). */
export const FONT_LABELS: Record<FontId, string> = {
  system: "Atkinson Hyperlegible",
  serif: "Source Serif 4",
  rounded: "Nunito",
  mono: "JetBrains Mono",
  hack: "Hack",
};

export const DEFAULT_THEME: ThemeId = "kura";
export const DEFAULT_FONT: FontId = "system";

export function isThemeId(value: unknown): value is ThemeId {
  return typeof value === "string" && (THEMES as string[]).includes(value);
}

export function isFontId(value: unknown): value is FontId {
  return typeof value === "string" && (FONTS as string[]).includes(value);
}

/** Fail closed to the default for unknown/missing values. */
export function normalizeTheme(value: unknown): ThemeId {
  return isThemeId(value) ? value : DEFAULT_THEME;
}

export function normalizeFont(value: unknown): FontId {
  return isFontId(value) ? value : DEFAULT_FONT;
}
