"use client";

import Link from "next/link";
import { useState } from "react";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import { Button, Micro, Para, Skeleton, Title } from "@/components/ui/kit";
import {
  useActiveChallenges,
  useChallengeTemplates,
  usePastChallenges,
} from "@/features/together/hooks";
import type { ChallengeTemplate } from "@/lib/api/types";
import { longDate } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";
import { useCouple } from "@/features/couple/hooks";
import {
  TOO_MANY,
  atLimit,
  currentSeason,
  ownerLabel,
  groupTemplates,
  overLabel,
  runningKeys,
} from "../challenges";
import { StartChallengeSheet } from "./start-challenge-sheet";

/** The challenges you have finished or ended, newest first. Nothing when there are none. */
export function PastChallenges() {
  const past = usePastChallenges();
  const couple = useCouple();
  const meId = couple.data?.me.id;
  const partner = couple.data?.partner?.display_name ?? "They";
  return (
    <>
      {(past.data ?? []).length > 0 ? (
        <section className="mt-8 flex flex-col pb-6">
          <Micro>Past challenges</Micro>
          <ol className="m-0 mt-1 list-none border-t border-line p-0">
            {(past.data ?? []).map((p) => (
              <li key={p.id} className="border-b border-line">
                <Link
                  href={routes.challenge(p.id)}
                  className="press flex min-h-[62px] items-center gap-3.5 py-3 text-ink no-underline"
                >
                  <span className="flex min-w-0 grow flex-col gap-0.5">
                    <span className="text-[16px] font-semibold">{p.title}</span>
                    <span className="text-support text-stone">
                      {overLabel(p.status)}
                      {p.kind === "mine"
                        ? ` · ${ownerLabel({ kind: "mine", can_edit: p.created_by === meId }, partner)}`
                        : ""}
                      {p.ended_at ? ` · ${longDate(p.ended_at.slice(0, 10))}` : ""}
                    </span>
                  </span>
                  <span className="shrink-0 text-[13px] font-semibold text-stone">
                    {p.my_done} of {p.days}
                  </span>
                  <Icon name="right" size={18} className="text-stone" />
                </Link>
              </li>
            ))}
          </ol>
        </section>
      ) : null}
    </>
  );
}

/** Pick one, make your own. Ones already running are marked and cannot be started twice.
 *  `onBack` is given when you came here from the list of running ones. */
export function ChallengeLibrary({ onBack }: { onBack?: () => void }) {
  const templates = useChallengeTemplates();
  const active = useActiveChallenges();
  const running = runningKeys(active.data);
  const full = atLimit(active.data);
  const [picked, setPicked] = useState<ChallengeTemplate | null>(null);
  const season = currentSeason(today());

  return (
    <>
      <QueryState
        queries={[templates]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(all) => (
          <Main>
            {onBack ? (
              <div className="pt-1">
                <Button variant="text" icon="left" onClick={onBack}>
                  Running challenges
                </Button>
              </div>
            ) : null}
            <div className="pt-2">
              <Title>A challenge together</Title>
              <Para className="mt-1.5">
                A few days of small things, one a day. You each mark your own, and either of you can
                skip a day without it counting against anything.
              </Para>
              {full ? <Para className="mt-2 text-plum">{TOO_MANY}</Para> : null}
            </div>

            {groupTemplates(all, season).map((g) => (
              <section key={g.category} className="mt-6 flex flex-col">
                <Micro>{g.label}</Micro>
                <ol className="m-0 mt-1 list-none border-t border-line p-0">
                  {g.templates.map((t) => (
                    <li key={t.key} className="border-b border-line">
                      <button
                        type="button"
                        disabled={running.has(t.key) || full}
                        onClick={() => setPicked(t)}
                        className="press flex min-h-[72px] w-full items-center gap-3.5 py-3 text-left disabled:opacity-55"
                      >
                        <span className="flex grow flex-col gap-0.5">
                          <span className="text-[16px] font-semibold text-ink">{t.title}</span>
                          <span className="text-support text-stone">{t.blurb}</span>
                        </span>
                        <span className="shrink-0 text-[13px] font-semibold text-stone">
                          {running.has(t.key) ? "Running" : `${t.days} days`}
                        </span>
                      </button>
                    </li>
                  ))}
                </ol>
              </section>
            ))}

            <Link
              href={routes.challengeNew}
              aria-disabled={full}
              tabIndex={full ? -1 : undefined}
              className={`press mt-6 flex min-h-[62px] items-center gap-3.5 border-y border-line text-ink no-underline ${full ? "pointer-events-none opacity-55" : ""}`}
            >
              <Icon name="plus" size={22} className="text-stone" />
              <span className="flex grow flex-col">
                <span className="text-[16px] font-semibold">Make our own</span>
                <span className="text-support text-stone">Write a line for each day.</span>
              </span>
              <Icon name="right" size={18} className="text-stone" />
            </Link>

            {onBack ? null : <PastChallenges />}
          </Main>
        )}
      </QueryState>
      <StartChallengeSheet template={picked} onClose={() => setPicked(null)} onStarted={onBack} />
    </>
  );
}
