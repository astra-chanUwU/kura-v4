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
    expect(screen.queryByRole("searchbox")).not.toBeInTheDocument();
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

  it("draws a font choice in that face and a theme as paper plus accent", () => {
    render(
      <ThemedSelect label="Font" value="serif">
        <option value="serif" data-font="serif">
          Source Serif 4
        </option>
      </ThemedSelect>,
    );
    expect(screen.getByRole("button", { name: "Font" })).toHaveAttribute("data-font", "serif");
    fireEvent.click(screen.getByRole("button", { name: "Font" }));
    const option = screen.getByRole("option", { name: "Source Serif 4" });
    expect(option).toHaveAttribute("data-font", "serif");
    expect(option).toHaveTextContent("Ag");
  });

  it("opens theme choices in a centered dialog", () => {
    const onChange = vi.fn();
    render(
      <ThemedSelect label="Theme" popup value="kura" onChange={onChange}>
        <option value="kura" data-swatch="#b49afa" data-paper="#100f15">
          Kura Dark
        </option>
        <option value="safebooru" data-swatch="#0b6bcb" data-paper="#eef4fc">
          Safebooru
        </option>
      </ThemedSelect>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Theme" }));
    const dialog = screen.getByRole("dialog", { name: "Theme" });
    expect(dialog).toBeInTheDocument();
    fireEvent.click(screen.getByRole("option", { name: /Safebooru/ }));
    expect(onChange).toHaveBeenCalledWith("safebooru");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("keeps a compact popup trigger named Theme without the theme name", () => {
    render(
      <ThemedSelect label="Theme" popup compact value="kura">
        <option value="kura" data-swatch="#b49afa" data-paper="#100f15">
          Kura Dark
        </option>
      </ThemedSelect>,
    );
    const button = screen.getByRole("button", { name: "Theme" });
    expect(button).toHaveAccessibleName("Theme");
    expect(button.textContent ?? "").not.toContain("Kura Dark");
  });

  it("filters popup choices from the search field", () => {
    const onChange = vi.fn();
    render(
      <ThemedSelect label="Theme" popup value="kura" onChange={onChange}>
        <option value="kura" data-swatch="#b49afa" data-paper="#100f15">
          Kura Dark
        </option>
        <option value="safebooru" data-swatch="#0b6bcb" data-paper="#eef4fc">
          Safebooru
        </option>
      </ThemedSelect>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Theme" }));
    const search = screen.getByRole("searchbox", { name: "Search Theme" });
    expect(search).toHaveAttribute("placeholder", "Search theme");
    fireEvent.change(search, { target: { value: "safe" } });
    expect(screen.getByRole("option", { name: /Safebooru/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /Kura Dark/ })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("option", { name: /Safebooru/ }));
    expect(onChange).toHaveBeenCalledWith("safebooru");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows no matches and ignores arrows when nothing fits", () => {
    render(
      <ThemedSelect label="Theme" popup value="kura">
        <option value="kura" data-swatch="#b49afa" data-paper="#100f15">
          Kura Dark
        </option>
        <option value="8008" data-swatch="#800080" data-paper="#111111">
          Eight
        </option>
      </ThemedSelect>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Theme" }));
    const search = screen.getByRole("searchbox", { name: "Search Theme" });
    fireEvent.change(search, { target: { value: "8008" } });
    expect(screen.getByRole("option", { name: /Eight/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /Kura Dark/ })).not.toBeInTheDocument();
    fireEvent.change(search, { target: { value: "zzz" } });
    expect(screen.getByText("No matches")).toBeInTheDocument();
    expect(screen.queryByRole("option")).not.toBeInTheDocument();
    fireEvent.keyDown(search, { key: "ArrowDown" });
    fireEvent.keyDown(search, { key: "Escape" });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Theme" })).toHaveFocus();
  });
});
