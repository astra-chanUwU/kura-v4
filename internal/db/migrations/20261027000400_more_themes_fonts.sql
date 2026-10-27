-- +goose Up
ALTER TABLE users DROP CONSTRAINT users_preferred_theme_check;
ALTER TABLE users ADD CONSTRAINT users_preferred_theme_check
    CHECK (preferred_theme IN ('kura', 'safebooru', 'rule34', 'umineko',
        'bernkastel', 'lambdadelta', 'beatrice', 'erika', 'battler', 'virgilia',
        'catppuccin', 'danbooru'));
ALTER TABLE users DROP CONSTRAINT users_preferred_font_check;
ALTER TABLE users ADD CONSTRAINT users_preferred_font_check
    CHECK (preferred_font IN ('system', 'serif', 'rounded', 'mono', 'hack'));

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_preferred_theme_check;
ALTER TABLE users ADD CONSTRAINT users_preferred_theme_check
    CHECK (preferred_theme IN ('kura', 'safebooru', 'rule34', 'umineko'));
ALTER TABLE users DROP CONSTRAINT users_preferred_font_check;
ALTER TABLE users ADD CONSTRAINT users_preferred_font_check
    CHECK (preferred_font IN ('system', 'serif', 'rounded', 'mono'));
