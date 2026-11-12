import { Link } from "@tanstack/react-router";
import { useUnreadNotifications } from "../hooks/useNotifications";
import styles from "./NotificationBell.module.css";

export function NotificationBell() {
  const unread = useUnreadNotifications();
  const label = unread > 0 ? `Notifications, ${unread} unread` : "Notifications";
  return (
    <Link className={styles.bell} to="/notifications" aria-label={label}>
      <BellIcon />
      {unread > 0 && <span className={styles.count}>{unread > 99 ? "99+" : unread}</span>}
    </Link>
  );
}

function BellIcon() {
  return (
    <svg
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M6 9a6 6 0 1 1 12 0c0 7 3 7 3 7H3s3 0 3-7" />
      <path d="M10 19a2 2 0 0 0 4 0" />
    </svg>
  );
}
