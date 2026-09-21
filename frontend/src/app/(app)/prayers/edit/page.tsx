'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useSavePoints, useWeek } from '@/lib/hooks';
import type { PrayerPoint } from '@/lib/types';
import { BareInput, BareTextarea, Button, ComposeBar, Skeleton } from '@/components/ui';

export default function EditPrayerPage() { return <Suspense fallback={null}><EditPrayer /></Suspense>; }

/** Feels like writing a note, not filling a form: bare fields, big type. */
function EditPrayer() {
  const params = useSearchParams(); const id = params.get('id');
  const week = useWeek(); const save = useSavePoints(); const router = useRouter();
  const [title, setTitle] = useState(''); const [text, setText] = useState(''); const [scripture, setScripture] = useState('');
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    if (!week.data || loaded) return;
    const p = week.data.points.find((x) => x.id === id);
    if (p) { setTitle(p.title); setText(p.text); setScripture(p.scripture ?? ''); }
    setLoaded(true);
  }, [week.data, id, loaded]);
  if (!week.data || !loaded) return <div className="px-6 pt-16"><Skeleton /></div>;
  const points = week.data.points; const idx = points.findIndex((x) => x.id === id);
  const done = () => {
    const t = title.trim(); if (!t) { router.back(); return; }
    const p: PrayerPoint = { id: id ?? Math.random().toString(36).slice(2, 10), title: t, text: text.trim(), scripture: scripture.trim() || undefined, position: idx === -1 ? points.length : idx };
    const next = idx === -1 ? [...points, p] : points.map((x) => (x.id === id ? { ...x, ...p } : x));
    save.mutate([next], { onSuccess: () => router.replace('/prayers/set') });
  };
  const remove = () => save.mutate([points.filter((x) => x.id !== id)], { onSuccess: () => router.replace('/prayers/set') });
  return (
    <>
      <ComposeBar cancelHref="/prayers/set" label={idx === -1 ? 'New prayer' : `Prayer ${idx + 1} of ${points.length}`} done="Done" onDone={done} busy={save.isPending} />
      <form onSubmit={(e) => { e.preventDefault(); done(); }} className="flex grow flex-col gap-[22px] px-6 pt-5">
        <BareInput label="Title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="What are we praying for?" autoFocus={idx === -1} className="h-11 text-title" />
        <BareTextarea label="Prayer" value={text} onChange={(e) => setText(e.target.value)} rows={4} placeholder="Write it the way you’d say it." className="min-h-[124px] text-[19px] leading-[1.6]" />
        <div className="border-t border-line pt-[18px]">
          <BareInput label="Scripture, optional" value={scripture} onChange={(e) => setScripture(e.target.value)} placeholder="Book, chapter and verse" className="h-11 text-bodylg font-semibold text-plum" />
        </div>
        {idx !== -1 ? <div className="border-t border-line pt-1.5"><Button variant="danger" icon="trash" onClick={remove} className="w-auto justify-start px-0">Delete this prayer</Button></div> : null}
        <button type="submit" className="sr-only">Done</button>
      </form>
    </>
  );
}
