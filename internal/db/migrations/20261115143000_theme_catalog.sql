-- +goose Up
-- Theme ids are checked in the accounts service, which mirrors
-- frontend/src/domain/themeCatalog.ts. A SQL IN-list cannot follow that
-- catalog without a migration for every palette.
ALTER TABLE users DROP CONSTRAINT users_preferred_theme_check;

-- +goose Down
UPDATE users SET preferred_theme = 'kura'
WHERE preferred_theme NOT IN (
    'kura', 'safebooru', 'rule34', 'umineko',
    'bernkastel', 'lambdadelta', 'beatrice', 'erika', 'battler', 'virgilia',
    'catppuccin', 'danbooru'
);
ALTER TABLE users ADD CONSTRAINT users_preferred_theme_check
    CHECK (preferred_theme IN (
        'kura', 'safebooru', 'rule34', 'umineko',
        'bernkastel', 'lambdadelta', 'beatrice', 'erika', 'battler', 'virgilia',
        'catppuccin', 'danbooru'
    ));
