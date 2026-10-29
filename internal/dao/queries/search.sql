-- name: CountBoardMatches :one
SELECT count(*)::bigint AS total
FROM board_match_ids(
    sqlc.arg(tags)::text[],
    sqlc.arg(exclude)::text[],
    sqlc.arg(q)::text,
    sqlc.arg(q_like)::text,
    sqlc.arg(artist_like)::text,
    sqlc.arg(include_nsfw)::bool
);

-- name: SearchPosts :many
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
    p.artist, p.source, p.source_id, p.thumbnail_key, p.width, p.height, p.rating,
    (p.hidden_at IS NOT NULL)::bool AS hidden, (p.deleted_at IS NOT NULL)::bool AS deleted,
    COALESCE(u.username, '')::text AS owner_username,
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
ORDER BY
    CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN p.created_at END ASC NULLS LAST,
    CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN p.created_at END DESC NULLS LAST,
    p.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: SuggestTags :many
SELECT t.tag, count(*)::bigint AS count
FROM post_tags t
JOIN posts p ON p.id = t.post_id
WHERE p.hidden_at IS NULL
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw')
  AND (
    sqlc.arg(q)::text = ''
    OR t.tag ILIKE sqlc.arg(q) || '%'
    OR (char_length(sqlc.arg(q)::text) >= 3 AND similarity(t.tag, sqlc.arg(q)::text) > 0.35)
  )
GROUP BY t.tag
ORDER BY
    CASE WHEN sqlc.arg(q)::text <> '' AND t.tag ILIKE sqlc.arg(q) || '%' THEN 0 ELSE 1 END,
    count(*) DESC,
    t.tag ASC
LIMIT sqlc.arg(row_limit);

-- name: PostNeighbors :one
WITH filtered AS (
    SELECT p.id,
        row_number() OVER (
            ORDER BY
                CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN p.created_at END ASC NULLS LAST,
                CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN p.created_at END DESC NULLS LAST,
                p.id DESC
        ) AS rn
    FROM board_match_ids(
        sqlc.arg(tags)::text[],
        sqlc.arg(exclude)::text[],
        sqlc.arg(q)::text,
        sqlc.arg(q_like)::text,
        sqlc.arg(artist_like)::text,
        sqlc.arg(include_nsfw)::bool
    ) m
    JOIN posts p ON p.id = m.id
)
SELECT
    COALESCE((SELECT rn - 1 FROM filtered WHERE id = sqlc.arg(post_id)::uuid), -1)::int AS idx,
    (SELECT count(*) FROM filtered)::bigint AS total,
    COALESCE((
        SELECT f2.id::text
        FROM filtered cur
        JOIN filtered f2 ON f2.rn = cur.rn - 1
        WHERE cur.id = sqlc.arg(post_id)::uuid
    ), '') AS prev_id,
    COALESCE((
        SELECT f2.id::text
        FROM filtered cur
        JOIN filtered f2 ON f2.rn = cur.rn + 1
        WHERE cur.id = sqlc.arg(post_id)::uuid
    ), '') AS next_id;
