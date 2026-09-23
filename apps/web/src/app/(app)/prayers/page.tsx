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
import { setterWord } from "@/features/prayers/derive";
import { usePendingCompletions, useWeek } from "@/features/prayers/hooks";
import { range } from "@/lib/dates";
import { routes } from "@/lib/routes";

export default function PrayersPage() {
  const week = useWeek();
  const couple = useCouple();
  const pending = usePendingCompletions();

  // Before a partner joins there is no week to wait for, and saying so is
  // kinder than letting the screen fail at them.
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
                    : "You’ll get a quiet nudge the moment the week is shared."
                }
                cta={
                  mine ? (
                    <LinkButton href={routes.prayersSet}>Set this week&rsquo;s prayers</LinkButton>
                  ) : (
                    <LinkButton href={routes.prayerMode({ quiet: true })} icon="moon">
                      Enter quiet prayer
                    </LinkButton>
                  )
                }
              />
            </Main>
          );
        }
        const done = w.my_completed.length;
        const total = w.points.length;
        return (
          <>
            <Main>
              <div className="pt-1.5">
                <Micro>
                  {range(w.week_start, w.week_end, true)} &middot; set by{" "}
                  {setterWord(w, me.id, partner)}
                </Micro>
              </div>
              <Title className="mt-2">This week&rsquo;s prayers</Title>
              <div className="mt-5 flex flex-col gap-2.5">
                <Segments total={total} done={done} />
                <Para size="support">
                  {done} of {total} prayed
                </Para>
              </div>
              <div className="mt-2 flex flex-col border-t border-line">
                {w.points.map((p, i) => (
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
            </Main>
            <BottomActions>
              <LinkButton href={routes.prayerMode()}>
                {done === total
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
