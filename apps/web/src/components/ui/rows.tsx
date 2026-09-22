'use client';

import Link from 'next/link';
import { type ReactNode } from 'react';
import { Icon, type IconName } from "@/components/icons";
import { cx } from './cx';
import { Micro } from './typography';

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

/** A labelled on/off preference: label left, Switch right. */
export function SwitchRow({ label, checked, onChange, last }: { label: string; checked: boolean; onChange: (v: boolean) => void; last?: boolean }) {
  return (
    <div className={cx('flex h-14 items-center gap-4', !last && 'border-b border-line')}>
      <span className="grow text-[16px] font-medium">{label}</span>
      <Switch checked={checked} onChange={onChange} label={label} />
    </div>
  );
}

/** A compact settings row: icon, label, optional value and chevron. Links when given href, else a button. */
export function SettingRow({ icon, label, value, href, onClick, tone = 'ink', chevron = true, last }: { icon: IconName; label: string; value?: string; href?: string; onClick?: () => void; tone?: 'ink' | 'red'; chevron?: boolean; last?: boolean }) {
  const cls = cx('press flex h-[54px] w-full items-center gap-3.5 text-left text-[16px] font-medium no-underline', tone === 'red' ? 'text-red' : 'text-ink', !last && 'border-b border-line');
  const inner = (
    <>
      <Icon name={icon} size={22} className={tone === 'red' ? 'text-red' : 'text-stone'} />
      <span className="grow">{label}</span>
      {value ? <span className="text-[15px] text-stone">{value}</span> : null}
      {chevron ? <Icon name="right" size={18} className="text-stone" /> : null}
    </>
  );
  if (href) return <Link href={href} className={cls}>{inner}</Link>;
  if (onClick) return <button type="button" onClick={onClick} className={cls}>{inner}</button>;
  return <div className={cls}>{inner}</div>;
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
