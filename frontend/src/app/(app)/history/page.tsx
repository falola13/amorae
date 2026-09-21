'use client';

import Link from 'next/link';
import { ME } from '@/lib/api';
import { useCouple, useHistory, useWeek } from '@/lib/hooks';
import { longDate, monthName } from '@/lib/dates';
import type { PrayerWeek } from '@/lib/types';
import { Icon } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { EmptyState, ErrorState, LinkButton, Micro, Para, Skeleton, Title, cx } from '@/components/ui';

function Entry({ w, now, last, partner }: { w: PrayerWeek; now?: boolean; last?: boolean; partner: string }) {
  const meta = now ? `${w.points.length} prayers · you ${w.myCompleted.length} of ${w.points.length} so far` : `${w.points.length} prayers · you ${w.myCompleted.length} of ${w.points.length} · ${partner} ${w.partnerCompleted.length} of ${w.points.length}`;
  return (
    <Link href={now ? '/prayers' : `/history/${w.id}`} className="press relative flex min-h-[84px] items-center gap-3 pl-[30px] text-ink no-underline">
      <span aria-hidden="true" className="absolute bottom-0 left-1 top-0 w-px bg-line" />
      <span aria-hidden="true" className={cx('absolute left-0 top-[22px] box-border h-[9px] w-[9px] rounded-full', now ? 'border-2 border-plum bg-plum' : 'border-[1.5px] border-edge bg-bg')} />
      <span className={cx('flex grow flex-col gap-0.5 py-3.5', !last && 'border-b border-line')}>
        <span className="flex items-center gap-2.5 text-bodylg font-semibold tracking-[-0.01em]">{longDate(w.start)}{now ? <span className="rounded-full bg-plum-tint px-2 py-0.5 text-[12px] font-bold text-plum">This week</span> : null}</span>
        <span className="text-[15px]">{w.setterId === ME ? 'You' : partner} set the prayers</span>
        <span className="text-support text-stone">{meta}</span>
      </span>
      <Icon name="right" size={18} className="text-stone" />
    </Link>
  );
}

export default function History() {
  const history = useHistory(); const week = useWeek(); const couple = useCouple();
  const partner = couple.data?.partner?.name ?? 'Your partner';
  if (history.isError) return <Main><div className="pt-1.5"><Title>History</Title></div><ErrorState title="We couldn’t load your history" text="Something went wrong on our side. Your prayers and everything you’ve marked are safe." onRetry={() => history.refetch()} secondary={<LinkButton href="/prayers" variant="text">Go to this week’s prayers</LinkButton>} /></Main>;
  if (!history.data) return <Main><div className="pt-1.5"><Title>History</Title></div><Skeleton lines={4} /></Main>;
  const all = [...(week.data && week.data.status === 'published' ? [week.data] : []), ...history.data];
  if (all.length === 0) return <Main><div className="pt-1.5"><Title>History</Title></div><EmptyState ghost="dots" title="Your first week will appear here" text="Each Sunday, the week you’ve just finished is kept here for the two of you to look back on." cta={<LinkButton href="/prayers">Go to this week’s prayers</LinkButton>} /></Main>;
  const groups: { month: string; weeks: PrayerWeek[] }[] = [];
  for (const w of all) { const m = monthName(w.start); const g = groups[groups.length - 1]; if (g && g.month === m) g.weeks.push(w); else groups.push({ month: m, weeks: [w] }); }
  return (
    <Main>
      <div className="pt-1.5"><Title>History</Title><Para className="mt-1.5">Every prayer week the two of you have shared.</Para></div>
      <div className="mt-2 flex flex-col pb-4">
        {groups.map((g, gi) => (
          <div key={g.month}>
            <div className="pb-1.5 pl-[30px] pt-[18px]"><Micro>{g.month}</Micro></div>
            {g.weeks.map((w, i) => <Entry key={w.id} w={w} now={w.id === week.data?.id} partner={partner} last={gi === groups.length - 1 && i === g.weeks.length - 1} />)}
          </div>
        ))}
      </div>
    </Main>
  );
}
