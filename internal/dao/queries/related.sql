-- name: RelatedPosts :many
-- Posts that share at least one tag with the anchor, excluding the anchor.
-- Column list matches GetPost without source_url.
WITH scored AS (
    SELECT t.post_id, count(DISTINCT t.tag)::int AS shared
    FROM post_tags t
    WHERE t.tag IN (SELECT tag FROM post_tags WHERE post_id = sqlc.arg(post_id)::uuid)
      AND t.post_id <> sqlc.arg(post_id)::uuid
    GROUP BY t.post_id
)
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
  p.artist, p.source, p.source_id, p.thumbnail_key, p.width, p.height, p.rating, p.favorite_count,
  (p.hidden_at IS NOT NULL)::bool AS hidden, (p.deleted_at IS NOT NULL)::bool AS deleted,
  COALESCE(u.username,'')::text AS owner_username,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM scored s
JOIN posts p ON p.id = s.post_id
LEFT JOIN users u ON u.id = p.owner_id
WHERE p.hidden_at IS NULL
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw')
  AND NOT EXISTS (
    SELECT 1 FROM post_tags blocked
    WHERE blocked.post_id = p.id
      AND blocked.tag = ANY(sqlc.arg(exclude)::text[])
  )
ORDER BY s.shared DESC, p.created_at DESC, p.id DESC
LIMIT sqlc.arg(row_limit);

-- name: RandomPost :one
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
  p.artist, p.source, p.source_id, p.source_url, p.thumbnail_key, p.width, p.height, p.rating, p.favorite_count,
  (p.hidden_at IS NOT NULL)::bool AS hidden, (p.deleted_at IS NOT NULL)::bool AS deleted,
  COALESCE(u.username,'')::text AS owner_username,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM posts p
LEFT JOIN users u ON u.id = p.owner_id
WHERE p.id IN (
    SELECT id FROM board_match_ids(
        sqlc.arg(tags)::text[],
        sqlc.arg(exclude)::text[],
        sqlc.arg(q)::text,
        sqlc.arg(q_like)::text,
        sqlc.arg(artist_like)::text,
        sqlc.arg(include_nsfw)::bool
    )
)
ORDER BY random()
LIMIT 1;
