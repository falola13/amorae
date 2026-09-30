"use client";

import { useState } from "react";

import { BareTextarea, Button, Sheet } from "@/components/ui/kit";
import { useAddMemory, useEventOutcome } from "@/features/together/hooks";
import type { Event } from "@/lib/api/types";
import { longDate } from "@/lib/dates";

// Everything is prefilled except the note, which the event can't supply. Marking the event done rides along.
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
  const outcome = useEventOutcome();
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
          // Only after the memory is stored — the other order could close the event while it never landed.
          if (!e.done) outcome.mutate({ id: e.id, outcome: "happened" });
          onClose();
          onKept?.();
        },
        // Queued offline, the event is left open: it closes only once the memory is known to have landed.
        onQueued: onClose,
      },
    );
  };

  return (
    <Sheet
      open={open}
      onClose={onClose}
      dirty={note !== (e.notes ?? "")}
      title="Keep this?"
      labelledBy="keep-h"
    >
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
