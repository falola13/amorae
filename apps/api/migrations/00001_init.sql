-- +goose Up
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    display_name  TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar_url    TEXT NULL,
    timezone      TEXT NOT NULL DEFAULT 'UTC',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ NULL,
    CONSTRAINT users_email_key UNIQUE (email)
);

CREATE TABLE couples (
    id UUID PRIMARY KEY,
    name TEXT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    relationship_start_date DATE NULL,
    created_by UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE couple_members (
id UUID PRIMARY KEY,
couple_id UUID NOT NULL REFERENCES couples (id),
user_id UUID NOT NULL REFERENCES users (id),
role TEXT NOT NULL DEFAULT 'partner',
joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
CONSTRAINT couple_members_couple_user_key UNIQUE (couple_id, user_id),
CONSTRAINT couple_members_user_id_key UNIQUE (user_id)
);

CREATE TYPE invitation_status as ENUM ('pending', 'accepted','revoked','expired');

CREATE TABLE couple_invitations (
    id UUID PRIMARY KEY,
    couple_id UUID NOT NULL REFERENCES couples (id),
    code TEXT UNIQUE NOT NULL,
    status invitation_status NOT NULL DEFAULT 'pending',
    created_by UUID NOT NULL REFERENCES users (id),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- +goose Down
-- Reverse dependency order: everything that references users goes first,
-- or Postgres refuses to drop it.
DROP TABLE couple_invitations;
DROP TYPE invitation_status;
DROP TABLE couple_members;
DROP TABLE couples;
DROP TABLE users;
