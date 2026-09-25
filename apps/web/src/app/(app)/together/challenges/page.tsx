"use client";

import { useState } from "react";
import { useCouple } from "@/features/couple/hooks";
import {
  useChallenge,
  useChallengeDay,
  useChallengeTemplates,
  useLeaveChallenge,
  useStartChallenge,
} from "@/features/together/hooks";
import { isApiError } from "@/lib/api/errors";
import { routes } from "@/lib/routes";
import { Main } from "@/components/layout/screen";
import { QueryState, inPage } from "@/components/ui/query-state";
import {
  BottomActions,
  Button,
  Micro,
  Para,
  Sheet,
  Skeleton,
  StatusMark,
  Title,
  TopBar,
  cx,
} from "@/components/ui/kit";

export default function Challenges() {
  const ch = useChallenge();
  const set = useChallengeDay();
  const leave = useLeaveChallenge();
  // There was no way out of a challenge once started, nor on to another once finished.
  const [ending, setEnding] = useState(false);
  const couple = useCouple();
  const partner = couple.data?.partner?.display_name ?? "They";

  // No active challenge is a 404, not an error state — offer to start one.
  if (isApiError(ch.error) && ch.error.code === "challenge_not_found") {
    return <StartAChallenge />;
  }
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <QueryState
        queries={[ch]}
        frame={inPage}
        loading={
          <Main>
            <Skeleton />
          </Main>
        }
      >
        {(c) => {
          // Each partner's "today" is their own first unanswered day, independent of the other's.
          const todayDay = c.days.find((d) => !d.done && !d.skipped);
          const finished = !todayDay;
          return (
            <>
              <Main>
                <div className="pt-2">
                  <Micro>{finished ? "Finished" : `Day ${todayDay.n} of ${c.days.length}`}</Micro>
                </div>
                <Title className="mt-2">{c.title}</Title>
                <Para className="mt-1.5">One small thing a day. Skip any day you need to.</Para>
                <ol className="m-0 mt-4 list-none border-t border-line p-0">
                  {c.days.map((d) => {
                    const isToday = todayDay?.n === d.n;
                    return (
                      <li
                        key={d.n}
                        className="flex min-h-[52px] items-center gap-3.5 border-b border-line"
                      >
                        {d.done ? (
                          <StatusMark done label="Done" />
                        ) : isToday ? (
                          <span
                            aria-hidden="true"
                            className="box-border h-[22px] w-[22px] shrink-0 rounded-full border-2 border-plum"
                          />
                        ) : (
                          <span
                            aria-hidden="true"
                            className="box-border h-[22px] w-[22px] shrink-0 rounded-full border-[1.5px] border-edge"
                          />
                        )}
                        <span className="flex grow flex-col">
                          <span className="text-[12px] font-semibold text-stone">
                            Day {d.n}
                            {d.skipped ? " · skipped" : ""}
                          </span>
                          <span
                            className={cx(
                              "text-[16px]",
                              isToday ? "font-semibold text-ink" : "font-medium text-stone",
                            )}
                          >
                            {d.text}
                          </span>
                        </span>
                        {/* Label only, never a control — you can see the partner's status but not change it (DEC-30). */}
                        {d.partner_done ? (
                          <span className="text-[12px] font-semibold text-stone">
                            {partner} did
                          </span>
                        ) : d.partner_skipped ? (
                          <span className="text-[12px] text-stone">{partner} skipped</span>
                        ) : null}
                        {isToday ? (
                          <span className="text-[12px] font-bold text-plum">Today</span>
                        ) : null}
                      </li>
                    );
                  })}
                </ol>
              </Main>
              {!finished ? (
                <BottomActions>
                  <Button
                    icon="check"
                    onClick={() => set.mutate({ n: todayDay.n, patch: { done: true } })}
                  >
                    I did today&rsquo;s
                  </Button>
                  <Button
                    variant="text"
                    onClick={() => set.mutate({ n: todayDay.n, patch: { skipped: true } })}
                  >
                    Skip today
                  </Button>
                  <Button variant="text" className="text-stone" onClick={() => setEnding(true)}>
                    End this challenge
                  </Button>
                </BottomActions>
              ) : (
                <BottomActions>
                  <Para size="support" className="text-center">
                    {c.days.length} days, done together. Nicely.
                  </Para>
                  <Button
                    variant="secondary"
                    loading={leave.isPending}
                    onClick={() => leave.mutate(undefined)}
                  >
                    Choose another
                  </Button>
                </BottomActions>
              )}
              <Sheet
                open={ending}
                onClose={() => setEnding(false)}
                title="End this challenge?"
                labelledBy="end-ch-h"
              >
                <Para>
                  It ends for both of you, and the days you&rsquo;ve each marked go with it. You can
                  start another straight after.
                </Para>
                <div className="flex flex-col gap-1">
                  <Button
                    variant="secondary"
                    className="border-red text-red"
                    loading={leave.isPending}
                    onClick={() =>
                      leave.mutate(undefined, {
                        onSuccess: () => setEnding(false),
                        onQueued: () => setEnding(false),
                      })
                    }
                  >
                    End it
                  </Button>
                  <Button variant="text" onClick={() => setEnding(false)}>
                    Keep going
                  </Button>
                </div>
              </Sheet>
            </>
          );
        }}
      </QueryState>
    </>
  );
}

function StartAChallenge() {
  const templates = useChallengeTemplates();
  const start = useStartChallenge();

  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
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
            <ol className="m-0 mt-5 list-none border-t border-line p-0">
              {all.map((t) => (
                <li key={t.key} className="border-b border-line">
                  <button
                    type="button"
                    disabled={start.isPending}
                    onClick={() => start.mutate(t.key)}
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
          </Main>
        )}
      </QueryState>
    </>
  );
}
