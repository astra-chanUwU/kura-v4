-- +goose Up
ALTER TABLE users
    ADD COLUMN posts_per_page integer NOT NULL DEFAULT 40
        CHECK (posts_per_page BETWEEN 12 AND 100);

-- +goose Down
ALTER TABLE users DROP COLUMN posts_per_page;
