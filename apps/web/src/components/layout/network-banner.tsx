"use client";

import { Banner } from "@/components/ui/kit";
import { useOnline, useSyncedCount } from "@/lib/query/offline";

/** Quiet status line. Never a full-screen error: content underneath stays usable. */
export function NetworkBanner() {
  const online = useOnline();
  const synced = useSyncedCount();
  if (!online)
    return (
      <Banner tone="amber" icon="offline">
        Offline &middot; Showing your saved content
      </Banner>
    );
  if (synced)
    return (
      <Banner tone="green" icon="sync">
        Back online &middot; {synced} {synced === 1 ? "change" : "changes"} synced
      </Banner>
    );
  return null;
}
