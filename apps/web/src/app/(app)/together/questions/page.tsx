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

/** Facing pages: yours on the left, theirs on the right, a hairline between. */
function Day({ day, partner, last }: { day: QuestionEntry; partner: string; last: boolean }) {
  const right = day.theirs
    ? null
    : day.mine
      ? `Sealed — ${partner} didn’t answer that day`
      : day.theirs_answered
        ? `Sealed — ${partner} answered, and yours wasn’t written`
        : `${partner} didn’t answer that day`;
  return (
    <article className={`flex flex-col gap-3 py-5 ${last ? "" : "border-b border-line"}`}>
      <span className="text-micro uppercase text-stone">{weekdayDate(day.asked_on)}</span>
      <h2 className="m-0 font-text text-bodylg font-medium">{day.question}</h2>
      <div className="grid grid-cols-2">
        <Page who="You" text={day.mine?.text} empty="You didn’t answer that day" />
        <Page who={partner} text={day.theirs?.text} empty={right} divided />
      </div>
    </article>
  );
}

function Page({
  who,
  text,
  empty,
  divided,
}: {
  who: string;
  text?: string;
  empty?: string | null;
  divided?: boolean;
}) {
  return (
    <div
      className={`flex min-w-0 flex-col gap-1 ${divided ? "border-l border-line pl-4" : "pr-4"}`}
    >
      <span className="text-micro font-semibold uppercase text-stone">{who}</span>
      {text ? (
        <p
          className="m-0 whitespace-pre-wrap break-words font-text text-[15px] leading-[1.55] text-ink md:text-bodylg"
          data-selectable
        >
          {text}
        </p>
      ) : (
        <p className="m-0 text-support italic text-stone">{empty}</p>
      )}
    </div>
  );
}
