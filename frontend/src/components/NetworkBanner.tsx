'use client';

import { useNetwork } from '@/lib/hooks';
import { Banner } from './ui';

/** Quiet status line. Never a full-screen error: content underneath stays usable. */
export function NetworkBanner() {
  const net = useNetwork();
  if (!net.online) return <Banner tone="amber" icon="offline">Offline &middot; Showing your saved content</Banner>;
  if (net.justSynced) return <Banner tone="green" icon="sync">Back online &middot; {net.justSynced} {net.justSynced === 1 ? 'change' : 'changes'} synced</Banner>;
  return null;
}
