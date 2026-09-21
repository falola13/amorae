'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useState } from 'react';
import { Icon, type IconName } from './Icon';
import { cx } from './ui';

const TABS: { label: string; icon: IconName; href: string; match: (p: string) => boolean }[] = [
  { label: 'Home', icon: 'home', href: '/', match: (p) => p === '/' },
  { label: 'Together', icon: 'together', href: '/together', match: (p) => p.startsWith('/together') },
  { label: 'Prayers', icon: 'book', href: '/prayers', match: (p) => p.startsWith('/prayers') },
  { label: 'History', icon: 'clock', href: '/history', match: (p) => p.startsWith('/history') },
  { label: 'Settings', icon: 'sliders', href: '/settings', match: (p) => p.startsWith('/settings') },
];

/** Hides while the on-screen keyboard is up (visualViewport shrinks), so the bar never sits on top of the keyboard. */
function useKeyboardOpen() {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const vv = window.visualViewport; if (!vv) return;
    const check = () => setOpen(window.innerHeight - vv.height > 150);
    vv.addEventListener('resize', check); return () => vv.removeEventListener('resize', check);
  }, []);
  return open;
}

export function TabBar() {
  const path = usePathname();
  const hidden = useKeyboardOpen();
  if (hidden) return null;
  return (
    <nav aria-label="Primary" className="sticky bottom-0 z-30 flex shrink-0 border-t border-line bg-bg px-1.5 pt-1.5" style={{ paddingBottom: 'max(var(--safe-bottom), 16px)' }}>
      {TABS.map((t) => {
        const on = t.match(path);
        return (
          <Link key={t.href} href={t.href} aria-current={on ? 'page' : undefined} className={cx('press flex h-[52px] grow basis-0 flex-col items-center justify-center gap-1 text-[11px] tracking-[0.02em] no-underline', on ? 'font-bold text-plum' : 'font-semibold text-stone')}>
            <Icon name={t.icon} size={22} strokeWidth={on ? 1.7 : 1.5} />
            <span>{t.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}
