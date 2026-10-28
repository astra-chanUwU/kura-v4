import { beforeEach, describe, expect, it } from "vitest";
import { boardMemoryFor, clearBoardMemory, readBoardMemory, writeBoardMemory } from "./boardMemory";

describe("board memory", () => {
  beforeEach(() => sessionStorage.clear());

  it("round-trips the filter for this viewer", () => {
    writeBoardMemory({
      userId: "u1",
      tags: ["Anime"],
      exclude: ["gif"],
      q: " cat ",
      cleared: false,
    });
    expect(boardMemoryFor("u1")).toEqual({
      userId: "u1",
      tags: ["anime"],
      exclude: ["gif"],
      q: "cat",
      cleared: false,
    });
    expect(boardMemoryFor("u2")).toBeNull();
  });

  it("remembers a cleared board as empty", () => {
    writeBoardMemory({ userId: "u1", tags: ["anime"], exclude: [], q: "", cleared: true });
    expect(readBoardMemory()).toEqual({
      userId: "u1",
      tags: [],
      exclude: [],
      q: "",
      cleared: true,
    });
  });

  it("ignores an empty filter that was not a clear", () => {
    writeBoardMemory({ userId: "u1", tags: [], exclude: [], q: " ", cleared: false });
    expect(readBoardMemory()).toBeNull();
  });

  it("drops corrupt storage", () => {
    sessionStorage.setItem("kura.board", "{");
    expect(readBoardMemory()).toBeNull();
  });

  it("clears on sign-out", () => {
    writeBoardMemory({ userId: "u1", tags: ["anime"], exclude: [], q: "", cleared: false });
    clearBoardMemory();
    expect(readBoardMemory()).toBeNull();
  });
});
