import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { HistoryActor } from "../types/api";
import { ActorCombobox } from "./ActorCombobox";

const ada: HistoryActor = { id: "u1", username: "ada" };
const astra: HistoryActor = { id: "u2", username: "astra" };

function renderBox(
  overrides: Partial<{
    value: HistoryActor | null;
    query: string;
    suggestions: HistoryActor[];
    loading: boolean;
  }> = {},
) {
  const onChange = vi.fn();
  const onQueryChange = vi.fn();
  const view = render(
    <ActorCombobox
      value={overrides.value ?? null}
      onChange={onChange}
      query={overrides.query ?? ""}
      onQueryChange={onQueryChange}
      suggestions={overrides.suggestions ?? []}
      loading={overrides.loading ?? false}
    />,
  );
  return { ...view, onChange, onQueryChange };
}

describe("ActorCombobox", () => {
  it("names the field Person and does not list people for an empty query", () => {
    const { rerender, onChange, onQueryChange } = renderBox({
      suggestions: [ada],
    });
    const input = screen.getByRole("combobox", { name: "Person" });
    expect(input).toHaveAttribute("placeholder", "Everyone");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "@ada" })).not.toBeInTheDocument();

    rerender(
      <ActorCombobox
        value={null}
        onChange={onChange}
        query="   "
        onQueryChange={onQueryChange}
        suggestions={[ada]}
      />,
    );
    expect(screen.queryByRole("option", { name: "@ada" })).not.toBeInTheDocument();
  });

  it("calls onChange with the clicked person's id and username", () => {
    const { onChange } = renderBox({ query: "a", suggestions: [ada, astra] });
    fireEvent.click(screen.getByRole("option", { name: "@astra" }));
    expect(onChange).toHaveBeenCalledWith({ id: "u2", username: "astra" });
  });

  it("moves the active option with the arrow keys and selects it with Enter", () => {
    const { onChange } = renderBox({ query: "a", suggestions: [ada, astra] });
    const input = screen.getByRole("combobox", { name: "Person" });
    expect(screen.getByRole("option", { name: "@ada" })).toHaveAttribute("aria-selected", "true");
    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(screen.getByRole("option", { name: "@astra" })).toHaveAttribute("aria-selected", "true");
    fireEvent.keyDown(input, { key: "ArrowUp" });
    expect(screen.getByRole("option", { name: "@ada" })).toHaveAttribute("aria-selected", "true");
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onChange).toHaveBeenCalledWith({ id: "u1", username: "ada" });
  });

  it("finishes a prefix in the field and accepts it with Tab or Right arrow", () => {
    const { onChange } = renderBox({ query: "ad", suggestions: [ada, astra] });
    const input = screen.getByRole("combobox", { name: "Person" }) as HTMLInputElement;
    expect(input).toHaveAttribute("aria-autocomplete", "both");
    expect(screen.getByText("a", { selector: "span" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "@ada" }).querySelector("mark")).toHaveTextContent(
      "ad",
    );

    input.setSelectionRange(1, 1);
    fireEvent.keyDown(input, { key: "ArrowRight" });
    expect(onChange).not.toHaveBeenCalled();

    input.setSelectionRange(2, 2);
    fireEvent.keyDown(input, { key: "ArrowRight" });
    expect(onChange).toHaveBeenCalledWith(ada);
  });

  it("accepts the completed name with Tab", () => {
    const { onChange } = renderBox({ query: "ast", suggestions: [astra, ada] });
    fireEvent.keyDown(screen.getByRole("combobox", { name: "Person" }), { key: "Tab" });
    expect(onChange).toHaveBeenCalledWith(astra);
  });

  it("does not invent a prefix when the match is in the middle of the name", () => {
    renderBox({ query: "tra", suggestions: [astra] });
    expect(screen.queryByText("a", { selector: "span" })).not.toBeInTheDocument();
    expect(screen.getByRole("option", { name: "@astra" }).querySelector("mark")).toHaveTextContent(
      "tra",
    );
  });

  it("says when nobody matches, and stays quiet while a search is still loading", () => {
    const { rerender, onChange, onQueryChange } = renderBox({ query: "zzz" });
    expect(screen.getByText("No one named that.")).toBeInTheDocument();

    rerender(
      <ActorCombobox
        value={null}
        onChange={onChange}
        query="zzz"
        onQueryChange={onQueryChange}
        suggestions={[]}
        loading
      />,
    );
    expect(screen.queryByText("No one named that.")).not.toBeInTheDocument();
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  it("keeps the suggestion list closed while a person is selected", () => {
    const { onChange, onQueryChange, rerender } = renderBox({
      value: ada,
      query: "ada",
      suggestions: [ada, astra],
    });
    expect(screen.getByText("@ada")).toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "Person" })).not.toBeInTheDocument();
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Show everyone's changes" }));
    expect(onChange).toHaveBeenCalledWith(null);
    expect(onQueryChange).toHaveBeenCalledWith("");

    rerender(
      <ActorCombobox
        value={null}
        onChange={onChange}
        query=""
        onQueryChange={onQueryChange}
        suggestions={[]}
      />,
    );
    expect(screen.getByRole("combobox", { name: "Person" })).toHaveFocus();
  });

  it("closes on Escape and on a click outside", () => {
    render(
      <div>
        <ActorCombobox
          value={null}
          onChange={vi.fn()}
          query="ada"
          onQueryChange={vi.fn()}
          suggestions={[ada]}
        />
        <button type="button">Elsewhere</button>
      </div>,
    );
    const input = screen.getByRole("combobox", { name: "Person" });
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    fireEvent.keyDown(input, { key: "Escape" });
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();

    fireEvent.keyDown(input, { key: "ArrowDown" });
    expect(screen.getByRole("listbox")).toBeInTheDocument();
    fireEvent.pointerDown(screen.getByRole("button", { name: "Elsewhere" }));
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});
