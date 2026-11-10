-- +goose Up
-- One open report per member per post. Dismiss closes the row; it does
-- not hide or delete the post. A later report from the same member is a
-- new row once the previous one is no longer open.
CREATE TABLE post_reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    reporter_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason text NOT NULL CHECK (reason IN ('illegal', 'underage', 'spam', 'wrong_rating', 'other')),
    note text NOT NULL DEFAULT '' CHECK (char_length(note) <= 500),
    status text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'dismissed')),
    created_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz,
    resolver_id uuid REFERENCES users(id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX post_reports_one_open ON post_reports (post_id, reporter_id) WHERE status = 'open';
CREATE INDEX post_reports_open_recent ON post_reports (created_at DESC, id DESC) WHERE status = 'open';

-- +goose Down
DROP TABLE post_reports;
