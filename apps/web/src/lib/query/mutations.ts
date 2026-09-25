"use client";

import {
  onlineManager,
  useMutation,
  useQueryClient,
  type MutateOptions,
  type MutationKey,
  type QueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import { notify } from "@/lib/store/toast";

/**
 * One write the app can make, declared once (features/<name>/writes.ts) and
 * used two ways: by the screen via useWrite(def), and by the QueryClient via
 * registerWrites() so a restored offline change — with no screen attached —
 * still knows its own API call and refetch (lib/query/persist.ts).
 */
interface BaseWriteDef<A, R> {
  mutationKey: MutationKey;
  mutationFn: (args: A) => Promise<R>;
  /** Queries that are stale once this succeeds or fails. */
  invalidates: readonly QueryKey[];
  /** Writes sharing a scope are sent one at a time, in order. */
  scope?: string;
  /** The screen shows this write's errors itself, so skip the global toast. */
  handlesError?: boolean;
  /**
   * Updates the cache (via qc.setQueryData) the moment the write starts, so
   * the UI reflects it instantly instead of after the round trip. `invalidates`
   * queries are cancelled first (so an in-flight reply can't overwrite this),
   * snapshotted, and restored automatically on failure — write only the happy path.
   */
  optimistic?: (qc: QueryClient, args: A) => void;
}

/**
 * Every write declares what happens if it's sent twice — no third option. A
 * queued write can be replayed (connection drops after the server applies it
 * but before the response arrives), so it may only be queued when replaying
 * it changes nothing (FR-PWA-009); otherwise it must refuse to queue at all.
 */
type Replay =
  | {
      /** Why a repeat send is the same as one send (natural key, replace, settled
       *  state) — a string, not a boolean, so the reasoning has to be written out. */
      idempotent: string;
      onlineOnly?: never;
      keyed?: never;
    }
  | {
      /** Send now or fail — never pause offline and send later. For writes that
       *  would do damage twice, or whose meaning depends on timing. */
      onlineOnly: true;
      idempotent?: never;
      keyed?: never;
    }
  | {
      /** Why a client key makes a replay safe. For creates, which have no
       *  natural key of their own: the API stores the reply against the key,
       *  so a second send replays the first answer instead of making a second
       *  row (FR-PWA-009). useWrite stamps the key into the variables, which
       *  is what carries it through storage and replay. */
      keyed: string;
      idempotent?: never;
      onlineOnly?: never;
    };

export type WriteDef<A, R> = BaseWriteDef<A, R> & Replay;

/** Any definition, whatever its argument and result types. */
export type AnyWriteDef = WriteDef<never, unknown>;

export const defineWrite = <A, R>(def: WriteDef<A, R>): WriteDef<A, R> => def;

const invalidate = (qc: QueryClient, keys: readonly QueryKey[]) =>
  Promise.all(keys.map((queryKey) => qc.invalidateQueries({ queryKey })));

export function useWrite<A, R>(def: WriteDef<A, R>) {
  const qc = useQueryClient();
  const mutation = useMutation({
    mutationKey: def.mutationKey,
    mutationFn: def.mutationFn,
    scope: def.scope ? { id: def.scope } : undefined,
    meta: def.handlesError ? { handlesError: true } : undefined,
    // "always" skips the offline pause: fails immediately instead of waiting for reconnect.
    networkMode: def.onlineOnly ? "always" : undefined,
    onMutate: def.optimistic ? (args) => applyOptimistic(qc, def, args) : undefined,
    onError: (_error, _args, rollback) => rollback?.(),
    onSettled: () => invalidate(qc, def.invalidates),
  });

  // Stamped here, never inside mutationFn: mutationFn runs again on a replay
  // and would mint a new key each time, which is the one thing that must not
  // happen. In the variables it is saved with the write and comes back with it.
  const stamp = (args: A): A => {
    if (!def.keyed) return args;
    const a = (args ?? {}) as { idempotencyKey?: string };
    return { ...a, idempotencyKey: a.idempotencyKey ?? crypto.randomUUID() } as A;
  };

  // Offline, a resumable write pauses and its onSuccess waits for the
  // connection — which left every "close when saved" sheet spinning. onQueued
  // is the screen's "done for now". onSuccess is dropped: firing hours later it
  // would close, or clear, whatever the person has opened since.
  const mutate = (args: A, opts?: WriteOptions<A, R>) => {
    const { onQueued, ...rest } = opts ?? {};
    if (def.onlineOnly || onlineManager.isOnline()) {
      mutation.mutate(stamp(args), rest);
      return;
    }
    mutation.mutate(stamp(args), { ...rest, onSuccess: undefined });
    notify("Saved on this phone. It’ll send when you’re back online.");
    onQueued?.();
  };
  return {
    ...mutation,
    mutate,
    mutateAsync: (args: A, opts?: Parameters<typeof mutation.mutateAsync>[1]) =>
      mutation.mutateAsync(stamp(args), opts),
  };
}

/** TanStack's per-call options, plus what to do when the write is queued offline. */
export type WriteOptions<A, R> = MutateOptions<R, Error, A, (() => void) | undefined> & {
  onQueued?: () => void;
};

/** Applies a write's optimistic change and hands back the undo. The cancel is
 *  load-bearing: without it, an in-flight refetch can land after and silently revert it. */
function applyOptimistic<A, R>(qc: QueryClient, def: WriteDef<A, R>, args: A) {
  const snapshot = def.invalidates.map((key) => [key, qc.getQueryData(key)] as const);
  for (const key of def.invalidates) void qc.cancelQueries({ queryKey: key });
  def.optimistic?.(qc, args);
  return () => {
    for (const [key, data] of snapshot) qc.setQueryData(key, data);
  };
}

/**
 * Makes each write resumable after a reload (see lib/query/persist.ts).
 * Online-only writes are skipped: they never pause, so there is nothing to
 * resume. Keyed writes are registered like idempotent ones — their key travels
 * in the saved variables, so a restored send carries the same one.
 */
export function registerWrites(qc: QueryClient, defs: readonly AnyWriteDef[]) {
  for (const def of defs) {
    if (def.onlineOnly) continue;
    qc.setMutationDefaults(def.mutationKey, {
      // Restored variables are saved JSON; the def is what knows their real type.
      mutationFn: def.mutationFn as (variables: unknown) => Promise<unknown>,
      scope: def.scope ? { id: def.scope } : undefined,
      onSettled: () => invalidate(qc, def.invalidates),
    });
  }
}

/** Whether a paused write can be sent after a reload, i.e. it was registered. */
export const isResumable = (qc: QueryClient, key: MutationKey | undefined): boolean =>
  key !== undefined && qc.getMutationDefaults(key).mutationFn !== undefined;
