"use client";

import Link from "next/link";
import { useState } from "react";
import { useCouple } from "@/features/couple/hooks";
import { useWeek } from "@/features/prayers/hooks";
import { useEvents, useMilestones } from "@/features/together/hooks";
import { occursOn, yearsBy } from "@/features/together/milestones";
import {
  addDays,
  addMonths,
  dayNum,
  iso,
  monthGrid,
  monthLabel,
  parse,
  sameMonth,
  startOfWeek,
  time12,
  weekdayDate,
} from "@/lib/dates";
import { routes } from "@/lib/routes";
import { Icon } from "@/components/icons";
import { today } from "@/lib/today";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import { BottomActions, LinkButton, Micro, Skeleton, Title, TopBar, cx } from "@/components/ui/kit";

/**
 * One thing on one day.
 *
 * `at` is what orders a day: "" for all-day, so anniversaries and things with
 * no time sit above the timed ones, then 08:30 before 19:00. `rank` breaks the
 * tie between all-day items from different sources, which used to be settled
 * by the order the three loops happened to run in — an order nobody chose.
 */
type Item = {
  at: string;
  rank: number;
  time: string;
  until?: string;
  title: string;
  sub: string;
  href: string;
};

const WEEKDAYS = ["S", "M", "T", "W", "T", "F", "S"];

/**
 * What the time column says.
 *
 * An event may carry an end without a start — nothing stops it, and the
 * composer offers the two independently — and that used to read as "All day",
 * which is the one thing it is not.
 */
function clock(start?: string, end?: string): { time: string; until?: string } {
  // An end that is not after its start is not a range. The API refuses to
  // save one, but rows predating that check exist, and "10:13 am to 10:00 am"
  // reads as a broken screen rather than as bad data. The detail screen still
  // shows both fields, which is where you would go to correct it.
  const ranged = Boolean(start && end && end > start);
  if (start) return { time: time12(start), until: ranged ? time12(end) : undefined };
  if (end) return { time: `Ends ${time12(end)}` };
  return { time: "All day" };
}

