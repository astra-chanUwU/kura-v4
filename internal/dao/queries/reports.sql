-- name: ReportPostAccess :one
SELECT owner_id, rating,
  (hidden_at IS NOT NULL)::bool AS hidden,
  (deleted_at IS NOT NULL)::bool AS deleted
FROM posts
WHERE id = sqlc.arg(id);

-- name: GetOpenPostReport :one
SELECT id
FROM post_reports
WHERE post_id = sqlc.arg(post_id)
  AND reporter_id = sqlc.arg(reporter_id)
  AND status = 'open';

-- name: CreatePostReport :one
INSERT INTO post_reports (post_id, reporter_id, reason, note)
VALUES (sqlc.arg(post_id), sqlc.arg(reporter_id), sqlc.arg(reason), sqlc.arg(note))
RETURNING id;

-- name: GetPostReport :one
SELECT r.id, r.post_id, r.reporter_id, r.reason, r.note, r.status, r.created_at,
  r.resolved_at, r.resolver_id,
  COALESCE(u.username, '')::text AS reporter_username
FROM post_reports r
JOIN users u ON u.id = r.reporter_id
WHERE r.id = sqlc.arg(id);

-- name: ListOpenPostReports :many
SELECT r.id, r.post_id, r.reporter_id, r.reason, r.note, r.status, r.created_at,
  r.resolved_at, r.resolver_id,
  COALESCE(u.username, '')::text AS reporter_username
FROM post_reports r
JOIN users u ON u.id = r.reporter_id
WHERE r.status = 'open'
ORDER BY r.created_at DESC, r.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountOpenPostReports :one
SELECT count(*)::bigint AS total
FROM post_reports
WHERE status = 'open';

-- name: DismissPostReport :one
WITH updated AS (
    UPDATE post_reports
    SET status = 'dismissed',
        resolved_at = now(),
        resolver_id = sqlc.arg(resolver_id)
    WHERE post_reports.id = sqlc.arg(id)
      AND post_reports.status = 'open'
    RETURNING id, post_id, reporter_id, reason, note, status, created_at, resolved_at, resolver_id
)
SELECT updated.id AS id,
  updated.post_id AS post_id,
  updated.reporter_id AS reporter_id,
  updated.reason AS reason,
  updated.note AS note,
  updated.status AS status,
  updated.created_at AS created_at,
  updated.resolved_at AS resolved_at,
  updated.resolver_id AS resolver_id,
  COALESCE(u.username, '')::text AS reporter_username
FROM updated
JOIN users u ON u.id = updated.reporter_id;
