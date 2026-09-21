'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { useCouple, useUpdateProfile } from '@/lib/hooks';
import { Icon } from '@/components/Icon';
import { Main } from '@/components/Screen';
import { BottomActions, Button, Field, Initial, Skeleton, Title, TopBar } from '@/components/ui';

export default function Profile() {
  const couple = useCouple(); const update = useUpdateProfile();
  const [name, setName] = useState(''); const [email, setEmail] = useState(''); const [tz, setTz] = useState('Africa/Lagos'); const [loaded, setLoaded] = useState(false); const [saved, setSaved] = useState(false);
  useEffect(() => { if (couple.data && !loaded) { setName(couple.data.me.name); setEmail(couple.data.me.email); setTz(couple.data.me.timezone); setLoaded(true); } }, [couple.data, loaded]);
  if (!couple.data || !loaded) return <><TopBar back="Settings" backHref="/settings" /><Main><Skeleton /></Main></>;
  const partner = couple.data.partner?.name;
  const dirty = name !== couple.data.me.name || email !== couple.data.me.email || tz !== couple.data.me.timezone;
  const save = () => update.mutate([{ name: name.trim(), email: email.trim(), timezone: tz }], { onSuccess: () => { setSaved(true); setTimeout(() => setSaved(false), 2000); } });
  return (
    <>
      <TopBar back="Settings" backHref="/settings" />
      <form onSubmit={(e) => { e.preventDefault(); save(); }} className="flex grow flex-col">
        <Main className="gap-[26px] pt-3">
          <Title>Profile</Title>
          <div className="flex items-center gap-4"><Initial letter={name[0] ?? 'A'} size={64} /><div className="flex flex-col"><div className="text-bodylg font-semibold">{name || 'You'}</div><div className="text-support text-stone">{partner ? `Praying with ${partner}` : 'Waiting for your partner'}</div></div></div>
          <div className="flex flex-col gap-[18px]">
            <Field label="First name" value={name} onChange={(e) => setName(e.target.value)} autoComplete="given-name" hint={partner ? `This is what ${partner} sees.` : 'This is what your partner will see.'} />
            <Field label="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" inputMode="email" />
            <div className="flex flex-col gap-2"><label htmlFor="tz" className="text-[13px] font-semibold text-stone">Timezone</label>
              <select id="tz" value={tz} onChange={(e) => setTz(e.target.value)} className="h-[52px] w-full appearance-none rounded-input border border-edge bg-surface px-4 text-[16px] text-ink">
                {['Africa/Lagos', 'America/New_York', 'Europe/London', 'Africa/Nairobi', 'Asia/Dubai'].map((z) => <option key={z} value={z}>{z.replace('_', ' ')}</option>)}
              </select></div>
            <Link href="#password" className="press flex h-[54px] items-center justify-between border-y border-line text-[16px] font-medium text-ink no-underline">Change password<Icon name="right" size={18} className="text-stone" /></Link>
          </div>
        </Main>
        <BottomActions><Button type="submit" disabled={!dirty && !saved} loading={update.isPending} icon={saved ? 'check' : undefined}>{saved ? 'Saved' : 'Save changes'}</Button></BottomActions>
      </form>
    </>
  );
}
