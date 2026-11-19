-- name: CreateCommentSafetyEvent :exec
INSERT INTO comment_safety_events (comment_id, post_id, author_id, action, categories, body)
VALUES (
  sqlc.narg(comment_id),
  sqlc.arg(post_id),
  sqlc.arg(author_id),
  sqlc.arg(action),
  sqlc.arg(categories)::text[],
  sqlc.arg(body)
);

-- Staff queue. An empty status, action, category, query, or resolver
-- skips that predicate. status "any" is normalized to empty before this
-- query. The text search is a substring of the body or the author name.
-- name: ListFilteredCommentSafetyEvents :many
SELECT e.id, e.comment_id, e.post_id, e.author_id, e.action, e.categories, e.body,
  e.status, e.created_at, e.resolved_at, e.resolver_id,
  COALESCE(u.username, '')::text AS author_username,
  COALESCE(resolver.username, '')::text AS resolver_username
FROM comment_safety_events e
JOIN users u ON u.id = e.author_id
LEFT JOIN users resolver ON resolver.id = e.resolver_id
WHERE (
    sqlc.arg(status)::text = ''
    OR e.status = sqlc.arg(status)::text
  )
  AND (
    sqlc.arg(action)::text = ''
    OR e.action = sqlc.arg(action)::text
  )
  AND (
    sqlc.arg(category)::text = ''
    OR sqlc.arg(category)::text = ANY (e.categories)
  )
  AND (
    sqlc.arg(query)::text = ''
    OR e.body ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR u.username ILIKE sqlc.arg(query)::text ESCAPE '\'
  )
  AND (
    sqlc.arg(resolver_id)::text = ''
    OR e.resolver_id = NULLIF(sqlc.arg(resolver_id)::text, '')::uuid
    OR EXISTS (
      SELECT 1 FROM comment_safety_decisions d
      WHERE d.event_id = e.id
        AND d.actor_id = NULLIF(sqlc.arg(resolver_id)::text, '')::uuid
    )
  )
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN e.created_at END ASC,
  CASE WHEN sqlc.arg(sort)::text = 'decided' THEN e.resolved_at END DESC NULLS LAST,
  CASE WHEN sqlc.arg(sort)::text <> 'oldest' AND sqlc.arg(sort)::text <> 'decided' THEN e.created_at END DESC,
  CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN e.id END ASC,
  CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN e.id END DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- One row per moderator who has acted on a review, with that person's
-- latest step. The DAO sorts these by last_at so the newest step is first.
-- name: ListCommentSafetyResolvers :many
SELECT DISTINCT ON (d.actor_id)
  d.actor_id AS resolver_id,
  d.actor_username AS username,
  d.action AS last_status,
  e.action AS last_action,
  e.categories AS last_categories,
  left(e.body, 80)::text AS last_body,
  d.created_at AS last_at,
  counts.decisions
FROM comment_safety_decisions d
JOIN comment_safety_events e ON e.id = d.event_id
JOIN (
  SELECT actor_id, count(*)::bigint AS decisions
  FROM comment_safety_decisions
  WHERE actor_id IS NOT NULL
  GROUP BY actor_id
) counts ON counts.actor_id = d.actor_id
WHERE d.actor_id IS NOT NULL
ORDER BY d.actor_id, d.created_at DESC, d.id DESC;

-- name: ListCommentSafetyDecisions :many
SELECT event_id, actor_id, actor_username, action, created_at
FROM comment_safety_decisions
WHERE event_id = ANY(sqlc.arg(event_ids)::uuid[])
ORDER BY created_at ASC, id ASC;

-- name: CountFilteredCommentSafetyEvents :one
SELECT count(*)::bigint AS total
FROM comment_safety_events e
JOIN users u ON u.id = e.author_id
WHERE (
    sqlc.arg(status)::text = ''
    OR e.status = sqlc.arg(status)::text
  )
  AND (
    sqlc.arg(action)::text = ''
    OR e.action = sqlc.arg(action)::text
  )
  AND (
    sqlc.arg(category)::text = ''
    OR sqlc.arg(category)::text = ANY (e.categories)
  )
  AND (
    sqlc.arg(query)::text = ''
    OR e.body ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR u.username ILIKE sqlc.arg(query)::text ESCAPE '\'
  )
  AND (
    sqlc.arg(resolver_id)::text = ''
    OR e.resolver_id = NULLIF(sqlc.arg(resolver_id)::text, '')::uuid
    OR EXISTS (
      SELECT 1 FROM comment_safety_decisions d
      WHERE d.event_id = e.id
        AND d.actor_id = NULLIF(sqlc.arg(resolver_id)::text, '')::uuid
    )
  );

