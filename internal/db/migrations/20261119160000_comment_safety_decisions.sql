-- +goose Up
-- Every dismiss, delete, restore, and reopen stays on the review.
-- The event row still stores who holds the current decision. This
-- table is the chain that got there, so a later delete does not
-- erase the person who restored it.
CREATE TABLE comment_safety_decisions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id uuid NOT NULL REFERENCES comment_safety_events(id) ON DELETE CASCADE,
    actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
    actor_username text NOT NULL CHECK (actor_username ~ '^[a-z0-9_]{3,24}$'),
    action text NOT NULL CHECK (action IN ('dismissed', 'removed', 'reopened', 'restored')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX comment_safety_decisions_event
    ON comment_safety_decisions (event_id, created_at ASC, id ASC);
CREATE INDEX comment_safety_decisions_actor_recent
    ON comment_safety_decisions (actor_id, created_at DESC, id DESC)
    WHERE actor_id IS NOT NULL;

-- Reviews closed before this table existed have one step: the person
-- currently stored on the event.
INSERT INTO comment_safety_decisions (event_id, actor_id, actor_username, action, created_at)
SELECT e.id, e.resolver_id, resolver.username,
    CASE WHEN e.status = 'removed' THEN 'removed' ELSE 'dismissed' END,
    COALESCE(e.resolved_at, e.created_at)
FROM comment_safety_events e
JOIN users resolver ON resolver.id = e.resolver_id
WHERE e.status IN ('dismissed', 'removed')
  AND e.resolver_id IS NOT NULL;

-- +goose Down
DROP TABLE comment_safety_decisions;
