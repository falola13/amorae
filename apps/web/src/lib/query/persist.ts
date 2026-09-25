"use client";

import { createSyncStoragePersister } from "@tanstack/query-sync-storage-persister";
import type { QueryClient } from "@tanstack/react-query";
import { persistQueryClient } from "@tanstack/react-query-persist-client";

import { isResumable } from "./mutations";

// Changes made offline survive the app being closed. Saves ONLY paused,
// resumable (registerWrites) writes — never the query cache, which would leave
// a couple's data on a shared device. Keyed by user id, cleared on logout.

const KEY_PREFIX = "amorae:offline-changes:";
const MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000;
const SAVE_THROTTLE_MS = 500;
// Bump when a write's variables change shape, so old saved changes are dropped.
const SCHEMA_VERSION = "1";

// The running persistence, so clearSignedInState can stop it.
let stopActive: (() => void) | null = null;

/** Starts saving and restoring this user's pending changes; returns a stop function. */
export function persistOfflineChanges(qc: QueryClient, userId: string): () => void {
  const storage = localStorageOrNull();
  if (!storage) return () => {};

  const [unsubscribe, restored] = persistQueryClient({
    queryClient: qc,
    persister: createSyncStoragePersister({
      storage,
      key: KEY_PREFIX + userId,
      throttleTime: SAVE_THROTTLE_MS,
    }),
    maxAge: MAX_AGE_MS,
    buster: SCHEMA_VERSION,
    dehydrateOptions: {
      shouldDehydrateQuery: () => false,
      shouldDehydrateMutation: (m) => m.state.isPaused && isResumable(qc, m.options.mutationKey),
    },
  });
  // Send restored (paused) changes now if online; otherwise React Query sends them on reconnect.
  restored.then(() => qc.resumePausedMutations()).catch(() => {});

  const stop = () => {
    unsubscribe();
    if (stopActive === stop) stopActive = null;
  };
  stopActive = stop;
  return stop;
}

/**
 * Clears the in-memory cache and saved offline changes on logout, session
 * expiry or account deletion. Order matters: clear the cache first so the
 * persister's next (throttled) write can't resurrect a pending change, then
 * stop it, delete the saved copy, and sweep again after that write lands.
 */
export function clearSignedInState(qc: QueryClient) {
  qc.clear();
  stopActive?.();
  removeSavedChanges();
  setTimeout(removeSavedChanges, SAVE_THROTTLE_MS * 2);
}

function removeSavedChanges() {
  const storage = localStorageOrNull();
  if (!storage) return;
  for (const key of Object.keys(storage)) {
    if (key.startsWith(KEY_PREFIX)) storage.removeItem(key);
  }
}

// localStorage can be missing or throw (private browsing, blocked storage) — then nothing persists.
function localStorageOrNull(): Storage | null {
  try {
    return typeof window === "undefined" ? null : window.localStorage;
  } catch {
    return null;
  }
}
