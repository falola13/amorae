"use client";

import Link from "next/link";
import { useState } from "react";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import {
  Button,
  Initial,
  LinkButton,
  Micro,
  Para,
  Row,
  Section,
  Segments,
  Skeleton,
} from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { isAbsence } from "@/lib/api/envelope";
import { readableMessage } from "@/lib/query/client";
import { whenLabel, yearsWord } from "@/features/anniversary/derive";
import { useAnniversary } from "@/features/anniversary/hooks";
import { useCouple } from "@/features/couple/hooks";
import { useInbox } from "@/features/notifications/hooks";
import { setterWord, todaysPoints } from "@/features/prayers/derive";
import { useWeek } from "@/features/prayers/hooks";
import { usePrefs } from "@/features/settings/hooks";
import { AnniversaryHome } from "./anniversary-home";
import { HomeChallengeCard } from "./challenge-card";
import { eventPhase, nowLabel } from "@/features/together/events";
import { useEvents, useGoals, useMilestones, useNudge } from "@/features/together/hooks";
import {
  celebrationCopy,
  daysAwayLabel,
  todaysMilestones,
  upcomingMilestones,
} from "@/features/together/milestones";
import type { Anniversary, Milestone, PrayerWeek } from "@/lib/api/types";
import {
  activeGoal,
  isAlone,
  nextUpcomingEvent,
  plannedCount,
  reminderLabel,
  todayEvent,
  upcomingEvents,
} from "./derive";
import {
  greeting,
  iso,
  naira,
  partOfDay,
  range,
  relativeDay,
  time12,
  weekdayDate,
} from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";

