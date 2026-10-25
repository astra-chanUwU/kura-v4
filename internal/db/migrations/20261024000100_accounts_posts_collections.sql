-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username text NOT NULL UNIQUE CHECK (username = lower(username) AND username ~ '^[a-z0-9_]{3,24}$'),
    display_name text NOT NULL DEFAULT '' CHECK (char_length(display_name) <= 60),
    bio text NOT NULL DEFAULT '' CHECK (char_length(bio) <= 500),
    pronouns text NOT NULL DEFAULT '' CHECK (char_length(pronouns) <= 40),
    location text NOT NULL DEFAULT '' CHECK (char_length(location) <= 80),
    website text NOT NULL DEFAULT '' CHECK (char_length(website) <= 200),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Password and passkeys are interchangeable sign-in methods on one account.
-- An account must always keep at least one; that rule lives in the service.
CREATE TABLE password_credentials (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE webauthn_credentials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id bytea NOT NULL UNIQUE,
    public_key bytea NOT NULL,
    attestation_type text NOT NULL DEFAULT '',
    transports text[] NOT NULL DEFAULT '{}',
    user_present boolean NOT NULL DEFAULT true,
    user_verified boolean NOT NULL DEFAULT false,
    backup_eligible boolean NOT NULL DEFAULT false,
    backup_state boolean NOT NULL DEFAULT false,
    aaguid bytea,
    sign_count bigint NOT NULL DEFAULT 0,
    display_name text NOT NULL DEFAULT 'Passkey',
    created_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);
CREATE INDEX webauthn_credentials_user_idx ON webauthn_credentials (user_id);

-- Only the SHA-256 hash of the session token is stored.
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_idx ON sessions (user_id);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);

-- Short-lived WebAuthn ceremonies (registration / login).
CREATE TABLE webauthn_challenges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose text NOT NULL CHECK (purpose IN ('register', 'login', 'add')),
    username text NOT NULL DEFAULT '',
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    session_data jsonb NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE posts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid REFERENCES users(id) ON DELETE SET NULL,
    title text NOT NULL DEFAULT '' CHECK (char_length(title) <= 200),
    original_name text NOT NULL DEFAULT '',
    storage_key text NOT NULL UNIQUE,
    content_type text NOT NULL,
    size_bytes bigint NOT NULL DEFAULT 0,
    sha256 text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX posts_created_idx ON posts (created_at DESC, id DESC);
CREATE UNIQUE INDEX posts_sha256_idx ON posts (sha256) WHERE sha256 <> '';

-- Favorites are always private to their owner.
CREATE TABLE favorites (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);
CREATE INDEX favorites_user_created_idx ON favorites (user_id, created_at DESC);

CREATE TABLE lists (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 80),
    description text NOT NULL DEFAULT '' CHECK (char_length(description) <= 300),
    visibility text NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX lists_owner_idx ON lists (owner_id, updated_at DESC);

-- Ordered set of posts; position is the share/display order.
CREATE TABLE list_items (
    list_id uuid NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    post_id uuid NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    position integer NOT NULL CHECK (position >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (list_id, post_id),
    UNIQUE (list_id, position) DEFERRABLE INITIALLY DEFERRED
);
CREATE INDEX list_items_order_idx ON list_items (list_id, position);

-- +goose Down
DROP TABLE list_items;
DROP TABLE lists;
DROP TABLE favorites;
DROP TABLE posts;
DROP TABLE webauthn_challenges;
DROP TABLE sessions;
DROP TABLE webauthn_credentials;
DROP TABLE password_credentials;
DROP TABLE users;
