-- +goose Up
-- Enough to recognise your own sessions in "where you're signed in", and
-- nothing more: no IP address and no location. The raw user agent stays here;
-- the API only ever returns a short label built from it ("Safari on iPhone").
-- last_used_at is touched at most once an hour per session, so an active
-- session costs one small write an hour rather than one per request.
ALTER TABLE sessions
ADD COLUMN last_used_at TIMESTAMPTZ,
ADD COLUMN user_agent TEXT;

-- +goose Down
ALTER TABLE sessions
DROP COLUMN last_used_at,
DROP COLUMN user_agent;
