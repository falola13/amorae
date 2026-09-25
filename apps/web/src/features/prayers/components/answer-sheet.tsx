"use client";

import { useState } from "react";

import { BareTextarea, Button, Sheet } from "@/components/ui/kit";
import { useAnswer } from "@/features/prayers/hooks";

// Note is optional and never blocks the submit. Also reused to edit the note afterwards.
export function AnswerSheet({
  open,
  onClose,
  pointId,
  title,
  note,
}: {
  open: boolean;
  onClose: () => void;
  pointId: string;
  title: string;
  /** The note already saved, when this is an edit rather than a first mark. */
  note?: string;
}) {
  const answer = useAnswer();
  // Seeded once at mount — the sheet unmounts when closed, so every open is a fresh mount with the current note.
  const [text, setText] = useState(note ?? "");

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    answer.mutate({ pointId, note: text }, { onSuccess: onClose });
  };

  return (
    <Sheet open={open} onClose={onClose} title="What happened?" labelledBy="answer-h">
      <form method="post" onSubmit={submit} className="contents" noValidate>
        <p className="m-0 -mt-2 text-support text-stone">{title}</p>
        <BareTextarea
          label="What happened"
          hideLabel
          rows={4}
          autoFocus
          placeholder="A line about how this was answered — or leave it blank."
          className="min-h-[110px] text-[19px] leading-[1.6]"
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <div className="flex flex-col gap-1">
          <Button type="submit" icon="check" loading={answer.isPending}>
            {note === undefined ? "Mark as answered" : "Save"}
          </Button>
          <Button type="button" variant="text" onClick={onClose}>
            Not now
          </Button>
        </div>
      </form>
    </Sheet>
  );
}
