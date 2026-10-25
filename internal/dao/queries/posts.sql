-- name: CreatePost :one
INSERT INTO posts (owner_id, original_name, storage_key, content_type, size_bytes, sha256, artist, source, source_id, thumbnail_key, width, height, rating)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id, owner_id, original_name, storage_key, content_type, size_bytes, created_at, artist, source, source_id, thumbnail_key, width, height, rating;

-- name: GetPost :one
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
  p.artist, p.source, p.source_id, p.thumbnail_key, p.width, p.height, p.rating,
  COALESCE(u.username,'')::text AS owner_username,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM posts p LEFT JOIN users u ON u.id=p.owner_id WHERE p.id=$1;

-- name: ListPosts :many
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
  p.artist, p.source, p.source_id, p.thumbnail_key, p.width, p.height, p.rating,
  COALESCE(u.username,'')::text AS owner_username,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM posts p LEFT JOIN users u ON u.id=p.owner_id WHERE ($3::bool OR p.rating='sfw') ORDER BY p.created_at DESC, p.id DESC LIMIT $1 OFFSET $2;

-- name: GetPostBySHA :one
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
  p.artist, p.source, p.source_id, p.thumbnail_key, p.width, p.height, p.rating,
  COALESCE(u.username,'')::text AS owner_username,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM posts p LEFT JOIN users u ON u.id=p.owner_id WHERE p.sha256=$1 AND $1<>'';

-- name: PostExistsBySHA :one
SELECT EXISTS(SELECT 1 FROM posts WHERE sha256=$1 AND $1<>'')::bool AS has;

-- name: SetPostDimensions :exec
UPDATE posts SET width=$2, height=$3 WHERE id=$1;

-- name: SetPostThumbnail :exec
UPDATE posts SET thumbnail_key=$2 WHERE id=$1;

-- name: UpdatePostArtist :exec
UPDATE posts SET artist=$2, rating=$3 WHERE id=$1;

-- name: DeletePostTags :exec
DELETE FROM post_tags WHERE post_id=$1;

-- name: InsertPostTag :exec
INSERT INTO post_tags (post_id, tag) VALUES ($1,$2) ON CONFLICT DO NOTHING;

-- name: ListPostIndex :many
SELECT p.id, p.artist, p.source, p.source_id, p.thumbnail_key, p.content_type, p.width, p.height, p.rating, p.created_at,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM posts p WHERE ($1::bool OR p.rating='sfw') ORDER BY p.created_at DESC, p.id DESC;

-- name: CountPostsByOwner :one
SELECT count(*) FROM posts WHERE owner_id=$1;

-- name: ListPostsByOwner :many
SELECT p.id, p.owner_id, p.original_name, p.storage_key, p.content_type, p.size_bytes, p.created_at,
  p.artist, p.source, p.source_id, p.thumbnail_key, p.width, p.height, p.rating,
  COALESCE(u.username,'')::text AS owner_username,
  COALESCE((SELECT string_agg(t.tag, ',' ORDER BY t.tag) FROM post_tags t WHERE t.post_id = p.id), '')::text AS tags_csv
FROM posts p LEFT JOIN users u ON u.id=p.owner_id WHERE p.owner_id=$1 ORDER BY p.created_at DESC, p.id DESC;
