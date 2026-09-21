'use client';

import Link from 'next/link';
import { forwardRef, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode, type TextareaHTMLAttributes } from 'react';
import { Icon, type IconName } from './Icon';

function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(' ');
}

/* ---------- Buttons ---------- */

type Variant = 'primary' | 'secondary' | 'text' | 'danger';

const VARIANT: Record<Variant, string> = {
  primary: 'h-[54px] rounded-btn bg-plum text-surface',
  secondary: 'h-[54px] rounded-btn border border-edge bg-transparent text-ink',
  text: 'h-11 rounded-btn bg-transparent text-plum',
  danger: 'h-11 rounded-btn bg-transparent text-red',
};

const BASE = 'press inline-flex w-full items-center justify-center gap-2 text-[16px] font-semibold disabled:opacity-100 disabled:bg-faint disabled:text-stone';

export function Button({ variant = 'primary', icon, loading, children, className, ...rest }: { variant?: Variant; icon?: IconName; loading?: boolean } & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button type="button" aria-busy={loading || undefined} className={cx(BASE, VARIANT[variant], className)} disabled={loading || rest.disabled} {...rest}>
      {loading ? <Spinner light={variant === 'primary'} /> : icon ? <Icon name={icon} size={20} /> : null}
      {children}
    </button>
  );
}

export function LinkButton({ href, variant = 'primary', icon, children, className, replace }: { href: string; variant?: Variant; icon?: IconName; children: ReactNode; className?: string; replace?: boolean }) {
  return (
    <Link href={href} replace={replace} className={cx(BASE, VARIANT[variant], 'no-underline', className)}>
      {icon ? <Icon name={icon} size={20} /> : null}
      {children}
    </Link>
  );
}

export function Spinner({ light }: { light?: boolean }) {
  return <span className={cx('inline-block h-[18px] w-[18px] animate-spin rounded-full border-2', light ? 'border-surface/35 border-t-surface' : 'border-faint border-t-plum')} />;
}

/* ---------- Type ---------- */

export function Micro({ children, className, tone = 'stone' }: { children: ReactNode; className?: string; tone?: 'stone' | 'plum' }) {
  return <div className={cx('text-micro uppercase', tone === 'plum' ? 'text-plum' : 'text-stone', className)}>{children}</div>;
}

export function Title({ children, className, as: As = 'h1', size = 'title' }: { children: ReactNode; className?: string; as?: 'h1' | 'h2'; size?: 'title' | 'display' | 'lg' }) {
  const s = size === 'display' ? 'text-display' : size === 'lg' ? 'text-[32px] leading-[1.15] font-semibold tracking-[-0.025em]' : 'text-title';
  return <As className={cx('m-0', s, className)}>{children}</As>;
}

export function Para({ children, className, size = 'body' }: { children: ReactNode; className?: string; size?: 'body' | 'lg' | 'support' }) {
  return <p className={cx('m-0 text-stone', size === 'lg' ? 'text-bodylg' : size === 'support' ? 'text-support' : 'text-body', className)}>{children}</p>;
}

/* ---------- Fields ---------- */

export const Field = forwardRef<HTMLInputElement, { label: string; hint?: string; error?: string; trailing?: ReactNode } & InputHTMLAttributes<HTMLInputElement>>(function Field({ label, hint, error, trailing, id, className, ...rest }, ref) {
  const fid = id ?? rest.name ?? label.toLowerCase().replace(/\W+/g, '-');
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={fid} className="text-[13px] font-semibold text-stone">{label}</label>
      <div className="relative flex">
        <input ref={ref} id={fid} aria-invalid={error ? true : undefined} aria-describedby={error ? `${fid}-err` : hint ? `${fid}-hint` : undefined}
          className={cx('h-[52px] w-full rounded-input border bg-surface px-4 text-[16px] text-ink focus:border-[1.5px] focus:border-plum', error ? 'border-[1.5px] border-red' : 'border-edge', className)} {...rest} />
        {trailing ? <div className="absolute right-1 top-1">{trailing}</div> : null}
      </div>
      {error ? <div id={`${fid}-err`} role="alert" className="flex items-center gap-1.5 text-[13px] text-red"><Icon name="alert" size={16} />{error}</div>
        : hint ? <div id={`${fid}-hint`} className="text-[13px] text-stone">{hint}</div> : null}
    </div>
  );
});

