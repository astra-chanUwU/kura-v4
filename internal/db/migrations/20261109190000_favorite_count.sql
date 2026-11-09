-- +goose Up
ALTER TABLE posts ADD COLUMN favorite_count integer NOT NULL DEFAULT 0;

-- Existing favorites are counted once, before the trigger starts maintaining the column.
UPDATE posts SET favorite_count = counts.n
FROM (
    SELECT post_id, count(*)::int AS n
    FROM favorites
    GROUP BY post_id
) counts
WHERE posts.id = counts.post_id;

-- +goose StatementBegin
CREATE FUNCTION favorites_count_apply() RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE posts
        SET favorite_count = favorite_count + 1
        WHERE id = NEW.post_id;
        RETURN NEW;
    END IF;
    UPDATE posts
    SET favorite_count = GREATEST(favorite_count - 1, 0)
    WHERE id = OLD.post_id;
    RETURN OLD;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER favorites_count_trg
AFTER INSERT OR DELETE ON favorites
FOR EACH ROW
EXECUTE FUNCTION favorites_count_apply();

CREATE INDEX posts_favorite_count_idx ON posts (favorite_count DESC, created_at DESC, id DESC);

ALTER TABLE saved_searches DROP CONSTRAINT saved_searches_sort_check;
ALTER TABLE saved_searches ADD CONSTRAINT saved_searches_sort_check CHECK (sort IN ('newest', 'oldest', 'favorites'));

-- +goose Down
DROP TRIGGER IF EXISTS favorites_count_trg ON favorites;
DROP FUNCTION IF EXISTS favorites_count_apply();
DROP INDEX IF EXISTS posts_favorite_count_idx;
ALTER TABLE posts DROP COLUMN favorite_count;
UPDATE saved_searches SET sort = 'newest' WHERE sort = 'favorites';
ALTER TABLE saved_searches DROP CONSTRAINT saved_searches_sort_check;
ALTER TABLE saved_searches ADD CONSTRAINT saved_searches_sort_check CHECK (sort IN ('newest', 'oldest'));
