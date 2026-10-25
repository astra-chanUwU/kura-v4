-- +goose Up
ALTER TABLE post_tags DROP CONSTRAINT IF EXISTS post_tags_tag_check;
ALTER TABLE post_tags ADD CONSTRAINT post_tags_tag_check CHECK (tag ~ '^[a-z0-9][a-z0-9_-]{0,39}$');

-- Canonicalize existing hyphenated tags to booru underscore form.
INSERT INTO post_tags (post_id, tag)
SELECT post_id, replace(tag, '-', '_') FROM post_tags WHERE tag LIKE '%-%'
ON CONFLICT DO NOTHING;
DELETE FROM post_tags WHERE tag LIKE '%-%';

-- +goose Down
ALTER TABLE post_tags DROP CONSTRAINT IF EXISTS post_tags_tag_check;
ALTER TABLE post_tags ADD CONSTRAINT post_tags_tag_check CHECK (tag ~ '^[a-z0-9][a-z0-9-]{0,39}$');
