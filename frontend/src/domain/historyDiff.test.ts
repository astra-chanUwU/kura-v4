import { describe, expect, it } from "vitest";
import { diffStates } from "./historyDiff";

describe("diffStates", () => {
  it("marks added and removed tags while keeping shared tags as context", () => {
    const lines = diffStates(
      { artist: "ana", tags: ["keep_me", "old_tag"], rating: "sfw", hidden: false, deleted: false },
      { artist: "ana", tags: ["keep_me", "new_tag"], rating: "sfw", hidden: false, deleted: false },
    );
    expect(lines).toContainEqual({ field: "tags", kind: "removed", text: "tags: old_tag" });
    expect(lines).toContainEqual({ field: "tags", kind: "added", text: "tags: new_tag" });
    expect(lines).toContainEqual({ field: "tags", kind: "unchanged", text: "tags: keep_me" });
    expect(lines).toContainEqual({ field: "artist", kind: "unchanged", text: "artist: ana" });
  });

  it("shows a changed scalar as a removed line plus an added line", () => {
    const lines = diffStates(
      { artist: "ana", tags: [], rating: "sfw", hidden: false, deleted: false },
      { artist: "bob", tags: [], rating: "nsfw", hidden: true, deleted: false },
    );
    expect(lines).toContainEqual({ field: "artist", kind: "removed", text: "artist: ana" });
    expect(lines).toContainEqual({ field: "artist", kind: "added", text: "artist: bob" });
    expect(lines).toContainEqual({ field: "rating", kind: "removed", text: "rating: SFW" });
    expect(lines).toContainEqual({ field: "rating", kind: "added", text: "rating: NSFW" });
    expect(lines).toContainEqual({ field: "hidden", kind: "removed", text: "hidden: no" });
    expect(lines).toContainEqual({ field: "hidden", kind: "added", text: "hidden: yes" });
  });

  it("shows a role change and a creation as additions only", () => {
    expect(diffStates({ role: "member" }, { role: "moderator" })).toEqual([
      { field: "role", kind: "removed", text: "role: member" },
      { field: "role", kind: "added", text: "role: moderator" },
    ]);
    const created = diffStates(
      { artist: "", tags: [], rating: "sfw", hidden: false, deleted: false },
      { artist: "ana", tags: ["new_tag"], rating: "sfw", hidden: false, deleted: false },
    );
    expect(created.some((line) => line.kind === "removed")).toBe(false);
    expect(created).toContainEqual({ field: "tags", kind: "added", text: "tags: new_tag" });
    expect(created).toContainEqual({ field: "artist", kind: "added", text: "artist: ana" });
  });

  it("reports no added or removed lines when nothing changed", () => {
    const state = { artist: "ana", tags: ["a"], rating: "sfw", hidden: false, deleted: false };
    const lines = diffStates(state, { ...state });
    expect(lines.length).toBeGreaterThan(0);
    expect(lines.every((line) => line.kind === "unchanged")).toBe(true);
  });
});
