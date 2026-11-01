-- +goose Up
-- A tag page is the shared description of one tag. The row is optional:
-- a tag can be on posts before anyone writes a description.
CREATE TABLE tag_pages (
    tag text PRIMARY KEY CHECK (tag ~ '^[a-z0-9][a-z0-9_]{0,39}$'),
    category text NOT NULL DEFAULT 'general' CHECK (category IN ('general', 'artist', 'character', 'copyright', 'meta')),
    description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
    updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- alias is the alternate name. target is the canonical tag.
-- Searching or adding the alias uses the target, the same way a booru alias does.
CREATE TABLE tag_aliases (
    alias text PRIMARY KEY CHECK (alias ~ '^[a-z0-9][a-z0-9_]{0,39}$'),
    target text NOT NULL CHECK (target ~ '^[a-z0-9][a-z0-9_]{0,39}$' AND target <> alias),
    created_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX tag_aliases_target_idx ON tag_aliases (target);

-- +goose Down
DROP TABLE tag_aliases;
DROP TABLE tag_pages;
