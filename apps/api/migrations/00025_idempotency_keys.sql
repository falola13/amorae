-- +goose Up
-- A key the client chose, and the reply it got — so a write sent twice does
-- not happen twice (FR-PWA-009).
--
-- Scoped to the user, not the couple: the key belongs to the device that made
-- it, and answering one partner with the other's reply would be a leak rather
-- than a bug. method and path are kept so the same key on a different request
-- can be refused instead of replaying an answer to a question nobody asked.
--
-- status is null while the request is still running. That is the difference
-- between "in flight" and "done", and it is what lets a second send be told
-- to wait rather than being answered with a reply that does not exist yet.
CREATE TABLE idempotency_keys (
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key        TEXT NOT NULL,
    method     TEXT NOT NULL,
    path       TEXT NOT NULL,
    status     SMALLINT NULL,
    body       BYTEA NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, key)
);

-- Rows are worth keeping only as long as a queued write might still be
-- replayed. The offline queue holds writes for seven days (lib/query/persist),
-- so anything older cannot be answering one.
CREATE INDEX idempotency_keys_created_idx ON idempotency_keys (created_at);

-- +goose Down
DROP TABLE idempotency_keys;
