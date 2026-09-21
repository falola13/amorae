'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState } from 'react';
import { useSetCompleted, useWeek } from '@/lib/hooks';
import { Icon } from '@/components/Icon';
import { Scripture } from '@/components/Scripture';
import { Button, Skeleton, cx } from '@/components/ui';

/** Distraction free. Own layout (no tab bar), paper background, one prayer at a time. */
export default function PrayerModePage() {
  return <Suspense fallback={null}><PrayerMode /></Suspense>;
}

function PrayerMode() {
  const week = useWeek(); const complete = useSetCompleted(); const router = useRouter();
  const params = useSearchParams();
  const quiet = params.get('quiet') === '1';
  const w = week.data;
  const [i, setI] = useState<number | null>(null);
  const [timer, setTimer] = useState<number | null>(null);

  useEffect(() => {
    if (!w || i !== null) return;
    const at = params.get('at');
    if (at) { setI(Number(at)); return; }
    const first = w.points.findIndex((p) => !w.myCompleted.includes(p.id));
    setI(first === -1 ? 0 : first);
  }, [w, i, params]);

  useEffect(() => {
    if (timer === null) return;
    const t = setInterval(() => setTimer((s) => (s === null ? null : s + 1)), 1000);
    return () => clearInterval(t);
  }, [timer]);

  if (quiet || (w && w.points.length === 0)) return <Quiet timer={timer} setTimer={setTimer} />;
  if (!w || i === null) return <div className="mx-auto flex min-h-[100dvh] max-w-[520px] flex-col bg-paper px-8 pt-24"><Skeleton /></div>;

  const p = w.points[i]; const n = w.points.length;
  const prayed = w.myCompleted.includes(p.id);
  const allDone = w.myCompleted.length === n;
  const last = i === n - 1;
  const mm = timer !== null ? `${Math.floor(timer / 60)}:${String(timer % 60).padStart(2, '0')}` : null;

  return (
    <div className="mx-auto flex min-h-[100dvh] w-full max-w-[520px] flex-col bg-paper">
      <div className="shrink-0" style={{ height: 'calc(var(--safe-top) + 12px)' }} />
      <div className="flex h-11 shrink-0 items-center justify-between px-3">
        <Link href="/prayers" aria-label="Leave prayer mode" className="press flex h-11 w-11 items-center justify-center text-ink"><Icon name="x" size={22} /></Link>
        <div aria-hidden="true" className="flex w-32 gap-1">{w.points.map((x, k) => <div key={x.id} className={cx('h-1 grow basis-0 rounded-full', k === i ? 'bg-plum' : w.myCompleted.includes(x.id) ? 'bg-plum-soft' : 'bg-faint')} />)}</div>
        <button type="button" aria-label={timer === null ? 'Start a quiet timer' : 'Stop the timer'} aria-pressed={timer !== null} onClick={() => setTimer(timer === null ? 0 : null)} className={cx('press flex h-11 min-w-11 items-center justify-center gap-1 px-1 text-[13px] font-semibold', timer === null ? 'text-stone' : 'tabular text-plum')}>
          <Icon name="clock" size={22} />{mm}
        </button>
      </div>

      <main key={p.id} className="flex grow animate-page flex-col justify-center gap-4 px-8 pb-4">
        <div aria-live="polite" className="tabular text-micro uppercase text-stone">Prayer {i + 1} of {n}</div>
        <h1 className="m-0 -mt-1 text-[30px] font-semibold leading-[1.18] tracking-[-0.025em]" data-selectable>{p.title}</h1>
        {p.text ? <p className="m-0 text-[19px] leading-[1.6]" data-selectable>{p.text}</p> : null}
        {p.scripture ? <Scripture reference={p.scripture} verse={p.verse} faint /> : null}
      </main>

      <div className="flex shrink-0 flex-col gap-1.5 px-6 pb-safe">
        {prayed ? (
          <div role="status" className="flex h-[54px] animate-rise items-center justify-between rounded-btn bg-green-tint pl-[18px] pr-1.5">
            <div className="flex items-center gap-2.5 text-[16px] font-semibold text-green">
              <svg width="20" height="20" viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><path d="M5 12.5l4.5 4.5L19 7.5" className="draw-check animate-draw" style={{ strokeDasharray: 24, strokeDashoffset: 24 }} /></svg>Prayed
            </div>
            <button type="button" onClick={() => complete.mutate({ pointId: p.id, done: false })} className="press h-11 px-3 text-[15px] font-semibold text-green">Undo</button>
          </div>
        ) : (
          <Button icon="check" onClick={() => complete.mutate({ pointId: p.id, done: true })}>I&rsquo;ve prayed</Button>
        )}
        <div className="flex items-center justify-between">
          <button type="button" onClick={() => setI(Math.max(0, i - 1))} disabled={i === 0} className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-ink disabled:opacity-35"><Icon name="left" size={20} />Previous</button>
          {allDone ? <button type="button" onClick={() => router.push('/prayers/done')} className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-plum">Finish<Icon name="right" size={20} /></button>
            : !last ? <button type="button" onClick={() => setI(Math.min(n - 1, i + 1))} className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-ink">Next<Icon name="right" size={20} /></button>
            : <Link href="/prayers" className="press flex h-12 items-center gap-1 px-2.5 text-[15px] font-semibold text-ink no-underline">Done for now<Icon name="right" size={20} /></Link>}
        </div>
      </div>
    </div>
  );
}

/** Quiet mode: no points to pray through, just space and an optional timer. */
function Quiet({ timer, setTimer }: { timer: number | null; setTimer: (v: number | null) => void }) {
  const mm = timer !== null ? `${Math.floor(timer / 60)}:${String(timer % 60).padStart(2, '0')}` : null;
  return (
    <div className="mx-auto flex min-h-[100dvh] w-full max-w-[520px] flex-col bg-paper">
      <div className="shrink-0" style={{ height: 'calc(var(--safe-top) + 12px)' }} />
      <div className="flex h-11 shrink-0 items-center px-3"><Link href="/" aria-label="Leave quiet prayer" className="press flex h-11 w-11 items-center justify-center text-ink"><Icon name="x" size={22} /></Link></div>
      <main className="flex grow animate-page flex-col justify-center gap-5 px-8 pb-6">
        <Icon name="moon" size={28} strokeWidth={1.4} className="text-plum" />
        <h1 className="m-0 text-display">Take a quiet moment.</h1>
        <p className="m-0 text-reading text-stone">No list today. Just you, and whatever is on your heart.</p>
        {mm ? <div aria-live="polite" className="tabular text-[40px] font-medium tracking-[-0.03em] text-plum">{mm}</div> : null}
      </main>
      <div className="flex shrink-0 flex-col gap-1 px-6 pb-safe">
        <Button variant={timer === null ? 'primary' : 'secondary'} icon="clock" onClick={() => setTimer(timer === null ? 0 : null)}>{timer === null ? 'Start a quiet timer' : 'Stop'}</Button>
      </div>
    </div>
  );
}