-- name: DismissCommentSafetyEvent :one
WITH updated AS (
    UPDATE comment_safety_events
    SET status = 'dismissed',
        resolved_at = now(),
        resolver_id = sqlc.arg(resolver_id)
    WHERE comment_safety_events.id = sqlc.arg(id)
      AND comment_safety_events.status = 'open'
    RETURNING id, comment_id, post_id, author_id, action, categories, body, status, created_at, resolved_at, resolver_id
), recorded AS (
    INSERT INTO comment_safety_decisions (event_id, actor_id, actor_username, action)
    SELECT updated.id, updated.resolver_id,
      (SELECT username FROM users WHERE id = updated.resolver_id),
      'dismissed'
    FROM updated
    RETURNING event_id
)
SELECT updated.id AS id,
  updated.comment_id AS comment_id,
  updated.post_id AS post_id,
  updated.author_id AS author_id,
  updated.action AS action,
  updated.categories AS categories,
  updated.body AS body,
  updated.status AS status,
  updated.created_at AS created_at,
  updated.resolved_at AS resolved_at,
  updated.resolver_id AS resolver_id,
  COALESCE(u.username, '')::text AS author_username
FROM updated
JOIN users u ON u.id = updated.author_id;

-- name: RemoveCommentSafetyEvent :one
WITH updated AS (
    UPDATE comment_safety_events
    SET status = 'removed',
        resolved_at = now(),
        resolver_id = sqlc.arg(resolver_id)
    WHERE comment_safety_events.id = sqlc.arg(id)
      AND comment_safety_events.status IN ('open', 'dismissed')
    RETURNING id, comment_id, post_id, author_id, action, categories, body, status, created_at, resolved_at, resolver_id
), recorded AS (
    INSERT INTO comment_safety_decisions (event_id, actor_id, actor_username, action)
    SELECT updated.id, updated.resolver_id,
      (SELECT username FROM users WHERE id = updated.resolver_id),
      'removed'
    FROM updated
    RETURNING event_id
)
SELECT updated.id AS id,
  updated.comment_id AS comment_id,
  updated.post_id AS post_id,
  updated.author_id AS author_id,
  updated.action AS action,
  updated.categories AS categories,
  updated.body AS body,
  updated.status AS status,
  updated.created_at AS created_at,
  updated.resolved_at AS resolved_at,
  updated.resolver_id AS resolver_id,
  COALESCE(u.username, '')::text AS author_username
FROM updated
JOIN users u ON u.id = updated.author_id;

-- name: ReopenCommentSafetyEvent :one
WITH updated AS (
    UPDATE comment_safety_events
    SET status = 'open',
        resolved_at = NULL,
        resolver_id = NULL
    WHERE comment_safety_events.id = sqlc.arg(id)
      AND comment_safety_events.status IN ('dismissed', 'removed')
    RETURNING id, comment_id, post_id, author_id, action, categories, body, status, created_at, resolved_at, resolver_id
), recorded AS (
    INSERT INTO comment_safety_decisions (event_id, actor_id, actor_username, action)
    SELECT updated.id, sqlc.arg(actor_id),
      (SELECT username FROM users WHERE id = sqlc.arg(actor_id)),
      CASE
        WHEN updated.action = 'flagged' AND updated.comment_id IS NULL THEN 'restored'
        ELSE 'reopened'
      END
    FROM updated
    RETURNING event_id
)
SELECT updated.id AS id,
  updated.comment_id AS comment_id,
  updated.post_id AS post_id,
  updated.author_id AS author_id,
  updated.action AS action,
  updated.categories AS categories,
  updated.body AS body,
  updated.status AS status,
  updated.created_at AS created_at,
  updated.resolved_at AS resolved_at,
  updated.resolver_id AS resolver_id,
  COALESCE(u.username, '')::text AS author_username
FROM updated
JOIN users u ON u.id = updated.author_id;

-- name: RestoreSafetyComment :one
INSERT INTO post_comments (post_id, author_id, body)
VALUES (sqlc.arg(post_id), sqlc.arg(author_id), sqlc.arg(body))
RETURNING id;

-- name: AttachSafetyComment :exec
UPDATE comment_safety_events
SET comment_id = sqlc.arg(comment_id)
WHERE id = sqlc.arg(id);

-- name: DeleteSafetyComment :execrows
DELETE FROM post_comments WHERE id = sqlc.arg(id);
