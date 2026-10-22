import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useSite } from "../hooks/useSite";
import { HomePage } from "./HomePage";

vi.mock("../hooks/useSite");
const mockUseSite = vi.mocked(useSite);
describe("home connection states", () => {
  it("offers recovery when the API is unavailable", () => {
    const retry = vi.fn();
    mockUseSite.mockReturnValue({
      site: undefined,
      loading: false,
      error: new Error("offline"),
      retry,
    });
    render(<HomePage />);
    expect(screen.getByRole("alert")).toHaveTextContent("couldn’t connect");
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(retry).toHaveBeenCalledOnce();
  });
  it("uses the server site name once connected", () => {
    mockUseSite.mockReturnValue({
      site: { name: "My archive", description: "" },
      loading: false,
      error: null,
      retry: vi.fn(),
    });
    render(<HomePage />);
    expect(screen.getByRole("status")).toHaveTextContent("My archive is connected");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
