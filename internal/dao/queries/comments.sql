-- name: CreateComment :one
INSERT INTO post_comments (post_id, author_id, body)
VALUES (sqlc.arg(post_id), sqlc.arg(author_id), sqlc.arg(body))
RETURNING id, post_id, author_id, body, created_at, updated_at;

-- name: GetComment :one
SELECT c.id, c.post_id, c.author_id, c.body, c.created_at, c.updated_at,
  COALESCE(u.username, '')::text AS author_username,
  COALESCE(u.avatar_key, '')::text AS author_avatar_key
FROM post_comments c
JOIN users u ON u.id = c.author_id
WHERE c.id = sqlc.arg(id);

-- name: ListCommentsByPost :many
SELECT c.id, c.post_id, c.author_id, c.body, c.created_at, c.updated_at,
  COALESCE(u.username, '')::text AS author_username,
  COALESCE(u.avatar_key, '')::text AS author_avatar_key
FROM post_comments c
JOIN users u ON u.id = c.author_id
WHERE c.post_id = sqlc.arg(post_id)
ORDER BY c.created_at ASC, c.id ASC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountCommentsByPost :one
SELECT count(*)::bigint AS total
FROM post_comments
WHERE post_id = sqlc.arg(post_id);

-- name: UpdateCommentBody :one
WITH updated AS (
    UPDATE post_comments
    SET body = sqlc.arg(body), updated_at = now()
    WHERE post_comments.id = sqlc.arg(id)
    RETURNING id, post_id, author_id, body, created_at, updated_at
)
SELECT updated.id AS id,
  updated.post_id AS post_id,
  updated.author_id AS author_id,
  updated.body AS body,
  updated.created_at AS created_at,
  updated.updated_at AS updated_at,
  COALESCE(u.username, '')::text AS author_username,
  COALESCE(u.avatar_key, '')::text AS author_avatar_key
FROM updated
JOIN users u ON u.id = updated.author_id;

-- name: DeleteComment :execrows
DELETE FROM post_comments WHERE id = sqlc.arg(id);

-- name: CommentPostAccess :one
SELECT owner_id, rating,
  (hidden_at IS NOT NULL)::bool AS hidden,
  (deleted_at IS NOT NULL)::bool AS deleted
FROM posts
WHERE id = sqlc.arg(id);
