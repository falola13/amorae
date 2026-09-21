'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { usePush } from '@/lib/hooks';
import { Icon } from '@/components/Icon';
import { SafeTop } from '@/components/Screen';
import { Button, Para, Title } from '@/components/ui';
import { Bottom } from '@/components/OnboardingBits';

function Preview({ when, text }: { when: string; text: string }) {
  return <div className="flex flex-col gap-0.5 border-b border-line py-3.5"><div className="text-[13px] font-semibold text-stone">{when}</div><div className="text-[16px] font-medium">{text}</div></div>;
}

export default function NotificationPermission() {
  const router = useRouter(); const qc = useQueryClient();
  const { request } = usePush();
  const [busy, setBusy] = useState(false);
  const finish = async () => { await api.setOnboarded({ notifications: true }); await qc.invalidateQueries(); router.replace('/'); };
  const enable = async () => { setBusy(true); await request(); await finish(); };
  return (
    <>
      <SafeTop />
      <div className="h-11 shrink-0" />
      <div className="flex grow flex-col gap-6 px-6 pt-3">
        <Icon name="bell" size={28} strokeWidth={1.4} className="text-plum" />
        <div className="flex flex-col gap-2"><Title>Stay close to what you&rsquo;ve planned</Title><Para>A few calm reminders about your shared life, and nothing else. For example:</Para></div>
        <div className="flex flex-col border-t border-line">
          <Preview when="On Sundays" text="Your new prayer week is ready." />
          <Preview when="Before something you’ve planned" text="You have a date tonight at 7:00 pm." />
          <Preview when="When your partner shares something" text="Adeola added something to your shared journal." />
        </div>
        <Para size="support">You choose which ones you get, anytime, in Settings.</Para>
      </div>
      <Bottom>
        <Button onClick={enable} loading={busy}>Enable notifications</Button>
        <Button variant="text" onClick={finish}>Not now</Button>
      </Bottom>
    </>
  );
}
