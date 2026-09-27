-- +goose Up
-- Where a person's own profile photo actually is, same reasoning as
-- memories' photo_id (00021_memory_photos.sql): the Cloudinary public id,
-- derived by the server from the user, never supplied by a client, so a
-- signed upload ticket can only ever write to that user's own picture.
-- avatar_url (00001_init.sql) went unused — nothing in the API ever read or
-- wrote it — so this is a fresh column rather than repurposing one whose
-- name promises a URL when what's stored is a public id.
ALTER TABLE users ADD COLUMN photo_id TEXT NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN photo_id;
