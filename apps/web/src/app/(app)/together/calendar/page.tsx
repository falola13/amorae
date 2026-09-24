"use client";

import Link from "next/link";
import { useState } from "react";
import { useCouple } from "@/features/couple/hooks";
import { useWeek } from "@/features/prayers/hooks";
import { useEvents, useMilestones } from "@/features/together/hooks";
import { occursOn, yearsBy } from "@/features/together/milestones";
import { addDays, dayNum, iso, parse, startOfWeek, time12, weekdayDate } from "@/lib/dates";
import { routes } from "@/lib/routes";
import { Icon } from "@/components/icons";
import { today } from "@/lib/today";
import { Main } from "@/components/layout/screen";
import { QueryState } from "@/components/ui/query-state";
import { BottomActions, LinkButton, Micro, Skeleton, Title, TopBar, cx } from "@/components/ui/kit";

// `at` orders a day's entries: "" for all-day, so anniversaries and things
// with no time sit above the timed ones, then 08:30 before 19:00. Three
// sources feed one day now, and the order they happen to be read in is not
// an order anybody meant.
type Item = { at: string; time: string; title: string; sub: string; href: string };

export default function Calendar() {
  const events = useEvents();
  const milestones = useMilestones();
  const week = useWeek();
  const couple = useCouple();
  const todayIso = iso(today());
  const [offset, setOffset] = useState(0);
  const [selected, setSelected] = useState<string | null>(null);
  const start = addDays(startOfWeek(today()), offset * 7);
  const days = Array.from({ length: 7 }, (_, i) => iso(addDays(start, i)));

  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="flex items-center justify-between pt-2">
          <Title>Calendar</Title>
          <div className="-mr-2.5 flex items-center">
            {offset !== 0 ? (
              <button
                type="button"
                onClick={() => {
                  setOffset(0);
                  setSelected(null);
                }}
                className="press h-11 px-2 text-[15px] font-semibold text-plum"
              >
                Today
              </button>
            ) : null}
            <button
              type="button"
              aria-label="Previous week"
              onClick={() => {
                setOffset((o) => o - 1);
                setSelected(null);
              }}
              className="press flex h-11 w-11 items-center justify-center text-ink"
            >
              <Icon name="left" size={22} />
            </button>
            <button
              type="button"
              aria-label="Next week"
              onClick={() => {
                setOffset((o) => o + 1);
                setSelected(null);
              }}
              className="press flex h-11 w-11 items-center justify-center text-ink"
            >
              <Icon name="right" size={22} />
            </button>
          </div>
        </div>
        <QueryState queries={[events]} loading={<Skeleton />}>
          {(eventsData) => {
            const byDay = new Map<string, Item[]>();
            const push = (d: string, it: Item) => byDay.set(d, [...(byDay.get(d) ?? []), it]);
            for (const e of eventsData)
              if (!e.done)
                push(e.date, {
                  at: e.start_time ?? "",
                  time: time12(e.start_time) || "All day",
                  title: e.title,
                  sub: e.location ?? (e.reminder ? `Reminder ${e.reminder}` : ""),
                  href: routes.event(e.id),
                });
            // The dates they keep come round every year, so the calendar asks
            // each day whether one falls on it rather than asking each date
            // when it is next due. Only the ones set to come round: a date
            // kept without that is part of their story, not their week.
            for (const d of days)
              for (const m of milestones.data ?? [])
                if (m.reminder && occursOn(m.date, d))
                  push(d, {
                    at: "",
                    time: "All day",
                    title: m.title,
                    sub: yearsBy(m.date, d) || m.sub || "",
                    href: routes.milestones,
                  });
            // The Sunday the current prayer week starts on is a calendar item
            // too — that Sunday, and no other. It used to land on every
            // Sunday of every week you paged to, so a week in 2029 promised
            // "a new prayer week begins" and the calendar could never be
            // empty, which made its own empty state unreachable. A week that
            // has not begun is not a plan (FR-CAL-001).
            if (week.data)
              for (const d of days)
                if (d === week.data.week_start)
                  push(d, {
                    at: "",
                    time: "All day",
                    title: "Weekly prayer",
                    sub: `A prayer week begins. ${week.data.setter_id === couple.data?.me.id ? "It’s your week." : `${couple.data?.partner?.display_name ?? "Your partner"} sets it.`}`,
                    href: routes.prayers,
                  });
            for (const items of byDay.values()) items.sort((a, b) => a.at.localeCompare(b.at));
            const shown = days.filter((d) =>
              selected ? d === selected : (byDay.get(d)?.length ?? 0) > 0 && d >= todayIso,
            );
            const list = shown.length ? shown : days.filter((d) => byDay.has(d));
            return (
              <>
                <div className="-mx-2 mt-3.5 flex border-b border-line pb-3">
                  {days.map((d) => {
                    const isToday = d === todayIso;
                    const sel = d === selected;
                    const dot = (byDay.get(d)?.length ?? 0) > 0;
                    return (
                      <button
                        key={d}
                        type="button"
                        aria-current={isToday ? "date" : undefined}
                        aria-pressed={sel}
                        onClick={() => setSelected(sel ? null : d)}
                        className="press flex grow basis-0 flex-col items-center gap-1 p-0"
                      >
                        <span className="text-[12px] font-semibold text-stone">
                          {"SMTWTFS"[parse(d).getDay()]}
                        </span>
                        <span
                          className={cx(
                            "tabular flex h-10 w-10 items-center justify-center rounded-full text-[16px] font-semibold",
                            isToday
                              ? "bg-plum font-bold text-surface"
                              : sel
                                ? "bg-plum-tint text-plum"
                                : "text-ink",
                          )}
                        >
                          {dayNum(d)}
                        </span>
                        <span
                          className={cx("h-1 w-1 rounded-full", dot ? "bg-plum" : "bg-transparent")}
                        />
                      </button>
                    );
                  })}
                </div>
                {list.length === 0 ? (
                  <div className="py-8 text-support text-stone">Nothing planned this week yet.</div>
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
                        <span className="tabular w-16 shrink-0 text-support text-stone">
                          {it.time}
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
                      // This line already said the right thing and did
                      // nothing. Tapping a day and then "Add an event" used to
                      // land you on today, so the one gesture the grid invites
                      // — pick a day, put something on it — was the one it
                      // would not do.
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
        <LinkButton href={routes.eventNew({ on: selected ?? undefined })} icon="plus">
          Add an event
        </LinkButton>
      </BottomActions>
    </>
  );
}
