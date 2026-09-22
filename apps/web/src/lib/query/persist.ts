"use client";

import { createSyncStoragePersister } from "@tanstack/query-sync-storage-persister";
import type { QueryClient } from "@tanstack/react-query";
import { persistQueryClient } from "@tanstack/react-query-persist-client";

import { isResumable } from "./mutations";

// Changes made offline survive the app being closed.
//
// What's saved: ONLY writes that are paused waiting for the connection, and
// only ones registered as resumable (registerWrites). Never the query cache:
// that would leave a couple's prayers and journal on the device for anyone
// who picks it up.
//
// Whose: saved under the signed-in user's id, so one person's pending
// changes can never be sent from another person's session on a shared
// device. Everything is cleared on logout (clearSignedInState).

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
    persister: createSyncStoragePersister({ storage, key: KEY_PREFIX + userId, throttleTime: SAVE_THROTTLE_MS }),
    maxAge: MAX_AGE_MS,
    buster: SCHEMA_VERSION,
    dehydrateOptions: {
      shouldDehydrateQuery: () => false,
      shouldDehydrateMutation: (m) => m.state.isPaused && isResumable(qc, m.options.mutationKey),
    },
  });
  // Restored changes are paused; send them now if we're online. If not,
  // React Query sends them when the connection returns.
  restored.then(() => qc.resumePausedMutations()).catch(() => {});

  const stop = () => {
    unsubscribe();
    if (stopActive === stop) stopActive = null;
  };
  stopActive = stop;
  return stop;
}

/**
 * Forgets everything the signed-in user left in this browser: the in-memory
 * cache and any saved offline changes. Call it on logout, session expiry and
 * account deletion.
 *
 * The order matters. The persister saves at most every SAVE_THROTTLE_MS,
 * always writing the latest state it was handed. Clearing the cache first
 * makes that latest state empty, so its last write can't bring a pending
 * change back; then we stop it, delete the saved copy, and sweep once more
 * after that last (empty) write has landed.
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

// localStorage can be missing or throw (private browsing, blocked storage);
// then changes simply aren't saved across restarts.
function localStorageOrNull(): Storage | null {
  try {
    return typeof window === "undefined" ? null : window.localStorage;
  } catch {
    return null;
  }
}
