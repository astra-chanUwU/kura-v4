-- +goose Up
ALTER TABLE users
    ADD COLUMN tag_suggestion_rows integer NOT NULL DEFAULT 1
        CHECK (tag_suggestion_rows BETWEEN 0 AND 10);

-- +goose Down
ALTER TABLE users DROP COLUMN tag_suggestion_rows;
