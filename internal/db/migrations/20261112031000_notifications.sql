-- +goose Up
-- Private notices for the uploader of a post. Who favorited a post stays
-- private: only the recipient can read these rows.
CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
    kind text NOT NULL CHECK (kind IN ('favorite', 'comment')),
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    comment_id uuid REFERENCES post_comments(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    read_at timestamptz
);
CREATE INDEX notifications_recipient_created_idx ON notifications (recipient_id, created_at DESC);

-- +goose Down
DROP TABLE notifications;
