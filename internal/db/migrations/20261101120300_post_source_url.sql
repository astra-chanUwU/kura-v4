-- +goose Up
ALTER TABLE posts ADD COLUMN source_url text NOT NULL DEFAULT '' CHECK (char_length(source_url) <= 300);
-- +goose Down
ALTER TABLE posts DROP COLUMN source_url;
