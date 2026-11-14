"use client";

import { Main } from "@/components/layout/screen";
import { Button, EmptyState, Para, Skeleton, Title, TopBar } from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { useCouple } from "@/features/couple/hooks";
import { usePastQuestions } from "@/features/questions/hooks";
import type { QuestionEntry } from "@/lib/api/types";
import { weekdayDate } from "@/lib/dates";
import { routes } from "@/lib/routes";

/** Every earlier day's question, with both answers where you wrote yours. */
export default function PastQuestions() {
  const past = usePastQuestions();
  const partner = (useCouple().data?.partner?.display_name ?? "your partner").split(" ")[0];

  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="pt-2">
          <Title>Past questions</Title>
        </div>
        <QueryState queries={[past]} loading={<Skeleton />}>
          {(data) => {
            const days = data.pages.flatMap((p) => p.items);
            if (days.length === 0) {
              return (
                <EmptyState
                  ghost="lines"
                  title="Nothing here yet."
                  text="Each day’s question lands here once the day is over."
                  cta={null}
                />
              );
            }
            return (
              <div className="flex flex-col pb-6">
                <Para className="mt-1.5">
                  A day’s answers open once you have both written yours that day. A day you skipped
                  stays sealed.
                </Para>
                {days.map((d, i) => (
                  <Day key={d.asked_on} day={d} partner={partner} last={i === days.length - 1} />
                ))}
                {past.hasNextPage ? (
                  <Button
                    variant="secondary"
                    className="mt-4"
                    loading={past.isFetchingNextPage}
                    onClick={() => past.fetchNextPage()}
                  >
                    Earlier
                  </Button>
                ) : null}
              </div>
            );
          }}
        </QueryState>
      </Main>
    </>
  );
}

function Day({ day, partner, last }: { day: QuestionEntry; partner: string; last: boolean }) {
  return (
    <article className={`flex flex-col gap-2 py-5 ${last ? "" : "border-b border-line"}`}>
      <span className="text-micro uppercase text-stone">{weekdayDate(day.asked_on)}</span>
      <h2 className="m-0 font-text text-bodylg font-medium">{day.question}</h2>
      {day.mine ? (
        <Line who="You" text={day.mine.text} />
      ) : (
        <Para size="support">You didn’t answer this one.</Para>
      )}
      {day.theirs ? (
        <Line who={partner} text={day.theirs.text} />
      ) : day.theirs_answered ? (
        <Para size="support">
          {partner} answered. It stays sealed, because yours wasn’t written.
        </Para>
      ) : (
        <Para size="support">{partner} didn’t answer this one.</Para>
      )}
    </article>
  );
}

function Line({ who, text }: { who: string; text: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-support font-semibold text-stone">{who}</span>
      <p className="m-0 font-text text-bodylg text-ink" data-selectable>
        {text}
      </p>
    </div>
  );
}
