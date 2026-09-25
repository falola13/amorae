"use client";

import { useState } from "react";

import { BareTextarea, Button, Sheet } from "@/components/ui/kit";
import { useAddMemory, useCompleteEvent } from "@/features/together/hooks";
import type { Event } from "@/lib/api/types";
import { longDate } from "@/lib/dates";

/**
 * Keeping a finished event.
 *
 * The button that opened onto this used to say "Save a moment from it" and
 * then navigate to the memories list — the whole list, with nothing carried
 * over. You were told the app would do something and then handed a blank
 * screen and the job of retyping what it already knew: the title, the day, the
 * place. This does the thing the button always said it did.
 *
 * Everything is prefilled and only the note is asked for, because the note is
 * the only part the event cannot supply. Marking the event done comes with it:
 * keeping something is the clearest way of saying it happened, and asking
 * twice would be asking somebody to file a form about their own evening.
 */
export function KeepAsMemorySheet({
  event: e,
  open,
  onClose,
  onKept,
}: {
  event: Event;
  open: boolean;
  onClose: () => void;
  onKept?: () => void;
}) {
  const add = useAddMemory();
  const complete = useCompleteEvent();
  const [note, setNote] = useState(e.notes ?? "");

  const keep = (ev: React.FormEvent) => {
    ev.preventDefault();
    add.mutate(
      {
        title: e.title,
        date: e.date,
        location: e.location ?? "",
        note: note.trim(),
        has_photo: false,
      },
      {
        onSuccess: () => {
          // Only after the memory is safely stored. If this fails the event
          // stays open, which is recoverable; the other order would quietly
          // close an event whose memory never landed.
          if (!e.done) complete.mutate({ id: e.id, done: true });
          onClose();
          onKept?.();
        },
      },
    );
  };

  return (
    <Sheet open={open} onClose={onClose} title="Keep this?" labelledBy="keep-h">
      <form method="post" onSubmit={keep} className="contents" noValidate>
        <p className="m-0 -mt-2 text-support text-stone">
          {e.title} &middot; {longDate(e.date)} {e.date.slice(0, 4)}
          {e.location ? ` · ${e.location}` : ""}
        </p>
        <BareTextarea
          label="What you want to remember"
          hideLabel
          rows={4}
          autoFocus
          placeholder="Anything worth remembering about it — or leave it as it is."
          className="min-h-[110px] text-[19px] leading-[1.6]"
          value={note}
          onChange={(ev) => setNote(ev.target.value)}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" icon="image" loading={add.isPending}>
            Keep it
          </Button>
          <Button type="button" variant="text" onClick={onClose}>
            Not now
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