/** Bare fields for compose screens: writing a note, not filling a form. */
export const BareInput = forwardRef<HTMLInputElement, { label: string } & InputHTMLAttributes<HTMLInputElement>>(function BareInput({ label, id, className, ...rest }, ref) {
  const fid = id ?? rest.name ?? label.toLowerCase().replace(/\W+/g, '-');
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={fid} className="text-micro uppercase text-stone">{label}</label>
      <input ref={ref} id={fid} className={cx('w-full border-0 bg-transparent p-0 text-ink outline-offset-[6px]', className)} {...rest} />
    </div>
  );
});

export const BareTextarea = forwardRef<HTMLTextAreaElement, { label: string; hideLabel?: boolean } & TextareaHTMLAttributes<HTMLTextAreaElement>>(function BareTextarea({ label, hideLabel, id, className, ...rest }, ref) {
  const fid = id ?? rest.name ?? label.toLowerCase().replace(/\W+/g, '-');
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={fid} className={hideLabel ? 'sr-only' : 'text-micro uppercase text-stone'}>{label}</label>
      <textarea ref={ref} id={fid} className={cx('w-full resize-none border-0 bg-transparent p-0 text-ink outline-offset-[6px]', className)} {...rest} />
    </div>
  );
});

/* ---------- Rows and lists ---------- */

export function Row({ href, icon, title, sub, trailing, onClick, last, className, children }: { href?: string; icon?: IconName; title: ReactNode; sub?: ReactNode; trailing?: ReactNode; onClick?: () => void; last?: boolean; className?: string; children?: ReactNode }) {
  const inner = (
    <>
      {icon ? <Icon name={icon} size={22} className="text-stone" /> : null}
      <span className="flex min-w-0 grow flex-col gap-px">
        <span className="text-[16px] font-semibold text-ink">{title}</span>
        {sub ? <span className="text-support text-stone">{sub}</span> : null}
        {children}
      </span>
      {trailing ?? (href || onClick ? <Icon name="right" size={18} className="text-stone" /> : null)}
    </>
  );
  const cls = cx('press flex min-h-[62px] w-full items-center gap-3.5 text-left no-underline', !last && 'border-b border-line', className);
  if (href) return <Link href={href} className={cls}>{inner}</Link>;
  if (onClick) return <button type="button" onClick={onClick} className={cls}>{inner}</button>;
  return <div className={cls}>{inner}</div>;
}

export function Section({ label, children, className, trailing }: { label: string; children: ReactNode; className?: string; trailing?: ReactNode }) {
  return (
    <section className={cx('flex flex-col', className)}>
      <div className="flex items-baseline justify-between pb-1"><Micro>{label}</Micro>{trailing}</div>
      {children}
    </section>
  );
}

export function StatusMark({ done, size = 22, label }: { done: boolean; size?: number; label?: string }) {
  if (done) {
    return (
      <span role="img" aria-label={label ?? 'Prayed'} className="flex shrink-0 items-center justify-center rounded-full bg-green-tint" style={{ width: size, height: size }}>
        <Icon name="check" size={size - 8} strokeWidth={2.2} className="text-green" />
      </span>
    );
  }
  return <span role="img" aria-label={label ?? 'Not yet prayed'} className="box-border shrink-0 rounded-full border-[1.5px] border-edge" style={{ width: size, height: size }} />;
}

export function Segments({ total, done, color = 'bg-plum', height = 6, className }: { total: number; done: number; color?: string; height?: number; className?: string }) {
  return (
    <div role="img" aria-label={`${done} of ${total} prayed`} className={cx('flex gap-1', className)}>
      {Array.from({ length: total }, (_, i) => (
        <div key={i} className={cx('grow basis-0 rounded-full', i < done ? color : 'bg-faint')} style={{ height }} />
      ))}
    </div>
  );
}

export function Bar({ pct, label }: { pct: number; label: string }) {
  return (
    <div role="progressbar" aria-valuenow={Math.round(pct)} aria-valuemin={0} aria-valuemax={100} aria-label={label} className="h-1.5 overflow-hidden rounded-full bg-faint">
      <div className="h-1.5 rounded-full bg-plum transition-[width] duration-300" style={{ width: `${Math.min(100, Math.max(0, pct))}%` }} />
    </div>
  );
}

export function Initial({ letter, size = 28 }: { letter: string; size?: number }) {
  return (
    <span aria-hidden="true" className="flex shrink-0 items-center justify-center rounded-full bg-plum-tint font-bold text-plum" style={{ width: size, height: size, fontSize: Math.round(size * 0.46) }}>{letter}</span>
  );
}

