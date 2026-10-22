-- +goose Up
CREATE TABLE site_settings (
    key text PRIMARY KEY,
    value text NOT NULL
);
INSERT INTO site_settings (key, value) VALUES
    ('site_name', 'Kura'),
    ('site_description', 'A home for your collection.');

CREATE TABLE jobs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    state text NOT NULL DEFAULT 'queued' CHECK (state IN ('queued', 'running', 'complete', 'failed')),
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 3 CHECK (max_attempts > 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    lease_until timestamptz,
    lease_token text,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX jobs_claim_idx ON jobs (state, available_at, id)
    WHERE state IN ('queued', 'running');

-- +goose Down
DROP TABLE jobs;
DROP TABLE site_settings;
