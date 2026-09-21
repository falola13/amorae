'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { useCouple, useDemoWeek, useInstall, usePrefs, useWeek } from '@/lib/hooks';
import { time12 } from '@/lib/dates';
import { ME } from '@/lib/api';
import { Icon, type IconName } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { Button, Micro, Para, Sheet, Title, cx } from '@/components/ui';

function SetRow({ icon, label, value, href, onClick, tone = 'ink', chevron = true, last }: { icon: IconName; label: string; value?: string; href?: string; onClick?: () => void; tone?: 'ink' | 'red'; chevron?: boolean; last?: boolean }) {
  const cls = cx('press flex h-[54px] w-full items-center gap-3.5 text-left text-[16px] font-medium no-underline', tone === 'red' ? 'text-red' : 'text-ink', !last && 'border-b border-line');
  const inner = <><Icon name={icon} size={22} className={tone === 'red' ? 'text-red' : 'text-stone'} /><span className="grow">{label}</span>{value ? <span className="text-[15px] text-stone">{value}</span> : null}{chevron ? <Icon name="right" size={18} className="text-stone" /> : null}</>;
  return href ? <Link href={href} className={cls}>{inner}</Link> : <button type="button" onClick={onClick} className={cls}>{inner}</button>;
}

function Group({ label, children }: { label: string; children: React.ReactNode }) {
  return <section className="mt-[26px] flex flex-col"><Micro className="pb-1.5">{label}</Micro>{children}</section>;
}

export default function Settings() {
  const couple = useCouple(); const prefs = usePrefs(); const install = useInstall(); const week = useWeek(); const demo = useDemoWeek();
  const router = useRouter(); const qc = useQueryClient();
  const [confirm, setConfirm] = useState<'logout' | 'delete' | null>(null);
  const me = couple.data?.me; const partner = couple.data?.partner?.name;
  const logout = async () => { await api.logout(); qc.clear(); router.replace('/welcome'); };
  const del = async () => { await api.deleteAccount(); qc.clear(); router.replace('/welcome'); };
  const mode = week.data ? (week.data.status === 'waiting' ? 'waiting' : week.data.setterId === ME ? 'mine' : 'partner') : 'partner';
  return (
    <>
      <Main>
        <div className="pt-1.5"><Title>Settings</Title></div>
        <Group label={partner ? `You and ${partner}` : 'You'}>
          <SetRow icon="user" label="Profile" value={me?.name} href="/settings/profile" />
          <SetRow icon="users" label="Couple" value={partner ? `With ${partner}` : 'Waiting for your partner'} href={partner ? '/settings/profile' : '/invite'} last />
        </Group>
        <Group label="Reminders">
          <SetRow icon="bell" label="Notifications" value={prefs.data ? (Object.values(prefs.data).some((v) => v === true) ? 'On' : 'Off') : ''} href="/settings/notifications" />
          <SetRow icon="clock" label="Prayer reminder" value={prefs.data?.prayerReminder ? time12(prefs.data.reminderTime) : 'Off'} href="/settings/notifications" />
          <SetRow icon="globe" label="Timezone" value={me?.timezone === 'Africa/Lagos' ? 'Lagos' : me?.timezone} href="/settings/profile" last />
        </Group>
        <Group label="App">
          <SetRow icon="phone" label="Install Amorae" value={install.standalone ? 'Installed' : 'Not yet'} href="/install" />
          <SetRow icon="lock" label="Privacy" href="#privacy" last />
        </Group>
        <Group label="Try the week states">
          <div role="radiogroup" aria-label="Demo week state" className="flex gap-1 py-1">
            {([['partner', 'Praying'], ['mine', 'Your week'], ['waiting', 'Waiting']] as const).map(([v, l]) => <button key={v} type="button" role="radio" aria-checked={mode === v} onClick={() => demo.mutate([v])} className={cx('press h-9 rounded-full px-3.5 text-[14px] font-semibold', mode === v ? 'bg-plum-tint text-plum' : 'text-stone')}>{l}</button>)}
          </div>
          <Para size="support">Preview only. The Go scheduler decides this for real, every Sunday.</Para>
        </Group>
        <Group label="Account">
          <SetRow icon="user" label="Account" value={me?.email} href="/settings/profile" />
          <SetRow icon="logout" label="Log out" chevron={false} onClick={() => setConfirm('logout')} />
          <SetRow icon="trash" label="Delete account" tone="red" chevron={false} onClick={() => setConfirm('delete')} last />
        </Group>
        <div className="pb-2 pt-[22px] text-[13px] text-stone">Amorae 1.0 &middot; Two hearts, one faith.</div>
      </Main>
      <Sheet open={confirm === 'logout'} onClose={() => setConfirm(null)} title="Log out of Amorae?" labelledBy="lo-h">
        <Para>Your prayers and everything you&rsquo;ve shared stay safe. You&rsquo;ll just need to log in again.</Para>
        <div className="flex flex-col gap-1 pt-1"><Button onClick={logout}>Log out</Button><Button variant="text" onClick={() => setConfirm(null)}>Stay logged in</Button></div>
      </Sheet>
      <Sheet open={confirm === 'delete'} onClose={() => setConfirm(null)} title="Delete your account?" labelledBy="del-h">
        <Para>This removes you from your space with {partner ?? 'your partner'} and deletes everything you wrote. It can&rsquo;t be undone.</Para>
        <div className="flex flex-col gap-1 pt-1"><Button variant="secondary" className="border-red text-red" onClick={del}>Delete my account</Button><Button variant="text" onClick={() => setConfirm(null)}>Keep my account</Button></div>
      </Sheet>
    </>
  );
}
