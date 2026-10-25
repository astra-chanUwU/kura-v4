-- +goose Up
-- Usernameless (discoverable) passkey sign-in stores its ceremony
-- under a fourth purpose.
ALTER TABLE webauthn_challenges DROP CONSTRAINT webauthn_challenges_purpose_check;
ALTER TABLE webauthn_challenges
    ADD CONSTRAINT webauthn_challenges_purpose_check
    CHECK (purpose IN ('register', 'login', 'add', 'discoverable'));

-- +goose Down
ALTER TABLE webauthn_challenges DROP CONSTRAINT webauthn_challenges_purpose_check;
ALTER TABLE webauthn_challenges
    ADD CONSTRAINT webauthn_challenges_purpose_check
    CHECK (purpose IN ('register', 'login', 'add'));
