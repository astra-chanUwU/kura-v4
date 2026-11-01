-- name: GetTagPage :one
SELECT t.tag, t.category, t.description, t.updated_at,
  COALESCE(u.username, '')::text AS updated_by_username
FROM tag_pages t
LEFT JOIN users u ON u.id = t.updated_by
WHERE t.tag = sqlc.arg(tag);

-- name: UpsertTagPage :exec
INSERT INTO tag_pages (tag, category, description, updated_by)
VALUES (sqlc.arg(tag), sqlc.arg(category), sqlc.arg(description), sqlc.arg(updated_by))
ON CONFLICT (tag) DO UPDATE
SET category = EXCLUDED.category,
    description = EXCLUDED.description,
    updated_by = EXCLUDED.updated_by,
    updated_at = now();

-- name: CountVisiblePostsForTag :one
SELECT count(*)::bigint AS total
FROM post_tags t
JOIN posts p ON p.id = t.post_id
WHERE t.tag = sqlc.arg(tag)
  AND p.hidden_at IS NULL
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw');

-- name: ListRecentPostsForTag :many
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
  p.artist, p.source, p.source_id, p.source_url, p.thumbnail_key, p.width, p.height, p.rating,
  (p.hidden_at IS NOT NULL)::bool AS hidden, (p.deleted_at IS NOT NULL)::bool AS deleted,
  COALESCE(u.username, '')::text AS owner_username,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM posts p
JOIN post_tags mine ON mine.post_id = p.id AND mine.tag = sqlc.arg(tag)
LEFT JOIN users u ON u.id = p.owner_id
WHERE p.hidden_at IS NULL
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw')
ORDER BY p.created_at DESC, p.id DESC
LIMIT 24;

-- name: GetTagAlias :one
SELECT target FROM tag_aliases WHERE alias = sqlc.arg(alias);

-- name: ListAliasesOfTarget :many
SELECT alias FROM tag_aliases WHERE target = sqlc.arg(target) ORDER BY alias;

-- name: TagIsAliasTarget :one
SELECT EXISTS(SELECT 1 FROM tag_aliases WHERE target = sqlc.arg(tag))::bool AS is_target;

-- name: ListTagAliasTargets :many
SELECT alias, target
FROM tag_aliases
WHERE alias = ANY(sqlc.arg(tags)::text[]);

-- name: DeleteConflictingAliasTags :exec
DELETE FROM post_tags AS old
USING post_tags AS kept
WHERE old.tag = sqlc.arg(alias)
  AND kept.post_id = old.post_id
  AND kept.tag = sqlc.arg(target);

-- name: RenamePostTag :exec
UPDATE post_tags SET tag = sqlc.arg(target) WHERE tag = sqlc.arg(alias);

-- name: InsertTagAlias :exec
INSERT INTO tag_aliases (alias, target, created_by)
VALUES (sqlc.arg(alias), sqlc.arg(target), sqlc.arg(created_by));

-- name: DeleteTagAlias :execrows
DELETE FROM tag_aliases
WHERE alias = sqlc.arg(alias) AND target = sqlc.arg(target);