export function HomeScreen() {
  const couple = useCouple();
  const week = useWeek();
  const events = useEvents();
  const goals = useGoals();
  // data ?? [] below would make a failed fetch look identical to an empty week, so track it separately.
  const eventsBroke = events.isError && !isAbsence(events.error);
  const goalsBroke = goals.isError && !isAbsence(goals.error);
  const prefs = usePrefs();
  const milestones = useMilestones();
  const inbox = useInbox();
  const anniversary = useAnniversary();
  const unread = inbox.data?.unread ?? 0;
  const now = today();
  const todayIso = iso(now);

  // Renders only once couple has loaded; other sources can fail independently.
  const coupleData = couple.data;
  const w = week.data;
  if (!coupleData || (!w && week.fetchStatus === "fetching")) {
    return (
      <Main>
        <div className="pt-4">
          <Skeleton lines={4} />
        </div>
      </Main>
    );
  }

  const me = coupleData.me;
  const partner = coupleData.partner?.display_name ?? "your partner";
  // Not a failure — no week exists until there's a partner; see isAlone.
  const alone = isAlone(Boolean(coupleData.partner), week.error);
  const upcoming = upcomingEvents(events.data ?? [], now);
  const todays = todayEvent(upcoming, todayIso, now);
  const nextEvent = nextUpcomingEvent(upcoming, todayIso, now);
  const goal = activeGoal(goals.data ?? []);
  const reminder = reminderLabel(prefs.data);

  const weekPrompt =
    w && w.status !== "published" ? <WeekPrompt week={w} meId={me.id} partner={partner} /> : null;

  // The anniversary takes Home over for the day — not a new colour on the
  // usual screen, a different one (anniversary-home.tsx).
  const theDay = anniversary.data?.days_until === 0 && !alone;
  if (theDay && anniversary.data) {
    return (
      <AnniversaryHome
        a={anniversary.data}
        couple={coupleData}
        todayIso={todayIso}
        todays={upcoming.filter((e) => e.date === todayIso)}
        nextEvent={nextEvent && nextEvent.date !== todayIso ? nextEvent : null}
        goal={goal}
        week={w}
        weekPrompt={weekPrompt}
        unread={unread}
      />
    );
  }

  // Counted only when every source answered; a failed one would undercount.
  const planned = w && events.data && goals.data ? plannedCount(upcoming, goal, w) : null;

  return (
    <Main>
      <div className="flex items-start justify-between pt-1.5">
        <div>
          <Micro>{weekdayDate(todayIso)}</Micro>
          {/* The masthead: a tab root has no back button, so the display title leads. */}
          <h1 className="m-0 mt-3 font-text text-display">
            <span className="font-light italic text-stone">{greeting(now)},</span>
            <br />
            {alone ? me.display_name : `${me.display_name} & ${partner}`}
          </h1>
        </div>
        <div className="-mr-2 flex items-center gap-1">
          <Link
            href={routes.inbox}
            aria-label={unread > 0 ? `Notifications, ${unread} unread` : "Notifications"}
            className="press relative flex h-11 w-11 items-center justify-center rounded-full text-ink"
          >
            <Icon name="bell" size={22} />
            {unread > 0 ? (
              <span
                aria-hidden="true"
                className="absolute right-2.5 top-2.5 h-2 w-2 rounded-full bg-plum"
              />
            ) : null}
          </Link>
          <Link
            href={routes.eventNew()}
            aria-label="Add something to our space"
            className="press flex h-11 w-11 items-center justify-center rounded-full text-ink"
          >
            <Icon name="plus" size={24} />
          </Link>
        </div>
      </div>

      <Celebrations
        milestones={milestones.data ?? []}
        meId={me.id}
        partner={partner}
        todayIso={todayIso}
        anniversary={anniversary.data}
      />
      {anniversary.data && !alone ? (
        <AnniversaryAhead a={anniversary.data} partner={partner} />
      ) : null}

      {weekPrompt}

      {alone ? (
        // The one raised leaf: the thing someone is waiting on.
        <div className="mt-8 flex flex-col items-start rounded-card border border-line bg-surface px-5 py-5">
          <Micro>Just you, so far</Micro>
          <h2 className="m-0 mt-2 font-text text-section">Amorae is for two.</h2>
          <Para className="mt-1.5">
            The week&rsquo;s prayers, the plans, the things you keep &mdash; none of it starts until{" "}
            {partner} is here too.
          </Para>
          <LinkButton href={routes.invite} className="mt-4">
            Invite them
          </LinkButton>
        </div>
      ) : null}

      {alone ? null : <HomeChallengeCard partner={partner} />}

      {/* Side by side from `lg`; a single column would leave the right half empty. */}
      <div className="flex flex-col lg:mt-8 lg:flex-row lg:items-start lg:gap-8">
        {todays ? (
          // Ruled above and below rather than boxed: there is only one leaf a screen.
          <Link
            href={routes.event(todays.id)}
            className="press mt-8 flex items-center gap-4 border-y border-line py-4 text-ink no-underline lg:mt-0 lg:flex-1"
          >
            <span className="flex grow flex-col gap-0.5">
              <Micro>{partOfDay(todays.start_time)}</Micro>
              <span className="mt-1 font-text text-section not-italic">{todays.title}</span>
              <span className="text-support text-stone">
                {eventPhase(todays, now) === "ongoing"
                  ? nowLabel(todays)
                  : time12(todays.start_time)}
                {(eventPhase(todays, now) === "ongoing" || todays.start_time) && todays.location
                  ? " · "
                  : ""}
                {todays.location ?? ""}
              </span>
            </span>
            <Icon name="right" size={18} className="text-stone" />
          </Link>
        ) : null}

        {/* Hidden entirely when alone with nothing planned — a heading with nothing under it reads as broken. */}
        {alone && !eventsBroke && !goalsBroke && !nextEvent && !goal ? null : (
          <Section
            label="This week"
            className="mt-8 lg:mt-0 lg:flex-1"
            trailing={
              planned !== null ? (
                <span className="lining text-support text-stone">
                  {planned} {planned === 1 ? "thing" : "things"} planned
                </span>
              ) : undefined
            }
          >
            {/* No week can exist without a partner (prayer alternates between two people), so skip it. */}
            {alone ? null : (
              <QueryState queries={[week]} loading={<Skeleton lines={1} />}>
                {(wk) => {
                  // Not shared yet: the prompt above says what to do; this row just says so.
                  // Content rows lose their leading icon: the words name the thing.
                  if (wk.status !== "published") {
                    return (
                      <Row
                        title="Weekly prayer"
                        sub={wk.status === "draft" ? "Not shared yet" : `${partner} is setting it`}
                        href={wk.status === "draft" ? routes.prayersSet : routes.prayers}
                      />
                    );
                  }
                  const total = todaysPoints(wk).length;
                  const done = wk.my_completed.length;
                  return (
                    <Row
                      title="Weekly prayer"
                      sub={
                        total === 0
                          ? `Set by ${setterWord(wk, me.id, partner)} · nothing set for today`
                          : `Set by ${setterWord(wk, me.id, partner)} · today ${done} of ${total} prayed`
                      }
                      href={routes.prayers}
                    >
                      {total > 0 ? (
                        <Segments
                          total={total}
                          done={done}
                          height={4}
                          className="mb-0.5 mt-1.5 w-24"
                        />
                      ) : null}
                    </Row>
                  );
                }}
              </QueryState>
            )}
            {eventsBroke ? (
              <Row
                icon="alert"
                title="Plans didn’t load"
                sub="Open events to try again"
                href={routes.events}
              />
            ) : nextEvent ? (
              <Row
                title={<span className="font-text text-bodylg font-normal">{nextEvent.title}</span>}
                sub={`${relativeDay(nextEvent.date, todayIso)}${nextEvent.start_time ? `, ${time12(nextEvent.start_time)}` : ""}`}
                href={routes.event(nextEvent.id)}
              />
            ) : null}
            {goalsBroke ? (
              <Row
                icon="alert"
                title="Goals didn’t load"
                sub="Open goals to try again"
                href={routes.goals}
              />
            ) : goal ? (
              <Row
                title={<span className="font-text text-bodylg font-normal">{goal.goal.title}</span>}
                sub={
                  goal.goal.unit === "naira"
                    ? `${naira(goal.total)} so far`
                    : `${goal.total} of ${goal.goal.target} ${goal.goal.unit_label ?? ""}`
                }
                href={routes.goal(goal.goal.id)}
              />
            ) : null}
          </Section>
        )}
      </div>

      {w && w.status === "published" ? (
        <PrayerFooter
          week={w}
          partner={partner}
          partnerPhoto={coupleData.partner?.photo_url}
          reminder={prefs.data ? reminder : null}
        />
      ) : null}
      {coupleData.partner ? <Nudge partner={partner} /> : null}
    </Main>
  );
}

