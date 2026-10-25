-- +goose Up
-- Posts are identified by their tags and artist, not a title.
ALTER TABLE posts DROP COLUMN title;

-- +goose Down
ALTER TABLE posts ADD COLUMN title text NOT NULL DEFAULT '' CHECK (char_length(title) <= 200);
