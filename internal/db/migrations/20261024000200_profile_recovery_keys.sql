-- +goose Up
ALTER TABLE users DROP COLUMN pronouns;

-- One active recovery key per account. Only the SHA-256 hash of the key is
-- stored (the key itself has 160 bits of entropy, like a session token);
-- the plaintext is shown to the user exactly once when it is issued.
CREATE TABLE recovery_keys (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    key_hash bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE recovery_keys;
ALTER TABLE users ADD COLUMN pronouns text NOT NULL DEFAULT '' CHECK (char_length(pronouns) <= 40);
