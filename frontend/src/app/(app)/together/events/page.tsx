'use client';

import { TODAY } from '@/lib/api';
import { useEvents } from '@/lib/hooks';
import { iso, relativeDay, time12 } from '@/lib/dates';
import { DateRow } from '@/components/DateRow';
import { Main } from '@/components/Screen';
import { BottomActions, EmptyState, LinkButton, Section, Skeleton, Title, TopBar } from '@/components/ui';

export default function Events() {
  const events = useEvents(); const today = iso(TODAY);
  if (!events.data) return <><TopBar back="Our space" backHref="/together" /><Main><Skeleton /></Main></>;
  const up = events.data.filter((e) => !e.done && e.date >= today).sort((a, b) => (a.date + (a.start ?? '')).localeCompare(b.date + (b.start ?? '')));
  const past = events.data.filter((e) => e.done || e.date < today).sort((a, b) => b.date.localeCompare(a.date));
  if (events.data.length === 0) return <><TopBar back="Our space" backHref="/together" /><Main><div className="pt-2"><Title>Events</Title></div><EmptyState ghost="lines" title="Nothing planned yet." text="Add something you’d love to do together." cta={<LinkButton href="/together/events/new" icon="plus">Add an event</LinkButton>} /></Main></>;
  return (
    <>
      <TopBar back="Our space" backHref="/together" />
      <Main>
        <div className="pt-2"><Title>Events</Title></div>
        {up.length ? <Section label="Coming up" className="mt-5">{up.map((e) => <DateRow key={e.id} date={e.date} title={e.title} sub={`${relativeDay(e.date, today)}, ${time12(e.start)}${e.date === today && e.location ? ` · ${e.location}` : ''}`} href={`/together/events/${e.id}`} />)}</Section> : null}
        {past.length ? <Section label="Earlier" className="mb-4 mt-6">{past.map((e) => <DateRow key={e.id} date={e.date} title={e.title} sub="Done together" past href={`/together/events/${e.id}`} action="Save a moment from it" />)}</Section> : null}
      </Main>
      <BottomActions><LinkButton href="/together/events/new" icon="plus">Add an event</LinkButton></BottomActions>
    </>
  );
}
