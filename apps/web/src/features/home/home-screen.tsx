"use client";

import Link from "next/link";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import {
  Initial,
  LinkButton,
  Micro,
  Para,
  Row,
  Section,
  Segments,
  Skeleton,
  Title,
} from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { isAbsence } from "@/lib/api/envelope";
import { useCouple } from "@/features/couple/hooks";
import { setterWord } from "@/features/prayers/derive";
import { useHistory, useWeek } from "@/features/prayers/hooks";
import { usePrefs } from "@/features/settings/hooks";
import { useEvents, useGoals } from "@/features/together/hooks";
import type { PrayerWeek } from "@/lib/api/types";
import {
  activeGoal,
  isAlone,
  lastWeekSummary,
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
  // A source that broke, as opposed to one with nothing in it. Worth telling
  // apart here more than anywhere: this screen reads `data ?? []`, so a
  // failed list used to look exactly like an empty week — the plan and the
  // next event simply were not there, and nothing said why. Quietly
  // reporting a smaller life than somebody actually has is worse than an
  // error, because there is nothing to notice and nothing to retry.
  const eventsBroke = events.isError && !isAbsence(events.error);
  const goalsBroke = goals.isError && !isAbsence(goals.error);
  const history = useHistory();
  const prefs = usePrefs();
  const now = today();
  const todayIso = iso(now);

  // The app shell renders this only once the couple has loaded. Every other
  // source can fail on its own (or not exist in the API yet) without taking
  // the greeting and the rest of the screen down with it.
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
  // Not a failure: there is no week until there are two of them, and until
  // then the week is never even asked for (see isAlone).
  const alone = isAlone(Boolean(coupleData.partner), week.error);
  const upcoming = upcomingEvents(events.data ?? [], today());
  const todays = todayEvent(upcoming, todayIso);
  const nextEvent = nextUpcomingEvent(upcoming, todayIso);
  const goal = activeGoal(goals.data ?? []);
  const lastWeek = lastWeekSummary(history.data, me.id);
  const reminder = reminderLabel(prefs.data);

  if (w && w.setter_id === me.id && w.status === "draft") {
    return (
      <Main>
        <div className="pt-1.5">
          <Micro>{range(w.week_start, w.week_end)}</Micro>
        </div>
        <Title size="display" className="mt-2">
          It&rsquo;s your week.
        </Title>
        <Para size="lg" className="mt-2.5">
          What would you like the two of you to pray about? {partner} will see your prayers once you
          share them.
        </Para>
        <div className="mt-7">
          <LinkButton href={routes.prayersSet}>Set this week&rsquo;s prayers</LinkButton>
        </div>
        <div className="mt-7 flex flex-col border-t border-line">
          {lastWeek ? (
            <InfoRow
              icon="clock"
              label="Last week"
              value={`you prayed ${lastWeek.prayed} of ${lastWeek.total}`}
              action="View"
              href={routes.history}
            />
          ) : null}
          <InfoRow
            icon="bell"
            label="Reminder"
            value={reminder}
            action="Change"
            href={routes.settingsNotifications}
          />
        </div>
      </Main>
    );
  }

  if (w && w.status === "waiting") {
    return (
      <Main>
        <div className="pt-1.5">
          <Micro>{range(w.week_start, w.week_end)}</Micro>
        </div>
        <Title size="display" className="mt-2">
          Your partner is setting this week&rsquo;s prayers.
        </Title>
        <Para size="lg" className="mt-2.5">
          We&rsquo;ll let you know the moment {partner} shares them. Until then, the time is yours.
        </Para>
        <div className="mt-6">
          <LinkButton href={routes.prayerMode({ quiet: true })} icon="moon">
            Enter quiet prayer
          </LinkButton>
        </div>
        <div className="mt-6 flex flex-col border-t border-line">
          {lastWeek ? (
            <Row
              icon="book"
              title="Last week’s prayers"
              sub={`${lastWeek.total} prayers ${lastWeek.iSetIt ? "you set" : "partner set"}`}
              href={routes.history}
            />
          ) : null}
          <Row
            icon="clock"
            title="Your prayer history"
            sub="Every week you’ve shared"
            href={routes.history}
          />
          <Row
            icon="bell"
            title="Prayer reminder"
            sub={prefs.data?.prayer_reminder ? `Every day at ${reminder}` : "Off"}
            href={routes.settingsNotifications}
            last
          />
        </div>
      </Main>
    );
  }

  // Counted only when every source answered; a failed one would undercount.
  const planned = w && events.data && goals.data ? plannedCount(upcoming, goal, w) : null;

  return (
    <Main>
      <div className="flex items-start justify-between pt-1.5">
        <div>
          <Micro>{weekdayDate(todayIso)}</Micro>
          <h1 className="m-0 mt-2 text-[30px] font-semibold leading-[1.18] tracking-[-0.022em]">
            <span className="font-medium text-stone">{greeting(now)},</span>
            <br />
            {alone ? me.display_name : `${me.display_name} & ${partner}`}
          </h1>
        </div>
        <Link
          href={routes.eventNew()}
          aria-label="Add something to our space"
          className="press -mr-2 flex h-11 w-11 items-center justify-center rounded-full text-ink"
        >
          <Icon name="plus" size={24} />
        </Link>
      </div>

      {/* Alone, this is the only thing on the screen worth doing, so it is
          the only thing offered. Everything Amorae has — the week, the plans,
          what gets kept — is built for two people, and a home screen that
          listed those as empty rows would be describing an app that does not
          work yet rather than one waiting for somebody. */}
      {alone ? (
        <div className="mt-6 flex flex-col items-start rounded-card border border-line bg-surface px-5 py-5">
          <Micro tone="plum">Just you, so far</Micro>
          <h2 className="m-0 mt-2 text-[21px] font-semibold tracking-[-0.015em]">
            Amorae is for two.
          </h2>
          <Para className="mt-1.5">
            The week&rsquo;s prayers, the plans, the things you keep &mdash; none of it starts until{" "}
            {partner} is here too.
          </Para>
          <LinkButton href={routes.invite} className="mt-4">
            Invite them
          </LinkButton>
        </div>
      ) : null}

      {/* Today's plan and this week sit side by side from `lg`, where a
          single column would leave the right half of the screen empty. With
          nothing on today, "This week" simply takes the row. */}
      <div className="flex flex-col lg:mt-6 lg:flex-row lg:items-start lg:gap-8">
        {todays ? (
          <Link
            href={routes.event(todays.id)}
            className="press mt-5 flex items-center gap-4 rounded-card border border-line bg-surface py-[18px] pl-5 pr-4 text-ink no-underline lg:mt-0 lg:flex-1"
          >
            <span className="flex grow flex-col gap-0.5">
              <Micro tone="plum">{partOfDay(todays.start_time)}</Micro>
              <span className="mt-1 text-[21px] font-semibold tracking-[-0.015em]">
                {todays.title}
              </span>
              <span className="text-[15px] text-stone">
                {time12(todays.start_time)}
                {todays.start_time && todays.location ? " · " : ""}
                {todays.location ?? ""}
              </span>
            </span>
            <Icon name="right" size={18} className="text-stone" />
          </Link>
        ) : null}

        {/* A heading with nothing under it is worse than no heading: it
            reads as something that failed to load. Alone and with nothing
            planned, there is genuinely nothing to put here, so the section
            does not appear at all. */}
        {alone && !eventsBroke && !goalsBroke && !nextEvent && !goal ? null : (
          <Section
            label="This week"
            className="mt-6 lg:mt-0 lg:flex-1"
            trailing={
              planned !== null ? (
                <span className="text-[13px] text-stone">
                  {planned} {planned === 1 ? "thing" : "things"} planned
                </span>
              ) : undefined
            }
          >
            {/* Before a partner joins there is no week and cannot be one: praying
              together alternates between two people, so one person has no turn
              to take. The API says exactly that (409 waiting_for_partner) and
              nothing here listened, so the very first thing a new account saw
              on its own home screen was a card saying this could not be
              loaded. It loaded fine. There was simply nothing to load yet. */}
            {alone ? null : (
              <QueryState queries={[week]} loading={<Skeleton lines={1} />}>
                {(wk) => (
                  <Row
                    icon="book"
                    title="Weekly prayer"
                    sub={`Set by ${setterWord(wk, me.id, partner)} · you’ve prayed ${wk.my_completed.length} of ${wk.points.length}`}
                    href={routes.prayers}
                  >
                    <Segments
                      total={wk.points.length}
                      done={wk.my_completed.length}
                      height={4}
                      className="mb-0.5 mt-1.5 w-24"
                    />
                  </Row>
                )}
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
                icon="calendar"
                title={nextEvent.title}
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
                icon="target"
                title={goal.goal.title}
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

      {w ? (
        <PrayerFooter week={w} partner={partner} reminder={prefs.data ? reminder : null} />
      ) : null}
    </Main>
  );
}

/** The prayer call to action and the partner's progress, shown only once the week has loaded. */
function PrayerFooter({
  week: w,
  partner,
  reminder,
}: {
  week: PrayerWeek;
  partner: string;
  reminder: string | null;
}) {
  const done = w.my_completed.length;
  const total = w.points.length;
  return (
    <>
      <div className="mt-[18px]">
        <LinkButton href={routes.prayerMode()}>
          {done === total ? "Open prayer mode" : done === 0 ? "Begin praying" : "Continue praying"}
        </LinkButton>
      </div>
      <div className="mt-3.5 flex items-center gap-2 pb-4 text-support text-stone">
        <Initial letter={partner[0]} size={18} />
        {partner} has prayed {w.partner_completed.length} of {total}
        {reminder ? <> &middot; reminder {reminder}</> : null}
      </div>
    </>
  );
}

function InfoRow({
  icon,
  label,
  value,
  action,
  href,
}: {
  icon: "clock" | "bell";
  label: string;
  value: string;
  action: string;
  href: string;
}) {
  return (
    <div className="flex min-h-[56px] items-center gap-3 border-b border-line">
      <Icon name={icon} size={22} className="text-stone" />
      <div className="grow text-[15px] text-stone">
        {label} <span className="font-semibold text-ink">{value}</span>
      </div>
      <Link
        href={href}
        className="press flex h-11 items-center px-0.5 text-[15px] font-semibold text-plum no-underline"
      >
        {action}
      </Link>
    </div>
  );
}
