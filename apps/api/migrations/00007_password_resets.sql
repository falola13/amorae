-- +goose Up
-- One row per reset link. Only the SHA-256 of the token is stored, exactly
-- like sessions, so a database leak can't be turned into working links.
-- used_at makes a link single-use; expires_at bounds how long it lives.
CREATE TABLE password_resets (
    token_hash BYTEA PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_resets_user_id_idx ON password_resets (user_id);

-- +goose Down
DROP TABLE password_resets;
