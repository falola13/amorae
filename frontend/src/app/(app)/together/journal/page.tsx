'use client';

import { useState } from 'react';
import { TODAY } from '@/lib/api';
import { useAddJournal, useCouple, useJournal } from '@/lib/hooks';
import { iso, relativeDay } from '@/lib/dates';
import type { JournalEntry } from '@/lib/types';
import { Main } from '@/components/Screen';
import { BareTextarea, BottomActions, Button, EmptyState, Initial, Para, Sheet, Skeleton, Title, TopBar, cx } from '@/components/ui';

const TAGS: JournalEntry['tag'][] = ['Gratitude', 'Reflection', 'Memory', 'Appreciation', 'Plans'];

export default function Journal() {
  const journal = useJournal(); const couple = useCouple(); const add = useAddJournal();
  const [open, setOpen] = useState(false); const [text, setText] = useState(''); const [tag, setTag] = useState<JournalEntry['tag']>('Gratitude');
  const me = couple.data?.me; const partner = couple.data?.partner?.name ?? 'Partner';
  const today = iso(TODAY);
  if (!journal.data) return <><TopBar back="Our space" backHref="/together" /><Main><Skeleton /></Main></>;
  const submit = () => { if (text.trim()) add.mutate([tag, text.trim()], { onSuccess: () => { setOpen(false); setText(''); } }); };
  const composer = (
    <Sheet open={open} onClose={() => setOpen(false)} title="Write something for us." labelledBy="j-h">
      <div role="radiogroup" aria-label="Kind of entry" className="-mx-1 flex flex-wrap gap-1">{TAGS.map((t) => <button key={t} type="button" role="radio" aria-checked={tag === t} onClick={() => setTag(t)} className={cx('press h-9 rounded-full px-3.5 text-[14px] font-semibold', tag === t ? 'bg-plum-tint text-plum' : 'text-stone')}>{t}</button>)}</div>
      <BareTextarea label="Entry" hideLabel value={text} onChange={(e) => setText(e.target.value)} rows={4} autoFocus placeholder="A line or two, in your own words." className="min-h-[110px] text-[19px] leading-[1.6]" />
      <div className="flex flex-col gap-1"><Button onClick={submit} loading={add.isPending}>Save to our journal</Button><Button variant="text" onClick={() => setOpen(false)}>Not now</Button></div>
    </Sheet>
  );
  if (journal.data.length === 0) return <><TopBar back="Our space" backHref="/together" /><Main><div className="pt-2"><Title>Journal</Title></div><EmptyState ghost="lines" title="A notebook for the two of you." text="Gratitude, a memory, something you noticed. Nothing here is public." cta={<Button icon="pencil" onClick={() => setOpen(true)}>Write something for us</Button>} /></Main>{composer}</>;
  return (
    <>
      <TopBar back="Our space" backHref="/together" />
      <Main>
        <div className="pt-2"><Title>Journal</Title><Para className="mt-1.5">A shared notebook, only for the two of you.</Para></div>
        <div className="mt-2 flex flex-col pb-4">
          {journal.data.map((j, i) => (
            <article key={j.id} className={cx('flex flex-col gap-2 py-[18px]', i < journal.data!.length - 1 && 'border-b border-line')}>
              <div className="flex items-center gap-2.5"><Initial letter={(j.authorId === me?.id ? (me?.name ?? 'Y') : partner)[0]} size={26} /><span className="grow text-support text-stone"><span className="font-semibold text-ink">{j.authorId === me?.id ? 'You' : partner}</span> &middot; {relativeDay(j.date, today)}</span><span className="text-[12px] font-bold uppercase tracking-[0.06em] text-stone">{j.tag}</span></div>
              <p className="m-0 text-bodylg leading-[1.6]" data-selectable>{j.text}</p>
            </article>
          ))}
        </div>
      </Main>
      <BottomActions><Button icon="pencil" onClick={() => setOpen(true)}>Write something for us</Button></BottomActions>
      {composer}
    </>
  );
}
