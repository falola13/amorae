"use client";

import Link from "next/link";
import { useState, type ReactNode } from "react";

import { Icon } from "@/components/icons";
import { Main } from "@/components/layout/screen";
import { Row } from "@/components/ui/kit";
import { OnTop, useRaiseGlass } from "@/features/anniversary/celebrate";
import { ordinal } from "@/features/anniversary/derive";
import { Heading, Keepsake } from "@/features/anniversary/sections";
import { todaysPoints } from "@/features/prayers/derive";
import { MemoryComposer } from "@/features/together/components/memory-composer";
import type { Anniversary, Couple, Event, Goal, PrayerWeek } from "@/lib/api/types";
import { naira, relativeDay, time12, weekdayDate } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { HomeChallengeCard } from "./challenge-card";

/**
 * Home, on the anniversary. Not the everyday screen in new colours: the
 * day leads — a keepsake with a glass to raise, the letter, what the two of
 * them are doing today — and the ordinary week follows underneath in the
 * occasion's own type.
 */
export function AnniversaryHome({
  a,
  couple,
  todayIso,
  todays,
  nextEvent,
  goal,
  week,
  weekPrompt,
  unread,
}: {
  a: Anniversary;
  couple: Couple;
  todayIso: string;
  /** Today's plans, soonest first. */
  todays: Event[];
  /** The next plan after today, if any. */
  nextEvent: Event | null;
  goal: { goal: Goal; total: number } | null;
  week: PrayerWeek | undefined;
  /** The "set this week's prayers" card, when the week isn't shared yet. */
  weekPrompt: ReactNode;
  unread: number;
}) {
  const me = couple.me.display_name.split(" ")[0];
  const partner = couple.partner?.display_name.split(" ")[0] ?? "your partner";
  const glass = useRaiseGlass({ years: a.years, partner });
  const [keeping, setKeeping] = useState(false);

  return (
    <Main className="pb-6">
      <div className="flex items-start justify-between pt-1.5">
        <div>
          <div className="text-[11px] font-bold uppercase tracking-[0.24em] text-gold">
            {weekdayDate(todayIso)}
          </div>
          <h1 className="m-0 mt-2 font-serif font-medium leading-[1.05]">
            <span className="block text-[26px] italic text-stone">Happy anniversary,</span>
            <span className="block text-[38px] tracking-[-0.01em] text-plum-dark">
              {me} <span className="italic text-gold">&amp;</span> {partner}
            </span>
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
            href={routes.eventNew({ on: todayIso })}
            aria-label="Add a plan for today"
            className="press flex h-11 w-11 items-center justify-center rounded-full text-ink"
          >
            <Icon name="plus" size={24} />
          </Link>
        </div>
      </div>

      <div className="mt-5">
        <Keepsake a={a} couple={couple} showDay onCheers={glass.raise} cheersBusy={glass.busy} />
      </div>

      <LetterCard a={a} partner={partner} />

      {weekPrompt}

      <section className="mt-12">
        <Heading sub={todays.length ? undefined : "Nothing planned yet — make it a day"}>
          Today, together
        </Heading>
        {todays.length ? (
          <div className="overflow-hidden rounded-[22px] border border-line bg-surface">
            {todays.map((e, i) => (
              <Link
                key={e.id}
                href={routes.event(e.id)}
                className={`press flex items-center gap-4 px-5 py-4 text-ink no-underline ${i > 0 ? "border-t border-line" : ""}`}
              >
                <span className="w-[62px] shrink-0 font-serif text-[18px] italic text-gold">
                  {e.start_time ? time12(e.start_time).replace(":00", "") : "today"}
                </span>
                <span className="flex grow flex-col">
                  <span className="text-[16px] font-semibold">{e.title}</span>
                  {e.location ? <span className="text-[13px] text-stone">{e.location}</span> : null}
                </span>
                <Icon name="right" size={18} className="text-stone" />
              </Link>
            ))}
          </div>
        ) : null}
        <div className="mt-3 grid grid-cols-2 gap-2.5">
          <Link
            href={`${routes.ourDay()}#ideas`}
            className="press flex flex-col rounded-[18px] bg-occasion px-4 py-4 text-ink no-underline"
          >
            <Icon name="calendar" size={20} className="text-plum" />
            <span className="mt-2 text-[15px] font-semibold">Ideas for today</span>
            <span className="text-[12px] text-stone">Add one in a tap</span>
          </Link>
          <button
            type="button"
            onClick={() => setKeeping(true)}
            className="press flex flex-col rounded-[18px] bg-occasion px-4 py-4 text-left text-ink"
          >
            <Icon name="image" size={20} className="text-plum" />
            <span className="mt-2 text-[15px] font-semibold">Keep today</span>
            <span className="text-[12px] text-stone">A photo of the day</span>
          </button>
          <Link
            href={`${routes.ourDay()}#questions`}
            className="press flex flex-col rounded-[18px] bg-occasion px-4 py-4 text-ink no-underline"
          >
            <Icon name="note" size={20} className="text-plum" />
            <span className="mt-2 text-[15px] font-semibold">For tonight</span>
            <span className="text-[12px] text-stone">Questions to ask</span>
          </Link>
          <Link
            href={`${routes.ourDay()}#prayer`}
            className="press flex flex-col rounded-[18px] bg-occasion px-4 py-4 text-ink no-underline"
          >
            <Icon name="book" size={20} className="text-plum" />
            <span className="mt-2 text-[15px] font-semibold">Pray it together</span>
            <span className="text-[12px] text-stone">For the year ahead</span>
          </Link>
        </div>
      </section>

      <section className="mt-12">
        <Heading>This week</Heading>
        <div className="overflow-hidden rounded-[22px] border border-line bg-surface px-4">
          {week ? <PrayerRow week={week} /> : null}
          {nextEvent ? (
            <Row
              icon="calendar"
              title={nextEvent.title}
              sub={`${relativeDay(nextEvent.date, todayIso)}${nextEvent.start_time ? `, ${time12(nextEvent.start_time)}` : ""}`}
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
              last
            />
          ) : (
            <Row
              icon="clock"
              title="Our story"
              sub="Everything you have kept"
              href={routes.history}
              last
            />
          )}
        </div>
        <HomeChallengeCard partner={couple.partner?.display_name ?? "your partner"} />
      </section>

      <div className="mt-10 flex flex-col items-center gap-3 text-center">
        <Link
          href={routes.ourDay()}
          className="press inline-flex h-[54px] w-full items-center justify-center gap-2 rounded-full text-[16px] font-bold text-[#3d1428] no-underline shadow-[0_10px_30px_-12px_rgba(233,199,127,0.8)]"
          style={{ background: "linear-gradient(100deg, #f6e2ae, #e2b866 55%, #f3d79b)" }}
        >
          Open our day <Icon name="right" size={18} />
        </Link>
        <p className="m-0 font-serif text-[18px] italic text-stone">
          Your {ordinal(a.years)} year, done. Here&rsquo;s to the next.
        </p>
      </div>

      {glass.layer}
      {keeping ? (
        <OnTop>
          <MemoryComposer
            open
            onClose={() => setKeeping(false)}
            onPhotoFailed={() => setKeeping(false)}
            title={`Our ${ordinal(a.years)} anniversary`}
          />
        </OnTop>
      ) : null}
    </Main>
  );
}

