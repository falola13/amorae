'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { TODAY } from '@/lib/api';
import { useSaveGoal } from '@/lib/hooks';
import { addDays, iso, longDate, naira } from '@/lib/dates';
import { BareTextarea, ComposeBar } from '@/components/ui';
import { PickRow } from '@/components/PickRow';

export default function NewGoal() {
  const router = useRouter(); const save = useSaveGoal();
  const [title, setTitle] = useState(''); const [why, setWhy] = useState(''); const [target, setTarget] = useState(''); const [isMoney, setIsMoney] = useState(true);
  const [start, setStart] = useState(iso(TODAY)); const [end, setEnd] = useState(iso(addDays(TODAY, 90)));
  const done = () => {
    if (!title.trim()) { router.back(); return; }
    const t = Number(target.replace(/[^\d]/g, '')) || 1;
    save.mutate([{ title: title.trim(), why: why.trim() || undefined, target: t, unit: isMoney ? 'naira' : 'count', start, end }], { onSuccess: () => router.replace('/together/goals') });
  };
  return (
    <>
      <ComposeBar cancelHref="/together/goals" label="New goal" onDone={done} busy={save.isPending} />
      <form onSubmit={(e) => { e.preventDefault(); done(); }} className="flex grow flex-col px-6 pt-5">
        <BareTextarea label="What would you like to build together?" value={title} onChange={(e) => setTitle(e.target.value)} rows={2} placeholder="Save &#8358;500,000 together" autoFocus className="min-h-[72px] text-title leading-[1.25]" />
        <div className="mt-4"><BareTextarea label="Why it matters" value={why} onChange={(e) => setWhy(e.target.value)} rows={2} placeholder="So December feels generous instead of tight." className="min-h-[56px] text-bodylg" /></div>
        <div className="mt-[18px] flex flex-col border-t border-line">
          <PickRow icon="target" label="Target" value={target} type="text" onChange={setTarget} placeholder={isMoney ? '₦500,000' : 'A number'} empty={target ? (isMoney ? naira(Number(target.replace(/[^\d]/g, '')) || 0) : target) : undefined} />
          <div className="flex h-[54px] items-center gap-3.5 border-b border-line text-[16px] font-medium">
            <span className="w-[22px]" /><span className="grow">Counted in</span>
            <div role="radiogroup" aria-label="Unit" className="flex gap-1">
              {[['Naira', true], ['Steps', false]].map(([l, v]) => <button key={String(l)} type="button" role="radio" aria-checked={isMoney === v} onClick={() => setIsMoney(v as boolean)} className={`press h-9 rounded-full px-3.5 text-[14px] font-semibold ${isMoney === v ? 'bg-plum-tint text-plum' : 'text-stone'}`}>{l}</button>)}
            </div>
          </div>
          <PickRow icon="calendar" label="Starts" value={start} type="date" onChange={setStart} empty={longDate(start)} />
          <PickRow icon="calendar" label="Ends" value={end} type="date" onChange={setEnd} empty={longDate(end)} last />
        </div>
        <button type="submit" className="sr-only">Save</button>
      </form>
    </>
  );
}
