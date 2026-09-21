'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { TODAY } from '@/lib/api';
import { useAppreciations, useCouple, useSendAppreciation, useUndoAppreciation } from '@/lib/hooks';
import { iso, relativeDay } from '@/lib/dates';
import { Icon } from '@/components/Icon';
import { BareTextarea, ComposeBar, Micro, Toast, cx } from '@/components/ui';

export default function Appreciation() {
  const router = useRouter();
  const couple = useCouple(); const list = useAppreciations(); const send = useSendAppreciation(); const undo = useUndoAppreciation();
  const [text, setText] = useState(''); const [sentId, setSentId] = useState<string | null>(null);
  const me = couple.data?.me; const partner = couple.data?.partner?.name ?? 'your partner';
  const today = iso(TODAY);
  const submit = () => { if (!text.trim()) return; send.mutate([text.trim()], { onSuccess: (a) => { setSentId(a.id); setText(''); setTimeout(() => setSentId(null), 5000); } }); };
  return (
    <>
      <ComposeBar cancelHref="/together" label={`To ${partner}`} done="Send" onDone={submit} busy={send.isPending} />
      <form onSubmit={(e) => { e.preventDefault(); submit(); }} className="flex flex-col gap-2.5 px-6 pt-6">
        <label htmlFor="ap" className="text-title leading-[1.2] text-stone">Something I appreciate about you&hellip;</label>
        <BareTextarea id="ap" label="Appreciation" hideLabel value={text} onChange={(e) => setText(e.target.value)} rows={4} autoFocus placeholder="One true sentence is plenty." className="min-h-[128px] text-reading" />
        <div className="flex items-center gap-2 text-support text-stone"><Icon name="bell" size={16} />{partner} will get one quiet notification.</div>
        <button type="submit" className="sr-only">Send</button>
      </form>
      <section className="mt-7 flex grow flex-col px-6">
        <Micro className="pb-0.5">Between you, lately</Micro>
        {(list.data ?? []).map((a, i, arr) => (
          <div key={a.id} className={cx('flex flex-col gap-1 py-3.5', i < arr.length - 1 && 'border-b border-line')}>
            <div className="text-[13px] font-semibold text-stone">From {a.fromId === me?.id ? 'you' : partner} &middot; {relativeDay(a.date, today)}</div>
            <div className="text-body leading-[1.55]" data-selectable>{a.text}</div>
          </div>
        ))}
        {list.data && list.data.length === 0 ? <div className="py-3.5 text-support text-stone">Nothing yet. Say the first one.</div> : null}
      </section>
      {sentId ? <Toast message={`Sent to ${partner}`} action="Undo" onAction={() => { undo.mutate([sentId]); setSentId(null); }} /> : null}
      <button type="button" className="sr-only" onClick={() => router.push('/together')}>Back to our space</button>
    </>
  );
}
