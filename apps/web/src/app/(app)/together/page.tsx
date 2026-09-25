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
import { useAnswered } from "@/features/prayers/hooks";
import { countdown, nextOccurrence } from "@/features/together/milestones";
import { iso, longDate, relativeDay, time12 } from "@/lib/dates";
import { isAbsence } from "@/lib/api/envelope";
import { routes } from "@/lib/routes";
import { Icon } from "@/components/icons";
import { today } from "@/lib/today";
import { Main } from "@/components/layout/screen";
import { Para, Row, Section, Title } from "@/components/ui/kit";

/**
 * What a row says under its title, and whether it says anything yet.
 *
 * A row here is the way into a screen; the line under it only summarises
 * what is behind it. So a summary that fails must not take the row with it —
 * which is exactly what happened when the events endpoint broke: the only
 * way through to Events and Calendar disappeared from the page whose whole
 * job is to be the way through.
 *
 * Three states, and the row stays where it is through all of them. Nothing
 * while it loads, because the row is a fixed height and nothing moves when
 * the line arrives. The summary once there is one. And a plain sentence if
 * it genuinely broke — short, because the screen the row leads to shows the
 * error properly and offers the retry, which is where somebody would go to
 * do anything about it anyway.
 *
 * An absence is not a break. A couple that has not formed yet gets a 404
 * from nearly all of this, and reading that as breakage would write "this
 * didn't load" across a page that loaded perfectly and is merely new. Those
 * fall through to the summary, which already knows how to say "nothing yet".
 */
function summary(
  query: { data: unknown; error: unknown; isError: boolean },
  line: string,
): string | undefined {
  if (query.data !== undefined) return line;
  if (!query.isError) return undefined;
  return isAbsence(query.error) ? line : "Couldn’t load just now";
}

export default function Together() {
  const couple = useCouple();
  const events = useEvents();
  const goals = useGoals();
  const challenge = useChallenge();
  const journal = useJournal();
  const memories = useMemories();
  const milestones = useMilestones();
  const answered = useAnswered();
  const appr = useAppreciations();
  const partner = couple.data?.partner?.display_name ?? "your partner";
  const todayIso = iso(today());

  // Every summary on this page, worked out from whatever has arrived. A
  // source that has not answered reads as empty here, and `summary` decides
  // whether that empty means "nothing yet" or "this broke".
  const upcoming = (events.data ?? [])
    .filter((e) => !e.done && e.date >= todayIso)
    .sort((a, b) => a.date.localeCompare(b.date));
  const nextEvent = upcoming[0];
  const activeGoals = (goals.data ?? []).filter((g) => !g.done).length;
  const challengeDay = challenge.data?.days.find((d) => !d.done && !d.skipped);
  const challengeLine = challenge.data
    ? challengeDay
      ? `${challenge.data.title}, day ${challengeDay.n}`
      : `${challenge.data.title}, finished`
    : "Something short, together";
  const lastWritten = (journal.data ?? [])[0];
  const lastAppreciated = (appr.data ?? [])[0];
  const lastMemory = (memories.data ?? [])[0];
  const soonestDate = (milestones.data ?? [])
    .filter((d) => d.reminder)
    .map((d) => ({ ...d, next: nextOccurrence(d.date, todayIso) }))
    .sort((a, b) => a.next.localeCompare(b.next))[0];
  const answeredCount = (answered.data ?? []).length;

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
          <Row
            icon="calendar"
            title="Calendar"
            sub={summary(
              events,
              nextEvent
                ? `${nextEvent.title} ${relativeDay(nextEvent.date, todayIso).toLowerCase()}${nextEvent.start_time ? ` at ${time12(nextEvent.start_time)}` : ""}`
                : "Nothing planned yet",
            )}
            href={routes.calendar}
          />
          <Row
            icon="pin"
            title="Events"
            sub={summary(
              events,
              upcoming.length ? `${upcoming.length} coming up` : "Add something you’d love to do",
            )}
            href={routes.events}
            last
          />
        </Section>
        <Section label="Grow" className="mt-[22px]">
          <Row
            icon="target"
            title="Goals"
            sub={summary(
              goals,
              activeGoals
                ? `${activeGoals} we’re working on`
                : "What would you like to build together?",
            )}
            href={routes.goals}
          />
          <Row
            icon="flag"
            title="Challenges"
            sub={summary(challenge, challengeLine)}
            href={routes.challenges}
            last
          />
        </Section>
        <Section label="Connect" className="mt-[22px]">
          <Row
            icon="pencil"
            title="Journal"
            sub={summary(
              journal,
              lastWritten
                ? `${lastWritten.author_id === couple.data?.me.id ? "You" : partner} wrote ${relativeDay(lastWritten.date, todayIso).toLowerCase()}`
                : "Write something for us",
            )}
            href={routes.journal}
          />
          <Row
            icon="note"
            title="Appreciation"
            sub={summary(
              appr,
              lastAppreciated
                ? `${lastAppreciated.from_id === couple.data?.me.id ? "You" : partner}, ${relativeDay(lastAppreciated.date, todayIso).toLowerCase()}`
                : "Something I appreciate about you",
            )}
            href={routes.appreciation}
            last
          />
        </Section>
        <Section label="Remember" className="mb-4 mt-[22px]">
          <Row
            icon="image"
            title="Memories"
            sub={summary(
              memories,
              lastMemory ? `Last saved ${longDate(lastMemory.date)}` : "Your story starts here",
            )}
            href={routes.memories}
          />
          <Row
            icon="bookmark"
            title="Milestones"
            sub={summary(
              milestones,
              soonestDate
                ? `${soonestDate.title} ${countdown(soonestDate.next, today()).toLowerCase()}`
                : "The dates that matter to us",
            )}
            href={routes.milestones}
          />
          <Row
            icon="check"
            title="Answered prayers"
            sub={summary(
              answered,
              answeredCount ? `${answeredCount} so far` : "What you’ve prayed for, and seen happen",
            )}
            href={routes.prayersAnswered}
            last
          />
        </Section>
      </div>
    </Main>
  );
}
