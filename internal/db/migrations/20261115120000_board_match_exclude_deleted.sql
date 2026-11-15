-- +goose Up
-- Deleted posts stay out of board search even when hidden_at was never set.
-- The match count, neighbors, random pick, and tag counts all use this function.
-- +goose StatementBegin
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
      AND p.deleted_at IS NULL
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
