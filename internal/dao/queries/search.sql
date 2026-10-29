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
  AND (
    sqlc.narg(after_id)::uuid IS NULL
    OR (
      CASE
        WHEN sqlc.arg(sort)::text = 'oldest' THEN
          p.created_at > sqlc.narg(after_at)::timestamptz
          OR (p.created_at = sqlc.narg(after_at)::timestamptz AND p.id < sqlc.narg(after_id)::uuid)
        ELSE
          p.created_at < sqlc.narg(after_at)::timestamptz
          OR (p.created_at = sqlc.narg(after_at)::timestamptz AND p.id < sqlc.narg(after_id)::uuid)
      END
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
WHERE p.id IN (
    SELECT id FROM board_match_ids(
        sqlc.arg(tags)::text[],
        sqlc.arg(exclude)::text[],
        sqlc.arg(match)::text,
        sqlc.arg(match_like)::text,
        ''::text,
        sqlc.arg(include_nsfw)::bool
    )
)
  AND (
    sqlc.arg(q)::text = ''
    OR strpos(t.tag, sqlc.arg(q)::text) = 1
    OR (char_length(sqlc.arg(q)::text) >= 3 AND similarity(t.tag, sqlc.arg(q)::text) > 0.35)
  )
GROUP BY t.tag
ORDER BY
    CASE WHEN sqlc.arg(q)::text <> '' AND strpos(t.tag, sqlc.arg(q)::text) = 1 THEN 0 ELSE 1 END,
    count(*) DESC,
    t.tag ASC
LIMIT sqlc.arg(row_limit);

-- name: PostNeighbors :one
-- One pass over the matches. Position is a count of rows that sort
-- earlier, and the neighbors are the single closest rows, so the pager
-- does not number every match.
WITH matched AS MATERIALIZED (
    SELECT p.id, p.created_at
    FROM board_match_ids(
        sqlc.arg(tags)::text[],
        sqlc.arg(exclude)::text[],
        sqlc.arg(q)::text,
        sqlc.arg(q_like)::text,
        sqlc.arg(artist_like)::text,
        sqlc.arg(include_nsfw)::bool
    ) m
    JOIN posts p ON p.id = m.id
),
cur AS (
    SELECT id, created_at FROM matched WHERE id = sqlc.arg(post_id)::uuid
)
SELECT
    CASE
        WHEN NOT EXISTS (SELECT 1 FROM cur) THEN -1
        ELSE (
            SELECT count(*)::int
            FROM matched p
            CROSS JOIN cur c
            WHERE CASE
                WHEN sqlc.arg(sort)::text = 'oldest' THEN
                    p.created_at < c.created_at OR (p.created_at = c.created_at AND p.id > c.id)
                ELSE
                    p.created_at > c.created_at OR (p.created_at = c.created_at AND p.id > c.id)
            END
        )
    END::int AS idx,
    (SELECT count(*) FROM matched)::bigint AS total,
    COALESCE((
        SELECT p.id::text
        FROM matched p
        CROSS JOIN cur c
        WHERE CASE
            WHEN sqlc.arg(sort)::text = 'oldest' THEN
                p.created_at < c.created_at OR (p.created_at = c.created_at AND p.id > c.id)
            ELSE
                p.created_at > c.created_at OR (p.created_at = c.created_at AND p.id > c.id)
        END
        ORDER BY
            CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN p.created_at END DESC NULLS LAST,
            CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN p.created_at END ASC NULLS LAST,
            p.id ASC
        LIMIT 1
    ), '') AS prev_id,
    COALESCE((
        SELECT p.id::text
        FROM matched p
        CROSS JOIN cur c
        WHERE CASE
            WHEN sqlc.arg(sort)::text = 'oldest' THEN
                p.created_at > c.created_at OR (p.created_at = c.created_at AND p.id < c.id)
            ELSE
                p.created_at < c.created_at OR (p.created_at = c.created_at AND p.id < c.id)
        END
        ORDER BY
            CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN p.created_at END ASC NULLS LAST,
            CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN p.created_at END DESC NULLS LAST,
            p.id DESC
        LIMIT 1
    ), '') AS next_id;
