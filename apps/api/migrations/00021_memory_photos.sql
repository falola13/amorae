-- +goose Up
-- Where a memory's photo actually is (FR-MEM-003).

-- The Cloudinary public id. Derived by the server from the couple and the
-- memory, never supplied by a client, so a signed upload ticket can only ever
-- write to that memory's own picture.
ALTER TABLE memories ADD COLUMN photo_id TEXT NULL;

-- has_photo was a flag beside the thing it described, and the two could
-- disagree: a memory could claim a picture that was never stored, or hold one
-- it said nothing about. There is one fact here — whether photo_id is set —
-- and the API derives the flag from it, the same reasoning that kept a
-- running total off the goals table.
ALTER TABLE memories DROP COLUMN has_photo;

-- +goose Down
ALTER TABLE memories ADD COLUMN has_photo BOOLEAN NOT NULL DEFAULT false;
UPDATE memories SET has_photo = photo_id IS NOT NULL;
ALTER TABLE memories DROP COLUMN photo_id;
