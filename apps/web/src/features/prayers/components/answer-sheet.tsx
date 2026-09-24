"use client";

import { useState } from "react";

import { BareTextarea, Button, Sheet } from "@/components/ui/kit";
import { useAnswer } from "@/features/prayers/hooks";

/**
 * Marking a prayer answered, and saying how.
 *
 * The note is optional and the button does not wait for it. Sometimes the
 * answer is the whole story — he got the job — and requiring a sentence first
 * would turn the gladdest action in the app into a piece of homework. The
 * field is offered, not demanded.
 *
 * The same sheet edits the note afterwards, because what you write in the
 * first ten seconds is rarely what you would write a week later.
 */
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
  // Seeded once, at mount. The parent only renders this while it is open and
  // <Sheet> is null when closed, so every opening is a fresh mount with the
  // note as it stands — which is what an effect here would have been faking,
  // at the cost of a cascading render.
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
