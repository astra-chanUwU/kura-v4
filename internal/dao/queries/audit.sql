-- name: InsertChangeHistory :one
INSERT INTO change_history (entity_type, entity_id, actor_id, actor_username, action, before_state, after_state, rollback_of)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, entity_type, entity_id, actor_id, actor_username, action, before_state, after_state, rollback_of, created_at;

-- name: GetChangeHistory :one
SELECT id, entity_type, entity_id, actor_id, actor_username, action, before_state, after_state, rollback_of, created_at FROM change_history WHERE id=$1;

-- name: ListChangeHistoryForEntity :many
SELECT id, entity_type, entity_id, actor_id, actor_username, action, before_state, after_state, rollback_of, created_at FROM change_history WHERE entity_type=$1 AND entity_id=$2 ORDER BY created_at DESC, id DESC LIMIT $3;

-- name: ListRecentChangeHistory :many
SELECT id, entity_type, entity_id, actor_id, actor_username, action, before_state, after_state, rollback_of, created_at FROM change_history ORDER BY created_at DESC, id DESC LIMIT $1;

-- name: ListChangeHistoryByActor :many
SELECT id, entity_type, entity_id, actor_id, actor_username, action, before_state, after_state, rollback_of, created_at FROM change_history WHERE actor_id=$1 ORDER BY created_at DESC, id DESC LIMIT $2;

-- Staff history page. The text search matches the actor and the snapshot
-- fields a person would type. It does not search the whole jsonb document:
-- keys such as hidden and deleted are on every post snapshot.
-- name: ListFilteredChangeHistory :many
SELECT id, entity_type, entity_id, actor_id, actor_username, action, before_state, after_state, rollback_of, created_at
FROM change_history
WHERE (
    sqlc.arg(actor_id)::text = ''
    OR actor_id = NULLIF(sqlc.arg(actor_id)::text, '')::uuid
  )
  AND (
    sqlc.arg(action)::text = ''
    OR action = sqlc.arg(action)::text
  )
  AND (
    sqlc.arg(entity)::text = ''
    OR entity_type = sqlc.arg(entity)::text
  )
  AND (
    sqlc.arg(query)::text = ''
    OR actor_username ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'artist', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'artist', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'role', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'role', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'rating', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'rating', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'source_url', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'source_url', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce((before_state->'tags')::text, '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce((after_state->'tags')::text, '') ILIKE sqlc.arg(query)::text ESCAPE '\'
  )
ORDER BY
  CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN created_at END ASC,
  CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN created_at END DESC,
  CASE WHEN sqlc.arg(sort)::text = 'oldest' THEN id END ASC,
  CASE WHEN sqlc.arg(sort)::text <> 'oldest' THEN id END DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountFilteredChangeHistory :one
SELECT count(*)::bigint AS total
FROM change_history
WHERE (
    sqlc.arg(actor_id)::text = ''
    OR actor_id = NULLIF(sqlc.arg(actor_id)::text, '')::uuid
  )
  AND (
    sqlc.arg(action)::text = ''
    OR action = sqlc.arg(action)::text
  )
  AND (
    sqlc.arg(entity)::text = ''
    OR entity_type = sqlc.arg(entity)::text
  )
  AND (
    sqlc.arg(query)::text = ''
    OR actor_username ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'artist', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'artist', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'role', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'role', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'rating', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'rating', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(before_state->>'source_url', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce(after_state->>'source_url', '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce((before_state->'tags')::text, '') ILIKE sqlc.arg(query)::text ESCAPE '\'
    OR coalesce((after_state->'tags')::text, '') ILIKE sqlc.arg(query)::text ESCAPE '\'
  );

-- name: CountMetaEditsByActor :one
SELECT count(*)::bigint AS total
FROM change_history
WHERE actor_id = sqlc.arg(actor_id)
  AND action = 'post.meta_updated';

-- name: LastMetaEditor :one
SELECT actor_username
FROM change_history
WHERE entity_type = 'post'
  AND entity_id = sqlc.arg(entity_id)
  AND action = 'post.meta_updated'
ORDER BY created_at DESC, id DESC
LIMIT 1;
