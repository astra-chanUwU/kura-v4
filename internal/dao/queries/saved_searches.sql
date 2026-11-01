-- name: ListSavedSearches :many
SELECT id, user_id, name, include_tags, exclude_tags, query, sort, created_at
FROM saved_searches
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: CountSavedSearches :one
SELECT count(*)::bigint AS total
FROM saved_searches
WHERE user_id = $1;

-- name: CreateSavedSearch :one
INSERT INTO saved_searches (user_id, name, include_tags, exclude_tags, query, sort)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, name, include_tags, exclude_tags, query, sort, created_at;

-- name: DeleteSavedSearch :execrows
DELETE FROM saved_searches WHERE id = $1 AND user_id = $2;
