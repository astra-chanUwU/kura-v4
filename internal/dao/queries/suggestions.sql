-- name: CreateTagSuggestion :one
INSERT INTO tag_suggestions (kind, source_tag, target_tag, reason, author_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, kind, source_tag, target_tag, reason, status, created_at,
  ''::text AS author_username;

-- name: GetTagSuggestion :one
SELECT s.id, s.kind, s.source_tag, s.target_tag, s.reason, s.status, s.created_at,
  COALESCE(u.username, '')::text AS author_username
FROM tag_suggestions s
LEFT JOIN users u ON u.id = s.author_id
WHERE s.id = $1;

-- name: ListTagSuggestions :many
SELECT s.id, s.kind, s.source_tag, s.target_tag, s.reason, s.status, s.created_at,
  COALESCE(u.username, '')::text AS author_username
FROM tag_suggestions s
LEFT JOIN users u ON u.id = s.author_id
WHERE s.status = sqlc.arg(status)
ORDER BY s.created_at DESC, s.id DESC;

-- name: ResolveTagSuggestion :one
UPDATE tag_suggestions AS s
SET status = sqlc.arg(status),
    resolved_by = sqlc.arg(resolved_by),
    resolved_at = now()
WHERE s.id = sqlc.arg(id)
  AND s.status = 'pending'
RETURNING s.id, s.kind, s.source_tag, s.target_tag, s.reason, s.status, s.created_at,
  COALESCE((SELECT username FROM users WHERE id = s.author_id), '')::text AS author_username;
