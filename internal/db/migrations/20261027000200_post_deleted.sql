-- +goose Up
-- Deleted is a stronger soft-remove than hidden: both keep the post off
-- the board and preserve its bytes, but the review page lists them apart.
ALTER TABLE posts ADD COLUMN deleted_at timestamptz;

-- +goose Down
ALTER TABLE posts DROP COLUMN deleted_at;
