-- +goose Up
ALTER TABLE users
    ADD COLUMN avatar_key text NOT NULL DEFAULT '',
    ADD COLUMN avatar_type text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users
    DROP COLUMN avatar_key,
    DROP COLUMN avatar_type;
