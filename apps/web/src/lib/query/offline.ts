"use client";

import { onlineManager, useMutationState, type MutationKey } from "@tanstack/react-query";
import { useEffect, useState, useSyncExternalStore } from "react";

// Read from React Query itself, not a second store, so the UI and the paused-mutation machinery can't disagree.

const subscribe = (onChange: () => void) => onlineManager.subscribe(onChange);

export function useOnline(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => onlineManager.isOnline(),
    () => true,
  );
}

/** Variables of changes waiting for the connection, e.g. to mark "will sync" rows. */
export function usePausedVariables<V>(mutationKey: MutationKey): V[] {
  return useMutationState({
    filters: { mutationKey, predicate: (m) => m.state.isPaused },
    select: (m) => m.state.variables as V,
  });
}

/** How many changes are waiting for the connection, e.g. to warn before logging out. */
export function usePausedCount(): number {
  return useMutationState({ filters: { predicate: (m) => m.state.isPaused } }).length;
}

const SYNCED_NOTICE_MS = 3500;

/** Count of paused changes just sent after reconnecting, for a "synced" notice;
 *  null otherwise. A later failure surfaces via the QueryClient's own error toast. */
export function useSyncedCount(): number | null {
  const paused = useMutationState({ filters: { predicate: (m) => m.state.isPaused } }).length;
  const online = useOnline();
  const [previous, setPrevious] = useState(paused);
  const [synced, setSynced] = useState<number | null>(null);

  // Derive from the change in `paused` during render (React's pattern for
  // "adjusting state when a value changes"), not in an effect.
  if (paused !== previous) {
    setPrevious(paused);
    if (online && paused === 0 && previous > 0) setSynced(previous);
  }

  useEffect(() => {
    if (synced === null) return;
    const timer = setTimeout(() => setSynced(null), SYNCED_NOTICE_MS);
    return () => clearTimeout(timer);
  }, [synced]);

  return synced;
}