/** What's today, and what's coming in the week ahead — a ruled block per date
 *  happening today (stacked if there's more than one), a countdown to the
 *  nearest one still to come, then a slim row for each after it. Nothing
 *  renders when there's nothing to say. */
function Celebrations({
  milestones,
  meId,
  partner,
  todayIso,
  anniversary,
}: {
  milestones: Milestone[];
  meId: string;
  partner: string;
  todayIso: string;
  anniversary?: Anniversary;
}) {
  const todays = todaysMilestones(milestones, todayIso);
  // The anniversary's own card (AnniversaryAhead) says it, with the letters.
  const soon = upcomingMilestones(milestones, todayIso).filter(
    (m) => !(anniversary && m.source === "anniversary" && aheadShown(anniversary)),
  );
  if (todays.length === 0 && soon.length === 0) return null;
  const [next, ...later] = soon;
  return (
    <div className="mt-6 flex flex-col gap-3">
      {todays.map((m) => {
        const { title, sub } = celebrationCopy(m, todayIso, meId, partner);
        if (m.source === "anniversary" && anniversary?.days_until === 0) {
          return (
            <Link
              key={m.id}
              href={routes.ourDay()}
              className="press relative flex items-center gap-5 overflow-hidden rounded-[26px] bg-keepsake px-6 py-6 text-[#fbeef2] no-underline shadow-[0_24px_50px_-30px_rgba(61,20,40,0.7)]"
            >
              <span className="lining font-serif text-[92px] font-semibold leading-[0.8] tracking-[-0.04em]">
                <span className="animate-shimmer text-foil">{anniversary.years}</span>
              </span>
              <span className="flex grow flex-col">
                <span className="text-[11px] font-bold uppercase tracking-[0.24em] text-[#e9c886]">
                  Happy anniversary
                </span>
                <span className="mt-1 font-serif text-[27px] font-medium italic leading-[1.1]">
                  {anniversary.years === 1 ? "year" : "years"} together today
                </span>
                <span className="mt-2 text-[14px] text-[#f0cbd8]">
                  {anniversary.theirs
                    ? `${partner.split(" ")[0]} wrote you a letter`
                    : "Your day is waiting"}{" "}
                  &rarr;
                </span>
              </span>
            </Link>
          );
        }
        // Today's date, set between rules in the serif rather than on a
        // gradient card: an everyday screen keeps everyday stock.
        return (
          <div key={m.id} className="flex flex-col items-start border-y border-line py-5">
            <h2 className="m-0 font-text text-title text-ink">{title}</h2>
            {sub ? <Para className="mt-1">{sub}</Para> : null}
          </div>
        );
      })}
      {next ? (
        // The nearest date as a countdown: the figure is the point, so it is
        // the largest thing here, in ink (plum would be decoration).
        <Link
          href={routes.milestones}
          className="press grid grid-cols-[auto_1fr] items-center gap-x-4 border-y border-line py-4 text-ink no-underline"
        >
          <span className="lining font-text text-[64px] font-light leading-[0.9] tracking-[-0.03em]">
            {next.days}
          </span>
          <span className="flex flex-col">
            <strong className="text-[16px] font-semibold leading-[1.35]">
              {next.days === 1 ? "day" : "days"} to {next.title}
            </strong>
            {next.sub ? <span className="text-support text-stone">{next.sub}</span> : null}
          </span>
        </Link>
      ) : null}
      {later.map((m) => (
        <Link
          key={m.id}
          href={routes.milestones}
          className="press -my-1 flex min-h-11 items-center py-1 text-ink no-underline"
        >
          <span className="text-support text-stone">
            {m.title} · {daysAwayLabel(m.days)}
          </span>
        </Link>
      ))}
    </div>
  );
}

