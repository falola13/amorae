'use client';

import Link from 'next/link';
import { useState } from 'react';
import { ME, TODAY } from '@/lib/api';
import { useEvents, useWeek } from '@/lib/hooks';
import { addDays, dayNum, iso, parse, startOfWeek, time12, weekdayDate } from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { BottomActions, LinkButton, Micro, Title, TopBar, cx } from '@/components/ui';

type Item = { time: string; title: string; sub: string; href: string };

export default function Calendar() {
  const events = useEvents(); const week = useWeek();
  const today = iso(TODAY);
  const [offset, setOffset] = useState(0);
  const [selected, setSelected] = useState<string | null>(null);
  const start = addDays(startOfWeek(TODAY), offset * 7);
  const days = Array.from({ length: 7 }, (_, i) => iso(addDays(start, i)));
  const byDay = new Map<string, Item[]>();
  const push = (d: string, it: Item) => byDay.set(d, [...(byDay.get(d) ?? []), it]);
  for (const e of events.data ?? []) if (!e.done) push(e.date, { time: time12(e.start) || 'All day', title: e.title, sub: e.location ?? (e.reminder ? `Reminder ${e.reminder}` : ''), href: `/together/events/${e.id}` });
  // The Sunday that starts a prayer week is a calendar item too.
  for (const d of days) if (parse(d).getDay() === 0) push(d, { time: 'All day', title: 'Weekly prayer', sub: week.data && d === week.data.start ? `A prayer week begins. ${week.data.setterId === ME ? 'It’s your week.' : 'Adeola sets it.'}` : 'A new prayer week begins.', href: '/prayers' });
  const shown = days.filter((d) => selected ? d === selected : (byDay.get(d)?.length ?? 0) > 0 && d >= today);
  const list = shown.length ? shown : days.filter((d) => byDay.has(d));
  return (
    <>
      <TopBar back="Our space" backHref="/together" />
      <Main>
        <div className="flex items-center justify-between pt-2">
          <Title>Calendar</Title>
          <div className="-mr-2.5 flex">
            <button type="button" aria-label="Previous week" onClick={() => { setOffset((o) => o - 1); setSelected(null); }} className="press flex h-11 w-11 items-center justify-center text-ink"><Icon name="left" size={22} /></button>
            <button type="button" aria-label="Next week" onClick={() => { setOffset((o) => o + 1); setSelected(null); }} className="press flex h-11 w-11 items-center justify-center text-ink"><Icon name="right" size={22} /></button>
          </div>
        </div>
        <div className="-mx-2 mt-3.5 flex border-b border-line pb-3">
          {days.map((d) => {
            const isToday = d === today; const sel = d === selected; const dot = (byDay.get(d)?.length ?? 0) > 0;
            return (
              <button key={d} type="button" aria-current={isToday ? 'date' : undefined} aria-pressed={sel} onClick={() => setSelected(sel ? null : d)} className="press flex grow basis-0 flex-col items-center gap-1 p-0">
                <span className="text-[12px] font-semibold text-stone">{'SMTWTFS'[parse(d).getDay()]}</span>
                <span className={cx('tabular flex h-10 w-10 items-center justify-center rounded-full text-[16px] font-semibold', isToday ? 'bg-plum font-bold text-surface' : sel ? 'bg-plum-tint text-plum' : 'text-ink')}>{dayNum(d)}</span>
                <span className={cx('h-1 w-1 rounded-full', dot ? 'bg-plum' : 'bg-transparent')} />
              </button>
            );
          })}
        </div>
        {list.length === 0 ? <div className="py-8 text-support text-stone">Nothing planned this week yet.</div> : null}
        {list.map((d) => (
          <section key={d} className="flex flex-col border-b border-line pb-2.5 pt-3.5">
            <Micro tone={d === today ? 'plum' : 'stone'} className="pb-1">{d === today ? 'Today, ' : ''}{weekdayDate(d)}</Micro>
            {(byDay.get(d) ?? []).map((it, i) => (
              <Link key={i} href={it.href} className="press flex min-h-12 items-baseline gap-3.5 py-1 text-ink no-underline">
                <span className="tabular w-16 shrink-0 text-support text-stone">{it.time}</span>
                <span className="flex flex-col"><span className="text-[16px] font-semibold">{it.title}</span>{it.sub ? <span className="text-support text-stone">{it.sub}</span> : null}</span>
              </Link>
            ))}
            {(byDay.get(d) ?? []).length === 0 ? <div className="py-2 text-support text-stone">Nothing planned. Add something you&rsquo;d love to do together.</div> : null}
          </section>
        ))}
      </Main>
      <BottomActions><LinkButton href="/together/events/new" icon="plus">Add an event</LinkButton></BottomActions>
    </>
  );
}
