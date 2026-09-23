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
import { useCouple } from "@/features/couple/hooks";
import { setterWord } from "@/features/prayers/derive";
import { useHistory, useWeek } from "@/features/prayers/hooks";
import { usePrefs } from "@/features/settings/hooks";
import { useEvents, useGoals } from "@/features/together/hooks";
import type { PrayerWeek } from "@/lib/api/types";
import {
  activeGoal,
  lastWeekSummary,
  nextUpcomingEvent,
  plannedCount,
  reminderLabel,
  tonightEvent,
  upcomingEvents,
} from "./derive";
import { greeting, iso, naira, range, relativeDay, time12, weekdayDate } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { today } from "@/lib/today";

export function HomeScreen() {
  const couple = useCouple();
  const week = useWeek();
  const events = useEvents();
  const goals = useGoals();
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
  const upcoming = upcomingEvents(events.data ?? [], todayIso);
  const tonight = tonightEvent(upcoming, todayIso);
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
            {me.display_name} &amp; {partner}
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

      {/* Tonight and this week sit side by side from `lg`, where a single
          column would leave the right half of the screen empty. With no event
          tonight, "This week" simply takes the row. */}
      <div className="flex flex-col lg:mt-6 lg:flex-row lg:items-start lg:gap-8">
        {tonight ? (
          <Link
            href={routes.event(tonight.id)}
            className="press mt-5 flex items-center gap-4 rounded-card border border-line bg-surface py-[18px] pl-5 pr-4 text-ink no-underline lg:mt-0 lg:flex-1"
          >
            <span className="flex grow flex-col gap-0.5">
              <Micro tone="plum">Tonight</Micro>
              <span className="mt-1 text-[21px] font-semibold tracking-[-0.015em]">
                {tonight.title}
              </span>
              <span className="text-[15px] text-stone">
                {time12(tonight.start_time)}
                {tonight.location ? ` · ${tonight.location}` : ""}
              </span>
            </span>
            <Icon name="right" size={18} className="text-stone" />
          </Link>
        ) : null}

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
          {nextEvent ? (
            <Row
              icon="calendar"
              title={nextEvent.title}
              sub={`${relativeDay(nextEvent.date, todayIso)}, ${time12(nextEvent.start_time)}`}
              href={routes.event(nextEvent.id)}
            />
          ) : null}
          {goal ? (
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
