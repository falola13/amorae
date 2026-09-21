'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { SafeTop } from '@/components/Screen';
import { Button, LinkButton, Micro, Para, Title, TopBar } from '@/components/ui';
import { Bottom, Fact } from '@/components/OnboardingBits';

export default function CreateCouple() {
  const router = useRouter(); const qc = useQueryClient();
  const [busy, setBusy] = useState(false);
  const create = async () => { setBusy(true); await api.createCouple(); await qc.invalidateQueries(); router.push('/invite'); };
  return (
    <>
      <SafeTop />
      <TopBar back="Back" backHref="/sign-up" />
      <div className="flex grow flex-col gap-5 px-6 pt-3">
        <div className="flex flex-col gap-2"><Micro>Step 2 of 3</Micro><Title>Start a space for the two of you</Title><Para>You&rsquo;ll get one private invitation to send your partner.</Para></div>
        <div className="flex flex-col border-t border-line">
          <Fact icon="users" title="Always exactly two" text="Only you and your partner can see what&rsquo;s written here." />
          <Fact icon="clock" title="A prayer week every Sunday" text="You&rsquo;ll set the first week. After that it alternates." />
          <Fact icon="globe" title="Lagos time (WAT)" text="Your weeks follow this timezone." trailing={<button type="button" className="press h-11 px-1 text-[15px] font-semibold text-plum">Change</button>} />
        </div>
      </div>
      <Bottom>
        <Button onClick={create} loading={busy}>Create our space</Button>
        <LinkButton href="/join" variant="text">I have an invitation code</LinkButton>
      </Bottom>
    </>
  );
}
