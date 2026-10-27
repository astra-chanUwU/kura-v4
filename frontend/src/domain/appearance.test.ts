import { describe, expect, it } from "vitest";
import {
  DEFAULT_FONT,
  DEFAULT_THEME,
  FONT_LABELS,
  FONTS,
  THEME_LABELS,
  THEMES,
  isFontId,
  isThemeId,
  normalizeFont,
  normalizeTheme,
} from "./appearance";

describe("appearance", () => {
  it("lists every theme and real font with labels", () => {
    expect(THEMES).toEqual([
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
    ]);
    expect(FONTS).toEqual(["system", "serif", "rounded", "mono", "hack"]);
    expect(THEME_LABELS.umineko).toBe("Umineko");
    expect(THEME_LABELS.catppuccin).toBe("Catppuccin");
    expect(FONT_LABELS.system).toBe("Atkinson Hyperlegible");
    expect(FONT_LABELS.mono).toBe("JetBrains Mono");
    expect(FONT_LABELS.hack).toBe("Hack");
    expect(DEFAULT_THEME).toBe("kura");
    expect(DEFAULT_FONT).toBe("system");
  });

  it("fails closed to defaults for unknown values", () => {
    expect(isThemeId("kura")).toBe(true);
    expect(isThemeId("neon")).toBe(false);
    expect(isFontId("mono")).toBe(true);
    expect(isFontId("comic")).toBe(false);
    expect(normalizeTheme("neon")).toBe("kura");
    expect(normalizeTheme(undefined)).toBe("kura");
    expect(normalizeFont("comic")).toBe("system");
    expect(normalizeFont(null)).toBe("system");
    expect(normalizeTheme("safebooru")).toBe("safebooru");
    expect(normalizeFont("rounded")).toBe("rounded");
  });
});
