"use client";

import Link from "next/link";
import { useState } from "react";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import { Micro, Para, Skeleton, Title } from "@/components/ui/kit";
import { useChallengeTemplates, usePastChallenges } from "@/features/together/hooks";
import type { ChallengeTemplate } from "@/lib/api/types";
import { longDate } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";
import { currentSeason, groupTemplates, overLabel } from "../challenges";
import { StartChallengeSheet } from "./start-challenge-sheet";

/** Nothing is running: pick one, make your own, or look back at the ones you have done. */
export function ChallengeLibrary() {
  const templates = useChallengeTemplates();
  const past = usePastChallenges();
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
            <div className="pt-2">
              <Title>A challenge together</Title>
              <Para className="mt-1.5">
                A few days of small things, one a day. You each mark your own, and either of you can
                skip a day without it counting against anything.
              </Para>
            </div>

            {groupTemplates(all, season).map((g) => (
              <section key={g.category} className="mt-6 flex flex-col">
                <Micro>{g.label}</Micro>
                <ol className="m-0 mt-1 list-none border-t border-line p-0">
                  {g.templates.map((t) => (
                    <li key={t.key} className="border-b border-line">
                      <button
                        type="button"
                        onClick={() => setPicked(t)}
                        className="press flex min-h-[72px] w-full items-center gap-3.5 py-3 text-left"
                      >
                        <span className="flex grow flex-col gap-0.5">
                          <span className="text-[16px] font-semibold text-ink">{t.title}</span>
                          <span className="text-support text-stone">{t.blurb}</span>
                        </span>
                        <span className="shrink-0 text-[13px] font-semibold text-stone">
                          {t.days} days
                        </span>
                      </button>
                    </li>
                  ))}
                </ol>
              </section>
            ))}

            <Link
              href={routes.challengeNew}
              className="press mt-6 flex min-h-[62px] items-center gap-3.5 border-y border-line text-ink no-underline"
            >
              <Icon name="plus" size={22} className="text-stone" />
              <span className="flex grow flex-col">
                <span className="text-[16px] font-semibold">Make our own</span>
                <span className="text-support text-stone">Write a line for each day.</span>
              </span>
              <Icon name="right" size={18} className="text-stone" />
            </Link>

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
          </Main>
        )}
      </QueryState>
      <StartChallengeSheet template={picked} onClose={() => setPicked(null)} />
    </>
  );
}
