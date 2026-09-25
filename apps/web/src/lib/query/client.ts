"use client";

import { MutationCache, QueryClient } from "@tanstack/react-query";

import { isApiError } from "@/lib/api/errors";
import { GENERIC_ERROR_MESSAGE } from "@/lib/api/envelope";
import { notify } from "@/lib/store/toast";

declare module "@tanstack/react-query" {
  interface Register {
    mutationMeta: {
      /** The caller shows this mutation's errors itself (e.g. inline form errors), so skip the toast. */
      handlesError?: boolean;
    };
  }
}

/** Prefer the API's per-field messages over the generic wrapper message
 *  ("Some fields are invalid."); joined, since a form wrong twice should say so once. */
function readableMessage(error: unknown): string {
  if (!isApiError(error)) return GENERIC_ERROR_MESSAGE;
  const fields = Object.values(error.fields ?? {}).filter(Boolean);
  return fields.length > 0 ? fields.join(" ") : error.message;
}

/**
 * Two policies, once, instead of per page:
 * - Failed mutations toast the API's message, unless `meta.handlesError` or a 401 (app shell handles that).
 * - Queries serve cached data offline. Mutations default to "online" mode, so
 *   an offline write is *paused* (optimistic update shows now, sent on reconnect)
 *   and survives a restart (persist.ts) — unless marked `onlineOnly` (mutations.ts), which fails instead.
 */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    mutationCache: new MutationCache({
      onError: (error, _variables, _context, mutation) => {
        if (mutation.meta?.handlesError) return;
        if (isApiError(error) && error.status === 401) return;
        notify(readableMessage(error));
      },
    }),
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: true,
        networkMode: "offlineFirst",
        // Retry server hiccups, never a 4xx: asking again won't change the answer.
        retry: (count, error) => !(isApiError(error) && error.status < 500) && count < 2,
      },
    },
  });
}
