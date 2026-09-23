-- +goose Up
-- Child rows disappear with the parent, the same way sessions already do.
-- couples.created_by is deliberately not cascading: deleting the creator
-- must not delete a couple that still has a partner. The user repository
-- reassigns created_by before it deletes that user.
ALTER TABLE couple_members DROP CONSTRAINT couple_members_couple_id_fkey;
ALTER TABLE couple_members
    ADD CONSTRAINT couple_members_couple_id_fkey
    FOREIGN KEY (couple_id) REFERENCES couples (id) ON DELETE CASCADE;

ALTER TABLE couple_members DROP CONSTRAINT couple_members_user_id_fkey;
ALTER TABLE couple_members
    ADD CONSTRAINT couple_members_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE couple_invitations DROP CONSTRAINT couple_invitations_couple_id_fkey;
ALTER TABLE couple_invitations
    ADD CONSTRAINT couple_invitations_couple_id_fkey
    FOREIGN KEY (couple_id) REFERENCES couples (id) ON DELETE CASCADE;

ALTER TABLE couple_invitations DROP CONSTRAINT couple_invitations_created_by_fkey;
ALTER TABLE couple_invitations
    ADD CONSTRAINT couple_invitations_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE;

-- Cascade deletes look rows up by these columns. A foreign key does not
-- create that index itself.
CREATE INDEX couple_invitations_couple_id_idx ON couple_invitations (couple_id);
CREATE INDEX couple_invitations_created_by_idx ON couple_invitations (created_by);

-- +goose Down
DROP INDEX couple_invitations_created_by_idx;
DROP INDEX couple_invitations_couple_id_idx;

ALTER TABLE couple_invitations DROP CONSTRAINT couple_invitations_created_by_fkey;
ALTER TABLE couple_invitations
    ADD CONSTRAINT couple_invitations_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users (id);

ALTER TABLE couple_invitations DROP CONSTRAINT couple_invitations_couple_id_fkey;
ALTER TABLE couple_invitations
    ADD CONSTRAINT couple_invitations_couple_id_fkey
    FOREIGN KEY (couple_id) REFERENCES couples (id);

ALTER TABLE couple_members DROP CONSTRAINT couple_members_user_id_fkey;
ALTER TABLE couple_members
    ADD CONSTRAINT couple_members_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users (id);

ALTER TABLE couple_members DROP CONSTRAINT couple_members_couple_id_fkey;
ALTER TABLE couple_members
    ADD CONSTRAINT couple_members_couple_id_fkey
    FOREIGN KEY (couple_id) REFERENCES couples (id);
