'use client';

import { useId } from 'react';
import { Icon, type IconName } from './Icon';
import { cx } from './ui';

/** A row on a compose screen that opens a native picker (date, time) or holds a short value. */
export function PickRow({ icon, label, value, empty, type, onChange, placeholder, last }: { icon: IconName; label: string; value: string; empty?: string; type?: 'date' | 'time' | 'text'; onChange?: (v: string) => void; placeholder?: string; last?: boolean }) {
  const id = useId();
  const shown = (value && type !== 'date' && type !== 'time') ? value : (empty || placeholder || '');
  return (
    <label htmlFor={id} className={cx('relative flex h-[54px] w-full cursor-pointer items-center gap-3.5 text-[16px] font-medium text-ink', !last && 'border-b border-line')}>
      <Icon name={icon} size={22} className="text-stone" />
      <span className="grow">{label}</span>
      <span className={cx('tabular', value ? 'font-semibold text-plum' : 'text-stone')}>{shown}</span>
      {onChange ? <input id={id} type={type ?? 'text'} value={value} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} className={cx('absolute inset-0 h-full w-full cursor-pointer opacity-0', type === 'text' && 'opacity-0')} aria-label={label} /> : null}
    </label>
  );
}
