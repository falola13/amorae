"use client";

import {
  useMutation,
  useQueryClient,
  type MutationKey,
  type QueryClient,
  type QueryKey,
} from "@tanstack/react-query";

/**
 * One write the app can make: its key, the API call, and what it makes
 * stale. A feature declares each write once (features/<name>/writes.ts) and
 * that single definition is used twice:
 *
 * - by the screen, through useWrite(def);
 * - by the QueryClient, through registerWrites(), so a change made offline
 *   can still be sent after the app is closed and reopened (lib/query/persist.ts).
 *   A restored change has no screen attached, so its API call and refetch
 *   have to come from here.
 *
 * Errors and offline pausing are handled centrally (lib/query/client.ts).
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
   * Change the cache the moment the write starts, rather than when it lands.
   *
   * This is what a tap feeling instant is made of: ticking a prayer should
   * look done while the request is still in the air, because on a phone on a
   * bad connection the alternative is half a second of a control that appears
   * not to have worked — and then the tap again, and the wondering.
   *
   * Write it against the cache with qc.setQueryData. Everything else is
   * handled: the queries in `invalidates` are cancelled first so a reply
   * already on its way cannot overwrite this, snapshotted before, and put
   * back exactly as they were if the write fails. So this only ever has to
   * describe the happy path.
   */
  optimistic?: (qc: QueryClient, args: A) => void;
}

/**
 * Every write says what happens if it is sent twice, and there is no third
 * option — which is the point.
 *
 * A queued write can be replayed: the connection drops after the server
 * applied it but before the response arrives, and the queue sends it again.
 * So a write may only be queued when replaying it changes nothing
 * (FR-PWA-009). The alternative is not "queue it and hope": it is to refuse
 * to queue it at all.
 */
type Replay =
  | {
      /**
       * Why sending this twice is the same as sending it once — the natural
       * key, the replace, or the state it settles on. Written out rather than
       * a boolean, because that sentence is where the thinking happens.
       */
      idempotent: string;
      onlineOnly?: never;
    }
  | {
      /**
       * Send now or fail; never pause offline and send later. For a write
       * that would do damage twice (creating a second memory) or whose
       * meaning depends on when it lands (undoing a note the partner has
       * since read).
       */
      onlineOnly: true;
      idempotent?: never;
    };

export type WriteDef<A, R> = BaseWriteDef<A, R> & Replay;

/** Any definition, whatever its argument and result types. */
export type AnyWriteDef = WriteDef<never, unknown>;

export const defineWrite = <A, R>(def: WriteDef<A, R>): WriteDef<A, R> => def;

const invalidate = (qc: QueryClient, keys: readonly QueryKey[]) =>
  Promise.all(keys.map((queryKey) => qc.invalidateQueries({ queryKey })));

export function useWrite<A, R>(def: WriteDef<A, R>) {
  const qc = useQueryClient();
  return useMutation({
    mutationKey: def.mutationKey,
    mutationFn: def.mutationFn,
    scope: def.scope ? { id: def.scope } : undefined,
    meta: def.handlesError ? { handlesError: true } : undefined,
    // "always" skips the offline pause: the request goes out and, with no
    // connection, fails straight away instead of waiting.
    networkMode: def.onlineOnly ? "always" : undefined,
    onMutate: def.optimistic ? (args) => applyOptimistic(qc, def, args) : undefined,
    onError: (_error, _args, rollback) => rollback?.(),
    onSettled: () => invalidate(qc, def.invalidates),
  });
}

/**
 * Applies a write's optimistic change and hands back the undo.
 *
 * The cancel matters more than it looks: without it a refetch already in
 * flight can land after the optimistic change and quietly put the old value
 * back, which reads as the tap having been ignored a moment later.
 */
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
 * Online-only writes are skipped: they never pause, so there is nothing to resume.
 */
export function registerWrites(qc: QueryClient, defs: readonly AnyWriteDef[]) {
  for (const def of defs) {
    if (def.onlineOnly) continue;
    qc.setMutationDefaults(def.mutationKey, {
      // A restored change carries its variables as saved JSON; the definition
      // is what knows their type.
      mutationFn: def.mutationFn as (variables: unknown) => Promise<unknown>,
      scope: def.scope ? { id: def.scope } : undefined,
      onSettled: () => invalidate(qc, def.invalidates),
    });
  }
}

/** Whether a paused write can be sent after a reload, i.e. it was registered. */
export const isResumable = (qc: QueryClient, key: MutationKey | undefined): boolean =>
  key !== undefined && qc.getMutationDefaults(key).mutationFn !== undefined;
