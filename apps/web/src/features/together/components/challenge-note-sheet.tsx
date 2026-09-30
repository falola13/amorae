"use client";

import { useState } from "react";

import { BareTextarea, Button, Sheet } from "@/components/ui/kit";

/** "What did you do?" after marking a day done: one optional line the other of you can read.
 *  Mounted only while open, so each opening starts from `initial`. */
export function ChallengeNoteSheet(props: {
  open: boolean;
  onClose: () => void;
  partner: string;
  /** Your note so far, when adding to one you already wrote. */
  initial?: string;
  /** True while marking a day done (Skip note still marks it); false when editing a note. */
  marking: boolean;
  busy?: boolean;
  onSave: (note: string) => void;
}) {
  return props.open ? <Inner {...props} /> : null;
}

function Inner({
  onClose,
  partner,
  initial = "",
  marking,
  busy,
  onSave,
}: Parameters<typeof ChallengeNoteSheet>[0]) {
  const [note, setNote] = useState(initial);
  return (
    <Sheet
      open
      onClose={onClose}
      dirty={note.trim() !== initial.trim()}
      title="What did you do?"
      labelledBy="ch-note-h"
    >
      <BareTextarea
        label="Your note"
        hideLabel
        rows={3}
        autoFocus
        maxLength={500}
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder={`A line for ${partner}, if you like.`}
        className="min-h-[84px] text-[19px] leading-[1.6]"
      />
      <div className="flex flex-col gap-1">
        <Button loading={busy} onClick={() => onSave(note.trim())}>
          Save
        </Button>
        {marking ? (
          <Button variant="text" disabled={busy} onClick={() => onSave("")}>
            Skip note
          </Button>
        ) : (
          <Button variant="text" onClick={onClose}>
            Not now
          </Button>
        )}
      </div>
    </Sheet>
  );
}
