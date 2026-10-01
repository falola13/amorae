"use client";

import Link from "next/link";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { Button, Micro, Para, Title } from "@/components/ui/kit";
import type { Challenge } from "@/lib/api/types";
import { routes } from "@/lib/routes";
import { MAX_ACTIVE, TOO_MANY, shownDay, todaysDay } from "../challenges";
import { PastChallenges } from "./challenge-library";

/** One line about a mark: yours, or theirs. */
function mark(done?: boolean, skipped?: boolean, note?: string): string {
  if (done) return note ? "Done · noted" : "Done";
  if (skipped) return "Skipped";
  return "Not yet";
}

function RunningCard({ c, partner }: { c: Challenge; partner: string }) {
  const d = todaysDay(c);
  return (
    <li className="list-none">
      <Link
        href={routes.challenge(c.id)}
        className="press flex flex-col gap-2 rounded-card border border-line bg-surface px-5 py-[18px] text-ink no-underline"
      >
        <span className="flex items-center gap-3">
          <span className="flex min-w-0 grow flex-col gap-0.5">
            <span className="text-[17px] font-semibold">{c.title}</span>
            <span className="text-[12px] font-semibold text-stone">
              Day {shownDay(c)} of {c.days.length}
            </span>
          </span>
          <Icon name="right" size={18} className="text-stone" />
        </span>
        {d ? (
          <>
            <span className="text-[15px] leading-snug text-ink">{d.text}</span>
            <span className="flex flex-wrap gap-x-4 gap-y-0.5 text-[13px] font-semibold text-stone">
              <span>You: {mark(d.done, d.skipped, d.note)}</span>
              <span>
                {partner}: {mark(d.partner_done, d.partner_skipped, d.partner_note)}
              </span>
            </span>
          </>
        ) : null}
      </Link>
    </li>
  );
}

/** Everything running now, a card each, then a way to start another and the past ones. */
export function ChallengeRunning({
  list,
  partner,
  onStartAnother,
}: {
  list: Challenge[];
  partner: string;
  onStartAnother: () => void;
}) {
  const full = list.length >= MAX_ACTIVE;
  return (
    <Main>
      <div className="pt-2">
        <Title>Running now</Title>
        <Para className="mt-1.5">One small thing a day. Skip any day you need to.</Para>
      </div>
      <ol className="m-0 mt-4 flex list-none flex-col gap-3 p-0">
        {list.map((c) => (
          <RunningCard key={c.id} c={c} partner={partner} />
        ))}
      </ol>
      <div className="mt-4 flex flex-col gap-1.5">
        <div>
          <Button variant="secondary" icon="plus" disabled={full} onClick={onStartAnother}>
            Start another
          </Button>
        </div>
        {full ? <Micro>{TOO_MANY}</Micro> : null}
      </div>
      <PastChallenges />
    </Main>
  );
}
