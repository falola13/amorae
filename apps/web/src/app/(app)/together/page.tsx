"use client";

import Link from "next/link";
import { useCouple } from "@/features/couple/hooks";
import {
  useAppreciations,
  useActiveChallenges,
  useEvents,
  useMilestones,
} from "@/features/together/hooks";
import { useTimeline } from "@/features/timeline/hooks";
import { useAnswered } from "@/features/prayers/hooks";
import { challengesLine } from "@/features/together/challenges";
import { eventPhase } from "@/features/together/events";
import type { Event } from "@/lib/api/types";
import { countdown, nextOccurrence } from "@/features/together/milestones";
import { iso, relativeDay, time12 } from "@/lib/dates";
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
  const challenge = useActiveChallenges();
  const story = useTimeline("all");
  const milestones = useMilestones();
  const answered = useAnswered();
  const appr = useAppreciations();
  const partner = couple.data?.partner?.display_name ?? "your partner";
  const todayIso = iso(today());

  const now = today();
  const phases = (events.data ?? []).map((e) => ({ e, phase: eventPhase(e, now) }));
  const byStart = (a: Event, b: Event) =>
    (a.date + (a.start_time ?? "")).localeCompare(b.date + (b.start_time ?? ""));
  const ongoingCount = phases.filter((x) => x.phase === "ongoing").length;
  const upcoming = phases
    .filter((x) => x.phase === "upcoming")
    .map((x) => x.e)
    .sort(byStart);
  const nextEvent = phases
    .filter((x) => x.phase !== "over")
    .map((x) => x.e)
    .sort(byStart)[0];
  const eventsLine = [
    ongoingCount ? `${ongoingCount} happening now` : "",
    upcoming.length ? `${upcoming.length} coming up` : "",
  ]
    .filter(Boolean)
    .join(" · ");
  // The shared calendar day, not "my first unanswered one" — the two of you
  // are on the same day now.
  const challengeLine = challengesLine(challenge.data ?? []);
  const lastAppreciated = (appr.data ?? [])[0];
  const latestStory = story.data?.pages[0]?.items[0];
  const soonestDate = (milestones.data ?? [])
    .filter((d) => d.reminder)
    .map((d) => ({ ...d, next: nextOccurrence(d.date, todayIso) }))
    .sort((a, b) => a.next.localeCompare(b.next))[0];
  const answeredCount = (answered.data ?? []).length;

  return (
    <Main>
      <div className="flex items-start justify-between pt-1.5">
        <div>
          <Title size="display">Our space</Title>
          <Para size="support" className="mt-2">
            Everything we&rsquo;re building together.
          </Para>
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
        <Section label="Dates & plans" className="mt-8">
          <Row
            title="Calendar"
            sub={summary(
              events,
              nextEvent
                ? `${nextEvent.title} ${relativeDay(nextEvent.date, todayIso).replace(/^(Today|Tomorrow)$/, (w) => w.toLowerCase())}${nextEvent.start_time ? ` at ${time12(nextEvent.start_time)}` : ""}`
                : "Nothing planned yet",
            )}
            href={routes.calendar}
          />
          <Row
            title="Events"
            sub={summary(events, eventsLine || "Add something you’d love to do")}
            href={routes.events}
          />
          <Row
            title="Important dates"
            sub={summary(
              milestones,
              soonestDate
                ? `${soonestDate.title} ${countdown(soonestDate.next, today()).toLowerCase()}`
                : "The dates that matter to us",
            )}
            href={routes.milestones}
            last
          />
        </Section>
        {/* Our story is read, not used, so it opens like a chapter: an ink rule,
            the title in the serif and the latest entry, rather than one more row. */}
        <Link
          href={routes.story}
          className="press mt-8 block border-b border-t-[1.5px] border-b-line border-t-ink pb-[22px] pt-5 text-ink no-underline"
        >
          <h2 className="m-0 font-text text-title">Our story</h2>
          <p className="m-0 mt-2 font-text text-[17px] italic text-stone">
            {summary(
              story,
              latestStory
                ? `Latest: ${latestStory.title}`
                : "Memories, journal and goals, all in one place",
            )}
          </p>
          <span className="-mb-3 inline-flex h-11 items-center gap-1 text-[15px] font-semibold text-plum">
            Read our story
            <Icon name="right" size={18} />
          </span>
        </Link>
        <Section label="Grow" className="mt-8">
          <Row
            title="Challenges"
            sub={summary(challenge, challengeLine)}
            href={routes.challenges}
            last
          />
        </Section>
        <Section label="Connect" className="mb-6 mt-8">
          <Row
            title="Appreciation"
            sub={summary(
              appr,
              lastAppreciated
                ? `${lastAppreciated.from_id === couple.data?.me.id ? "You" : partner}, ${relativeDay(lastAppreciated.date, todayIso).replace(/^(Today|Tomorrow)$/, (w) => w.toLowerCase())}`
                : "Something I appreciate about you",
            )}
            href={routes.appreciation}
          />
          <Row
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
