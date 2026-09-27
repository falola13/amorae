"use client";

import { Main } from "@/components/layout/screen";
import {
  BottomActions,
  EmptyState,
  LinkButton,
  Micro,
  Para,
  Segments,
  Skeleton,
  Title,
} from "@/components/ui/kit";
import { QueryState, inPage } from "@/components/ui/query-state";
import { PrayerRow } from "@/components/ui/prayer-row";
import { useCouple } from "@/features/couple/hooks";
import { DaysRow } from "@/features/prayers/components/days-row";
import { daysLabel, setterWord, todaysPoints } from "@/features/prayers/derive";
import { usePendingCompletions, useWeek } from "@/features/prayers/hooks";
import { range } from "@/lib/dates";
import { routes } from "@/lib/routes";
import Link from "next/link";

export default function PrayersPage() {
  const week = useWeek();
  const couple = useCouple();
  const pending = usePendingCompletions();

  // No week exists before a partner joins, so show that state explicitly rather than letting the query fail.
  if (couple.data && !couple.data.partner) {
    return (
      <Main>
        <div className="pt-1.5">
          <Title>This week&rsquo;s prayers</Title>
        </div>
        <EmptyState
          ghost="dots"
          title="Once your partner joins."
          text="Praying together takes two: one of you sets the week, the other responds, and it alternates from there."
          cta={<LinkButton href={routes.invite}>Invite your partner</LinkButton>}
        />
      </Main>
    );
  }

  return (
    <QueryState
      queries={[week, couple]}
      frame={inPage}
      loading={
        <Main>
          <div className="pt-4">
            <Skeleton lines={5} />
          </div>
        </Main>
      }
    >
      {(w, coupleData) => {
        const me = coupleData.me;
        const partner = coupleData.partner?.display_name ?? "your partner";
        if (w.status !== "published") {
          const mine = w.setter_id === me.id;
          return (
            <Main>
              <div className="pt-1.5">
                <Micro>{range(w.week_start, w.week_end, true)}</Micro>
              </div>
              <Title className="mt-2">This week&rsquo;s prayers</Title>
              <EmptyState
                ghost="dots"
                title={mine ? "Nothing shared yet." : `${partner} is still writing.`}
                text={
                  mine
                    ? "Write what’s on your heart and share it when it feels ready."
                    : `It’s ${partner}’s turn this week, but you can pitch in and write it with them.`
                }
                cta={
                  <LinkButton href={routes.prayersSet}>
                    {mine ? "Set this week’s prayers" : "Write or edit this week"}
                  </LinkButton>
                }
              />
            </Main>
          );
        }
        const todays = todaysPoints(w);
        // The rest of the week, so a Thursday prayer isn't invisible on Monday.
        const later = w.points.filter((p) => !todays.includes(p));
        const done = w.my_completed.length;
        const total = todays.length;
        return (
          <>
            <Main>
              <div className="pt-1.5">
                <Micro>
                  {range(w.week_start, w.week_end, true)} &middot; set by{" "}
                  {setterWord(w, me.id, partner)}
                </Micro>
              </div>
              <div className="mt-2 flex items-baseline justify-between gap-3">
                <Title>This week&rsquo;s prayers</Title>
                <Link
                  href={routes.prayersSet}
                  className="press shrink-0 text-support font-semibold text-plum no-underline"
                >
                  Edit the week
                </Link>
              </div>
              <div className="mt-5 flex flex-col gap-2.5">
                <Segments total={total} done={done} />
                <Para size="support">
                  {total === 0 ? "Nothing set for today" : `Today · ${done} of ${total} prayed`}
                </Para>
              </div>
              <div className="mt-4">
                <DaysRow days={w.days} today={w.today} partnerName={partner} />
              </div>
              {total === 0 ? (
                <Para className="mt-5 text-stone">Rest, or pray freely.</Para>
              ) : (
                <div className="mt-2 flex flex-col border-t border-line">
                  {todays.map((p, i) => (
                    <PrayerRow
                      key={p.id}
                      p={p}
                      index={i}
                      done={w.my_completed.includes(p.id)}
                      href={routes.prayer(p.id)}
                      note={pending.has(p.id) ? "Will sync when you’re back online" : undefined}
                    />
                  ))}
                </div>
              )}
              {later.length > 0 ? (
                <section className="mt-6">
                  <Micro>Other days this week</Micro>
                  <div className="mt-2 flex flex-col border-t border-line">
                    {later.map((p) => (
                      <Link
                        key={p.id}
                        href={routes.prayer(p.id)}
                        className="press flex min-h-[54px] items-center justify-between gap-3 border-b border-line py-2 text-ink no-underline"
                      >
                        <span className="text-[16px] font-medium text-stone">{p.title}</span>
                        <span className="shrink-0 text-[13px] font-semibold text-stone">
                          {daysLabel(p.weekdays)}
                        </span>
                      </Link>
                    ))}
                  </div>
                </section>
              ) : null}
            </Main>
            <BottomActions>
              <LinkButton href={routes.prayerMode()}>
                {total === 0
                  ? "Open prayer mode"
                  : done >= total
                    ? "Open prayer mode"
                    : done === 0
                      ? "Begin praying"
                      : "Continue praying"}
              </LinkButton>
            </BottomActions>
          </>
        );
      }}
    </QueryState>
  );
}
