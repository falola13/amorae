'use client';

import { useCouple, useNetwork, useWeek } from '@/lib/hooks';
import { ME, readQueue } from '@/lib/api';
import { range } from '@/lib/dates';
import { Main } from '@/components/Screen';
import { PrayerRow } from '@/components/PrayerRow';
import { BottomActions, EmptyState, LinkButton, Micro, Para, Segments, Skeleton, Title } from '@/components/ui';

export default function Prayers() {
  const week = useWeek(); const couple = useCouple(); const net = useNetwork();
  const w = week.data; const partner = couple.data?.partner?.name ?? 'your partner';
  if (!w) return <Main><div className="pt-4"><Skeleton lines={5} /></div></Main>;
  if (w.status !== 'published') {
    const mine = w.setterId === ME;
    return (
      <Main>
        <div className="pt-1.5"><Micro>{range(w.start, w.end, true)}</Micro></div>
        <Title className="mt-2">This week&rsquo;s prayers</Title>
        <EmptyState ghost="dots" title={mine ? 'Nothing shared yet.' : `${partner} is still writing.`} text={mine ? 'Write what’s on your heart and share it when it feels ready.' : 'You’ll get a quiet nudge the moment the week is shared.'}
          cta={mine ? <LinkButton href="/prayers/set">Set this week&rsquo;s prayers</LinkButton> : <LinkButton href="/prayers/mode?quiet=1" icon="moon">Enter quiet prayer</LinkButton>} />
      </Main>
    );
  }
  const done = w.myCompleted.length; const total = w.points.length;
  const queued = new Set(net.online ? [] : readQueue().map((q) => (q.payload as { pointId?: string }).pointId));
  return (
    <>
      <Main>
        <div className="pt-1.5"><Micro>{range(w.start, w.end, true)} &middot; set by {w.setterId === ME ? 'you' : partner}</Micro></div>
        <Title className="mt-2">This week&rsquo;s prayers</Title>
        <div className="mt-5 flex flex-col gap-2.5"><Segments total={total} done={done} /><Para size="support">{done} of {total} prayed</Para></div>
        <div className="mt-2 flex flex-col border-t border-line">
          {w.points.map((p, i) => <PrayerRow key={p.id} p={p} index={i} done={w.myCompleted.includes(p.id)} href={`/prayers/${p.id}`} note={queued.has(p.id) ? 'Will sync when you’re back online' : undefined} />)}
        </div>
      </Main>
      <BottomActions><LinkButton href="/prayers/mode">{done === total ? 'Open prayer mode' : done === 0 ? 'Begin praying' : 'Continue praying'}</LinkButton></BottomActions>
    </>
  );
}
