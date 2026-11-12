-- name: CreateTagSuggestion :one
INSERT INTO tag_suggestions (kind, source_tag, target_tag, reason, author_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, kind, source_tag, target_tag, reason, status, created_at, resolved_at,
  ''::text AS author_username,
  ''::text AS resolver_username;

-- name: GetTagSuggestion :one
SELECT s.id, s.kind, s.source_tag, s.target_tag, s.reason, s.status, s.created_at,
  s.resolved_at,
  COALESCE(u.username, '')::text AS author_username,
  COALESCE(r.username, '')::text AS resolver_username
FROM tag_suggestions s
LEFT JOIN users u ON u.id = s.author_id
LEFT JOIN users r ON r.id = s.resolved_by
WHERE s.id = $1;

-- name: ListTagSuggestions :many
SELECT s.id, s.kind, s.source_tag, s.target_tag, s.reason, s.status, s.created_at,
  s.resolved_at,
  COALESCE(u.username, '')::text AS author_username,
  COALESCE(r.username, '')::text AS resolver_username
FROM tag_suggestions s
LEFT JOIN users u ON u.id = s.author_id
LEFT JOIN users r ON r.id = s.resolved_by
WHERE s.status = sqlc.arg(status)
ORDER BY COALESCE(s.resolved_at, s.created_at) DESC, s.id DESC;

-- name: ResolveTagSuggestion :one
-- from_status keeps a second moderator from overwriting a decision that
-- landed between the read and this update.
UPDATE tag_suggestions AS s
SET status = sqlc.arg(status),
    resolved_by = sqlc.arg(resolved_by),
    resolved_at = now()
WHERE s.id = sqlc.arg(id)
  AND s.status = sqlc.arg(from_status)
RETURNING s.id, s.kind, s.source_tag, s.target_tag, s.reason, s.status, s.created_at,
  s.resolved_at,
  COALESCE((SELECT username FROM users WHERE id = s.author_id), '')::text AS author_username,
  COALESCE((SELECT username FROM users WHERE id = s.resolved_by), '')::text AS resolver_username;
