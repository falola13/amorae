"use client";

import { Banner } from "@/components/ui/kit";
import { useUpdate } from "@/lib/pwa/update";
import { useOnline, usePausedCount, useSyncedCount } from "@/lib/query/offline";

/** Quiet status line. Never a full-screen error: content underneath stays usable. */
export function NetworkBanner() {
  const online = useOnline();
  const synced = useSyncedCount();
  const waiting = usePausedCount();
  const updated = useUpdate((s) => s.ready);
  if (!online)
    return (
      <Banner tone="amber" icon="offline">
        {/* Queued changes aren't in the lists until they send, so say they exist. */}
        Offline &middot;{" "}
        {waiting
          ? `${waiting} ${waiting === 1 ? "change" : "changes"} waiting to send`
          : "Showing your saved content"}
      </Banner>
    );
  if (synced)
    return (
      <Banner tone="green" icon="sync">
        Back online &middot; {synced} {synced === 1 ? "change" : "changes"} synced
      </Banner>
    );
  // Last, and until acted on: it matters, but never more than being offline.
  // Not while changes wait to send — a reload then would strand nothing (they
  // persist), but it would look like it had.
  if (updated && !waiting)
    return (
      <Banner tone="green" icon="sync">
        <span className="grow">A new version of Amorae is ready</span>
        <button
          type="button"
          onClick={() => window.location.reload()}
          className="press -my-2 h-11 px-1 font-bold underline"
        >
          Refresh
        </button>
      </Banner>
    );
  return null;
}
