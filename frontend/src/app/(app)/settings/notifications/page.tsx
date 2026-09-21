'use client';

import { useCouple, usePrefs, useSavePrefs } from '@/lib/hooks';
import { time12 } from '@/lib/dates';
import type { NotificationPrefs } from '@/lib/types';
import { Main } from '@/components/Screen';
import { Micro, Para, Skeleton, Switch, Title, TopBar, cx } from '@/components/ui';

type BoolKey = { [K in keyof NotificationPrefs]: NotificationPrefs[K] extends boolean ? K : never }[keyof NotificationPrefs];

export default function NotificationSettings() {
  const prefs = usePrefs(); const save = useSavePrefs(); const couple = useCouple();
  const p = prefs.data; const partner = couple.data?.partner?.name ?? 'your partner';
  if (!p) return <><TopBar back="Settings" backHref="/settings" /><Main><Skeleton /></Main></>;
  const set = (k: BoolKey) => (v: boolean) => save.mutate([{ [k]: v }]);
  const Row = ({ k, label, last }: { k: BoolKey; label: string; last?: boolean }) => (
    <div className={cx('flex h-14 items-center gap-4', !last && 'border-b border-line')}><span className="grow text-[16px] font-medium">{label}</span><Switch checked={p[k]} onChange={set(k)} label={label} /></div>
  );
  const Group = ({ label, children }: { label: string; children: React.ReactNode }) => <section className="mt-[22px] flex flex-col"><Micro className="pb-0.5">{label}</Micro>{children}</section>;
  return (
    <>
      <TopBar back="Settings" backHref="/settings" />
      <Main>
        <div className="pt-3"><Title>Notifications</Title><Para className="mt-1.5">Few, calm, and only about the two of you.</Para></div>
        <Group label="Faith">
          <Row k="newWeek" label="New prayer week" />
          <Row k="prayerReminder" label="Prayer reminder" />
          <label className={cx('relative flex h-[54px] items-center justify-between pl-4 text-[16px] font-medium', !p.prayerReminder && 'opacity-50')}>Reminder time
            <span className="tabular flex items-center gap-1.5 font-semibold text-plum">{time12(p.reminderTime)}, every day</span>
            <input type="time" value={p.reminderTime} disabled={!p.prayerReminder} onChange={(e) => save.mutate([{ reminderTime: e.target.value }])} className="absolute inset-0 h-full w-full cursor-pointer opacity-0" aria-label="Reminder time" />
          </label>
        </Group>
        <Group label="Plans"><Row k="eventReminders" label="Event reminders" /><Row k="importantDates" label="Important dates" last /></Group>
        <Group label="Together"><Row k="appreciation" label={`Appreciation from ${partner}`} /><Row k="journal" label="Journal entries" /><Row k="goals" label="Goal updates" /><Row k="challenges" label="Challenge reminders" last /></Group>
        <Para size="support" className="mb-4 mt-4">Never more than one reminder a day.</Para>
      </Main>
    </>
  );
}
