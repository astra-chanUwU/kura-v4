-- name: AddFavorite :exec
INSERT INTO favorites (user_id, post_id) VALUES ($1,$2) ON CONFLICT DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM favorites WHERE user_id=$1 AND post_id=$2;

-- name: ListFavoritePostIDs :many
SELECT post_id FROM favorites WHERE user_id=$1;

-- name: CountFavorites :one
SELECT count(*)::bigint AS total
FROM favorites f JOIN posts p ON p.id = f.post_id
WHERE f.user_id = sqlc.arg(user_id)
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw');

-- name: ListFavorites :many
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at, p.rating,
  COALESCE(u.username,'')::text AS owner_username
FROM favorites f JOIN posts p ON p.id=f.post_id LEFT JOIN users u ON u.id=p.owner_id
WHERE f.user_id = sqlc.arg(user_id)
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw')
ORDER BY f.created_at DESC, p.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CreateList :one
INSERT INTO lists (owner_id, title, description, visibility) VALUES ($1,$2,$3,$4)
RETURNING id, owner_id, title, description, visibility, created_at, updated_at;

-- name: GetList :one
SELECT l.id, l.owner_id, l.title, l.description, l.visibility, l.created_at, l.updated_at,
  COALESCE(u.username,'')::text AS owner_username, COALESCE(u.display_name,'')::text AS owner_display_name,
  (SELECT count(*) FROM list_items i WHERE i.list_id=l.id)::bigint AS item_count
FROM lists l LEFT JOIN users u ON u.id=l.owner_id WHERE l.id=$1;

-- name: ListListsByOwner :many
SELECT l.id, l.owner_id, l.title, l.description, l.visibility, l.created_at, l.updated_at,
  COALESCE(u.username,'')::text AS owner_username, COALESCE(u.display_name,'')::text AS owner_display_name,
  (SELECT count(*) FROM list_items i WHERE i.list_id=l.id)::bigint AS item_count
FROM lists l LEFT JOIN users u ON u.id=l.owner_id WHERE l.owner_id=$1 ORDER BY l.updated_at DESC;

-- name: ListPublicListsByOwner :many
SELECT l.id, l.owner_id, l.title, l.description, l.visibility, l.created_at, l.updated_at,
  COALESCE(u.username,'')::text AS owner_username, COALESCE(u.display_name,'')::text AS owner_display_name,
  (SELECT count(*) FROM list_items i WHERE i.list_id=l.id)::bigint AS item_count
FROM lists l LEFT JOIN users u ON u.id=l.owner_id WHERE l.owner_id=$1 AND l.visibility='public' ORDER BY l.updated_at DESC;

-- name: UpdateList :one
UPDATE lists SET title=$2, description=$3, visibility=$4, updated_at=now() WHERE id=$1 AND owner_id=$5
RETURNING id, owner_id, title, description, visibility, created_at, updated_at;

-- name: DeleteList :execrows
DELETE FROM lists WHERE id=$1 AND owner_id=$2;

-- name: CountListItems :one
SELECT count(*)::bigint AS total
FROM list_items i JOIN posts p ON p.id = i.post_id
WHERE i.list_id = sqlc.arg(list_id)
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw');

-- name: ListItemIDs :many
SELECT post_id FROM list_items WHERE list_id = $1 ORDER BY position, post_id;

-- name: ListItems :many
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at, p.rating,
  COALESCE(u.username,'')::text AS owner_username
FROM list_items i JOIN posts p ON p.id=i.post_id LEFT JOIN users u ON u.id=p.owner_id
WHERE i.list_id = sqlc.arg(list_id)
  AND (sqlc.arg(include_nsfw)::bool OR p.rating = 'sfw')
ORDER BY i.position
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: NextListPosition :one
SELECT COALESCE(max(position)+1,0)::int AS pos FROM list_items WHERE list_id=$1;

-- name: AddListItem :exec
INSERT INTO list_items (list_id, post_id, position) VALUES ($1,$2,$3) ON CONFLICT (list_id, post_id) DO NOTHING;

-- name: RemoveListItem :exec
DELETE FROM list_items WHERE list_id=$1 AND post_id=$2;

-- name: SetListItemPosition :exec
UPDATE list_items SET position=$3 WHERE list_id=$1 AND post_id=$2;

-- name: TouchList :exec
UPDATE lists SET updated_at=now() WHERE id=$1;
