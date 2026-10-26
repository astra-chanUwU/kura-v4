import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ThemedSelect } from "./ThemedSelect";

describe("ThemedSelect", () => {
  it("opens a themed listbox and selects with the keyboard", () => {
    const onChange = vi.fn();
    render(
      <ThemedSelect label="Theme" value="kura" onChange={onChange}>
        <option value="kura">Kura Dark</option>
        <option value="rule34">Rule34</option>
      </ThemedSelect>,
    );
    const button = screen.getByRole("button", { name: "Theme" });
    expect(button).toHaveTextContent("Kura Dark");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    fireEvent.keyDown(button, { key: "ArrowDown" });
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    expect(screen.getByRole("option", { name: /Kura Dark/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    fireEvent.keyDown(button, { key: "ArrowDown" });
    fireEvent.keyDown(button, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith("rule34");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  it("keeps form callers working through a hidden input", () => {
    render(
      <form aria-label="List form">
        <ThemedSelect name="visibility" label="Visibility" defaultValue="private">
          <option value="private">Private</option>
          <option value="public">Public</option>
        </ThemedSelect>
      </form>,
    );
    const form = screen.getByRole("form", { name: "List form" });
    expect(new FormData(form as HTMLFormElement).get("visibility")).toBe("private");
    fireEvent.click(screen.getByRole("button", { name: "Visibility" }));
    fireEvent.click(screen.getByRole("option", { name: /Public/ }));
    expect(new FormData(form as HTMLFormElement).get("visibility")).toBe("public");
  });
});
