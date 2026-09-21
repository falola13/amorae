'use client';

import { useState } from 'react';
import { TODAY } from '@/lib/api';
import { useAddMemory, useMemories } from '@/lib/hooks';
import { iso, longDate, monthName } from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { BareInput, BareTextarea, BottomActions, Button, EmptyState, Micro, Sheet, Skeleton, Title, TopBar } from '@/components/ui';

function Photo({ h, label }: { h: number; label: string }) {
  return <div role="img" aria-label={label} className="flex items-center justify-center gap-2 rounded-btn bg-photo text-[13px] font-semibold text-stone" style={{ height: h }}><Icon name="image" size={18} />Photo</div>;
}

export default function Memories() {
  const memories = useMemories(); const add = useAddMemory();
  const [open, setOpen] = useState(false); const [title, setTitle] = useState(''); const [note, setNote] = useState(''); const [where, setWhere] = useState('');
  if (!memories.data) return <><TopBar back="Our space" backHref="/together" /><Main><Skeleton /></Main></>;
  const submit = () => { if (title.trim()) add.mutate([{ title: title.trim(), date: iso(TODAY), location: where.trim() || undefined, note: note.trim() || undefined, hasPhoto: false }], { onSuccess: () => { setOpen(false); setTitle(''); setNote(''); setWhere(''); } }); };
  const composer = (
    <Sheet open={open} onClose={() => setOpen(false)} title="Save a moment from today." labelledBy="m-h">
      <BareInput label="What happened" value={title} onChange={(e) => setTitle(e.target.value)} autoFocus placeholder="Our first Amorae date night" className="h-11 text-[22px] font-semibold tracking-[-0.02em]" />
      <BareInput label="Where, optional" value={where} onChange={(e) => setWhere(e.target.value)} placeholder="Lekki" className="h-10 text-body" />
      <BareTextarea label="A short note, optional" value={note} onChange={(e) => setNote(e.target.value)} rows={2} placeholder="We stayed until they stacked the chairs." className="min-h-[56px] text-body" />
      <div className="flex flex-col gap-1"><Button onClick={submit} loading={add.isPending}>Save this moment</Button><Button variant="text" icon="image">Add a photo</Button></div>
    </Sheet>
  );
  const sorted = [...memories.data].sort((a, b) => b.date.localeCompare(a.date));
  if (sorted.length === 0) return <><TopBar back="Our space" backHref="/together" /><Main><div className="pt-2"><Title>Memories</Title></div><EmptyState ghost="frames" title="Your story starts here." text="Save your first shared moment." cta={<Button icon="plus" onClick={() => setOpen(true)}>Save a moment</Button>} /></Main>{composer}</>;
  let lastMonth = '';
  return (
    <>
      <TopBar back="Our space" backHref="/together" />
      <Main>
        <div className="pt-2"><Title>Memories</Title></div>
        <div className="mt-3.5 flex flex-col gap-[26px] pb-4">
          {sorted.map((m) => {
            const month = `${monthName(m.date)} ${m.date.slice(0, 4)}`; const showMonth = month !== lastMonth; lastMonth = month;
            return (
              <div key={m.id} className="flex flex-col gap-3.5">
                {showMonth ? <Micro>{month}</Micro> : null}
                <article className="flex flex-col gap-1">
                  {m.hasPhoto ? <Photo h={m.note ? 190 : 120} label={m.title} /> : null}
                  <div className="mt-2 text-bodylg font-semibold tracking-[-0.01em]">{m.title}</div>
                  <div className="text-support text-stone">{longDate(m.date)} {m.date.slice(0, 4)}{m.location ? ` · ${m.location}` : ''}</div>
                  {m.note ? <p className="m-0 mt-0.5 text-[15px] leading-[1.55] text-stone" data-selectable>{m.note}</p> : null}
                </article>
              </div>
            );
          })}
        </div>
      </Main>
      <BottomActions><Button icon="plus" onClick={() => setOpen(true)}>Save a moment from today</Button></BottomActions>
      {composer}
    </>
  );
}
