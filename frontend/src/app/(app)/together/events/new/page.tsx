'use client';

import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { TODAY } from '@/lib/api';
import { useEvent, useSaveEvent } from '@/lib/hooks';
import { iso, longDate, time12 } from '@/lib/dates';
import { BareInput, BareTextarea, ComposeBar } from '@/components/ui';
import { PickRow } from '@/components/PickRow';

export default function NewEventPage() { return <Suspense fallback={null}><EventForm /></Suspense>; }

function EventForm() {
  const router = useRouter(); const params = useSearchParams(); const editId = params.get('edit');
  const existing = useEvent(editId ?? '__none');
  const save = useSaveEvent();
  const [title, setTitle] = useState(''); const [date, setDate] = useState(iso(TODAY)); const [start, setStart] = useState(''); const [end, setEnd] = useState('');
  const [location, setLocation] = useState(''); const [reminder, setReminder] = useState('1 hour before'); const [notes, setNotes] = useState('');
  const [loaded, setLoaded] = useState(!editId);
  useEffect(() => { if (editId && existing.data && !loaded) { const e = existing.data; setTitle(e.title); setDate(e.date); setStart(e.start ?? ''); setEnd(e.end ?? ''); setLocation(e.location ?? ''); setReminder(e.reminder ?? ''); setNotes(e.notes ?? ''); setLoaded(true); } }, [editId, existing.data, loaded]);
  const done = () => {
    if (!title.trim()) { router.back(); return; }
    save.mutate([{ id: editId ?? undefined, title: title.trim(), date, start: start || undefined, end: end || undefined, location: location.trim() || undefined, reminder: reminder || undefined, notes: notes.trim() || undefined }], { onSuccess: () => router.replace(editId ? `/together/events/${editId}` : '/together/events') });
  };
  return (
    <>
      <ComposeBar cancelHref={editId ? `/together/events/${editId}` : '/together/events'} label={editId ? 'Edit event' : 'New event'} onDone={done} busy={save.isPending} />
      <form onSubmit={(e) => { e.preventDefault(); done(); }} className="flex grow flex-col px-6 pt-5">
        <BareInput label="What would you like to do together?" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Date night" autoFocus={!editId} className="h-11 text-title" />
        <div className="mt-[18px] flex flex-col border-t border-line">
          <PickRow icon="calendar" label="Date" value={date} type="date" onChange={setDate} empty={longDate(date)} />
          <PickRow icon="clock" label="Starts" value={start} type="time" onChange={setStart} empty={start ? time12(start) : "Add a time"} />
          <PickRow icon="clock" label="Ends" value={end} type="time" onChange={setEnd} empty={end ? time12(end) : "Add a time"} />
          <PickRow icon="pin" label="Location" value={location} type="text" onChange={setLocation} placeholder="Add a place" />
          <PickRow icon="bell" label="Reminder" value={reminder} type="text" onChange={setReminder} placeholder="None" last />
        </div>
        <div className="mt-5"><BareTextarea label="Notes" value={notes} onChange={(e) => setNotes(e.target.value)} rows={3} placeholder="Anything worth remembering for the day." className="min-h-[84px] text-bodylg" /></div>
        <button type="submit" className="sr-only">Save</button>
      </form>
    </>
  );
}
