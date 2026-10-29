-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS posts_artist_trgm_idx ON posts USING gin (lower(artist) gin_trgm_ops);
CREATE INDEX IF NOT EXISTS post_tags_tag_trgm_idx ON post_tags USING gin (tag gin_trgm_ops);

-- One predicate for the board, the post pager, and the agent.
-- want_tags is AND, skip_tags drops a post that carries any of them,
-- query matches an artist or a tag (substring or a close spelling),
-- artist_like is a separate artist substring used by the agent.
CREATE OR REPLACE FUNCTION board_match_ids(
    want_tags text[],
    skip_tags text[],
    query text,
    query_like text,
    artist_like text,
    include_nsfw boolean
) RETURNS TABLE (id uuid)
LANGUAGE sql
STABLE
AS $$
    SELECT p.id
    FROM posts p
    WHERE p.hidden_at IS NULL
      AND (include_nsfw OR p.rating = 'sfw')
      AND (
        cardinality(COALESCE(want_tags, '{}'::text[])) = 0
        OR (
            SELECT count(DISTINCT t.tag)
            FROM post_tags t
            WHERE t.post_id = p.id
              AND t.tag = ANY(COALESCE(want_tags, '{}'::text[]))
        ) = cardinality(COALESCE(want_tags, '{}'::text[]))
      )
      AND NOT EXISTS (
        SELECT 1
        FROM post_tags t
        WHERE t.post_id = p.id
          AND t.tag = ANY(COALESCE(skip_tags, '{}'::text[]))
      )
      AND (
        COALESCE(artist_like, '') = ''
        OR lower(p.artist) ILIKE artist_like ESCAPE '\'
      )
      AND (
        COALESCE(query, '') = ''
        OR lower(p.artist) ILIKE query_like ESCAPE '\'
        OR similarity(lower(p.artist), query) > 0.35
        OR EXISTS (
            SELECT 1
            FROM post_tags t
            WHERE t.post_id = p.id
              AND (
                t.tag ILIKE query_like ESCAPE '\'
                OR similarity(t.tag, query) > 0.35
              )
        )
      );
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS board_match_ids(text[], text[], text, text, text, boolean);
DROP INDEX IF EXISTS post_tags_tag_trgm_idx;
DROP INDEX IF EXISTS posts_artist_trgm_idx;
DROP EXTENSION IF EXISTS pg_trgm;
-- +goose StatementEnd
