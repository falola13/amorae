'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useEffect } from 'react';
import { useMe } from '@/lib/hooks';
import { Screen } from '@/components/Screen';
import { TabBar } from '@/components/TabBar';
import { NetworkBanner } from '@/components/NetworkBanner';
import { SplashView } from '@/components/SplashView';

/** Signed-in shell: auth gate, offline banner, tab bar. */
export default function AppLayout({ children }: { children: React.ReactNode }) {
  const me = useMe();
  const router = useRouter();
  const path = usePathname();
  const focused = path.startsWith('/prayers/mode'); // distraction-free: no shell, no tab bar
  useEffect(() => {
    if (me.data && !me.data.session) router.replace('/welcome');
    else if (me.data && !me.data.onboarded.couple) router.replace('/couple');
  }, [me.data, router]);
  if (!me.data || !me.data.session || !me.data.onboarded.couple) return <SplashView />;
  if (focused) return <>{children}</>;
  return (
    <Screen>
      <div className="shrink-0" style={{ height: 'calc(var(--safe-top) + 8px)' }} />
      <NetworkBanner />
      {children}
      <TabBar />
    </Screen>
  );
}