export function Switch({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <button type="button" role="switch" aria-checked={checked} aria-label={label} onClick={() => onChange(!checked)} className="press flex h-11 w-[52px] shrink-0 items-center bg-transparent p-0">
      <span className={cx('flex h-8 w-[52px] items-center rounded-full transition-colors duration-200', checked ? 'bg-plum' : 'bg-edge')}>
        <span className={cx('h-7 w-7 rounded-full bg-surface transition-[margin] duration-200', checked ? 'ml-[22px]' : 'ml-0.5')} />
      </span>
    </button>
  );
}

export function Checkbox({ checked, onChange, children }: { checked: boolean; onChange: (v: boolean) => void; children: ReactNode }) {
  return (
    <button type="button" role="checkbox" aria-checked={checked} onClick={() => onChange(!checked)} className={cx('press flex h-12 w-full items-center gap-3.5 bg-transparent p-0 text-left text-[16px] font-medium', checked ? 'text-stone' : 'text-ink')}>
      {checked ? (
        <span className="flex h-[22px] w-[22px] shrink-0 items-center justify-center rounded-[7px] bg-green-tint"><Icon name="check" size={14} strokeWidth={2.2} className="text-green" /></span>
      ) : (
        <span className="box-border h-[22px] w-[22px] shrink-0 rounded-[7px] border-[1.5px] border-edge" />
      )}
      {children}
    </button>
  );
}

/* ---------- Chrome ---------- */

export function TopBar({ back, backHref, right, center, onBack }: { back?: string; backHref?: string; right?: ReactNode; center?: ReactNode; onBack?: () => void }) {
  const cls = 'press flex h-11 items-center gap-0.5 pl-1 pr-2 text-[16px] font-medium text-ink no-underline';
  return (
    <div className="flex h-11 shrink-0 items-center justify-between px-3">
      {backHref ? <Link href={backHref} className={cls}><Icon name="left" size={22} />{back}</Link>
        : onBack ? <button type="button" onClick={onBack} className={cls}><Icon name="left" size={22} />{back}</button> : <span />}
      {center ? <div className="text-[13px] font-semibold text-stone">{center}</div> : null}
      {right ?? <span />}
    </div>
  );
}

/** Compose bar: Cancel and Save stay in the top bar, reachable with the keyboard open. */
export function ComposeBar({ cancelHref, label, done = 'Save', onDone, onCancel, busy }: { cancelHref?: string; label: string; done?: string; onDone?: () => void; onCancel?: () => void; busy?: boolean }) {
  const c = 'press flex h-11 items-center px-3 text-[16px] font-medium text-ink no-underline';
  return (
    <div className="flex h-11 shrink-0 items-center justify-between px-3">
      {cancelHref ? <Link href={cancelHref} className={c}>Cancel</Link> : <button type="button" onClick={onCancel} className={c}>Cancel</button>}
      <div className="text-[13px] font-semibold text-stone">{label}</div>
      <button type="button" onClick={onDone} disabled={busy} className="press flex h-11 items-center px-3 text-[16px] font-bold text-plum disabled:text-stone">{busy ? <Spinner /> : done}</button>
    </div>
  );
}

/** Sticky action area above the tab bar or the home indicator. */
export function BottomActions({ children, className, safe }: { children: ReactNode; className?: string; safe?: boolean }) {
  return <div className={cx('flex shrink-0 flex-col gap-1 px-6 pt-3', safe ? 'pb-safe' : 'pb-3', className)}>{children}</div>;
}

export function Banner({ tone, icon, children }: { tone: 'amber' | 'green'; icon: IconName; children: ReactNode }) {
  const t = tone === 'amber' ? 'bg-amber-tint text-amber' : 'bg-green-tint text-green';
  return (
    <div role="status" className={cx('mx-4 flex animate-rise items-center gap-2.5 rounded-input px-3.5 py-2.5 text-support font-semibold', t)}>
      <Icon name={icon} size={18} strokeWidth={1.7} />{children}
    </div>
  );
}

