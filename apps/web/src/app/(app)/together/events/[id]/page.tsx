'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useCompleteEvent, useEvent, useChecklist } from '@/features/together/hooks';
import { time12, weekdayDate } from '@/lib/dates';
import { routes } from '@/lib/routes';
import { Icon, type IconName } from '@/components/icons';
import { Main } from '@/components/layout/screen';
import { QueryState } from '@/components/ui/query-state';
import { BottomActions, Button, Checkbox, Micro, Para, Section, Skeleton, Title, TopBar } from '@/components/ui/kit';

function Meta({ icon, text, action, href }: { icon: IconName; text: string; action?: string; href?: string }) {
  return (
    <div className="flex min-h-[52px] items-center gap-3.5 border-b border-line"><Icon name={icon} size={22} className="text-stone" /><span className="grow text-[16px]">{text}</span>
      {action && href ? <Link href={href} className="press flex h-11 items-center px-0.5 text-[15px] font-semibold text-plum no-underline">{action}</Link> : null}</div>
  );
}

export default function EventDetail() {
  const { id } = useParams<{ id: string }>(); const router = useRouter();
  const ev = useEvent(id); const toggle = useChecklist(); const complete = useCompleteEvent();
  const edit = ev.data ? <Link href={routes.eventNew({ edit: ev.data.id })} className="press flex h-11 items-center px-3 text-[16px] font-semibold text-plum no-underline">Edit</Link> : undefined;
  return (
    <>
      <TopBar back="Events" backHref={routes.events} right={edit} />
      <QueryState queries={[ev]} loading={<Main><Skeleton /></Main>}>
        {(e) => (
          <>
            <Main>
              <div className="pt-3"><Micro>{weekdayDate(e.date)}</Micro></div>
              <Title size="lg" className="mt-2">{e.title}</Title>
              <div className="mt-5 flex flex-col border-t border-line">
                {e.start_time ? <Meta icon="clock" text={`${time12(e.start_time)}${e.end_time ? ` to ${time12(e.end_time)}` : ''}`} /> : null}
                {e.location ? <Meta icon="pin" text={e.location} /> : null}
                {e.reminder ? <Meta icon="bell" text={`Reminder ${e.reminder}`} action="Change" href={routes.eventNew({ edit: e.id })} /> : null}
              </div>
              {e.checklist.length ? <Section label="Before we go" className="mt-[22px]">{e.checklist.map((c) => <Checkbox key={c.id} checked={c.done} onChange={(v) => toggle.mutate({ id: e.id, item: c.id, done: v })}>{c.text}</Checkbox>)}</Section> : null}
              {e.notes ? <section className="mt-[18px] flex flex-col gap-1.5"><Micro>Notes</Micro><p className="m-0 text-body" data-selectable>{e.notes}</p></section> : null}
            </Main>
            <BottomActions>
              {e.done ? <Button variant="secondary" icon="image" onClick={() => router.push(routes.memories)}>Save a moment from it</Button>
                : <Button variant="secondary" icon="check" onClick={() => complete.mutate({ id: e.id, done: true }, { onSuccess: () => router.replace(routes.events) })}>Mark as done</Button>}
              {e.done ? <Button variant="text" onClick={() => complete.mutate({ id: e.id, done: false })}>Not done yet</Button> : <Para size="support" className="hidden">&nbsp;</Para>}
            </BottomActions>
          </>
        )}
      </QueryState>
    </>
  );
}