/** The letter, as the first thing after the keepsake. */
function LetterCard({ a, partner }: { a: Anniversary; partner: string }) {
  const [title, sub] = a.theirs
    ? [`A letter from ${partner}`, "Sealed until today — open it now"]
    : a.mine
      ? [
          `${partner} can read your letter now`,
          a.theirs_sealed ? "" : `If ${partner} writes you one, it will be here`,
        ]
      : [`Write ${partner} a letter`, "They can read it the moment you keep it"];
  return (
    <Link
      href={a.theirs || a.mine ? `${routes.ourDay()}#letters` : routes.ourDayLetter}
      className="press mt-4 flex items-center gap-4 overflow-hidden rounded-[22px] px-5 py-4 text-[#fbeef2] no-underline"
      style={{ background: "linear-gradient(160deg, #6b2a4c, #4a1a35)" }}
    >
      <span
        aria-hidden="true"
        className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full"
        style={{
          background: "radial-gradient(circle at 35% 30%, #f6e2ae, #c99a4b 55%, #8a5d1f)",
          boxShadow: "0 3px 8px rgba(0,0,0,0.35)",
        }}
      >
        <Icon
          name={a.theirs ? "heart" : "letter"}
          size={20}
          strokeWidth={2}
          className="text-[#5a2042]"
        />
      </span>
      <span className="flex grow flex-col">
        <span className="font-serif text-[22px] italic leading-tight">{title}</span>
        {sub ? <span className="text-[13px] text-[#e9c886]">{sub}</span> : null}
      </span>
      <Icon name="right" size={18} className="text-[#f0cbd8]" />
    </Link>
  );
}

function PrayerRow({ week }: { week: PrayerWeek }) {
  if (week.status !== "published") {
    return (
      <Row
        icon="book"
        title="Weekly prayer"
        sub={week.status === "draft" ? "Not shared yet" : "Being set"}
        href={week.status === "draft" ? routes.prayersSet : routes.prayers}
      />
    );
  }
  const total = todaysPoints(week).length;
  const done = week.my_completed.length;
  return (
    <Row
      icon="book"
      title="Weekly prayer"
      sub={total === 0 ? "Nothing set for today" : `Today ${done} of ${total} prayed`}
      href={done < total ? routes.prayerMode() : routes.prayers}
    />
  );
}