export function Sheet({ open, title, onClose, children, labelledBy }: { open: boolean; title: string; onClose: () => void; children: ReactNode; labelledBy: string }) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-40">
      <button type="button" aria-label="Close" onClick={onClose} className="absolute inset-0 animate-fade bg-ink/40" />
      <div role="dialog" aria-modal="true" aria-labelledby={labelledBy} className="absolute inset-x-0 bottom-0 flex animate-sheet flex-col gap-4 rounded-t-sheet bg-bg px-6 pt-2.5 pb-safe">
        <div aria-hidden="true" className="h-1 w-9 self-center rounded-full bg-faint" />
        <h2 id={labelledBy} className="m-0 mt-2 text-[24px] font-semibold leading-tight tracking-[-0.02em]">{title}</h2>
        {children}
      </div>
    </div>
  );
}

export function Toast({ message, action, onAction }: { message: string; action?: string; onAction?: () => void }) {
  return (
    <div role="status" className="fixed inset-x-6 z-50 flex h-[52px] animate-rise items-center justify-between gap-2.5 rounded-btn bg-ink pl-4 pr-2 text-[15px] font-medium text-surface" style={{ bottom: 'calc(92px + var(--safe-bottom))' }}>
      {message}
      {action ? <button type="button" onClick={onAction} className="press h-11 px-2.5 text-[15px] font-bold text-surface">{action}</button> : null}
    </div>
  );
}

export function Skeleton({ lines = 3 }: { lines?: number }) {
  return (
    <div aria-hidden="true" className="flex flex-col gap-3 pt-2">
      {Array.from({ length: lines }, (_, i) => <div key={i} className="h-3.5 animate-breathe rounded-lg bg-faint" style={{ width: `${70 - i * 14}%` }} />)}
    </div>
  );
}

export function Ghost({ kind = 'lines' }: { kind?: 'lines' | 'frames' | 'bars' | 'dots' }) {
  if (kind === 'frames') {
    return (
      <div aria-hidden="true" className="flex gap-2.5 pb-3.5">
        <span className="h-[72px] w-24 rounded-input border-[1.5px] border-dashed border-edge opacity-70" /><span className="h-[72px] w-[72px] rounded-input bg-faint opacity-60" /><span className="h-[72px] w-12 rounded-input bg-faint opacity-30" />
      </div>
    );
  }
  if (kind === 'bars') {
    return <div aria-hidden="true" className="flex flex-col gap-4 pb-3.5">{[200, 150, 100].map((w, i) => <div key={w} className="h-1.5 rounded-full bg-faint" style={{ width: w, opacity: 1 - i * 0.35 }} />)}</div>;
  }
  if (kind === 'dots') {
    return (
      <div aria-hidden="true" className="flex flex-col gap-[18px] pb-3.5">
        {[150, 120, 96].map((w, i) => <div key={w} className="flex items-center gap-4" style={{ opacity: 1 - i * 0.35 }}><span className="box-border h-[9px] w-[9px] rounded-full border-[1.5px] border-edge" /><span className="h-2.5 rounded-full bg-faint" style={{ width: w }} /></div>)}
      </div>
    );
  }
  return (
    <div aria-hidden="true" className="flex flex-col gap-4 pb-3.5">
      {[150, 120, 96].map((w, i) => <div key={w} className="flex items-center gap-4" style={{ opacity: 1 - i * 0.35 }}><span className="h-7 w-7 rounded-lg bg-faint" /><span className="h-2.5 rounded-full bg-faint" style={{ width: w }} /></div>)}
    </div>
  );
}

export function EmptyState({ ghost, title, text, cta }: { ghost: 'lines' | 'frames' | 'bars' | 'dots'; title: string; text: string; cta: ReactNode }) {
  return (
    <div className="flex grow flex-col justify-center gap-3.5 pb-12">
      <Ghost kind={ghost} />
      <h2 className="m-0 text-[26px] font-semibold leading-tight tracking-[-0.02em]">{title}</h2>
      <Para size="lg">{text}</Para>
      <div className="mt-3">{cta}</div>
    </div>
  );
}

export function ErrorState({ title, text, onRetry, secondary }: { title: string; text: string; onRetry: () => void; secondary?: ReactNode }) {
  return (
    <div role="alert" className="flex grow flex-col justify-center gap-3.5 pb-10">
      <Icon name="alert" size={28} strokeWidth={1.4} className="text-red" />
      <h2 className="m-0 text-[24px] font-semibold leading-tight tracking-[-0.02em]">{title}</h2>
      <Para>{text}</Para>
      <div className="mt-2.5 flex flex-col gap-1"><Button icon="sync" onClick={onRetry}>Try again</Button>{secondary}</div>
    </div>
  );
}

export { cx };
