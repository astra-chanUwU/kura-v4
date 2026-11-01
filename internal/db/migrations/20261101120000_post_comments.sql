-- +goose Up
CREATE TABLE post_comments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX post_comments_post_idx ON post_comments (post_id, created_at ASC, id ASC);

-- +goose Down
DROP TABLE post_comments;
