-- +goose Up
ALTER TABLE couple_members
ADD COLUMN updated_at TIMESTAMPTZ DEFAULT now();

-- +goose Down
ALTER TABLE couple_members
DROP COLUMN updated_at;