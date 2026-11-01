-- +goose Up
CREATE TABLE saved_searches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
    include_tags text[] NOT NULL DEFAULT '{}',
    exclude_tags text[] NOT NULL DEFAULT '{}',
    query text NOT NULL DEFAULT '' CHECK (char_length(query) <= 80),
    sort text NOT NULL DEFAULT 'newest' CHECK (sort IN ('newest', 'oldest')),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);
CREATE INDEX saved_searches_user_idx ON saved_searches (user_id, created_at DESC);

-- +goose Down
DROP TABLE saved_searches;
