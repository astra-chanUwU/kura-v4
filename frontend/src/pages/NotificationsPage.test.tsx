import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ToastsProvider } from "../components/Toasts";
import type { NotificationPage } from "../types/notification";
import { NotificationsPage } from "./NotificationsPage";

const hoisted = vi.hoisted(() => ({
  listNotifications: vi.fn(),
  markNotificationsRead: vi.fn(),
}));

const notices: NotificationPage = {
  total: 2,
  unread: 1,
  notifications: [
    {
      id: "n-unread",
      kind: "comment",
      post_id: "post-9",
      actor_username: "bea",
      read: false,
      created_at: "2026-11-01T12:00:00Z",
    },
    {
      id: "n-read",
      kind: "favorite",
      post_id: "post-3",
      actor_username: "cy",
      read: true,
      created_at: "2026-11-02T08:00:00Z",
    },
  ],
};

vi.mock("@tanstack/react-router", async () => {
  const React = await import("react");
  return {
    Link: ({
      to,
      params,
      children,
      ...props
    }: {
      to: string;
      params?: Record<string, string>;
      children?: ReactNode;
    }) => {
      let href = to;
      for (const [key, value] of Object.entries(params ?? {}))
        href = href.replace(`$${key}`, value);
      return React.createElement("a", { href, ...props }, children);
    },
  };
});
vi.mock("../hooks/useSession", () => ({
  useMe: () => ({
    user: { id: "u1", username: "ana", role: "member" },
    loading: false,
    signedIn: true,
  }),
}));
vi.mock("../api/notifications", async () => {
  const actual =
    await vi.importActual<typeof import("../api/notifications")>("../api/notifications");
  return {
    ...actual,
    listNotifications: hoisted.listNotifications,
    markNotificationsRead: hoisted.markNotificationsRead,
  };
});

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ToastsProvider>
        <NotificationsPage />
      </ToastsProvider>
    </QueryClientProvider>,
  );
}

describe("NotificationsPage", () => {
  beforeEach(() => {
    hoisted.listNotifications.mockReset();
    hoisted.markNotificationsRead.mockReset();
    hoisted.listNotifications.mockResolvedValue(notices);
    hoisted.markNotificationsRead.mockResolvedValue(undefined);
  });

  it("marks every notice from Mark all as read", async () => {
    renderPage();
    expect(screen.queryByRole("button", { name: "Mark as read" })).not.toBeInTheDocument();
    fireEvent.click(await screen.findByRole("button", { name: "Mark all as read" }));
    await waitFor(() => expect(hoisted.markNotificationsRead).toHaveBeenCalledWith([]));
  });

  it("marks only the unread notice that was opened", async () => {
    renderPage();
    const unread = await screen.findByRole("link", { name: /@bea commented on your post/ });
    expect(unread).toHaveAttribute("href", "/posts/post-9");
    expect(fireEvent.click(unread)).toBe(true);
    await waitFor(() => expect(hoisted.markNotificationsRead).toHaveBeenCalledTimes(1));
    expect(hoisted.markNotificationsRead).toHaveBeenCalledWith(["n-unread"]);
    expect(hoisted.markNotificationsRead).not.toHaveBeenCalledWith([]);
    expect(unread).toHaveAttribute("href", "/posts/post-9");

    hoisted.markNotificationsRead.mockClear();
    const read = screen.getByRole("link", { name: /@cy favorited your post/ });
    expect(read).toHaveAttribute("href", "/posts/post-3");
    expect(fireEvent.click(read)).toBe(true);
    expect(hoisted.markNotificationsRead).not.toHaveBeenCalled();
  });

  it("toasts a failed mark and still lets the post link activate", async () => {
    hoisted.markNotificationsRead.mockRejectedValue(
      new Error("Could not mark that notice. Try again later."),
    );
    renderPage();
    const unread = await screen.findByRole("link", { name: /@bea commented on your post/ });
    expect(fireEvent.click(unread)).toBe(true);
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not mark that notice.");
    expect(unread).toHaveAttribute("href", "/posts/post-9");
    expect(hoisted.markNotificationsRead).toHaveBeenCalledWith(["n-unread"]);
    expect(hoisted.markNotificationsRead).not.toHaveBeenCalledWith([]);
  });
});
