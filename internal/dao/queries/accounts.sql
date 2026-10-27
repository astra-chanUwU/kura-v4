-- name: CreateUserWithID :one
INSERT INTO users (id, username, display_name) VALUES ($1, $2, $3) RETURNING id, username, display_name, bio, location, website, created_at, avatar_key, avatar_type, show_nsfw, default_include_tags, default_exclude_tags, preferred_theme, preferred_font, role;

-- name: CreateUser :one
INSERT INTO users (username, display_name) VALUES ($1, $2) RETURNING id, username, display_name, bio, location, website, created_at, avatar_key, avatar_type, show_nsfw, default_include_tags, default_exclude_tags, preferred_theme, preferred_font, role;

-- name: GetUserByUsername :one
SELECT id, username, display_name, bio, location, website, created_at, avatar_key, avatar_type, show_nsfw, default_include_tags, default_exclude_tags, preferred_theme, preferred_font, role FROM users WHERE username = $1;

-- name: GetUserByID :one
SELECT id, username, display_name, bio, location, website, created_at, avatar_key, avatar_type, show_nsfw, default_include_tags, default_exclude_tags, preferred_theme, preferred_font, role FROM users WHERE id = $1;

-- name: UpdateProfile :one
UPDATE users SET display_name=$2, bio=$3, location=$4, website=$5, updated_at=now()
WHERE id=$1 RETURNING id, username, display_name, bio, location, website, created_at, avatar_key, avatar_type, show_nsfw, default_include_tags, default_exclude_tags, preferred_theme, preferred_font, role;

-- name: UpsertPassword :exec
INSERT INTO password_credentials (user_id, password_hash) VALUES ($1,$2)
ON CONFLICT (user_id) DO UPDATE SET password_hash=EXCLUDED.password_hash, updated_at=now();

-- name: GetPasswordHash :one
SELECT password_hash FROM password_credentials WHERE user_id = $1;

-- name: DeletePassword :exec
DELETE FROM password_credentials WHERE user_id = $1;

-- name: HasPassword :one
SELECT EXISTS(SELECT 1 FROM password_credentials WHERE user_id=$1)::bool AS has;

-- name: InsertWebauthnCredential :one
INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, transports, user_present, user_verified, backup_eligible, backup_state, aaguid, sign_count, display_name)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id;

-- name: ListWebauthnCredentials :many
SELECT id, user_id, credential_id, public_key, attestation_type, transports, user_present, user_verified, backup_eligible, backup_state, aaguid, sign_count, display_name, created_at, last_used_at
FROM webauthn_credentials WHERE user_id=$1 ORDER BY created_at;

-- name: TouchWebauthnCredential :exec
UPDATE webauthn_credentials SET sign_count=$2, last_used_at=now() WHERE id=$1;

-- name: DeleteWebauthnCredential :execrows
DELETE FROM webauthn_credentials WHERE id=$1 AND user_id=$2;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1,$2,$3);

-- name: GetSessionUserID :one
SELECT user_id FROM sessions WHERE token_hash=$1 AND expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash=$1;

-- name: CreateChallenge :one
INSERT INTO webauthn_challenges (purpose, username, user_id, session_data, expires_at) VALUES ($1,$2,$3,$4,$5) RETURNING id;

-- name: TakeChallenge :one
DELETE FROM webauthn_challenges WHERE id=$1 AND expires_at > now() RETURNING purpose, username, user_id, session_data;

-- name: SetRecoveryKey :exec
INSERT INTO recovery_keys (user_id, key_hash) VALUES ($1,$2)
ON CONFLICT (user_id) DO UPDATE SET key_hash=EXCLUDED.key_hash, created_at=now();

-- name: GetRecoveryKey :one
SELECT key_hash, created_at FROM recovery_keys WHERE user_id = $1;

-- name: DeleteSessionsForUser :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteCredentialsForUser :exec
DELETE FROM webauthn_credentials WHERE user_id = $1;

-- name: UpdateAvatar :exec
UPDATE users SET avatar_key=$2, avatar_type=$3, updated_at=now() WHERE id=$1;

-- name: UpdateShowNSFW :exec
UPDATE users SET show_nsfw=$2, nsfw_ack_at = CASE WHEN $2 THEN now() ELSE nsfw_ack_at END, updated_at=now() WHERE id=$1;

-- name: UpdateTagPreferences :exec
UPDATE users SET default_include_tags=$2, default_exclude_tags=$3, updated_at=now() WHERE id=$1;

-- name: UpdateAppearance :exec
UPDATE users SET preferred_theme=$2, preferred_font=$3, updated_at=now() WHERE id=$1;

-- name: ListUsers :many
SELECT id, username, display_name, bio, location, website, created_at, avatar_key, avatar_type, show_nsfw, default_include_tags, default_exclude_tags, preferred_theme, preferred_font, role FROM users ORDER BY created_at, username;

-- name: UpdateUserRole :one
UPDATE users SET role=$2, updated_at=now() WHERE id=$1 RETURNING id, username, display_name, bio, location, website, created_at, avatar_key, avatar_type, show_nsfw, default_include_tags, default_exclude_tags, preferred_theme, preferred_font, role;

-- name: CountUsersByRole :one
SELECT count(*) FROM users WHERE role=$1;
