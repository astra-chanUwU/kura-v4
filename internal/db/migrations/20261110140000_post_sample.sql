-- +goose Up
-- Medium JPEG for the post page. NULL until post.derive stores one.
ALTER TABLE posts
    ADD COLUMN sample_key text;

-- +goose Down
ALTER TABLE posts
    DROP COLUMN sample_key;
