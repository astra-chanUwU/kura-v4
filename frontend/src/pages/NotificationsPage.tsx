import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { notificationPageSize } from "../api/notifications";
import { shortError, useToasts } from "../components/Toasts";
import { useMarkNotificationsRead, useNotifications } from "../hooks/useNotifications";
import { useMe } from "../hooks/useKura";
import type { NotificationItem } from "../types/notification";
import styles from "./NotificationsPage.module.css";

function line(item: NotificationItem): string {
  const who = item.actor_username ? `@${item.actor_username}` : "Someone";
  if (item.kind === "comment") return `${who} commented on your post`;
  return `${who} favorited your post`;
}

function when(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

export function NotificationsPage() {
  const { user, loading } = useMe();
  const toasts = useToasts();
  const [offset, setOffset] = useState(0);
  const query = useNotifications(offset);
  const mark = useMarkNotificationsRead();
  const items = query.data?.notifications ?? [];
  const total = query.data?.total ?? 0;
  const unread = query.data?.unread ?? 0;
  const waiting = !!user && query.isPending && items.length === 0;

  return (
    <section className={styles.page} aria-labelledby="notifications-title">
      <div className={styles.head}>
        <h1 id="notifications-title">Notifications</h1>
        {!!user && unread > 0 && (
          <button
            type="button"
            className={styles.button}
            disabled={mark.isPending}
            onClick={() =>
              mark.mutate([], {
                onError: (error: unknown) => toasts.push(shortError(error), "error"),
              })
            }
          >
            Mark as read
          </button>
        )}
      </div>
      {loading && <p className={styles.muted}>Loading…</p>}
      {!loading && !user && <p className={styles.muted}>Sign in to see notifications.</p>}
      {waiting && <p className={styles.muted}>Loading…</p>}
      {!!user && query.error && items.length === 0 && (
        <p role="alert" className={styles.error}>
          {shortError(query.error)}
        </p>
      )}
      {!!user && !waiting && !query.error && items.length === 0 && (
        <p className={styles.muted}>No notifications.</p>
      )}
      {items.length > 0 && (
        <ul className={styles.list}>
          {items.map((item) => (
            <li key={item.id} className={item.read ? styles.item : styles.unread}>
              <Link to="/posts/$id" params={{ id: item.post_id }} className={styles.link}>
                <span>{line(item)}</span>
                <time className={styles.time} dateTime={item.created_at}>
                  {when(item.created_at)}
                </time>
              </Link>
            </li>
          ))}
        </ul>
      )}
      {total > notificationPageSize && (
        <div className={styles.pager}>
          <button
            type="button"
            className={styles.button}
            disabled={offset === 0}
            onClick={() => setOffset((current) => Math.max(0, current - notificationPageSize))}
          >
            Previous
          </button>
          <button
            type="button"
            className={styles.button}
            disabled={offset + items.length >= total}
            onClick={() => setOffset((current) => current + notificationPageSize)}
          >
            Next
          </button>
        </div>
      )}
    </section>
  );
}
