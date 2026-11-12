-- +goose Up
-- A member proposes an alias or implication. A moderator accepts it,
-- which applies the same rules as adding the alias or implication directly.
CREATE TABLE tag_suggestions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind text NOT NULL CHECK (kind IN ('alias', 'implication')),
    source_tag text NOT NULL CHECK (source_tag ~ '^[a-z0-9][a-z0-9_]{0,39}$'),
    target_tag text NOT NULL CHECK (target_tag ~ '^[a-z0-9][a-z0-9_]{0,39}$'),
    reason text NOT NULL DEFAULT '' CHECK (char_length(reason) <= 300),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    author_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    resolved_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    resolved_at timestamptz,
    CHECK (source_tag <> target_tag)
);
CREATE UNIQUE INDEX tag_suggestions_pending_idx
    ON tag_suggestions (kind, source_tag, target_tag)
    WHERE status = 'pending';
CREATE INDEX tag_suggestions_status_idx ON tag_suggestions (status, created_at DESC);

-- +goose Down
DROP TABLE tag_suggestions;