/** The days before the anniversary when a letter can be written. */
const aheadShown = (a: Anniversary) => a.days_until > 0 && a.writable;

/** In the days before: the anniversary, and the letters — written, waiting, or not yet. */
function AnniversaryAhead({ a, partner }: { a: Anniversary; partner: string }) {
  if (!aheadShown(a)) return null;
  const first = partner.split(" ")[0];
  const line = a.theirs_sealed
    ? `${first} has written you something. It opens ${whenLabel(a.days_until)}.`
    : a.mine
      ? `Your letter is sealed until ${whenLabel(a.days_until) === "tomorrow" ? "tomorrow" : "the day"}.`
      : `Write ${first} a letter to open on the day.`;
  return (
    <Link
      href={routes.ourDay()}
      className="press mt-4 flex items-center gap-4 rounded-card bg-occasion px-5 py-4 text-ink no-underline"
    >
      <Icon name={a.theirs_sealed ? "letter" : "heart"} size={24} className="shrink-0 text-plum" />
      <span className="flex grow flex-col">
        <span className="text-[17px] font-semibold">
          {yearsWord(a.years)} {whenLabel(a.days_until)}
        </span>
        <span className="text-support text-stone">{line}</span>
      </span>
      <Icon name="right" size={18} className="text-stone" />
    </Link>
  );
}

