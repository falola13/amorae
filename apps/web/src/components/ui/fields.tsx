'use client';

import { forwardRef, type InputHTMLAttributes, type ReactNode, type TextareaHTMLAttributes } from 'react';
import { Icon } from "@/components/icons";
import { cx } from './cx';

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
export const BareInput = forwardRef<HTMLInputElement, { label: string; hideLabel?: boolean; error?: string } & InputHTMLAttributes<HTMLInputElement>>(function BareInput({ label, hideLabel, error, id, className, ...rest }, ref) {
  const fid = id ?? rest.name ?? label.toLowerCase().replace(/\W+/g, '-');
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={fid} className={hideLabel ? 'sr-only' : 'text-micro uppercase text-stone'}>{label}</label>
      <input ref={ref} id={fid} aria-invalid={error ? true : undefined} aria-describedby={error ? `${fid}-err` : undefined} className={cx('w-full border-0 bg-transparent p-0 text-ink outline-offset-[6px]', className)} {...rest} />
      {error ? <div id={`${fid}-err`} role="alert" className="flex items-center gap-1.5 text-[13px] text-red"><Icon name="alert" size={16} />{error}</div> : null}
    </div>
  );
});

export const BareTextarea = forwardRef<HTMLTextAreaElement, { label: string; hideLabel?: boolean; error?: string } & TextareaHTMLAttributes<HTMLTextAreaElement>>(function BareTextarea({ label, hideLabel, error, id, className, ...rest }, ref) {
  const fid = id ?? rest.name ?? label.toLowerCase().replace(/\W+/g, '-');
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={fid} className={hideLabel ? 'sr-only' : 'text-micro uppercase text-stone'}>{label}</label>
      <textarea ref={ref} id={fid} aria-invalid={error ? true : undefined} aria-describedby={error ? `${fid}-err` : undefined} className={cx('w-full resize-none border-0 bg-transparent p-0 text-ink outline-offset-[6px]', className)} {...rest} />
      {error ? <div id={`${fid}-err`} role="alert" className="flex items-center gap-1.5 text-[13px] text-red"><Icon name="alert" size={16} />{error}</div> : null}
    </div>
  );
});
