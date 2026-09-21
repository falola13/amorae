'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCompleteEvent, useEvent, useToggleChecklist } from '@/lib/hooks';
import { time12, weekdayDate } from '@/lib/dates';
import { Icon, type IconName } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { BottomActions, Button, Checkbox, Micro, Para, Section, Skeleton, Title, TopBar } from '@/components/ui';

function Meta({ icon, text, action, href }: { icon: IconName; text: string; action?: string; href?: string }) {
  return (
    <div className="flex min-h-[52px] items-center gap-3.5 border-b border-line"><Icon name={icon} size={22} className="text-stone" /><span className="grow text-[16px]">{text}</span>
      {action && href ? <Link href={href} className="press flex h-11 items-center px-0.5 text-[15px] font-semibold text-plum no-underline">{action}</Link> : null}</div>
  );
}

export default function EventDetail() {
  const { id } = useParams<{ id: string }>(); const router = useRouter();
  const ev = useEvent(id); const toggle = useToggleChecklist(); const complete = useCompleteEvent();
  const e = ev.data;
  if (!e) return <><TopBar back="Events" backHref="/together/events" /><Main><Skeleton /></Main></>;
  const edit = <Link href={`/together/events/new?edit=${e.id}`} className="press flex h-11 items-center px-3 text-[16px] font-semibold text-plum no-underline">Edit</Link>;
  return (
    <>
      <TopBar back="Events" backHref="/together/events" right={edit} />
      <Main>
        <div className="pt-3"><Micro>{weekdayDate(e.date)}</Micro></div>
        <Title size="lg" className="mt-2">{e.title}</Title>
        <div className="mt-5 flex flex-col border-t border-line">
          {e.start ? <Meta icon="clock" text={`${time12(e.start)}${e.end ? ` to ${time12(e.end)}` : ''}`} /> : null}
          {e.location ? <Meta icon="pin" text={e.location} /> : null}
          {e.reminder ? <Meta icon="bell" text={`Reminder ${e.reminder}`} action="Change" href={`/together/events/new?edit=${e.id}`} /> : null}
        </div>
        {e.checklist.length ? <Section label="Before we go" className="mt-[22px]">{e.checklist.map((c) => <Checkbox key={c.id} checked={c.done} onChange={(v) => toggle.mutate([e.id, c.id, v])}>{c.text}</Checkbox>)}</Section> : null}
        {e.notes ? <section className="mt-[18px] flex flex-col gap-1.5"><Micro>Notes</Micro><p className="m-0 text-body" data-selectable>{e.notes}</p></section> : null}
      </Main>
      <BottomActions>
        {e.done ? <Button variant="secondary" icon="image" onClick={() => router.push('/together/memories')}>Save a moment from it</Button>
          : <Button variant="secondary" icon="check" onClick={() => complete.mutate([e.id, true], { onSuccess: () => router.replace('/together/events') })}>Mark as done</Button>}
        {e.done ? <Button variant="text" onClick={() => complete.mutate([e.id, false])}>Not done yet</Button> : <Para size="support" className="hidden">&nbsp;</Para>}
      </BottomActions>
    </>
  );
}
