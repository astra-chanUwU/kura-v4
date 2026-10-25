-- +goose Up
ALTER TABLE posts ADD COLUMN rating text NOT NULL DEFAULT 'sfw' CHECK (rating IN ('sfw', 'nsfw'));
ALTER TABLE users
    ADD COLUMN show_nsfw boolean NOT NULL DEFAULT false,
    ADD COLUMN nsfw_ack_at timestamptz;

-- +goose Down
ALTER TABLE users DROP COLUMN show_nsfw, DROP COLUMN nsfw_ack_at;
ALTER TABLE posts DROP COLUMN rating;
