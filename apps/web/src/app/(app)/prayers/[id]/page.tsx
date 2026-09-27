"use client";

import Link from "next/link";
import { useParams } from "next/navigation";

import { Main } from "@/components/layout/screen";
import {
  BottomActions,
  Button,
  LinkButton,
  Para,
  Skeleton,
  Title,
  TopBar,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { Scripture } from "@/components/ui/scripture";
import { useCouple } from "@/features/couple/hooks";
import { AnsweredBlock } from "@/features/prayers/components/answered-block";
import { daysLabel, isForDay, todaysPoints, weekdayOf } from "@/features/prayers/derive";
import { useSetCompleted, useWeek } from "@/features/prayers/hooks";
import { routes } from "@/lib/routes";

export default function PrayerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const week = useWeek();
  const couple = useCouple();
  const complete = useSetCompleted();
  const loading = (
    <Main>
      <div className="pt-4">
        <Skeleton />
      </div>
    </Main>
  );

  return (
    <QueryState queries={[week]} frame={inPage} loading={loading}>
      {(w) => {
        const p = w.points.find((x) => x.id === id);
        if (!p) return loading;
        const idx = w.points.findIndex((x) => x.id === id);
        const done = w.my_completed.includes(p.id);
        const forToday = w.today !== undefined && isForDay(p, weekdayOf(w.today));
        const todayIdx = forToday ? todaysPoints(w).findIndex((x) => x.id === id) : -1;
        const setter =
          w.setter_id === couple.data?.me.id
            ? "you"
            : (couple.data?.partner?.display_name ?? "your partner");
        return (
          <>
            <TopBar
              back="Prayers"
              backHref={routes.prayers}
              right={
                <div className="flex items-center gap-3 pr-3">
                  <span className="tabular text-support font-semibold text-stone">
                    {idx + 1} of {w.points.length}
                  </span>
                  <Link
                    href={routes.prayerEdit(p.id)}
                    className="press text-support font-semibold text-plum no-underline"
                  >
                    Edit
                  </Link>
                </div>
              }
            />
            <Main className="gap-5 pt-5">
              <div className="flex flex-col gap-2.5">
                <Title size="lg">{p.title}</Title>
                <div className="text-support text-stone">
                  Added by {setter} &middot; {daysLabel(p.weekdays)}
                </div>
              </div>
              {p.text ? <p className="m-0 text-[19px] leading-[1.6]">{p.text}</p> : null}
              {p.scripture ? <Scripture reference={p.scripture} verse={p.verse} /> : null}
              <AnsweredBlock p={p} />
            </Main>
            <BottomActions>
              {!forToday ? (
                <Para className="text-stone">
                  Not set for today &mdash; {daysLabel(p.weekdays)}.
                </Para>
              ) : done ? (
                <Button
                  variant="secondary"
                  icon="check"
                  onClick={() => complete.mutate({ pointId: p.id, done: false })}
                >
                  Prayed &middot; tap to undo
                </Button>
              ) : (
                <Button icon="check" onClick={() => complete.mutate({ pointId: p.id, done: true })}>
                  I&rsquo;ve prayed
                </Button>
              )}
              {forToday ? (
                <LinkButton href={routes.prayerMode({ at: todayIdx })} variant="text" icon="moon">
                  Open in prayer mode
                </LinkButton>
              ) : null}
            </BottomActions>
          </>
        );
      }}
    </QueryState>
  );
}
