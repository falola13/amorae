"use client";

import Link from "next/link";
import { useState } from "react";

import { Button, Micro } from "@/components/ui/kit";
import { ChallengeNoteSheet } from "@/features/together/components/challenge-note-sheet";
import { homeChallengeRows } from "@/features/together/challenges";
import { useActiveChallenges, useChallengeDay } from "@/features/together/hooks";
import { routes } from "@/lib/routes";

/** Today's open, unmarked prompt for each running challenge (at most three), one row each.
 *  A row goes once you have marked it; the card goes when none are left. The note sheet
 *  outlives the card. */
export function HomeChallengeCard({ partner }: { partner: string }) {
  const active = useActiveChallenges();
  const set = useChallengeDay();
  const [noting, setNoting] = useState<{ id: string; n: number } | null>(null);
  const rows = homeChallengeRows(active.data);

  const save = (id: string, n: number, note: string) => {
    const close = () => setNoting(null);
    // One request for the mark and the note, so the last day is not closed off before its note.
    set.mutate(
      { id, n, patch: { done: true, ...(note ? { note } : {}) } },
      { onSuccess: close, onQueued: close },
    );
  };

  return (
    <>
      {rows.length > 0 ? (
        <section
          aria-label="Today’s challenges"
          className="mt-5 flex flex-col gap-1 rounded-card border border-line bg-surface px-5 py-[18px]"
        >
          <Micro tone="plum">Today’s challenges</Micro>
          <ul className="m-0 flex list-none flex-col p-0">
            {rows.map(({ c, day }) => (
              <li
                key={c.id}
                className="flex flex-col gap-2.5 border-b border-line py-3 last:border-b-0"
              >
                <Link
                  href={routes.challenge(c.id)}
                  className="press text-[17px] font-semibold leading-snug tracking-[-0.01em] text-ink no-underline"
                >
                  {c.title} · Day {day.n}: {day.text}
                </Link>
                <Button icon="check" onClick={() => setNoting({ id: c.id, n: day.n })}>
                  Done
                </Button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      <ChallengeNoteSheet
        open={noting !== null}
        onClose={() => setNoting(null)}
        partner={partner}
        marking
        busy={set.isPending}
        onSave={(note) => noting !== null && save(noting.id, noting.n, note)}
      />
    </>
  );
}
