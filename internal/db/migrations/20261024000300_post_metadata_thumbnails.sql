-- +goose Up
ALTER TABLE posts
    ADD COLUMN artist text NOT NULL DEFAULT '',
    ADD COLUMN source text NOT NULL DEFAULT '',
    ADD COLUMN source_id text NOT NULL DEFAULT '',
    ADD COLUMN thumbnail_key text NOT NULL DEFAULT '';

-- Communal tags for posts, maintained from the post page.
CREATE TABLE post_tags (
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag text NOT NULL CHECK (tag ~ '^[a-z0-9][a-z0-9-]{0,39}$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, tag)
);
CREATE INDEX post_tags_tag_idx ON post_tags (tag);

-- +goose Down
DROP TABLE post_tags;
ALTER TABLE posts
    DROP COLUMN artist,
    DROP COLUMN source,
    DROP COLUMN source_id,
    DROP COLUMN thumbnail_key;
