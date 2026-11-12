-- name: CreateNotification :exec
INSERT INTO notifications (recipient_id, actor_id, kind, post_id, comment_id)
VALUES (
  sqlc.arg(recipient_id),
  sqlc.narg(actor_id),
  sqlc.arg(kind),
  sqlc.arg(post_id),
  sqlc.narg(comment_id)
);

-- name: ListNotifications :many
SELECT n.id,
  n.kind,
  n.post_id,
  n.comment_id,
  COALESCE(u.username, '')::text AS actor_username,
  (n.read_at IS NOT NULL)::bool AS is_read,
  n.created_at
FROM notifications n
LEFT JOIN users u ON u.id = n.actor_id
WHERE n.recipient_id = sqlc.arg(recipient_id)
ORDER BY n.created_at DESC, n.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountNotifications :one
SELECT count(*)::bigint AS total
FROM notifications
WHERE recipient_id = sqlc.arg(recipient_id);

-- name: CountUnreadNotifications :one
SELECT count(*)::bigint AS unread
FROM notifications
WHERE recipient_id = sqlc.arg(recipient_id)
  AND read_at IS NULL;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET read_at = now()
WHERE id = sqlc.arg(id)
  AND recipient_id = sqlc.arg(recipient_id)
  AND read_at IS NULL;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET read_at = now()
WHERE recipient_id = sqlc.arg(recipient_id)
  AND read_at IS NULL;

-- name: NotificationPostOwner :one
SELECT COALESCE(owner_id::text, '')::text AS owner_id
FROM posts
WHERE id = sqlc.arg(id);
