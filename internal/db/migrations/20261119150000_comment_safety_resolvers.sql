-- +goose Up
-- One row per moderator for "who decided" and "what they did last".
CREATE INDEX comment_safety_events_resolver_recent
    ON comment_safety_events (resolver_id, resolved_at DESC, id DESC)
    WHERE resolver_id IS NOT NULL AND status IN ('dismissed', 'removed');

-- +goose Down
DROP INDEX comment_safety_events_resolver_recent;
