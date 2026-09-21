'use client';

import type { ReactNode } from 'react';
import { cx } from './ui';

/**
 * One screen = one column that fills the dynamic viewport.
 * Top padding respects the iOS status bar in standalone mode. Content scrolls inside `main`
 * so the tab bar and bottom actions stay put.
 */
export function Screen({ children, className, tone = 'bg' }: { children: ReactNode; className?: string; tone?: 'bg' | 'paper' }) {
  return (
    <div className={cx('mx-auto flex min-h-[100dvh] w-full max-w-[520px] flex-col', tone === 'paper' ? 'bg-paper' : 'bg-bg', className)}>
      {children}
    </div>
  );
}

export function SafeTop({ className }: { className?: string }) {
  return <div className={cx('shrink-0', className)} style={{ height: 'calc(var(--safe-top) + 42px)' }} />;
}

export function Main({ children, className, pad = true }: { children: ReactNode; className?: string; pad?: boolean }) {
  return <main className={cx('flex grow animate-page flex-col', pad && 'px-6', className)}>{children}</main>;
}
