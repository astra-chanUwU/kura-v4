import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { shortError, ToastsProvider, useToasts } from "./Toasts";

function Trigger() {
  const toasts = useToasts();
  return (
    <button
      type="button"
      onClick={() =>
        toasts.push(
          "anime.png — Already on the board. Extra detail that should not matter.",
          "error",
        )
      }
    >
      fail
    </button>
  );
}

describe("toasts", () => {
  it("shows a dismissible toast when pushed", () => {
    render(
      <ToastsProvider>
        <Trigger />
      </ToastsProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "fail" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Already on the board.");
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("shortens long errors to the first sentence", () => {
    expect(shortError(new Error("Already on the board. It wasn't added again."))).toBe(
      "Already on the board.",
    );
  });
});