/**
 * One tap, no message, nothing to reply to. It says what happened in place
 * rather than as a toast, including the refusals — that they are asleep, or
 * that today's three are spent — because those are the answer, not errors.
 */
function Nudge({ partner }: { partner: string }) {
  const nudge = useNudge();
  const [said, setSaid] = useState<string | null>(null);
  const first = partner.split(" ")[0];

  if (said) {
    return (
      <p className="m-0 pb-4 pt-1 text-support text-stone" role="status">
        {said}
      </p>
    );
  }
  return (
    <div className="pb-4 pt-1">
      <Button
        variant="text"
        icon="together"
        loading={nudge.isPending}
        onClick={() =>
          nudge.mutate(undefined, {
            onSuccess: ({ left }) =>
              setSaid(
                `${first} will know you were thinking of them. ${
                  left === 0 ? "That was today’s last." : `${left} more today.`
                }`,
              ),
            onError: (e) => setSaid(readableMessage(e)),
          })
        }
      >
        Thinking of {first}
      </Button>
    </div>
  );
}

/** The prayer call to action and the partner's progress, shown only once the week has loaded. */
function PrayerFooter({
  week: w,
  partner,
  partnerPhoto,
  reminder,
}: {
  week: PrayerWeek;
  partner: string;
  partnerPhoto?: string | null;
  reminder: string | null;
}) {
  const done = w.my_completed.length;
  const total = todaysPoints(w).length;
  const partnerToday = w.partner_completed.length > 0;
  return (
    <>
      <div className="mt-4">
        <LinkButton href={routes.prayerMode()}>
          {total === 0 || done >= total
            ? "Open prayer mode"
            : done === 0
              ? "Begin praying"
              : "Continue praying"}
        </LinkButton>
      </div>
      <div className="mt-3 flex items-center gap-2 pb-4 text-support text-stone">
        <Initial letter={partner[0]} photoUrl={partnerPhoto} name={partner} size={22} />
        {total === 0
          ? "Nothing set for today"
          : partnerToday
            ? `${partner} has prayed today`
            : `${partner} hasn’t prayed yet today`}
        {reminder ? <> &middot; reminder {reminder}</> : null}
      </div>
    </>
  );
}

/**
 * The week's prayers aren't shared yet. Once this was the whole of Home,
 * which hid everything else for days at a time; now it is the first card,
 * with the rest of the week underneath it.
 */
function WeekPrompt({
  week: w,
  meId,
  partner,
}: {
  week: PrayerWeek;
  meId: string;
  partner: string;
}) {
  const first = partner.split(" ")[0];
  const mine = w.setter_id === meId;
  const waiting = w.status === "waiting";
  return (
    // The screen's one raised leaf while it shows: the week is waiting on someone.
    <div className="mt-8 flex flex-col items-start rounded-card border border-line bg-surface px-5 py-5">
      <Micro className="lining">{range(w.week_start, w.week_end)}</Micro>
      <h2 className="m-0 mt-2 font-text text-section">
        {waiting
          ? `${first} is setting this week’s prayers`
          : mine
            ? "It’s your week to set the prayers"
            : `It’s ${first}’s turn this week`}
      </h2>
      <Para className="mt-1.5">
        {waiting
          ? "We’ll let you know the moment they’re shared."
          : mine
            ? `What would you like the two of you to pray about? ${first} will see them once you share.`
            : "You can still write it if you’d like to."}
      </Para>
      {waiting ? (
        <Link
          href={routes.prayerMode({ quiet: true })}
          className="press mt-3 inline-flex h-11 items-center gap-1.5 text-[15px] font-semibold text-plum no-underline"
        >
          <Icon name="moon" size={18} /> Enter quiet prayer
        </Link>
      ) : (
        <LinkButton
          href={routes.prayersSet}
          variant={mine ? "primary" : "secondary"}
          className="mt-4"
        >
          {mine ? "Set this week’s prayers" : "Write or edit this week"}
        </LinkButton>
      )}
    </div>
  );
}
