'use client';

import Link from 'next/link';
import { TODAY } from '@/lib/api';
import { useAppreciations, useChallenge, useCouple, useEvents, useGoals, useJournal, useMemories, useMilestones } from '@/lib/hooks';
import { daysUntil, iso, longDate, relativeDay, time12 } from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { Para, Row, Section, Title } from '@/components/ui';

export default function Together() {
  const couple = useCouple(); const events = useEvents(); const goals = useGoals(); const challenge = useChallenge(); const journal = useJournal(); const memories = useMemories(); const milestones = useMilestones(); const appr = useAppreciations();
  const partner = couple.data?.partner?.name ?? 'your partner';
  const today = iso(TODAY);
  const up = (events.data ?? []).filter((e) => !e.done && e.date >= today).sort((a, b) => a.date.localeCompare(b.date));
  const next = up[0];
  const active = (goals.data ?? []).filter((g) => !g.done).length;
  const day = challenge.data?.days.find((d) => !d.done && !d.skipped);
  const j = journal.data?.[0]; const m = memories.data?.[0];
  const upcomingDates = (milestones.data ?? []).map((d) => ({ ...d, in: daysUntil(nextOccurrence(d.date, today), TODAY) })).filter((d) => d.in >= 0).sort((a, b) => a.in - b.in)[0];
  const lastAppr = appr.data?.[0];
  return (
    <Main>
      <div className="flex items-start justify-between pt-1.5">
        <div><Title>Our space</Title><Para className="mt-1.5">Everything we&rsquo;re building together.</Para></div>
        <Link href="/together/events/new" aria-label="Add something to our space" className="press -mr-2.5 -mt-1 flex h-11 w-11 items-center justify-center text-ink"><Icon name="plus" size={24} /></Link>
      </div>
      <Section label="Plan" className="mt-[22px]">
        <Row icon="calendar" title="Calendar" sub={next ? `${next.title} ${relativeDay(next.date, today).toLowerCase()} at ${time12(next.start)}` : 'Nothing planned yet'} href="/together/calendar" />
        <Row icon="pin" title="Events" sub={up.length ? `${up.length} coming up` : 'Add something you’d love to do'} href="/together/events" last />
      </Section>
      <Section label="Grow" className="mt-[22px]">
        <Row icon="target" title="Goals" sub={active ? `${active} we’re working on` : 'What would you like to build together?'} href="/together/goals" />
        <Row icon="flag" title="Challenges" sub={challenge.data ? (day ? `${challenge.data.title}, day ${day.n}` : `${challenge.data.title}, finished`) : ''} href="/together/challenges" last />
      </Section>
      <Section label="Connect" className="mt-[22px]">
        <Row icon="pencil" title="Journal" sub={j ? `${j.authorId === couple.data?.me.id ? 'You' : partner} wrote ${relativeDay(j.date, today).toLowerCase()}` : 'Write something for us'} href="/together/journal" />
        <Row icon="note" title="Appreciation" sub={lastAppr ? `${lastAppr.fromId === couple.data?.me.id ? 'You' : partner}, ${relativeDay(lastAppr.date, today).toLowerCase()}` : 'Something I appreciate about you'} href="/together/appreciation" last />
      </Section>
      <Section label="Remember" className="mb-4 mt-[22px]">
        <Row icon="image" title="Memories" sub={m ? `Last saved ${longDate(m.date)}` : 'Your story starts here'} href="/together/memories" />
        <Row icon="bookmark" title="Milestones" sub={upcomingDates ? `${upcomingDates.title} in ${upcomingDates.in} days` : 'The dates that matter to us'} href="/together/milestones" last />
      </Section>
    </Main>
  );
}

function nextOccurrence(date: string, today: string) {
  if (date >= today) return date;
  const [, m, d] = date.split('-'); const y = Number(today.slice(0, 4));
  const thisYear = `${y}-${m}-${d}`; return thisYear >= today ? thisYear : `${y + 1}-${m}-${d}`;
}
