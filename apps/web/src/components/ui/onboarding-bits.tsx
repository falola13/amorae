'use client';

import type { ReactNode } from 'react';
import { Icon, type IconName } from "@/components/icons";

export function Bottom({ children }: { children: ReactNode }) {
  return <div className="flex shrink-0 flex-col gap-2 px-6 pt-4 pb-safe">{children}</div>;
}

export function Fact({ icon, title, text, trailing }: { icon: IconName; title: string; text: string; trailing?: ReactNode }) {
  return (
    <div className="flex items-center gap-3.5 border-b border-line py-4">
      <Icon name={icon} size={22} className="text-plum" />
      <div className="flex grow flex-col gap-0.5"><div className="text-[16px] font-semibold">{title}</div><div className="text-support text-stone">{text}</div></div>
      {trailing}
    </div>
  );
}

export function ShowButton({ shown, onClick }: { shown: boolean; onClick: () => void }) {
  return <button type="button" onClick={onClick} className="press h-11 px-3 text-support font-semibold text-plum" aria-pressed={shown}>{shown ? 'Hide' : 'Show'}</button>;
}
