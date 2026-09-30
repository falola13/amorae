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

// A 404 from an unpaired couple is treated as "nothing yet", not an error.
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

  const upcoming = (events.data ?? [])
    .filter((e) => !e.done && e.date >= todayIso)
    .sort((a, b) => a.date.localeCompare(b.date));
  const nextEvent = upcoming[0];
  const activeGoals = (goals.data ?? []).filter((g) => !g.done).length;
  // The shared calendar day, not "my first unanswered one" — the two of you
  // are on the same day now.
  const challengeLine = challenge.data
    ? challenge.data.status === "active"
      ? `${challenge.data.title}, day ${challenge.data.today_n} of ${challenge.data.days.length}`
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
