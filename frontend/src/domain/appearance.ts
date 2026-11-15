// Pure appearance logic mirroring internal/accounts (Themes/Fonts).
// No React, no transport.
export type ThemeId =
  | "kura"
  | "safebooru"
  | "rule34"
  | "umineko"
  | "bernkastel"
  | "lambdadelta"
  | "beatrice"
  | "erika"
  | "battler"
  | "virgilia"
  | "catppuccin"
  | "danbooru";
export type FontId = "system" | "serif" | "rounded" | "mono" | "hack";

export const THEMES: ThemeId[] = [
  "kura",
  "safebooru",
  "rule34",
  "umineko",
  "bernkastel",
  "lambdadelta",
  "beatrice",
  "erika",
  "battler",
  "virgilia",
  "catppuccin",
  "danbooru",
];
export const FONTS: FontId[] = ["system", "serif", "rounded", "mono", "hack"];

export const THEME_LABELS: Record<ThemeId, string> = {
  kura: "Kura Dark",
  safebooru: "Safebooru",
  rule34: "Rule34",
  umineko: "Umineko",
  bernkastel: "Bernkastel",
  lambdadelta: "Lambdadelta",
  beatrice: "Beatrice",
  erika: "Erika",
  battler: "Battler",
  virgilia: "Virgilia",
  catppuccin: "Catppuccin",
  danbooru: "Danbooru",
};

export const THEME_SWATCHES: Record<ThemeId, string> = {
  kura: "#b49afa",
  safebooru: "#0b6bcb",
  rule34: "#8bc34a",
  umineko: "#c9a227",
  bernkastel: "#4a90d9",
  lambdadelta: "#ff69b4",
  beatrice: "#d4a84b",
  erika: "#00b4d8",
  battler: "#e85a70",
  virgilia: "#8b1a2d",
  catppuccin: "#cba6f7",
  danbooru: "#0069df",
};

/** Page color behind each accent, so a theme chip shows paper and ink. */
export const THEME_PAPERS: Record<ThemeId, string> = {
  kura: "#100f15",
  safebooru: "#eef4fc",
  rule34: "#0b0f08",
  umineko: "#120d10",
  bernkastel: "#06090f",
  lambdadelta: "#120810",
  beatrice: "#0f0a08",
  erika: "#060a0f",
  battler: "#0f0608",
  virgilia: "#e8eaf0",
  catppuccin: "#1e1e2e",
  danbooru: "#f5f7fa",
};

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
