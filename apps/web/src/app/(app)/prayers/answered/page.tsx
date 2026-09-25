"use client";

import { Main } from "@/components/layout/screen";
import {
  EmptyState,
  LinkButton,
  Micro,
  Ornament,
  Para,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { useAnswered } from "@/features/prayers/hooks";
import type { AnsweredPrayer } from "@/lib/api/types";
import { longDate, monthName } from "@/lib/dates";
import { routes } from "@/lib/routes";

// Deliberately no count or filter of unanswered prayers.
function Answered({ a }: { a: AnsweredPrayer }) {
  return (
    <article className="flex flex-col gap-1 border-b border-line py-[18px]">
      <div className="text-bodylg font-semibold tracking-[-0.01em]">{a.title}</div>
      {a.answer_note ? (
        <p className="m-0 mt-0.5 text-[15px] leading-[1.55] text-stone" data-selectable>
          {a.answer_note}
        </p>
      ) : null}
      {/* Both dates shown: the gap between them is the point. */}
      <div className="mt-0.5 text-support text-stone">
        Prayed {longDate(a.week_start)} {a.week_start.slice(0, 4)}
        {a.answered_on ? ` · answered ${longDate(a.answered_on)}` : ""}
      </div>
    </article>
  );
}

const keptLine = (n: number) =>
  n === 1 ? "One prayer answered so far." : `${n} prayers answered so far.`;

export default function AnsweredPage() {
  const answered = useAnswered();
  return (
    <>
      <TopBar back="Prayers" backHref={routes.prayers} />
      <Main>
        <div className="pt-2">
          <Title>Answered</Title>
        </div>
        <QueryState queries={[answered]} loading={<Skeleton lines={4} />}>
          {(items) => {
            if (items.length === 0) {
              return (
                <EmptyState
                  ghost="dots"
                  title="Nothing marked yet."
                  text="When something you prayed for happens, mark it — from the prayer itself, however long afterwards. It will be kept here."
                  cta={<LinkButton href={routes.prayers}>Go to this week’s prayers</LinkButton>}
                />
              );
            }
            const groups: { month: string; items: AnsweredPrayer[] }[] = [];
            for (const a of items) {
              const on = a.answered_on ?? a.answered_at!.slice(0, 10);
              const m = `${monthName(on)} ${on.slice(0, 4)}`;
              const g = groups[groups.length - 1];
              if (g && g.month === m) g.items.push(a);
              else groups.push({ month: m, items: [a] });
            }
            return (
              <>
                <Para className="mt-1.5">
                  What the two of you have prayed for, and seen happen.
                </Para>
                <div className="mt-2 flex flex-col pb-4">
                  {groups.map((g) => (
                    <div key={g.month}>
                      <div className="flex items-center gap-3 pb-1 pt-[18px]">
                        <Micro>{g.month}</Micro>
                        <span className="h-px grow bg-line" />
                      </div>
                      {g.items.map((a) => (
                        <Answered key={a.id} a={a} />
                      ))}
                    </div>
                  ))}
                  <Ornament caption={keptLine(items.length)} />
                </div>
              </>
            );
          }}
        </QueryState>
      </Main>
    </>
  );
}
