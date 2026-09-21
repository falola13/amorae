'use client';

import Link from 'next/link';
import { ME, PARTNER, TODAY } from '@/lib/api';
import { useCouple, useEvents, useGoals, useWeek } from '@/lib/hooks';
import { greeting, iso, naira, range, relativeDay, time12, weekdayDate } from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { Initial, LinkButton, Micro, Para, Row, Section, Segments, Skeleton, Title } from '@/components/ui';

export default function Home() {
  const couple = useCouple(); const week = useWeek(); const events = useEvents(); const goals = useGoals();
  const me = couple.data?.me.name ?? ''; const partner = couple.data?.partner?.name ?? 'your partner';
  const w = week.data;
  const today = iso(TODAY);
  const upcoming = (events.data ?? []).filter((e) => !e.done && e.date >= today).sort((a, b) => (a.date + (a.start ?? '')).localeCompare(b.date + (b.start ?? '')));
  const tonight = upcoming.find((e) => e.date === today);
  const nextEvent = upcoming.find((e) => e.date !== today);
  const goal = (goals.data ?? []).find((g) => !g.done);
  const goalTotal = goal ? goal.progress.reduce((s, p) => s + p.amount, 0) : 0;
  const planned = upcoming.length + (goal ? 1 : 0) + (w && w.status === 'published' ? 1 : 0);

  if (!w || !couple.data) return <Main><div className="pt-4"><Skeleton lines={4} /></div></Main>;

  // Your week to set
  if (w.setterId === ME && w.status === 'draft') {
    return (
      <Main>
        <div className="pt-1.5"><Micro>{range(w.start, w.end)}</Micro></div>
        <Title size="display" className="mt-2">It&rsquo;s your week.</Title>
        <Para size="lg" className="mt-2.5">What would you like the two of you to pray about? {partner} will see your prayers once you share them.</Para>
        <div className="mt-7"><LinkButton href="/prayers/set">Set this week&rsquo;s prayers</LinkButton></div>
        <div className="mt-7 flex flex-col border-t border-line">
          <InfoRow icon={<Icon name="clock" size={22} className="text-stone" />} label="Last week" value="you prayed 5 of 5" action="View" href="/history" />
          <InfoRow icon={<Icon name="bell" size={22} className="text-stone" />} label="Reminder" value="today, 9:00 pm" action="Change" href="/settings/notifications" />
        </div>
      </Main>
    );
  }

  // Partner is setting
  if (w.status === 'waiting') {
    return (
      <Main>
        <div className="pt-1.5"><Micro>{range(w.start, w.end)}</Micro></div>
        <Title size="display" className="mt-2">Your partner is setting this week&rsquo;s prayers.</Title>
        <Para size="lg" className="mt-2.5">We&rsquo;ll let you know the moment {partner} shares them. Until then, the time is yours.</Para>
        <div className="mt-6"><LinkButton href="/prayers/mode?quiet=1" icon="moon">Enter quiet prayer</LinkButton></div>
        <div className="mt-6 flex flex-col border-t border-line">
          <Row icon="book" title="Last week&rsquo;s prayers" sub="5 prayers you set" href="/history" />
          <Row icon="clock" title="Your prayer history" sub="Every week you&rsquo;ve shared" href="/history" />
          <Row icon="bell" title="Prayer reminder" sub="Every day at 9:00 pm" href="/settings/notifications" last />
        </div>
      </Main>
    );
  }

  const done = w.myCompleted.length; const total = w.points.length;
  const partnerDone = w.partnerCompleted.length;
  return (
    <Main>
      <div className="flex items-start justify-between pt-1.5">
        <div>
          <Micro>{weekdayDate(today)}</Micro>
          <h1 className="m-0 mt-2 text-[30px] font-semibold leading-[1.18] tracking-[-0.022em]"><span className="font-medium text-stone">{greeting(TODAY)},</span><br />{me} &amp; {partner}</h1>
        </div>
        <Link href="/together/events/new" aria-label="Add something to our space" className="press -mr-2 flex h-11 w-11 items-center justify-center rounded-full text-ink"><Icon name="plus" size={24} /></Link>
      </div>

      {tonight ? (
        <Link href={`/together/events/${tonight.id}`} className="press mt-5 flex items-center gap-4 rounded-card border border-line bg-surface py-[18px] pl-5 pr-4 text-ink no-underline">
          <span className="flex grow flex-col gap-0.5"><Micro tone="plum">Tonight</Micro><span className="mt-1 text-[21px] font-semibold tracking-[-0.015em]">{tonight.title}</span><span className="text-[15px] text-stone">{time12(tonight.start)}{tonight.location ? ` · ${tonight.location}` : ''}</span></span>
          <Icon name="right" size={18} className="text-stone" />
        </Link>
      ) : null}

      <Section label="This week" className="mt-6" trailing={<span className="text-[13px] text-stone">{planned} {planned === 1 ? 'thing' : 'things'} planned</span>}>
        <Row icon="book" title="Weekly prayer" sub={`Set by ${w.setterId === PARTNER ? partner : 'you'} · you’ve prayed ${done} of ${total}`} href="/prayers">
          <Segments total={total} done={done} height={4} className="mb-0.5 mt-1.5 w-24" />
        </Row>
        {nextEvent ? <Row icon="calendar" title={nextEvent.title} sub={`${relativeDay(nextEvent.date, today)}, ${time12(nextEvent.start)}`} href={`/together/events/${nextEvent.id}`} /> : null}
        {goal ? <Row icon="target" title={goal.title} sub={goal.unit === 'naira' ? `${naira(goalTotal)} so far` : `${goalTotal} of ${goal.target} ${goal.unitLabel ?? ''}`} href={`/together/goals/${goal.id}`} /> : null}
      </Section>

      <div className="mt-[18px]"><LinkButton href="/prayers/mode">{done === total ? 'Open prayer mode' : done === 0 ? 'Begin praying' : 'Continue praying'}</LinkButton></div>
      <div className="mt-3.5 flex items-center gap-2 pb-4 text-support text-stone"><Initial letter={partner[0]} size={18} />{partner} has prayed {partnerDone} of {total} &middot; reminder 9:00 pm</div>
    </Main>
  );
}

function InfoRow({ icon, label, value, action, href }: { icon: React.ReactNode; label: string; value: string; action: string; href: string }) {
  return (
    <div className="flex min-h-[56px] items-center gap-3 border-b border-line">
      {icon}<div className="grow text-[15px] text-stone">{label} <span className="font-semibold text-ink">{value}</span></div>
      <Link href={href} className="press flex h-11 items-center px-0.5 text-[15px] font-semibold text-plum no-underline">{action}</Link>
    </div>
  );
}
