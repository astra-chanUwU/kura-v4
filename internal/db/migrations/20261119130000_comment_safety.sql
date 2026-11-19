-- +goose Up
-- Moderator review rows for comments that were blocked before publish
-- or flagged after publish. Dismiss closes the row. Remove closes the
-- row and deletes the comment when one was stored.
CREATE TABLE comment_safety_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    comment_id uuid REFERENCES post_comments(id) ON DELETE SET NULL,
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action text NOT NULL CHECK (action IN ('blocked', 'flagged')),
    categories text[] NOT NULL CHECK (
        cardinality(categories) >= 1
        AND categories <@ ARRAY['phone', 'telegram', 'whatsapp', 'discord', 'signal', 'snapchat', 'kik', 'email', 'solicitation']::text[]
    ),
    body text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'dismissed', 'removed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz,
    resolver_id uuid REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX comment_safety_events_open_recent ON comment_safety_events (created_at DESC, id DESC) WHERE status = 'open';

-- +goose Down
DROP TABLE comment_safety_events;
