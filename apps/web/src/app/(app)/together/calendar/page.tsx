"use client";

import Link from "next/link";
import { useState } from "react";
import { useCouple } from "@/features/couple/hooks";
import { useWeek } from "@/features/prayers/hooks";
import { eventOwnerLabel } from "@/features/together/events";
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

// `at` orders items within a day ("" sorts before timed entries); `rank` breaks ties among all-day items.
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

function clock(start?: string, end?: string): { time: string; until?: string } {
  // Guards against pre-validation rows where end <= start (would render as a bogus range).
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
  const meId = couple.data?.me.id;
  const partnerName = couple.data?.partner?.display_name;
  const todayIso = iso(today());

  // A single anchor (not a week offset) keeps week/month toggling on the same date.
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
              // Selects a day when entering month view so the grid isn't left with an empty list under it.
              if (!month && !selected) setSelected(sameMonth(anchor, todayIso) ? todayIso : anchor);
              setMonth((m) => !m);
            }}
            aria-pressed={month}
            className="press -mr-2 h-11 rounded-btn px-2.5 text-[15px] font-semibold text-plum"
          >
            {month ? "Week" : "Month"}
          </button>
        </div>

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

        {/* Milestones are gated by QueryState so a failed fetch shows an error, not an empty list. Week stays ungated: there's genuinely no week until a partner joins. */}
        <QueryState queries={[events, milestones]} loading={<Skeleton />}>
          {(eventsData, milestoneData) => {
            const byDay = new Map<string, Item[]>();
            const push = (d: string, it: Item) => byDay.set(d, [...(byDay.get(d) ?? []), it]);

            for (const e of eventsData)
              if (!e.done) {
                const owner = eventOwnerLabel(e, meId, partnerName);
                const base = e.location ?? (e.reminder ? `Reminder ${e.reminder}` : "");
                push(e.date, {
                  at: e.start_time ?? "",
                  rank: 1,
                  ...clock(e.start_time, e.end_time),
                  title: e.title,
                  sub: owner ? (base ? `${base} · ${owner}` : owner) : base,
                  href: routes.event(e.id),
                });
              }

            // Checks each visible day against each milestone (not each milestone's next occurrence) and only for ones with a reminder set.
            for (const d of days)
              for (const m of milestoneData)
                if (m.reminder && occursOn(m.date, d))
                  push(d, {
                    at: "",
                    rank: 0,
                    time: "All day",
                    title: m.title,
                    sub: yearsBy(m.date, d, m.year_known) || m.sub || "",
                    href: routes.milestones,
                  });

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

            // Nothing selected means "what's ahead": a week containing today
            // opens on today, not on the start of the week, so the first
            // thing read is not yesterday. Paging to a past week, or tapping
            // a day, is a deliberate look back and shows everything.
            const list = selected
              ? [selected]
              : days.filter((d) => (byDay.get(d)?.length ?? 0) > 0 && (!atToday || d >= todayIso));

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
                  // Full weekday + item count, since the visual dot marker is aria-hidden decoration.
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
                  {/* One mark per thing, to three. A single dot said "something"
                      for a day with one errand and a day with an anniversary,
                      a dinner and the week's prayers on it. */}
                  <span aria-hidden="true" className="flex h-1 items-center gap-[3px]">
                    {Array.from({ length: Math.min(count, 3) }, (_, i) => (
                      <span
                        key={i}
                        className={cx("h-1 w-1 rounded-full", outside ? "bg-edge" : "bg-plum")}
                      />
                    ))}
                  </span>
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
                          className={cx(
                            "grow basis-0 text-center text-[12px] font-semibold",
                            i === 0 || i === 6 ? "text-edge" : "text-stone",
                          )}
                        >
                          {w}
                        </span>
                      ))}
                    </div>
                    {/* Ruled, not boxed. Numbers floating in space read as a
                        list; full boxes read as a spreadsheet. A line under
                        each week is what a printed calendar does. */}
                    {Array.from({ length: days.length / 7 }, (_, r) => (
                      <div
                        key={r}
                        className={cx(
                          "flex pt-2",
                          r < days.length / 7 - 1 && "border-b border-line pb-2",
                        )}
                      >
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
                  <section key={d} className="flex flex-col pb-2.5 pt-3.5">
                    <div className="flex items-center gap-3 pb-1.5">
                      <Micro tone={d === todayIso ? "plum" : "stone"}>
                        {d === todayIso ? "Today, " : ""}
                        {weekdayDate(d)}
                      </Micro>
                      <span className="h-px grow bg-line" />
                    </div>
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
                          {it.sub ? (
                            <span className="text-support text-stone">{it.sub}</span>
                          ) : null}
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
        {/* Falls back to the anchored week's first day (not today) so paging away and adding still targets the right day. */}
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
