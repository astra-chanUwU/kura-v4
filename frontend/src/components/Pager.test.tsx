import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Pager } from "./Pager";

describe("Pager", () => {
  it("renders nothing when everything fits on one page", () => {
    const { container } = render(
      <Pager page={1} total={10} pageSize={40} onPage={vi.fn()} />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("jumps to a clamped page via Go, ignoring empty submit and same page", () => {
    const onPage = vi.fn();
    render(<Pager page={2} total={200} pageSize={40} onPage={onPage} />);

    const input = screen.getByRole("spinbutton", { name: "Go to page" });
    const go = screen.getByRole("button", { name: "Go" });

    fireEvent.change(input, { target: { value: "" } });
    fireEvent.click(go);
    expect(onPage).not.toHaveBeenCalled();

    fireEvent.change(input, { target: { value: "2" } });
    fireEvent.click(go);
    expect(onPage).not.toHaveBeenCalled();

    fireEvent.change(input, { target: { value: "99" } });
    fireEvent.click(go);
    expect(onPage).toHaveBeenCalledTimes(1);
    expect(onPage).toHaveBeenCalledWith(5);
  });

  it("resets empty input to the current page on blur", () => {
    render(<Pager page={3} total={200} pageSize={40} onPage={vi.fn()} />);
    const input = screen.getByRole("spinbutton", { name: "Go to page" });
    fireEvent.change(input, { target: { value: "" } });
    fireEvent.blur(input);
    expect(input).toHaveValue(3);
  });

  it("prefers Enter submit and keeps number-button prefetch", () => {
    const onPage = vi.fn();
    const onPrefetch = vi.fn();
    render(
      <Pager page={1} total={200} pageSize={40} onPage={onPage} onPrefetch={onPrefetch} />,
    );

    fireEvent.focus(screen.getByRole("button", { name: "2" }));
    expect(onPrefetch).toHaveBeenCalledWith(2);

    const input = screen.getByRole("spinbutton", { name: "Go to page" });
    fireEvent.change(input, { target: { value: "4" } });
    fireEvent.submit(input.closest("form")!);
    expect(onPage).toHaveBeenCalledWith(4);
  });
});
