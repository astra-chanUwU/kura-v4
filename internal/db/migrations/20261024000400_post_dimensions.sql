-- +goose Up
ALTER TABLE posts
    ADD COLUMN width integer NOT NULL DEFAULT 0,
    ADD COLUMN height integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE posts
    DROP COLUMN width,
    DROP COLUMN height;
