'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { Icon } from '@/components/Icon';
import { SafeTop } from '@/components/Screen';
import { Button, LinkButton, Micro, Para, Title, TopBar, cx } from '@/components/ui';
import { Bottom } from '@/components/OnboardingBits';

export default function JoinCouple() {
  const router = useRouter(); const qc = useQueryClient();
  const [code, setCode] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const looksValid = /^[A-Z0-9]{4}-?[A-Z0-9]{4}$/i.test(code.trim());
  const join = async () => {
    setBusy(true); setError(null);
    try { await api.joinCouple(code); await qc.invalidateQueries(); router.push('/install'); } catch (e) { setError((e as Error).message); } finally { setBusy(false); }
  };
  return (
    <form onSubmit={(e) => { e.preventDefault(); join(); }} className="flex grow flex-col">
      <SafeTop />
      <TopBar back="Back" backHref="/couple" />
      <div className="flex grow flex-col gap-7 px-6 pt-3">
        <div className="flex flex-col gap-2"><Micro>Step 2 of 3</Micro><Title>Join your partner</Title><Para>Enter the code from the invitation they sent you.</Para></div>
        <div className="flex flex-col gap-2.5">
          <label htmlFor="code" className="text-[13px] font-semibold text-stone">Invitation code</label>
          <input id="code" value={code} onChange={(e) => setCode(e.target.value.toUpperCase())} autoComplete="off" autoCapitalize="characters" spellCheck={false} placeholder="XXXX-XXXX" aria-invalid={!!error}
            className={cx('tabular h-16 w-full rounded-input border-[1.5px] bg-surface px-4 text-center text-[24px] font-semibold tracking-[0.14em] text-ink', error ? 'border-red' : looksValid ? 'border-plum' : 'border-edge')} />
          {error ? <div role="alert" className="flex items-center gap-2 text-support text-red"><Icon name="alert" size={18} />{error}</div>
            : looksValid ? <div role="status" className="flex items-center gap-2 text-support font-medium text-green"><Icon name="check" size={18} strokeWidth={2} />This invitation is from Adeola.</div> : null}
        </div>
      </div>
      <Bottom>
        <Button type="submit" disabled={!looksValid} loading={busy}>{looksValid ? 'Join Adeola' : 'Join'}</Button>
        <LinkButton href="/couple" variant="text">Start a new space instead</LinkButton>
      </Bottom>
    </form>
  );
}
