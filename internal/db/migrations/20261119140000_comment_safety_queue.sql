-- +goose Up
-- The open-only index still serves the default queue. These cover
-- status filters and "any status" sorts once the log is large.
CREATE INDEX comment_safety_events_status_recent
    ON comment_safety_events (status, created_at DESC, id DESC);
CREATE INDEX comment_safety_events_recent
    ON comment_safety_events (created_at DESC, id DESC);

-- +goose Down
DROP INDEX comment_safety_events_recent;
DROP INDEX comment_safety_events_status_recent;
