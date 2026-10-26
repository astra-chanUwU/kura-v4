-- +goose Up
ALTER TABLE users
    ADD COLUMN default_include_tags text[] NOT NULL DEFAULT '{}',
    ADD COLUMN default_exclude_tags text[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE users DROP COLUMN default_include_tags, DROP COLUMN default_exclude_tags;
