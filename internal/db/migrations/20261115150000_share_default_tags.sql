-- +goose Up
-- Default board tags stay on the account until the owner shares them.
-- The public profile then offers those tags to copy.
ALTER TABLE users ADD COLUMN share_default_tags boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN share_default_tags;
