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
