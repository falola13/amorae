"use client";

import Link from "next/link";
import { useCouple } from "@/features/couple/hooks";
import {
  useAppreciations,
  useChallenge,
  useEvents,
  useGoals,
  useJournal,
  useMemories,
  useMilestones,
} from "@/features/together/hooks";
import { countdown, nextOccurrence } from "@/features/together/milestones";
import { iso, longDate, relativeDay, time12 } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { Icon } from "@/components/icons";
import { today } from "@/lib/today";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import { Para, Row, Section, Skeleton, Title } from "@/components/ui/kit";

export default function Together() {
  const couple = useCouple();
  const events = useEvents();
  const goals = useGoals();
  const challenge = useChallenge();
  const journal = useJournal();
  const memories = useMemories();
  const milestones = useMilestones();
  const appr = useAppreciations();
  const partner = couple.data?.partner?.display_name ?? "your partner";
  const todayIso = iso(today());
  return (
    <Main>
      <div className="flex items-start justify-between pt-1.5">
        <div>
          <Title>Our space</Title>
          <Para className="mt-1.5">Everything we&rsquo;re building together.</Para>
        </div>
        <Link
          href={routes.eventNew()}
          aria-label="Add an event"
          className="press -mr-2.5 -mt-1 flex h-11 w-11 items-center justify-center text-ink"
        >
          <Icon name="plus" size={24} />
        </Link>
      </div>
      {/* Four independent sections: stacked on a phone, two columns from `lg`,
          where the width is there and a single column would leave half the
          screen empty. */}
      <div className="flex flex-col lg:grid lg:grid-cols-2 lg:gap-x-8">
        <Section label="Plan" className="mt-[22px]">
          <QueryState queries={[events]} loading={<Skeleton lines={2} />}>
            {(eventsData) => {
              const up = eventsData
                .filter((e) => !e.done && e.date >= todayIso)
                .sort((a, b) => a.date.localeCompare(b.date));
              const next = up[0];
              return (
                <>
                  <Row
                    icon="calendar"
                    title="Calendar"
                    sub={
                      next
                        ? `${next.title} ${relativeDay(next.date, todayIso).toLowerCase()}${next.start_time ? ` at ${time12(next.start_time)}` : ""}`
                        : "Nothing planned yet"
                    }
                    href={routes.calendar}
                  />
                  <Row
                    icon="pin"
                    title="Events"
                    sub={up.length ? `${up.length} coming up` : "Add something you’d love to do"}
                    href={routes.events}
                    last
                  />
                </>
              );
            }}
          </QueryState>
        </Section>
        <Section label="Grow" className="mt-[22px]">
          <QueryState queries={[goals, challenge]} loading={<Skeleton lines={2} />}>
            {(goalsData, challengeData) => {
              const active = goalsData.filter((g) => !g.done).length;
              const day = challengeData.days.find((d) => !d.done && !d.skipped);
              return (
                <>
                  <Row
                    icon="target"
                    title="Goals"
                    sub={
                      active
                        ? `${active} we’re working on`
                        : "What would you like to build together?"
                    }
                    href={routes.goals}
                  />
                  <Row
                    icon="flag"
                    title="Challenges"
                    sub={
                      day
                        ? `${challengeData.title}, day ${day.n}`
                        : `${challengeData.title}, finished`
                    }
                    href={routes.challenges}
                    last
                  />
                </>
              );
            }}
          </QueryState>
        </Section>
        <Section label="Connect" className="mt-[22px]">
          <QueryState queries={[journal, appr]} loading={<Skeleton lines={2} />}>
            {(journalData, apprData) => {
              const j = journalData[0];
              const lastAppr = apprData[0];
              return (
                <>
                  <Row
                    icon="pencil"
                    title="Journal"
                    sub={
                      j
                        ? `${j.author_id === couple.data?.me.id ? "You" : partner} wrote ${relativeDay(j.date, todayIso).toLowerCase()}`
                        : "Write something for us"
                    }
                    href={routes.journal}
                  />
                  <Row
                    icon="note"
                    title="Appreciation"
                    sub={
                      lastAppr
                        ? `${lastAppr.from_id === couple.data?.me.id ? "You" : partner}, ${relativeDay(lastAppr.date, todayIso).toLowerCase()}`
                        : "Something I appreciate about you"
                    }
                    href={routes.appreciation}
                    last
                  />
                </>
              );
            }}
          </QueryState>
        </Section>
        {/* One QueryState each, as this component asks for: these two rows
            need nothing from each other, and sharing one meant the module
            that is not built yet hid the one that is. */}
        <Section label="Remember" className="mb-4 mt-[22px]">
          <QueryState queries={[memories]} loading={<Skeleton lines={1} />}>
            {(memoriesData) => {
              const m = memoriesData[0];
              return (
                <Row
                  icon="image"
                  title="Memories"
                  sub={m ? `Last saved ${longDate(m.date)}` : "Your story starts here"}
                  href={routes.memories}
                />
              );
            }}
          </QueryState>
          <QueryState queries={[milestones]} loading={<Skeleton lines={1} />}>
            {(milestonesData) => {
              const soonest = milestonesData
                .filter((d) => d.reminder)
                .map((d) => ({ ...d, next: nextOccurrence(d.date, todayIso) }))
                .sort((a, b) => a.next.localeCompare(b.next))[0];
              return (
                <Row
                  icon="bookmark"
                  title="Milestones"
                  sub={
                    soonest
                      ? `${soonest.title} ${countdown(soonest.next, today()).toLowerCase()}`
                      : "The dates that matter to us"
                  }
                  href={routes.milestones}
                  last
                />
              );
            }}
          </QueryState>
        </Section>
      </div>
    </Main>
  );
}
