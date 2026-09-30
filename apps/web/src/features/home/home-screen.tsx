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
  Title,
} from "@/components/ui/kit";
import { QueryState } from "@/components/ui/query-state";
import { isAbsence } from "@/lib/api/envelope";
import { readableMessage } from "@/lib/query/client";
import { useCouple } from "@/features/couple/hooks";
import { useInbox } from "@/features/notifications/hooks";
import { setterWord, todaysPoints } from "@/features/prayers/derive";
import { useHistory, useWeek } from "@/features/prayers/hooks";
import { usePrefs } from "@/features/settings/hooks";
import { HomeChallengeCard } from "./challenge-card";
import { eventPhase, nowLabel } from "@/features/together/events";
import { useEvents, useGoals, useMilestones, useNudge } from "@/features/together/hooks";
import {
  celebrationCopy,
  daysAwayLabel,
  todaysMilestones,
  upcomingMilestones,
} from "@/features/together/milestones";
import type { Milestone, PrayerWeek } from "@/lib/api/types";
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
  // data ?? [] below would make a failed fetch look identical to an empty week, so track it separately.
  const eventsBroke = events.isError && !isAbsence(events.error);
  const goalsBroke = goals.isError && !isAbsence(goals.error);
  const history = useHistory();
  const prefs = usePrefs();
  const milestones = useMilestones();
  const inbox = useInbox();
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
  const lastWeek = lastWeekSummary(history.data, me.id);
  const reminder = reminderLabel(prefs.data);

  if (w && w.status === "draft") {
    const mine = w.setter_id === me.id;
    return (
      <Main>
        <div className="pt-1.5">
          <Micro>{range(w.week_start, w.week_end)}</Micro>
        </div>
        <Title size="display" className="mt-2">
          {mine ? "It’s your week." : "This week’s prayers"}
        </Title>
        <Para size="lg" className="mt-2.5">
          {mine
            ? `What would you like the two of you to pray about? ${partner} will see your prayers once you share them.`
            : `It’s ${partner}’s turn this week — you can still write it.`}
        </Para>
        <div className="mt-7">
          <LinkButton href={routes.prayersSet}>
            {mine ? "Set this week’s prayers" : "Write or edit this week"}
          </LinkButton>
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
      />

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

      {alone ? null : <HomeChallengeCard partner={partner} />}

      {/* Side by side from `lg`; a single column would leave the right half empty. */}
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
            className="mt-6 lg:mt-0 lg:flex-1"
            trailing={
              planned !== null ? (
                <span className="text-[13px] text-stone">
                  {planned} {planned === 1 ? "thing" : "things"} planned
                </span>
              ) : undefined
            }
          >
            {/* No week can exist without a partner (prayer alternates between two people), so skip it. */}
            {alone ? null : (
              <QueryState queries={[week]} loading={<Skeleton lines={1} />}>
                {(wk) => {
                  const total = todaysPoints(wk).length;
                  const done = wk.my_completed.length;
                  return (
                    <Row
                      icon="book"
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

/** The icon for a milestone: cake for a birthday, heart for an anniversary,
 *  gift for anything else worth marking. */
const celebrationIcon = (m: Milestone) =>
  m.source === "birthday" ? "cake" : m.source === "anniversary" ? "heart" : "gift";

/** What's today, and what's coming in the week ahead — a calm card per date
 *  happening today (stacked if there's more than one), then a slim row for
 *  each one still a few days off. Nothing renders when there's nothing to say. */
function Celebrations({
  milestones,
  meId,
  partner,
  todayIso,
}: {
  milestones: Milestone[];
  meId: string;
  partner: string;
  todayIso: string;
}) {
  const todays = todaysMilestones(milestones, todayIso);
  const soon = upcomingMilestones(milestones, todayIso);
  if (todays.length === 0 && soon.length === 0) return null;
  return (
    <div className="mt-6 flex flex-col gap-3">
      {todays.map((m) => {
        const { title, sub } = celebrationCopy(m, todayIso, meId, partner);
        return (
          <div key={m.id} className="flex flex-col items-start rounded-card bg-celebrate px-5 py-5">
            <Icon name={celebrationIcon(m)} size={26} strokeWidth={1.4} className="text-plum" />
            <h2 className="m-0 mt-3 text-[26px] font-semibold leading-tight tracking-[-0.02em] text-plum-dark">
              {title}
            </h2>
            {sub ? <Para className="mt-1">{sub}</Para> : null}
          </div>
        );
      })}
      {soon.map((m) => (
        <Link
          key={m.id}
          href={routes.milestones}
          className="press -my-1 flex items-center gap-2.5 py-1 text-ink no-underline"
        >
          <Icon name={celebrationIcon(m)} size={18} className="text-stone" />
          <span className="text-support text-stone">
            {m.title} · {daysAwayLabel(m.days)}
          </span>
        </Link>
      ))}
    </div>
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
      <div className="mt-[18px]">
        <LinkButton href={routes.prayerMode()}>
          {total === 0 || done >= total
            ? "Open prayer mode"
            : done === 0
              ? "Begin praying"
              : "Continue praying"}
        </LinkButton>
      </div>
      <div className="mt-3.5 flex items-center gap-2 pb-4 text-support text-stone">
        <Initial letter={partner[0]} photoUrl={partnerPhoto} name={partner} size={18} />
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
