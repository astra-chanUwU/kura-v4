-- +goose Up
ALTER TABLE users
    ADD COLUMN preferred_theme text NOT NULL DEFAULT 'kura'
        CHECK (preferred_theme IN ('kura', 'safebooru', 'rule34', 'umineko')),
    ADD COLUMN preferred_font text NOT NULL DEFAULT 'system'
        CHECK (preferred_font IN ('system', 'serif', 'rounded', 'mono'));

-- +goose Down
ALTER TABLE users DROP COLUMN preferred_theme, DROP COLUMN preferred_font;
