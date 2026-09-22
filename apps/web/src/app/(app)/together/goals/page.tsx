'use client';

import Link from 'next/link';
import { useGoals } from '@/features/together/hooks';
import { goalSub, sumOf } from '@/features/together/goals';
import type { Goal } from '@/lib/api/types';
import { routes } from '@/lib/routes';
import { Icon } from '@/components/icons';
import { Main } from '@/components/layout/screen';
import { QueryState } from '@/components/ui/query-state';
import { Bar, BottomActions, EmptyState, LinkButton, Micro, Para, Skeleton, Title, TopBar } from '@/components/ui/kit';


function GoalRow({ g }: { g: Goal }) {
  const pct = (sumOf(g) / g.target) * 100;
  return (
    <Link href={routes.goal(g.id)} className="press flex items-center gap-3.5 border-b border-line py-[18px] text-ink no-underline">
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
  return (
    <>
      <TopBar back="Our space" backHref={routes.together} />
      <Main>
        <div className="pt-2"><Title>Goals</Title></div>
        <QueryState queries={[goals]} loading={<Skeleton />}>
          {(goalsData) => {
            if (goalsData.length === 0) {
              return <EmptyState ghost="bars" title="What would you like to build together?" text="A goal belongs to both of you. Start with something small." cta={<LinkButton href={routes.goalNew} icon="plus">Add a goal</LinkButton>} />;
            }
            const active = goalsData.filter((g) => !g.done); const done = goalsData.filter((g) => g.done);
            return (
              <>
                <Para className="mt-1.5">What we&rsquo;re working toward, together.</Para>
                <section className="mt-3 flex flex-col">{active.map((g) => <GoalRow key={g.id} g={g} />)}</section>
                {done.length ? <section className="mb-4 mt-[22px] flex flex-col"><Micro>Finished</Micro>{done.map((g) => <GoalRow key={g.id} g={g} />)}</section> : null}
              </>
            );
          }}
        </QueryState>
      </Main>
      <BottomActions><LinkButton href={routes.goalNew} icon="plus">Add a goal</LinkButton></BottomActions>
    </>
  );
}
