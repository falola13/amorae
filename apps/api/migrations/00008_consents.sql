-- +goose Up
-- One row per consent event, never updated or overwritten, so the history
-- of what someone agreed to, and when, is always recoverable.
-- kind is one of 'terms', 'privacy', 'age_18', 'faith_content'.
CREATE TABLE user_consents (
    id             UUID PRIMARY KEY,
    user_id        UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind           TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX user_consents_user_id_idx ON user_consents (user_id);

-- +goose Down
DROP TABLE user_consents;
