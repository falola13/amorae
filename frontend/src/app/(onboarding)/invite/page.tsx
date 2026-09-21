'use client';

import { useState } from 'react';
import { useCouple } from '@/lib/hooks';
import { Icon } from '@/components/Icon';
import { SafeTop } from '@/components/Screen';
import { Button, LinkButton, Micro, Para, Title, TopBar } from '@/components/ui';
import { Bottom } from '@/components/OnboardingBits';

export default function InvitePartner() {
  const couple = useCouple();
  const code = couple.data?.inviteCode ?? '••••-••••';
  const [copied, setCopied] = useState(false);
  const copy = async () => { try { await navigator.clipboard.writeText(code); setCopied(true); setTimeout(() => setCopied(false), 2000); } catch { /* clipboard blocked */ } };
  const share = async () => {
    const data = { title: 'Amorae', text: `Join me on Amorae, our private space. Invitation code: ${code}`, url: typeof location !== 'undefined' ? `${location.origin}/join?code=${code}` : '' };
    if (navigator.share) { try { await navigator.share(data); } catch { /* cancelled */ } } else copy();
  };
  return (
    <>
      <SafeTop />
      <TopBar back="Back" backHref="/couple" />
      <div className="flex grow flex-col gap-6 px-6 pt-3">
        <div className="flex flex-col gap-2"><Micro>Step 3 of 3</Micro><Title>Invite your partner</Title><Para>Send this to one person. It works once, and expires in 7 days.</Para></div>
        <div className="flex flex-col items-center gap-1.5 rounded-card border border-line bg-surface px-4 pb-3 pt-7">
          <div className="text-[13px] font-semibold text-stone">Your invitation code</div>
          <div className="tabular text-[30px] font-semibold tracking-[0.14em]" data-selectable>{code}</div>
          <Button variant="text" icon={copied ? 'check' : 'copy'} onClick={copy} className="w-auto px-4">{copied ? 'Copied' : 'Copy code'}</Button>
        </div>
        <div role="status" className="flex items-center gap-2.5 text-support text-stone"><span className="h-2 w-2 animate-breathe rounded-full bg-amber" />Waiting for your partner to join. We&rsquo;ll tell you when they do.</div>
      </div>
      <Bottom>
        <Button icon="share" onClick={share}>Share invitation</Button>
        <LinkButton href="/install" variant="text">Continue for now</LinkButton>
      </Bottom>
      <span className="hidden"><Icon name="share" /></span>
    </>
  );
}
