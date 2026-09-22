'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useCouple } from '@/features/couple/hooks';
import { usePublish, useSavePoints, useWeek } from '@/features/prayers/hooks';
import { useDragReorder } from '@/features/prayers/use-drag-reorder';
import { range } from '@/lib/dates';
import { routes } from '@/lib/routes';
import type { PrayerPoint } from '@/lib/api/types';
import { Icon } from '@/components/icons';
import { Main } from '@/components/layout/screen';
import { Button, LinkButton, Micro, Para, Sheet, Skeleton } from '@/components/ui/kit';
import { QueryState } from '@/components/ui/query-state';

const MAX = 10;

/** Setter's list: reorder by drag handle (pointer) or the up/down keys on a focused handle. */
export default function SetPrayers() {
  const week = useWeek(); const couple = useCouple(); const save = useSavePoints(); const publish = usePublish(); const router = useRouter();
  const [confirm, setConfirm] = useState(false);
  const [saved, setSaved] = useState(true);
  const partner = couple.data?.partner?.display_name ?? 'your partner';
  const commit = (next: PrayerPoint[]) => { setSaved(false); save.mutate(next, { onSuccess: () => setSaved(true) }); };
  const drag = useDragReorder(week.data?.points ?? null, commit);

  return (
    <QueryState queries={[week]} loading={<Main><div className="pt-4"><Skeleton /></div></Main>}>
      {(w) => {
        const items = drag.items ?? w.points;
        const doPublish = () => publish.mutate(undefined, { onSuccess: () => router.replace(routes.home) });
        return (
          <>
            <div className="flex h-11 shrink-0 items-center justify-between px-3">
              <Link href={routes.home} aria-label="Close" className="press flex h-11 w-11 items-center justify-center text-ink"><Icon name="x" size={22} /></Link>
              <div role="status" className="flex items-center gap-1.5 pr-3 text-[13px] text-stone">{saved ? <><Icon name="check" size={16} strokeWidth={2} className="text-green" />Draft saved</> : 'Saving…'}</div>
            </div>
            <Main>
              <div className="pt-3"><Micro>{range(w.week_start, w.week_end)}</Micro></div>
              <h1 className="m-0 mt-2 text-[30px] font-semibold leading-[1.18] tracking-[-0.022em]">It&rsquo;s your week.</h1>
              <Para className="mt-1.5">Write what&rsquo;s on your heart. Up to {MAX} prayers.</Para>
              <ol className="m-0 mt-6 list-none border-t border-line p-0" {...drag.listProps}>
                {items.map((p, i) => (
                  <li key={p.id} className="flex min-h-[72px] items-center gap-3 border-b border-line">
                    <span className="tabular w-[22px] self-start pt-[17px] text-[13px] font-semibold text-stone">{String(i + 1).padStart(2, '0')}</span>
                    <Link href={routes.prayerEdit(p.id)} className="press flex min-h-[72px] grow flex-col justify-center gap-0.5 text-ink no-underline">
                      <span className="text-bodylg font-semibold tracking-[-0.01em]">{p.title || 'Untitled prayer'}</span>
                      {p.text ? <span className="text-support text-stone">{p.text}</span> : null}
                    </Link>
                    <button type="button" aria-label={`Reorder ${p.title}. Use the arrow keys to move it.`} {...drag.handleProps(i)}
                      onKeyDown={(e) => { if (e.key === 'ArrowUp') { e.preventDefault(); drag.moveByKey(i, i - 1); } if (e.key === 'ArrowDown') { e.preventDefault(); drag.moveByKey(i, i + 1); } }}
                      className="press -mr-3 flex h-11 w-11 touch-none cursor-grab items-center justify-center text-stone active:cursor-grabbing"><Icon name="grip" size={22} strokeWidth={2.2} /></button>
                  </li>
                ))}
                {items.length < MAX ? (
                  <li><Link href={routes.prayerEdit()} className="press flex h-[60px] items-center gap-3 text-[16px] font-semibold text-plum no-underline"><span className="flex w-[22px]"><Icon name="plus" size={20} strokeWidth={1.8} /></span>Add a prayer</Link></li>
                ) : <li className="py-4 text-support text-stone">That&rsquo;s the ten for this week.</li>}
              </ol>
            </Main>
            <div className="flex shrink-0 flex-col gap-2 px-6 pt-4 pb-safe">
              <Button onClick={() => setConfirm(true)} disabled={items.length === 0 || !saved}>Publish prayers</Button>
              <LinkButton href={routes.home} variant="text">Save draft and finish later</LinkButton>
            </div>

            <Sheet open={confirm} onClose={() => setConfirm(false)} title="Ready to share these prayers?" labelledBy="pub-h">
              <Para>{partner} will see all {items.length}. Once {partner === 'your partner' ? 'they start' : 'she starts'} praying, they can&rsquo;t be changed, so the week stays the same for both of you.</Para>
              <ol className="m-0 list-none border-y border-line py-1">
                {items.map((p, i) => <li key={p.id} className="flex h-10 items-center gap-3.5 text-[16px] font-medium"><span className="tabular w-[22px] text-[13px] font-semibold text-stone">{String(i + 1).padStart(2, '0')}</span>{p.title}</li>)}
              </ol>
              <div className="flex flex-col gap-1 pt-1">
                <Button onClick={doPublish} loading={publish.isPending}>{publish.isPending ? 'Sharing' : `Share with ${partner}`}</Button>
                <Button variant="text" onClick={() => setConfirm(false)}>Keep editing</Button>
              </div>
            </Sheet>
          </>
        );
      }}
    </QueryState>
  );
}
