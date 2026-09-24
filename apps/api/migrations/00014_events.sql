-- +goose Up
-- Events the couple plans together. There is no owner and no participants
-- list: both partners are implicit participants in everything (DEC-16), so
-- the only thing an event belongs to is the couple.

CREATE TABLE events (
    id         UUID PRIMARY KEY,
    couple_id  UUID NOT NULL REFERENCES couples (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    -- A calendar day, not an instant: "dinner on Friday" does not move when
    -- somebody travels. The time, if there is one, is a wall clock read in
    -- the couple's timezone.
    date       DATE NOT NULL,
    start_time TIME NULL,
    end_time   TIME NULL,
    location   TEXT NULL,
    -- Free text as the person wrote it ("an hour before"), not a duration to
    -- compute with. When reminders are delivered this becomes structured;
    -- until then, storing a parsed value would be inventing precision.
    reminder   TEXT NULL,
    notes      TEXT NULL,
    done       BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every list of events is "this couple's, by day".
CREATE INDEX events_couple_date_idx ON events (couple_id, date);

CREATE TABLE event_checklist_items (
    id         UUID PRIMARY KEY,
    event_id   UUID NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    -- 0-based, the order they were written in.
    position   SMALLINT NOT NULL,
    text       TEXT NOT NULL,
    -- One flag for the couple, not one each: a checklist is the things that
    -- need doing between them, and "who ticked it" is not a question either
    -- of them is asking. Prayer completion is per person for the opposite
    -- reason — praying is something each of them does themselves.
    done       BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT event_checklist_position_key UNIQUE (event_id, position)
);

CREATE INDEX event_checklist_event_idx ON event_checklist_items (event_id);

-- +goose Down
DROP TABLE event_checklist_items;
DROP TABLE events;
