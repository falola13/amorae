'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useParams } from 'next/navigation';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { useCouple } from '@/features/couple/hooks';
import { useAddProgress, useGoal } from '@/features/together/hooks';
import { sumOf } from '@/features/together/goals';
import { progressSchema } from '@/lib/api/schemas';
import { dayNum, daysUntil, naira, range, shortMonth } from '@/lib/dates';
import { routes } from '@/lib/routes';
import { today } from '@/lib/today';
import { Main } from '@/components/layout/screen';
import { QueryState } from '@/components/ui/query-state';
import { BareInput, Bar, BottomActions, Button, Initial, Micro, Para, Section, Sheet, Skeleton, Title, TopBar } from '@/components/ui/kit';

// The sheet's amount field is still free-typed text while the couple is
// entering it; strip non-digits and pipe the result through progressSchema's
// own "amount" rule so the message and the positive-number check stay in sync.
const progressFormSchema = z.object({
  amount: z.string().transform((v) => Number(v.replace(/[^\d]/g, ''))).pipe(progressSchema.shape.amount),
});
type ProgressFormInput = z.input<typeof progressFormSchema>;
type ProgressFormOutput = z.output<typeof progressFormSchema>;

export default function GoalDetail() {
  const { id } = useParams<{ id: string }>();
  const goal = useGoal(id); const couple = useCouple(); const add = useAddProgress();
  const [open, setOpen] = useState(false);
  const { register, handleSubmit, reset, formState: { errors } } = useForm<ProgressFormInput, unknown, ProgressFormOutput>({
    resolver: zodResolver(progressFormSchema),
    defaultValues: { amount: '' },
  });
  const close = () => { reset(); setOpen(false); };
  return (
    <>
      <TopBar back="Goals" backHref={routes.goals} />
      <QueryState queries={[goal]} loading={<Main><Skeleton /></Main>}>
        {(g) => {
          const total = sumOf(g); const pct = (total / g.target) * 100; const left = Math.max(0, g.target - total);
          const weeks = Math.max(0, Math.round(daysUntil(g.end_date, today()) / 7));
          const fmt = (n: number) => (g.unit === 'naira' ? naira(n) : `${n} ${g.unit_label ?? ''}`.trim());
          const partner = couple.data?.partner?.display_name ?? 'Partner';
          const onSubmit = (v: ProgressFormOutput) => add.mutate({ id: g.id, amount: v.amount }, { onSuccess: close });
          return (
            <>
              <Main>
                <div className="pt-3"><Micro>{range(g.start_date, g.end_date)}</Micro></div>
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
                      <li key={p.id} className={`flex min-h-[52px] items-center gap-3 ${i === arr.length - 1 ? '' : 'border-b border-line'}`}><Initial letter={(p.user_id === couple.data?.me.id ? (couple.data?.me.display_name ?? 'F') : partner)[0]} /><span className="grow text-[16px]">{fmt(p.amount)}</span><span className="tabular text-support text-stone">{dayNum(p.date)} {shortMonth(p.date)}</span></li>
                    ))}</ol>
                  </Section>
                ) : null}
              </Main>
              {!g.done ? <BottomActions><Button icon="plus" onClick={() => setOpen(true)}>Add progress</Button></BottomActions> : null}
              <Sheet open={open} onClose={close} title="What did you add?" labelledBy="prog-h">
                <form method="post" onSubmit={handleSubmit(onSubmit)} className="contents" noValidate>
                  <BareInput label="Amount" hideLabel inputMode="numeric" autoFocus placeholder={g.unit === 'naira' ? '₦40,000' : '1'} className="tabular h-16 w-full rounded-input border border-edge bg-surface px-4 text-[28px] font-semibold text-ink" error={errors.amount?.message} {...register('amount')} />
                  <div className="flex flex-col gap-1"><Button type="submit" loading={add.isPending}>Add it</Button><Button type="button" variant="text" onClick={close}>Not now</Button></div>
                </form>
              </Sheet>
            </>
          );
        }}
      </QueryState>
    </>
  );
}
