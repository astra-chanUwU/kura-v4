import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { UploadDialog } from "./UploadDialog";

const upload = vi.hoisted(() => vi.fn());

vi.mock("../hooks/useKura", () => ({
  duplicateOf: () => undefined,
  useInvalidate: () => vi.fn(),
  useMe: () => ({ me: undefined }),
  usePostActions: () => ({ upload }),
}));

describe("UploadDialog", () => {
  beforeEach(() => {
    upload.mockReset();
    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:preview");
    vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  });

  it("shows the upload limit message", async () => {
    upload.mockRejectedValue(new Error("Upload limit reached. Try again later."));
    const file = new File(["x"], "cap.png", { type: "image/png" });
    render(<UploadDialog initialFiles={[file]} onClose={() => {}} />);
    fireEvent.click(await screen.findByRole("button", { name: "Add 1 to the board" }));
    expect(await screen.findByText("Upload limit reached. Try again later.")).toBeInTheDocument();
  });
});
