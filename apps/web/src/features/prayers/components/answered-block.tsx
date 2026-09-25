"use client";

import { useState } from "react";

import { Icon } from "@/components/icons";
import { Button } from "@/components/ui/kit";
import { useUnanswer } from "@/features/prayers/hooks";
import { longDate } from "@/lib/dates";
import type { PrayerPoint } from "@/lib/api/types";

import { AnswerSheet } from "./answer-sheet";

// No "not answered" state drawn here, only an unobtrusive offer — an unanswered prayer isn't a task left undone.
export function AnsweredBlock({ p }: { p: PrayerPoint }) {
  const [open, setOpen] = useState(false);
  const unanswer = useUnanswer();

  if (!p.answered_at) {
    return (
      <>
        <div className="flex">
          <button
            type="button"
            onClick={() => setOpen(true)}
            className="press -ml-1 flex items-center gap-2 rounded-btn px-1 py-1.5 text-[15px] font-semibold text-plum"
          >
            <Icon name="check" size={18} />
            Mark as answered
          </button>
        </div>
        {open ? (
          <AnswerSheet open onClose={() => setOpen(false)} pointId={p.id} title={p.title} />
        ) : null}
      </>
    );
  }

  return (
    <>
      <div className="flex flex-col gap-2 rounded-card bg-green-tint px-4 py-3.5">
        <div className="flex items-center gap-2 text-green">
          <Icon name="check" size={18} />
          <span className="text-support font-bold">
            Answered · {longDate(p.answered_on ?? p.answered_at.slice(0, 10))}
          </span>
        </div>
        {p.answer_note ? (
          <p className="m-0 text-[17px] leading-[1.55] text-ink" data-selectable>
            {p.answer_note}
          </p>
        ) : null}
        <div className="-mb-1.5 -ml-2 flex">
          <Button variant="text" onClick={() => setOpen(true)} className="w-auto px-2">
            {p.answer_note ? "Edit note" : "Add a note"}
          </Button>
          <Button
            variant="text"
            onClick={() => unanswer.mutate({ pointId: p.id })}
            className="w-auto px-2"
          >
            Undo
          </Button>
        </div>
      </div>
      {open ? (
        <AnswerSheet
          open
          onClose={() => setOpen(false)}
          pointId={p.id}
          title={p.title}
          note={p.answer_note ?? ""}
        />
      ) : null}
    </>
  );
}
