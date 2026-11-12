export type NotificationKind = "favorite" | "comment";

export interface NotificationItem {
  id: string;
  kind: NotificationKind;
  post_id: string;
  comment_id?: string;
  actor_username: string;
  read: boolean;
  created_at: string;
}

export interface NotificationPage {
  total: number;
  unread: number;
  notifications: NotificationItem[];
}
