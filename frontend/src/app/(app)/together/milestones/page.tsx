'use client';

import { useState } from 'react';
import { TODAY } from '@/lib/api';
import { useAddMilestone, useMilestones } from '@/lib/hooks';
import { daysUntil, iso, longDate } from '@/lib/dates';
import { DateRow } from '@/components/DateRow';
import { Main } from '@/components/Screen';
import { BareInput, BottomActions, Button, Para, Section, Sheet, Skeleton, Title, TopBar } from '@/components/ui';
import { PickRow } from '@/components/PickRow';

function nextOccurrence(date: string, today: string) {
  if (date >= today) return date;
  const [, m, d] = date.split('-'); const y = Number(today.slice(0, 4));
  const t = `${y}-${m}-${d}`; return t >= today ? t : `${y + 1}-${m}-${d}`;
}

export default function Milestones() {
  const ms = useMilestones(); const add = useAddMilestone();
  const [open, setOpen] = useState(false); const [title, setTitle] = useState(''); const [date, setDate] = useState(iso(TODAY)); const [sub, setSub] = useState('');
  const today = iso(TODAY);
  if (!ms.data) return <><TopBar back="Our space" backHref="/together" /><Main><Skeleton /></Main></>;
  const upcoming = ms.data.filter((d) => d.reminder).map((d) => ({ ...d, next: nextOccurrence(d.date, today) })).sort((a, b) => a.next.localeCompare(b.next));
  const story = ms.data.filter((d) => !d.reminder).sort((a, b) => b.date.localeCompare(a.date));
  const submit = () => { if (title.trim()) add.mutate([{ title: title.trim(), date, sub: sub.trim() || undefined, reminder: date >= today }], { onSuccess: () => { setOpen(false); setTitle(''); setSub(''); } }); };
  return (
    <>
      <TopBar back="Our space" backHref="/together" />
      <Main>
        <div className="pt-2"><Title>Milestones</Title><Para className="mt-1.5">The dates that matter to us.</Para></div>
        {upcoming.length ? <Section label="Coming up" className="mt-[18px]">{upcoming.map((d) => <DateRow key={d.id} date={d.next} title={d.title} sub={d.sub} right={`in ${daysUntil(d.next, TODAY)} days`} />)}</Section> : null}
        {story.length ? <Section label="Our story so far" className="mb-4 mt-6">{story.map((d) => <DateRow key={d.id} date={d.date} title={d.title} sub={d.sub ?? d.date.slice(0, 4)} />)}</Section> : null}
      </Main>
      <BottomActions><Button icon="plus" onClick={() => setOpen(true)}>Add a date</Button></BottomActions>
      <Sheet open={open} onClose={() => setOpen(false)} title="A date worth keeping." labelledBy="d-h">
        <BareInput label="What is it" value={title} onChange={(e) => setTitle(e.target.value)} autoFocus placeholder="Our engagement" className="h-11 text-[22px] font-semibold tracking-[-0.02em]" />
        <div className="flex flex-col border-t border-line"><PickRow icon="calendar" label="Date" value={date} type="date" onChange={setDate} empty={longDate(date)} last /></div>
        <BareInput label="A note, optional" value={sub} onChange={(e) => setSub(e.target.value)} placeholder="Three years together" className="h-10 text-body" />
        <div className="flex flex-col gap-1"><Button onClick={submit} loading={add.isPending}>Keep this date</Button><Button variant="text" onClick={() => setOpen(false)}>Not now</Button></div>
      </Sheet>
    </>
  );
}
