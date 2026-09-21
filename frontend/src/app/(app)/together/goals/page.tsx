'use client';

import Link from 'next/link';
import { useGoals } from '@/lib/hooks';
import { goalSub, sumOf } from '@/lib/goals';
import type { Goal } from '@/lib/types';
import { Icon } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { Bar, BottomActions, EmptyState, LinkButton, Micro, Para, Skeleton, Title, TopBar } from '@/components/ui';


function GoalRow({ g }: { g: Goal }) {
  const pct = (sumOf(g) / g.target) * 100;
  return (
    <Link href={`/together/goals/${g.id}`} className="press flex items-center gap-3.5 border-b border-line py-[18px] text-ink no-underline">
      <span className="flex grow flex-col gap-0.5">
        <span className="text-bodylg font-semibold tracking-[-0.01em]">{g.title}</span>
        {g.done ? <span className="flex items-center gap-1.5 text-support font-semibold text-green"><Icon name="check" size={16} strokeWidth={2} />Done together</span>
          : <span className="mt-2 flex flex-col gap-2"><Bar pct={pct} label={goalSub(g)} /><span className="text-support text-stone">{goalSub(g)}</span></span>}
      </span>
      <Icon name="right" size={18} className="text-stone" />
    </Link>
  );
}

export default function Goals() {
  const goals = useGoals();
  if (!goals.data) return <><TopBar back="Our space" backHref="/together" /><Main><Skeleton /></Main></>;
  const active = goals.data.filter((g) => !g.done); const done = goals.data.filter((g) => g.done);
  if (goals.data.length === 0) return <><TopBar back="Our space" backHref="/together" /><Main><div className="pt-2"><Title>Goals</Title></div><EmptyState ghost="bars" title="What would you like to build together?" text="A goal belongs to both of you. Start with something small." cta={<LinkButton href="/together/goals/new" icon="plus">Add a goal</LinkButton>} /></Main></>;
  return (
    <>
      <TopBar back="Our space" backHref="/together" />
      <Main>
        <div className="pt-2"><Title>Goals</Title><Para className="mt-1.5">What we&rsquo;re working toward, together.</Para></div>
        <section className="mt-3 flex flex-col">{active.map((g) => <GoalRow key={g.id} g={g} />)}</section>
        {done.length ? <section className="mb-4 mt-[22px] flex flex-col"><Micro>Finished</Micro>{done.map((g) => <GoalRow key={g.id} g={g} />)}</section> : null}
      </Main>
      <BottomActions><LinkButton href="/together/goals/new" icon="plus">Add a goal</LinkButton></BottomActions>
    </>
  );
}
