'use client';

import { type ReactNode } from 'react';
import { Icon } from "@/components/icons";
import { Button } from './buttons';
import { Para } from './typography';

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

export function Alert({ message, variant = "error" }: { message: string; variant?: "error" | "success" }) {
  const tone = variant === "error" ? "bg-red-tint text-red" : "bg-green-tint text-green";
  return (
    <div role={variant === "error" ? "alert" : "status"} className={`flex items-center gap-2.5 rounded-input px-3.5 py-2.5 text-support font-semibold ${tone}`}>
      <Icon name={variant === "error" ? "alert" : "check"} size={18} />
      {message}
    </div>
  );
}