export default function Calendar() {
  const events = useEvents();
  const milestones = useMilestones();
  const week = useWeek();
  const couple = useCouple();
  const todayIso = iso(today());

  // One anchor rather than a week offset, so switching between week and month
  // keeps you where you were instead of throwing you back to today — and so
  // "next" means the next of whatever you are actually looking at.
  const [anchor, setAnchor] = useState(todayIso);
  const [month, setMonth] = useState(false);
  const [selected, setSelected] = useState<string | null>(null);

  const anchorDate = parse(anchor);
  const days = month
    ? monthGrid(anchorDate)
    : Array.from({ length: 7 }, (_, i) => iso(addDays(startOfWeek(anchorDate), i)));
  const step = (n: number) =>
    setAnchor(iso(month ? addMonths(anchorDate, n) : addDays(anchorDate, n * 7)));
  const atToday = month ? sameMonth(anchor, todayIso) : days.includes(todayIso);

  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="flex items-center justify-between pt-2">
          <Title>Calendar</Title>
          <button
            type="button"
            onClick={() => {
              // Entering the month, land on a day rather than on nothing: a
              // grid with no day chosen has an empty half-screen under it.
              if (!month && !selected) setSelected(sameMonth(anchor, todayIso) ? todayIso : anchor);
              setMonth((m) => !m);
            }}
            aria-pressed={month}
            className="press -mr-2 h-11 rounded-btn px-2.5 text-[15px] font-semibold text-plum"
          >
            {month ? "Week" : "Month"}
          </button>
        </div>

        {/* The calendar never said which month you were in. Day numbers alone
            repeat every month, so paging more than a few weeks out left you
            reading 1–31 with no way to know where you had got to. */}
        <div className="mt-1 flex items-center gap-1">
          <button
            type="button"
            aria-label={month ? "Previous month" : "Previous week"}
            onClick={() => step(-1)}
            className="press -ml-2.5 flex h-11 w-11 items-center justify-center text-ink"
          >
            <Icon name="left" size={22} />
          </button>
          <h2 className="m-0 grow text-[17px] font-semibold tracking-[-0.01em]">
            {monthLabel(month ? anchor : days[0])}
            {!month && !sameMonth(days[0], days[6]) ? (
              <span className="text-stone"> – {monthLabel(days[6])}</span>
            ) : null}
          </h2>
          {!atToday ? (
            <button
              type="button"
              onClick={() => {
                setAnchor(todayIso);
                setSelected(month ? todayIso : null);
              }}
              className="press h-11 px-2 text-[15px] font-semibold text-plum"
            >
              Today
            </button>
          ) : null}
          <button
            type="button"
            aria-label={month ? "Next month" : "Next week"}
            onClick={() => step(1)}
            className="press -mr-2.5 flex h-11 w-11 items-center justify-center text-ink"
          >
            <Icon name="right" size={22} />
          </button>
        </div>

        {/* Milestones sit inside the gate now. They used to be read as
            `milestones.data ?? []`, so a failed request looked exactly like a
            couple who had kept no dates — the calendar quietly reported a
            smaller life than they have, with nothing to notice and nothing to
            retry. The week stays outside it on purpose: there genuinely is no
            week until a partner joins. */}
        <QueryState queries={[events, milestones]} loading={<Skeleton />}>
          {(eventsData, milestoneData) => {
            const byDay = new Map<string, Item[]>();
            const push = (d: string, it: Item) => byDay.set(d, [...(byDay.get(d) ?? []), it]);

            for (const e of eventsData)
              if (!e.done)
                push(e.date, {
                  at: e.start_time ?? "",
                  rank: 1,
                  ...clock(e.start_time, e.end_time),
                  title: e.title,
                  sub: e.location ?? (e.reminder ? `Reminder ${e.reminder}` : ""),
                  href: routes.event(e.id),
                });

            // The dates they keep come round every year, so the calendar asks
            // each day whether one falls on it rather than asking each date
            // when it is next due. Only the ones set to come round: a date
            // kept without that is part of their story, not their week.
            for (const d of days)
              for (const m of milestoneData)
                if (m.reminder && occursOn(m.date, d))
                  push(d, {
                    at: "",
                    rank: 0,
                    time: "All day",
                    title: m.title,
                    sub: yearsBy(m.date, d) || m.sub || "",
                    href: routes.milestones,
                  });

            // The Sunday the current prayer week starts on, and no other.
            if (week.data)
              for (const d of days)
                if (d === week.data.week_start)
                  push(d, {
                    at: "",
                    rank: 2,
                    time: "All day",
                    title: "Weekly prayer",
                    sub: `A prayer week begins. ${week.data.setter_id === couple.data?.me.id ? "It's your week." : `${couple.data?.partner?.display_name ?? "Your partner"} sets it.`}`,
                    href: routes.prayers,
                  });

            for (const items of byDay.values())
              items.sort((a, b) => a.at.localeCompare(b.at) || a.rank - b.rank);

            // A day is listed when it is the one you picked, or — with none
            // picked — when it has something on it. The old rule also hid
            // days before today, but fell back to showing them when nothing
            // else in the week qualified, so whether a past day appeared
            // depended on what else happened to be in that week. Predictable
            // beats clever: everything in view that has something on it.
            const list = selected
              ? [selected]
              : days.filter((d) => (byDay.get(d)?.length ?? 0) > 0);

            const cell = (d: string) => {
              const isToday = d === todayIso;
              const sel = d === selected;
              const count = byDay.get(d)?.length ?? 0;
              const outside = month && !sameMonth(d, anchor);
              return (
                <button
                  key={d}
                  type="button"
                  aria-current={isToday ? "date" : undefined}
                  aria-pressed={sel}
                  // "S, 29, button" told a screen reader almost nothing — and
                  // S is both Sunday and Saturday. The count goes here because
                  // the dot that carries it visually is decoration.
                  aria-label={`${weekdayDate(d)}${isToday ? ", today" : ""}${
                    count ? `, ${count} ${count === 1 ? "thing" : "things"}` : ", nothing planned"
                  }`}
                  onClick={() => setSelected(sel ? null : d)}
                  className="press flex grow basis-0 flex-col items-center gap-1 p-0"
                >
                  {!month ? (
                    <span className="text-[12px] font-semibold text-stone">
                      {WEEKDAYS[parse(d).getDay()]}
                    </span>
                  ) : null}
                  <span
                    className={cx(
                      "tabular flex h-10 w-10 items-center justify-center rounded-full text-[16px] font-semibold",
                      isToday
                        ? "bg-plum font-bold text-surface"
                        : sel
                          ? "bg-plum-tint text-plum"
                          : outside
                            ? "text-edge"
                            : "text-ink",
                    )}
                  >
                    {dayNum(d)}
                  </span>
                  <span
                    aria-hidden="true"
                    className={cx(
                      "h-1 w-1 rounded-full",
                      count ? (outside ? "bg-edge" : "bg-plum") : "bg-transparent",
                    )}
                  />
                </button>
              );
            };

            return (
              <>
                {month ? (
                  <div className="-mx-2 mt-3 border-b border-line pb-3">
                    <div className="flex">
                      {WEEKDAYS.map((w, i) => (
                        <span
                          key={i}
                          aria-hidden="true"
                          className="grow basis-0 text-center text-[12px] font-semibold text-stone"
                        >
                          {w}
                        </span>
                      ))}
                    </div>
                    {Array.from({ length: days.length / 7 }, (_, r) => (
                      <div key={r} className="mt-1.5 flex">
                        {days.slice(r * 7, r * 7 + 7).map(cell)}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="-mx-2 mt-3.5 flex border-b border-line pb-3">
                    {days.map(cell)}
                  </div>
                )}

                {list.length === 0 ? (
                  <div className="py-8 text-support text-stone">
                    Nothing planned {month ? "this month" : "this week"} yet.
                  </div>
                ) : null}

                {list.map((d) => (
                  <section key={d} className="flex flex-col border-b border-line pb-2.5 pt-3.5">
                    <Micro tone={d === todayIso ? "plum" : "stone"} className="pb-1">
                      {d === todayIso ? "Today, " : ""}
                      {weekdayDate(d)}
                    </Micro>
                    {(byDay.get(d) ?? []).map((it) => (
                      <Link
                        key={it.href}
                        href={it.href}
                        className="press flex min-h-12 items-baseline gap-3.5 py-1 text-ink no-underline"
                      >
                        <span className="tabular w-[68px] shrink-0 text-support text-stone">
                          {it.time}
                          {it.until ? (
                            <span className="block text-[12px] opacity-75">to {it.until}</span>
                          ) : null}
                        </span>
                        <span className="flex flex-col">
                          <span className="text-[16px] font-semibold">{it.title}</span>
                          {it.sub ? <span className="text-support text-stone">{it.sub}</span> : null}
                        </span>
                      </Link>
                    ))}
                    {(byDay.get(d) ?? []).length === 0 ? (
                      <Link
                        href={routes.eventNew({ on: d })}
                        className="press py-2 text-support text-plum no-underline"
                      >
                        Nothing planned. Add something you&rsquo;d love to do together.
                      </Link>
                    ) : null}
                  </section>
                ))}
              </>
            );
          }}
        </QueryState>
      </Main>
      <BottomActions>
        {/* Without a day picked this falls back to the composer's own default,
            which is today — so paging three weeks out and tapping Add put the
            thing on the wrong day. Anchor the week you are actually looking
            at instead. */}
        <LinkButton
          href={routes.eventNew({ on: selected ?? (atToday ? undefined : days[0]) })}
          icon="plus"
        >
          Add an event
        </LinkButton>
      </BottomActions>
    </>
  );
}
