"use client";

import Link from "next/link";
import { useState } from "react";

import { Button, Micro } from "@/components/ui/kit";
import { ChallengeNoteSheet } from "@/features/together/components/challenge-note-sheet";
import { homeChallengeDay } from "@/features/together/challenges";
import { useChallenge, useChallengeDay } from "@/features/together/hooks";
import { routes } from "@/lib/routes";

/** Today's challenge day, offered on Home until you have marked it. Hidden once you have,
 *  or when nothing is running or today is not open. The note sheet outlives the card. */
export function HomeChallengeCard({ partner }: { partner: string }) {
  const challenge = useChallenge();
  const set = useChallengeDay();
  const [noting, setNoting] = useState<number | null>(null);
  const day = homeChallengeDay(challenge.data);

  const save = (n: number, note: string) => {
    const close = () => setNoting(null);
    // One request for the mark and the note, so the last day is not closed off before its note.
    set.mutate(
      { n, patch: { done: true, ...(note ? { note } : {}) } },
      { onSuccess: close, onQueued: close },
    );
  };

  return (
    <>
      {day ? (
        <section
          aria-label="Today’s challenge"
          className="mt-5 flex flex-col gap-3 rounded-card border border-line bg-surface px-5 py-[18px]"
        >
          <Micro tone="plum">{challenge.data?.title}</Micro>
          <p className="m-0 text-[19px] font-semibold leading-snug tracking-[-0.01em] text-ink">
            Day {day.n} · {day.text}
          </p>
          <div className="flex items-center gap-2">
            <Button icon="check" className="grow" onClick={() => setNoting(day.n)}>
              Done
            </Button>
            <Link
              href={routes.challenges}
              className="press flex h-[52px] items-center px-4 text-[16px] font-semibold text-plum no-underline"
            >
              Open
            </Link>
          </div>
        </section>
      ) : null}
      <ChallengeNoteSheet
        open={noting !== null}
        onClose={() => setNoting(null)}
        partner={partner}
        marking
        busy={set.isPending}
        onSave={(note) => noting !== null && save(noting, note)}
      />
    </>
  );
}
