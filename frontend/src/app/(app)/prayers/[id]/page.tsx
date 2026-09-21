'use client';

import { useParams } from 'next/navigation';
import { useCouple, useSetCompleted, useWeek } from '@/lib/hooks';
import { ME } from '@/lib/api';
import { Main } from '@/components/Screen';
import { Scripture } from '@/components/Scripture';
import { BottomActions, Button, LinkButton, Skeleton, Title, TopBar } from '@/components/ui';

export default function PrayerDetail() {
  const { id } = useParams<{ id: string }>();
  const week = useWeek(); const couple = useCouple(); const complete = useSetCompleted();
  const w = week.data; const p = w?.points.find((x) => x.id === id);
  if (!w || !p) return <Main><div className="pt-4"><Skeleton /></div></Main>;
  const idx = w.points.findIndex((x) => x.id === id);
  const done = w.myCompleted.includes(p.id);
  const setter = w.setterId === ME ? 'you' : (couple.data?.partner?.name ?? 'your partner');
  return (
    <>
      <TopBar back="Prayers" backHref="/prayers" right={<div className="tabular pr-3 text-support font-semibold text-stone">{idx + 1} of {w.points.length}</div>} />
      <Main className="gap-5 pt-5">
        <div className="flex flex-col gap-2.5"><Title size="lg">{p.title}</Title><div className="text-support text-stone">Added by {setter} on Sunday</div></div>
        {p.text ? <p className="m-0 text-[19px] leading-[1.6]">{p.text}</p> : null}
        {p.scripture ? <Scripture reference={p.scripture} verse={p.verse} /> : null}
      </Main>
      <BottomActions>
        {done ? <Button variant="secondary" icon="check" onClick={() => complete.mutate({ pointId: p.id, done: false })}>Prayed &middot; tap to undo</Button>
          : <Button icon="check" onClick={() => complete.mutate({ pointId: p.id, done: true })}>I&rsquo;ve prayed</Button>}
        <LinkButton href={`/prayers/mode?at=${idx}`} variant="text" icon="moon">Open in prayer mode</LinkButton>
      </BottomActions>
    </>
  );
}
