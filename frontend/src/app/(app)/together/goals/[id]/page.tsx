'use client';

import { useParams } from 'next/navigation';
import { useState } from 'react';
import { ME, TODAY } from '@/lib/api';
import { useAddProgress, useCouple, useGoal } from '@/lib/hooks';
import { dayNum, daysUntil, naira, range, shortMonth } from '@/lib/dates';
import { sumOf } from '@/lib/goals';
import { Main } from '@/components/Screen';
import { Bar, BottomActions, Button, Initial, Micro, Para, Section, Sheet, Skeleton, Title, TopBar } from '@/components/ui';

export default function GoalDetail() {
  const { id } = useParams<{ id: string }>();
  const goal = useGoal(id); const couple = useCouple(); const add = useAddProgress();
  const [open, setOpen] = useState(false); const [amount, setAmount] = useState('');
  const g = goal.data;
  if (!g) return <><TopBar back="Goals" backHref="/together/goals" /><Main><Skeleton /></Main></>;
  const total = sumOf(g); const pct = (total / g.target) * 100; const left = Math.max(0, g.target - total);
  const weeks = Math.max(0, Math.round(daysUntil(g.end, TODAY) / 7));
  const fmt = (n: number) => (g.unit === 'naira' ? naira(n) : `${n} ${g.unitLabel ?? ''}`.trim());
  const partner = couple.data?.partner?.name ?? 'Partner';
  const submit = () => { const n = Number(amount.replace(/[^\d]/g, '')); if (n > 0) add.mutate([g.id, n], { onSuccess: () => { setOpen(false); setAmount(''); } }); };
  return (
    <>
      <TopBar back="Goals" backHref="/together/goals" />
      <Main>
        <div className="pt-3"><Micro>{range(g.start, g.end)}</Micro></div>
        <Title className="mt-2">{g.title}</Title>
        {g.why ? <Para className="mt-1.5">{g.why}</Para> : null}
        <section className="mt-6 flex flex-col gap-3">
          <div className="flex items-baseline gap-2"><span className="tabular text-display tracking-[-0.03em]">{fmt(total)}</span><span className="text-[15px] text-stone">of {fmt(g.target)}</span></div>
          <Bar pct={pct} label={`${fmt(total)} of ${fmt(g.target)}`} />
          <div className="text-support text-stone">{g.done ? 'Done together.' : `${fmt(left)} to go, with ${weeks} ${weeks === 1 ? 'week' : 'weeks'} left.`}</div>
        </section>
        {g.progress.length ? (
          <Section label="What we’ve added" className="mb-4 mt-6">
            <ol className="m-0 list-none p-0">{[...g.progress].sort((a, b) => b.date.localeCompare(a.date)).map((p, i, arr) => (
              <li key={p.id} className={`flex min-h-[52px] items-center gap-3 ${i === arr.length - 1 ? '' : 'border-b border-line'}`}><Initial letter={(p.by === ME ? (couple.data?.me.name ?? 'F') : partner)[0]} /><span className="grow text-[16px]">{fmt(p.amount)}</span><span className="tabular text-support text-stone">{dayNum(p.date)} {shortMonth(p.date)}</span></li>
            ))}</ol>
          </Section>
        ) : null}
      </Main>
      {!g.done ? <BottomActions><Button icon="plus" onClick={() => setOpen(true)}>Add progress</Button></BottomActions> : null}
      <Sheet open={open} onClose={() => setOpen(false)} title="What did you add?" labelledBy="prog-h">
        <label htmlFor="amt" className="sr-only">Amount</label>
        <input id="amt" inputMode="numeric" autoFocus value={amount} onChange={(e) => setAmount(e.target.value)} placeholder={g.unit === 'naira' ? '₦40,000' : '1'} className="tabular h-16 w-full rounded-input border border-edge bg-surface px-4 text-[28px] font-semibold text-ink" />
        <div className="flex flex-col gap-1"><Button onClick={submit} loading={add.isPending}>Add it</Button><Button variant="text" onClick={() => setOpen(false)}>Not now</Button></div>
      </Sheet>
    </>
  );
}
