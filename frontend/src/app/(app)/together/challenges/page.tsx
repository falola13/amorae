'use client';

import { useChallenge, useChallengeDay } from '@/lib/hooks';
import { Main } from '@/components/Screen';
import { BottomActions, Button, Micro, Para, Skeleton, StatusMark, Title, TopBar, cx } from '@/components/ui';

export default function Challenges() {
  const ch = useChallenge(); const set = useChallengeDay();
  const c = ch.data;
  if (!c) return <><TopBar back="Our space" backHref="/together" /><Main><Skeleton /></Main></>;
  const today = c.days.find((d) => !d.done && !d.skipped);
  const finished = !today;
  return (
    <>
      <TopBar back="Our space" backHref="/together" />
      <Main>
        <div className="pt-2"><Micro>{finished ? 'Finished' : `Day ${today.n} of ${c.days.length}`}</Micro></div>
        <Title className="mt-2">{c.title}</Title>
        <Para className="mt-1.5">One small thing a day. Skip any day you need to.</Para>
        <ol className="m-0 mt-4 list-none border-t border-line p-0">
          {c.days.map((d) => {
            const isToday = today?.n === d.n;
            return (
              <li key={d.n} className="flex min-h-[52px] items-center gap-3.5 border-b border-line">
                {d.done ? <StatusMark done label="Done" /> : isToday ? <span aria-hidden="true" className="box-border h-[22px] w-[22px] shrink-0 rounded-full border-2 border-plum" /> : <span aria-hidden="true" className="box-border h-[22px] w-[22px] shrink-0 rounded-full border-[1.5px] border-faint" />}
                <span className="flex grow flex-col"><span className="text-[12px] font-semibold text-stone">Day {d.n}{d.skipped ? ' · skipped' : ''}</span><span className={cx('text-[16px]', isToday ? 'font-semibold text-ink' : 'font-medium text-stone')}>{d.text}</span></span>
                {isToday ? <span className="text-[12px] font-bold text-plum">Today</span> : null}
              </li>
            );
          })}
        </ol>
      </Main>
      {!finished ? (
        <BottomActions>
          <Button icon="check" onClick={() => set.mutate([today.n, { done: true }])}>We did today&rsquo;s</Button>
          <Button variant="text" onClick={() => set.mutate([today.n, { skipped: true }])}>Skip today</Button>
        </BottomActions>
      ) : <BottomActions><Para size="support" className="text-center">Seven days, done together. Nicely.</Para></BottomActions>}
    </>
  );
}
