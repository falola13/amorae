"use client";

import { useMutation, useQueryClient, type MutationKey, type QueryClient, type QueryKey } from "@tanstack/react-query";

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
export interface WriteDef<A, R> {
  mutationKey: MutationKey;
  mutationFn: (args: A) => Promise<R>;
  /** Queries that are stale once this succeeds or fails. */
  invalidates: readonly QueryKey[];
  /** Writes sharing a scope are sent one at a time, in order. */
  scope?: string;
  /** The screen shows this write's errors itself, so skip the global toast. */
  handlesError?: boolean;
}

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
    onSettled: () => invalidate(qc, def.invalidates),
  });
}

/** Makes each write resumable after a reload (see lib/query/persist.ts). */
export function registerWrites(qc: QueryClient, defs: readonly AnyWriteDef[]) {
  for (const def of defs) {
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
