-- +goose Up
-- Roles: visitor is anonymous (no row); stored roles are the signed-in tiers.
ALTER TABLE users
    ADD COLUMN role text NOT NULL DEFAULT 'member'
    CHECK (role IN ('super_admin', 'admin', 'moderator', 'member'));

-- Hidden posts are removed from the public board but never deleted:
-- originals stay in storage so a hide/delete can always be rolled back.
ALTER TABLE posts
    ADD COLUMN hidden_at timestamptz,
    ADD COLUMN hidden_by uuid REFERENCES users(id) ON DELETE SET NULL;

-- Append-only change history. before_state/after_state are JSON snapshots:
-- post -> {"artist","tags","rating","hidden"}; user -> {"role"}.
-- A rollback restores before_state and writes its own row (rollback_of),
-- so the log itself is never rewritten.
CREATE TABLE change_history (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type text NOT NULL CHECK (entity_type IN ('post', 'user')),
    entity_id uuid NOT NULL,
    actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
    actor_username text NOT NULL DEFAULT '',
    action text NOT NULL CHECK (action IN (
        'post.created', 'post.meta_updated', 'post.hidden', 'post.unhidden',
        'post.deleted', 'user.role_changed', 'rollback'
    )),
    before_state jsonb NOT NULL DEFAULT '{}',
    after_state jsonb NOT NULL DEFAULT '{}',
    rollback_of uuid REFERENCES change_history(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX change_history_entity_idx ON change_history (entity_type, entity_id, created_at DESC);
CREATE INDEX change_history_actor_idx ON change_history (actor_id, created_at DESC);
CREATE INDEX change_history_created_idx ON change_history (created_at DESC);

-- +goose Down
DROP TABLE change_history;
ALTER TABLE posts DROP COLUMN hidden_by, DROP COLUMN hidden_at;
ALTER TABLE users DROP COLUMN role;
